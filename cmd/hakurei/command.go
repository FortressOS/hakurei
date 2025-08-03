package main

import (
	"context"
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

	"hakurei.app/command"
	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/app"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/hlog"
	"hakurei.app/system"
	"hakurei.app/system/dbus"
)

func buildCommand(out io.Writer) command.Command {
	var (
		flagVerbose bool
		flagJSON    bool
	)
	c := command.New(out, log.Printf, "hakurei", func([]string) error { internal.InstallOutput(flagVerbose); return nil }).
		Flag(&flagVerbose, "v", command.BoolFlag(false), "Increase log verbosity").
		Flag(&flagJSON, "json", command.BoolFlag(false), "Serialise output in JSON when applicable")

	c.Command("shim", command.UsageInternal, func([]string) error { app.ShimMain(); return errSuccess })

	c.Command("app", "Load app from configuration file", func(args []string) error {
		if len(args) < 1 {
			log.Fatal("app requires at least 1 argument")
		}

		// config extraArgs...
		config := tryPath(args[0])
		config.Args = append(config.Args, args[1:]...)

		runApp(config)
		panic("unreachable")
	})

	{
		var (
			dbusConfigSession string
			dbusConfigSystem  string
			mpris             bool
			dbusVerbose       bool

			fid      string
			aid      int
			groups   command.RepeatableFlag
			homeDir  string
			userName string

			wayland, x11, dBus, pulse bool
		)

		c.NewCommand("run", "Configure and start a permissive default sandbox", func(args []string) error {
			// initialise config from flags
			config := &hst.Config{
				ID:   fid,
				Args: args,
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
						hlog.PrintBaseError(err, "cannot obtain uid from setuid wrapper:")
						os.Exit(1)
					} else {
						us = strconv.Itoa(uid)
					}

					if u, err := user.LookupId(us); err != nil {
						hlog.Verbosef("cannot look up uid %s", us)
						passwd = &user.User{
							Uid:      us,
							Gid:      us,
							Username: "chronos",
							Name:     "Hakurei Permissive Default",
							HomeDir:  container.FHSVarEmpty,
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

			config.Identity = aid
			config.Groups = groups
			config.Data = homeDir
			config.Username = userName

			if wayland {
				config.Enablements |= system.EWayland
			}
			if x11 {
				config.Enablements |= system.EX11
			}
			if dBus {
				config.Enablements |= system.EDBus
			}
			if pulse {
				config.Enablements |= system.EPulse
			}

			// parse D-Bus config file from flags if applicable
			if dBus {
				if dbusConfigSession == "builtin" {
					config.SessionBus = dbus.NewConfig(fid, true, mpris)
				} else {
					if conf, err := dbus.NewConfigFromFile(dbusConfigSession); err != nil {
						log.Fatalf("cannot load session bus proxy config from %q: %s", dbusConfigSession, err)
					} else {
						config.SessionBus = conf
					}
				}

				// system bus proxy is optional
				if dbusConfigSystem != "nil" {
					if conf, err := dbus.NewConfigFromFile(dbusConfigSystem); err != nil {
						log.Fatalf("cannot load system bus proxy config from %q: %s", dbusConfigSystem, err)
					} else {
						config.SystemBus = conf
					}
				}

				// override log from configuration
				if dbusVerbose {
					config.SessionBus.Log = true
					config.SystemBus.Log = true
				}
			}

			// invoke app
			runApp(config)
			panic("unreachable")
		}).
			Flag(&dbusConfigSession, "dbus-config", command.StringFlag("builtin"),
				"Path to session bus proxy config file, or \"builtin\" for defaults").
			Flag(&dbusConfigSystem, "dbus-system", command.StringFlag("nil"),
				"Path to system bus proxy config file, or \"nil\" to disable").
			Flag(&mpris, "mpris", command.BoolFlag(false),
				"Allow owning MPRIS D-Bus path, has no effect if custom config is available").
			Flag(&dbusVerbose, "dbus-log", command.BoolFlag(false),
				"Force buffered logging in the D-Bus proxy").
			Flag(&fid, "id", command.StringFlag(""),
				"Reverse-DNS style Application identifier, leave empty to inherit instance identifier").
			Flag(&aid, "a", command.IntFlag(0),
				"Application identity").
			Flag(nil, "g", &groups,
				"Groups inherited by all container processes").
			Flag(&homeDir, "d", command.StringFlag("os"),
				"Container home directory").
			Flag(&userName, "u", command.StringFlag("chronos"),
				"Passwd user name within sandbox").
			Flag(&wayland, "wayland", command.BoolFlag(false),
				"Enable connection to Wayland via security-context-v1").
			Flag(&x11, "X", command.BoolFlag(false),
				"Enable direct connection to X11").
			Flag(&dBus, "dbus", command.BoolFlag(false),
				"Enable proxied connection to D-Bus").
			Flag(&pulse, "pulse", command.BoolFlag(false),
				"Enable direct connection to PulseAudio")
	}

	var showFlagShort bool
	c.NewCommand("show", "Show live or local app configuration", func(args []string) error {
		switch len(args) {
		case 0: // system
			printShowSystem(os.Stdout, showFlagShort, flagJSON)

		case 1: // instance
			name := args[0]
			config, entry := tryShort(name)
			if config == nil {
				config = tryPath(name)
			}
			printShowInstance(os.Stdout, time.Now().UTC(), entry, config, showFlagShort, flagJSON)

		default:
			log.Fatal("show requires 1 argument")
		}
		return errSuccess
	}).Flag(&showFlagShort, "short", command.BoolFlag(false), "Omit filesystem information")

	var psFlagShort bool
	c.NewCommand("ps", "List active instances", func(args []string) error {
		printPs(os.Stdout, time.Now().UTC(), state.NewMulti(std.Paths().RunDirPath), psFlagShort, flagJSON)
		return errSuccess
	}).Flag(&psFlagShort, "short", command.BoolFlag(false), "Print instance id")

	c.Command("version", "Display version information", func(args []string) error {
		fmt.Println(internal.Version())
		return errSuccess
	})

	c.Command("license", "Show full license text", func(args []string) error {
		fmt.Println(license)
		return errSuccess
	})

	c.Command("template", "Produce a config template", func(args []string) error {
		printJSON(os.Stdout, false, hst.Template())
		return errSuccess
	})

	c.Command("help", "Show this help message", func([]string) error {
		c.PrintHelp()
		return errSuccess
	})

	return c
}

func runApp(config *hst.Config) {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop() // unreachable
	a := app.MustNew(ctx, std)

	rs := new(app.RunState)
	if sa, err := a.Seal(config); err != nil {
		hlog.PrintBaseError(err, "cannot seal app:")
		internal.Exit(1)
	} else {
		internal.Exit(app.PrintRunStateErr(rs, sa.Run(rs)))
	}

	*(*int)(nil) = 0 // not reached
}
