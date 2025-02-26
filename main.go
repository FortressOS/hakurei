package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"sync"
	"syscall"
	"time"

	"git.gensokyo.uk/security/fortify/command"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/seccomp"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/app"
	init0 "git.gensokyo.uk/security/fortify/internal/app/init"
	"git.gensokyo.uk/security/fortify/internal/app/shim"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
	"git.gensokyo.uk/security/fortify/internal/sys"
	"git.gensokyo.uk/security/fortify/system"
)

var (
	errSuccess = errors.New("success")

	//go:embed LICENSE
	license string
)

func init() { fmsg.Prepare("fortify") }

var std sys.State = new(sys.Std)

func main() {
	// early init argv0 check, skips root check and duplicate PR_SET_DUMPABLE
	init0.TryArgv0()

	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		log.Printf("cannot set SUID_DUMP_DISABLE: %s", err)
		// not fatal: this program runs as the privileged user
	}

	if os.Geteuid() == 0 {
		log.Fatal("this program must not run as root")
	}

	buildCommand(os.Stderr).MustParse(os.Args[1:], func(err error) {
		fmsg.Verbosef("command returned %v", err)
		if errors.Is(err, errSuccess) {
			fmsg.BeforeExit()
			os.Exit(0)
		}
	})
	log.Fatal("unreachable")
}

