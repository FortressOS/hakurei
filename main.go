package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/linux"
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

var os = new(linux.Std)

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
		w := tabwriter.NewWriter(os.Stdout(), 0, 1, 4, ' ', 0)
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
		if s, err := json.MarshalIndent(fst.Template(), "", "  "); err != nil {
			fmsg.Fatalf("cannot generate template: %v", err)
			panic("unreachable")
		} else {
			fmt.Println(string(s))
		}
		fmsg.Exit(0)
	case "help": // print help message
		flag.CommandLine.Usage()
		fmsg.Exit(0)
	case "ps": // print all state info
		var w *tabwriter.Writer
		state.MustPrintLauncherStateSimpleGlobal(&w, os.Paths().RunDirPath)
		if w != nil {
			if err := w.Flush(); err != nil {
				fmsg.Println("cannot format output:", err)
			}
		} else {
			fmt.Println("No information available")
		}

		fmsg.Exit(0)
	case "show": // pretty-print app info
		if len(args) != 2 {
			fmsg.Fatal("show requires 1 argument")
		}

		likePrefix := false
		if len(args[1]) <= 32 {
			likePrefix = true
			for _, c := range args[1] {
				if c >= '0' && c <= '9' {
					continue
				}
				if c >= 'a' && c <= 'f' {
					continue
				}
				likePrefix = false
				break
			}
		}

		var (
			config   *fst.Config
			instance *state.State
		)

		// try to match from state store
		if likePrefix && len(args[1]) >= 8 {
			fmsg.VPrintln("argument looks like prefix")

			s := state.NewMulti(os.Paths().RunDirPath)
			if entries, err := state.Join(s); err != nil {
				fmsg.Printf("cannot join store: %v", err)
				// drop to fetch from file
			} else {
				for id := range entries {
					v := id.String()
					if strings.HasPrefix(v, args[1]) {
						// match, use config from this state entry
						instance = entries[id]
						config = instance.Config
						break
					}

					fmsg.VPrintf("instance %s skipped", v)
				}
			}
		}

		if config == nil {
			fmsg.VPrintf("reading from file")

			config = new(fst.Config)
			if f, err := os.Open(args[1]); err != nil {
				fmsg.Fatalf("cannot access config file %q: %s", args[1], err)
				panic("unreachable")
			} else if err = json.NewDecoder(f).Decode(&config); err != nil {
				fmsg.Fatalf("cannot parse config file %q: %s", args[1], err)
				panic("unreachable")
			}
		}

		printShow(instance, config)
		fmsg.Exit(0)
	case "app": // launch app from configuration file
		if len(args) < 2 {
			fmsg.Fatal("app requires at least 1 argument")
		}

		config := new(fst.Config)
		if f, err := os.Open(args[1]); err != nil {
			fmsg.Fatalf("cannot access config file %q: %s", args[1], err)
			panic("unreachable")
		} else if err = json.NewDecoder(f).Decode(&config); err != nil {
			fmsg.Fatalf("cannot parse config file %q: %s", args[1], err)
			panic("unreachable")
		}

		// append extra args
		config.Command = append(config.Command, args[2:]...)

		// invoke app
		runApp(config)
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
				if uid, err := os.Uid(aid); err != nil {
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
	default:
		fmsg.Fatalf("%q is not a valid command", args[0])
	}

	panic("unreachable")
}

func runApp(config *fst.Config) {
	a, err := app.New(os)
	if err != nil {
		fmsg.Fatalf("cannot create app: %s\n", err)
	} else if err = a.Seal(config); err != nil {
		logBaseError(err, "cannot seal app:")
		fmsg.Exit(1)
	} else if err = a.Start(); err != nil {
		logBaseError(err, "cannot start app:")
	}

	var r int
	// wait must be called regardless of result of start
	if r, err = a.Wait(); err != nil {
		if r < 1 {
			r = 1
		}
		logWaitError(err)
	}
	if err = a.WaitErr(); err != nil {
		fmsg.Println("inner wait failed:", err)
	}
	fmsg.Exit(r)
	panic("unreachable")
}
