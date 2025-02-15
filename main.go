package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/seccomp"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/linux"
	init0 "git.gensokyo.uk/security/fortify/internal/priv/init"
	"git.gensokyo.uk/security/fortify/internal/priv/shim"
	"git.gensokyo.uk/security/fortify/internal/state"
	"git.gensokyo.uk/security/fortify/internal/system"
)

var (
	flagVerbose bool
	flagJSON    bool

	//go:embed LICENSE
	license string
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
	flag.BoolVar(&flagJSON, "json", false, "Format output in JSON when applicable")
}

var sys linux.System = new(linux.Std)

type gl []string

func (g *gl) String() string {
	if g == nil {
		return "<nil>"
	}
	return strings.Join(*g, " ")
}

func (g *gl) Set(v string) error {
	*g = append(*g, v)
	return nil
}

func main() {
	// early init argv0 check, skips root check and duplicate PR_SET_DUMPABLE
	init0.TryArgv0()

	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		fmsg.Printf("cannot set SUID_DUMP_DISABLE: %s", err)
		// not fatal: this program runs as the privileged user
	}

	if os.Geteuid() == 0 {
		fmsg.Fatal("this program must not run as root")
		panic("unreachable")
	}

	flag.CommandLine.Usage = func() {
		fmt.Println()
		fmt.Println("Usage:\tfortify [-v] [--json] COMMAND [OPTIONS]")
		fmt.Println()
		fmt.Println("Commands:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		commands := [][2]string{
			{"app", "Launch app defined by the specified config file"},
			{"run", "Configure and start a permissive default sandbox"},
			{"show", "Show the contents of an app configuration"},
			{"ps", "List active apps and their state"},
			{"version", "Show fortify version"},
			{"license", "Show full license text"},
			{"template", "Produce a config template"},
			{"help", "Show this help message"},
		}
		for _, c := range commands {
			_, _ = fmt.Fprintf(w, "\t%s\t%s\n", c[0], c[1])
		}
		if err := w.Flush(); err != nil {
			fmt.Printf("fortify: cannot write command list: %v\n", err)
		}
		fmt.Println()
	}
	flag.Parse()
	fmsg.SetVerbose(flagVerbose)

	args := flag.Args()
	if len(args) == 0 {
		flag.CommandLine.Usage()
		fmsg.Exit(0)
	}

	switch args[0] {
	case "version": // print version string
		if v, ok := internal.Check(internal.Version); ok {
			fmt.Println(v)
		} else {
			fmt.Println("impure")
		}
		fmsg.Exit(0)
	case "license": // print embedded license
		fmt.Println(license)
		fmsg.Exit(0)
	case "template": // print full template configuration
		printJSON(os.Stdout, false, fst.Template())
		fmsg.Exit(0)
	case "help": // print help message
		flag.CommandLine.Usage()
		fmsg.Exit(0)
	case "ps": // print all state info
		set := flag.NewFlagSet("ps", flag.ExitOnError)
		var short bool
		set.BoolVar(&short, "short", false, "Print instance id")

		// Ignore errors; set is set for ExitOnError.
		_ = set.Parse(args[1:])

		printPs(os.Stdout, time.Now().UTC(), state.NewMulti(sys.Paths().RunDirPath), short)
		fmsg.Exit(0)
	case "show": // pretty-print app info
		set := flag.NewFlagSet("show", flag.ExitOnError)
		var short bool
		set.BoolVar(&short, "short", false, "Omit filesystem information")

		// Ignore errors; set is set for ExitOnError.
		_ = set.Parse(args[1:])

		switch len(set.Args()) {
		case 0: // system
			printShowSystem(os.Stdout, short)
		case 1: // instance
			name := set.Args()[0]
			config, instance := tryShort(name)
			if config == nil {
				config = tryPath(name)
			}
			printShowInstance(os.Stdout, time.Now().UTC(), instance, config, short)
		default:
			fmsg.Fatal("show requires 1 argument")
		}

		fmsg.Exit(0)
	case "app": // launch app from configuration file
		if len(args) < 2 {
			fmsg.Fatal("app requires at least 1 argument")
		}

		// config extraArgs...
		config := tryPath(args[1])
		config.Command = append(config.Command, args[2:]...)

		// invoke app
		runApp(config)
		panic("unreachable")
	case "run": // run app in permissive defaults usage pattern
		set := flag.NewFlagSet("run", flag.ExitOnError)

		var (
			dbusConfigSession string
			dbusConfigSystem  string
			mpris             bool
			dbusVerbose       bool

			fid         string
			aid         int
			groups      gl
			homeDir     string
			userName    string
			enablements [system.ELen]bool
		)

		set.StringVar(&dbusConfigSession, "dbus-config", "builtin", "Path to D-Bus proxy config file, or \"builtin\" for defaults")
		set.StringVar(&dbusConfigSystem, "dbus-system", "nil", "Path to system D-Bus proxy config file, or \"nil\" to disable")
		set.BoolVar(&mpris, "mpris", false, "Allow owning MPRIS D-Bus path, has no effect if custom config is available")
		set.BoolVar(&dbusVerbose, "dbus-log", false, "Force logging in the D-Bus proxy")

		set.StringVar(&fid, "id", "", "App ID, leave empty to disable security context app_id")
		set.IntVar(&aid, "a", 0, "Fortify application ID")
		set.Var(&groups, "g", "Groups inherited by the app process")
		set.StringVar(&homeDir, "d", "os", "Application home directory")
		set.StringVar(&userName, "u", "chronos", "Passwd name within sandbox")
		set.BoolVar(&enablements[system.EWayland], "wayland", false, "Allow Wayland connections")
		set.BoolVar(&enablements[system.EX11], "X", false, "Share X11 socket and allow connection")
		set.BoolVar(&enablements[system.EDBus], "dbus", false, "Proxy D-Bus connection")
		set.BoolVar(&enablements[system.EPulse], "pulse", false, "Share PulseAudio socket and cookie")

		// Ignore errors; set is set for ExitOnError.
		_ = set.Parse(args[1:])

		// initialise config from flags
		config := &fst.Config{
			ID:      fid,
			Command: set.Args(),
		}

		if aid < 0 || aid > 9999 {
			fmsg.Fatalf("aid %d out of range", aid)
			panic("unreachable")
		}

		// resolve home/username from os when flag is unset
		var (
			passwd     *user.User
			passwdOnce sync.Once
			passwdFunc = func() {
				var us string
				if uid, err := sys.Uid(aid); err != nil {
					fmsg.Fatalf("cannot obtain uid from fsu: %v", err)
				} else {
					us = strconv.Itoa(uid)
				}

				if u, err := user.LookupId(us); err != nil {
					fmsg.VPrintf("cannot look up uid %s", us)
					passwd = &user.User{
						Uid:      us,
						Gid:      us,
						Username: "chronos",
						Name:     "Fortify",
						HomeDir:  "/var/empty",
					}
				} else {
					passwd = u
				}
			}
		)

		if homeDir == "os" {
			passwdOnce.Do(passwdFunc)
			homeDir = passwd.HomeDir
		}

		if userName == "chronos" {
			passwdOnce.Do(passwdFunc)
			userName = passwd.Username
		}

		config.Confinement.AppID = aid
		config.Confinement.Groups = groups
		config.Confinement.Outer = homeDir
		config.Confinement.Username = userName

		// enablements from flags
		for i := system.Enablement(0); i < system.Enablement(system.ELen); i++ {
			if enablements[i] {
				config.Confinement.Enablements.Set(i)
			}
		}

		// parse D-Bus config file from flags if applicable
		if enablements[system.EDBus] {
			if dbusConfigSession == "builtin" {
				config.Confinement.SessionBus = dbus.NewConfig(fid, true, mpris)
			} else {
				if c, err := dbus.NewConfigFromFile(dbusConfigSession); err != nil {
					fmsg.Fatalf("cannot load session bus proxy config from %q: %s", dbusConfigSession, err)
				} else {
					config.Confinement.SessionBus = c
				}
			}

			// system bus proxy is optional
			if dbusConfigSystem != "nil" {
				if c, err := dbus.NewConfigFromFile(dbusConfigSystem); err != nil {
					fmsg.Fatalf("cannot load system bus proxy config from %q: %s", dbusConfigSystem, err)
				} else {
					config.Confinement.SystemBus = c
				}
			}

			// override log from configuration
			if dbusVerbose {
				config.Confinement.SessionBus.Log = true
				config.Confinement.SystemBus.Log = true
			}
		}

		// invoke app
		runApp(config)

	// internal commands
	case "shim":
		shim.Main()
		fmsg.Exit(0)
	case "init":
		init0.Main()
		fmsg.Exit(0)

	default:
		fmsg.Fatalf("%q is not a valid command", args[0])
	}

	panic("unreachable")
}

func runApp(config *fst.Config) {
	rs := new(app.RunState)
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop() // unreachable

	if fmsg.Verbose() {
		seccomp.CPrintln = fmsg.Println
	}

	if a, err := app.New(sys); err != nil {
		fmsg.Fatalf("cannot create app: %s\n", err)
	} else if err = a.Seal(config); err != nil {
		logBaseError(err, "cannot seal app:")
		fmsg.Exit(1)
	} else if err = a.Run(ctx, rs); err != nil {
		if !rs.Start {
			logBaseError(err, "cannot start app:")
		} else {
			logWaitError(err)
		}

		if rs.ExitCode == 0 {
			rs.ExitCode = 126
		}
	}
	if rs.WaitErr != nil {
		fmsg.Println("inner wait failed:", rs.WaitErr)
	}
	fmsg.Exit(rs.ExitCode)
	panic("unreachable")
}