func buildCommand(out io.Writer) command.Command {
	var (
		flagVerbose bool
		flagJSON    bool
	)
	c := command.New(out, log.Printf, "fortify", func([]string) error {
		fmsg.Store(flagVerbose)
		if flagVerbose {
			seccomp.CPrintln = log.Println
		}
		return nil
	}).
		Flag(&flagVerbose, "v", command.BoolFlag(false), "Print debug messages to the console").
		Flag(&flagJSON, "json", command.BoolFlag(false), "Serialise output as JSON when applicable")

	c.Command("app", "Launch app defined by the specified config file", func(args []string) error {
		if len(args) < 1 {
			log.Fatal("app requires at least 1 argument")
		}

		// config extraArgs...
		config := tryPath(args[0])
		config.Command = append(config.Command, args[1:]...)

		// invoke app
		runApp(app.MustNew(std), config)
		panic("unreachable")
	})

	{
		var (
			dbusConfigSession string
			dbusConfigSystem  string
			mpris             bool
			dbusVerbose       bool

			fid         string
			aid         int
			groups      command.RepeatableFlag
			homeDir     string
			userName    string
			enablements [system.ELen]bool
		)

		c.NewCommand("run", "Configure and start a permissive default sandbox", func(args []string) error {
			// initialise config from flags
			config := &fst.Config{
				ID:      fid,
				Command: args,
			}

			if aid < 0 || aid > 9999 {
				log.Fatalf("aid %d out of range", aid)
			}

			// resolve home/username from os when flag is unset
			var (
				passwd     *user.User
				passwdOnce sync.Once
				passwdFunc = func() {
					var us string
					if uid, err := std.Uid(aid); err != nil {
						fmsg.PrintBaseError(err, "cannot obtain uid from fsu:")
						os.Exit(1)
					} else {
						us = strconv.Itoa(uid)
					}

					if u, err := user.LookupId(us); err != nil {
						fmsg.Verbosef("cannot look up uid %s", us)
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
					if conf, err := dbus.NewConfigFromFile(dbusConfigSession); err != nil {
						log.Fatalf("cannot load session bus proxy config from %q: %s", dbusConfigSession, err)
					} else {
						config.Confinement.SessionBus = conf
					}
				}

				// system bus proxy is optional
				if dbusConfigSystem != "nil" {
					if conf, err := dbus.NewConfigFromFile(dbusConfigSystem); err != nil {
						log.Fatalf("cannot load system bus proxy config from %q: %s", dbusConfigSystem, err)
					} else {
						config.Confinement.SystemBus = conf
					}
				}

				// override log from configuration
				if dbusVerbose {
					config.Confinement.SessionBus.Log = true
					config.Confinement.SystemBus.Log = true
				}
			}

			// invoke app
			runApp(app.MustNew(std), config)
			panic("unreachable")
		}).
			Flag(&dbusConfigSession, "dbus-config", command.StringFlag("builtin"),
				"Path to D-Bus proxy config file, or \"builtin\" for defaults").
			Flag(&dbusConfigSystem, "dbus-system", command.StringFlag("nil"),
				"Path to system D-Bus proxy config file, or \"nil\" to disable").
			Flag(&mpris, "mpris", command.BoolFlag(false),
				"Allow owning MPRIS D-Bus path, has no effect if custom config is available").
			Flag(&dbusVerbose, "dbus-log", command.BoolFlag(false),
				"Force logging in the D-Bus proxy").
			Flag(&fid, "id", command.StringFlag(""),
				"App ID, leave empty to disable security context app_id").
			Flag(&aid, "a", command.IntFlag(0),
				"Fortify application ID").
			Flag(nil, "g", &groups,
				"Groups inherited by the app process").
			Flag(&homeDir, "d", command.StringFlag("os"),
				"Application home directory").
			Flag(&userName, "u", command.StringFlag("chronos"),
				"Passwd name within sandbox").
			Flag(&enablements[system.EWayland], "wayland", command.BoolFlag(false),
				"Allow Wayland connections").
			Flag(&enablements[system.EX11], "X", command.BoolFlag(false),
				"Share X11 socket and allow connection").
			Flag(&enablements[system.EDBus], "dbus", command.BoolFlag(false),
				"Proxy D-Bus connection").
			Flag(&enablements[system.EPulse], "pulse", command.BoolFlag(false),
				"Share PulseAudio socket and cookie")
	}

	var showFlagShort bool
	c.NewCommand("show", "Show the contents of an app configuration", func(args []string) error {
		switch len(args) {
		case 0: // system
			printShowSystem(os.Stdout, showFlagShort, flagJSON)

		case 1: // instance
			name := args[0]
			config, instance := tryShort(name)
			if config == nil {
				config = tryPath(name)
			}
			printShowInstance(os.Stdout, time.Now().UTC(), instance, config, showFlagShort, flagJSON)

		default:
			log.Fatal("show requires 1 argument")
		}
		return errSuccess
	}).Flag(&showFlagShort, "short", command.BoolFlag(false), "Omit filesystem information")

	var psFlagShort bool
	c.NewCommand("ps", "List active apps and their state", func(args []string) error {
		printPs(os.Stdout, time.Now().UTC(), state.NewMulti(std.Paths().RunDirPath), psFlagShort, flagJSON)
		return errSuccess
	}).Flag(&psFlagShort, "short", command.BoolFlag(false), "Print instance id")

	c.Command("version", "Show fortify version", func(args []string) error {
		if v, ok := internal.Check(internal.Version); ok {
			fmt.Println(v)
		} else {
			fmt.Println("impure")
		}
		return errSuccess
	})

	c.Command("license", "Show full license text", func(args []string) error {
		fmt.Println(license)
		return errSuccess
	})

	c.Command("template", "Produce a config template", func(args []string) error {
		printJSON(os.Stdout, false, fst.Template())
		return errSuccess
	})

	c.Command("help", "Show this help message", func([]string) error {
		c.PrintHelp()
		return errSuccess
	})

	// internal commands
	c.Command("shim", command.UsageInternal, func([]string) error { shim.Main(); return errSuccess })
	c.Command("init", command.UsageInternal, func([]string) error { init0.Main(); return errSuccess })

	return c
}

func runApp(a fst.App, config *fst.Config) {
	rs := new(fst.RunState)
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop() // unreachable

	if sa, err := a.Seal(config); err != nil {
		fmsg.PrintBaseError(err, "cannot seal app:")
		internal.Exit(1)
	} else if err = sa.Run(ctx, rs); err != nil {
		if rs.Time == nil {
			fmsg.PrintBaseError(err, "cannot start app:")
		} else {
			logWaitError(err)
		}

		if rs.ExitCode == 0 {
			rs.ExitCode = 126
		}
	}
	if rs.RevertErr != nil {
		var stateStoreError *app.StateStoreError
		if !errors.As(rs.RevertErr, &stateStoreError) || stateStoreError == nil {
			fmsg.PrintBaseError(rs.RevertErr, "generic fault during cleanup:")
			goto out
		}

		if stateStoreError.Err != nil {
			if len(stateStoreError.Err) == 2 {
				if stateStoreError.Err[0] != nil {
					if joinedErrs, ok := stateStoreError.Err[0].(interface{ Unwrap() []error }); !ok {
						fmsg.PrintBaseError(stateStoreError.Err[0], "generic fault during revert:")
					} else {
						for _, err := range joinedErrs.Unwrap() {
							if err != nil {
								fmsg.PrintBaseError(err, "fault during revert:")
							}
						}
					}
				}
				if stateStoreError.Err[1] != nil {
					log.Printf("cannot close store: %v", stateStoreError.Err[1])
				}
			} else {
				log.Printf("fault during cleanup: %v",
					errors.Join(stateStoreError.Err...))
			}
		}

		if stateStoreError.OpErr != nil {
			log.Printf("blind revert due to store fault: %v",
				stateStoreError.OpErr)
		}

		if stateStoreError.DoErr != nil {
			fmsg.PrintBaseError(stateStoreError.DoErr, "state store operation unsuccessful:")
		}

		if stateStoreError.Inner && stateStoreError.InnerErr != nil {
			fmsg.PrintBaseError(stateStoreError.InnerErr, "cannot destroy state entry:")
		}

	out:
		if rs.ExitCode == 0 {
			rs.ExitCode = 128
		}
	}
	if rs.WaitErr != nil {
		log.Println("inner wait failed:", rs.WaitErr)
	}
	internal.Exit(rs.ExitCode)
}
