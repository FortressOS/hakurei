package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"strconv"
	"sync"
	"time"

	"hakurei.app/command"
	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/app"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/hlog"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
	"hakurei.app/system/dbus"
)

func buildCommand(ctx context.Context, out io.Writer) command.Command {
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

		app.Main(ctx, std, config)
		panic("unreachable")
	})

	{
		var (
			flagDBusConfigSession string
			flagDBusConfigSystem  string
			flagDBusMpris         bool
			flagDBusVerbose       bool

			flagID       string
			flagIdentity int
			flagGroups   command.RepeatableFlag
			flagHomeDir  string
			flagUserName string

			flagWayland, flagX11, flagDBus, flagPulse bool
		)

		c.NewCommand("run", "Configure and start a permissive default sandbox", func(args []string) error {
			// initialise config from flags
			config := &hst.Config{
				ID:   flagID,
				Args: args,
			}

			if flagIdentity < 0 || flagIdentity > 9999 {
				log.Fatalf("identity %d out of range", flagIdentity)
			}

			// resolve home/username from os when flag is unset
			var (
				passwd     *user.User
				passwdOnce sync.Once
				passwdFunc = func() {
					us := strconv.Itoa(sys.MustUid(std, flagIdentity))
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

			if flagHomeDir == "os" {
				passwdOnce.Do(passwdFunc)
				flagHomeDir = passwd.HomeDir
			}

			if flagUserName == "chronos" {
				passwdOnce.Do(passwdFunc)
				flagUserName = passwd.Username
			}

			config.Identity = flagIdentity
			config.Groups = flagGroups
			config.Username = flagUserName

			if a, err := container.NewAbs(flagHomeDir); err != nil {
				log.Fatal(err.Error())
				return err
			} else {
				config.Home = a
			}

			var e system.Enablement
			if flagWayland {
				e |= system.EWayland
			}
			if flagX11 {
				e |= system.EX11
			}
			if flagDBus {
				e |= system.EDBus
			}
			if flagPulse {
				e |= system.EPulse
			}
			config.Enablements = hst.NewEnablements(e)

			// parse D-Bus config file from flags if applicable
			if flagDBus {
				if flagDBusConfigSession == "builtin" {
					config.SessionBus = dbus.NewConfig(flagID, true, flagDBusMpris)
				} else {
					if conf, err := dbus.NewConfigFromFile(flagDBusConfigSession); err != nil {
						log.Fatalf("cannot load session bus proxy config from %q: %s", flagDBusConfigSession, err)
					} else {
						config.SessionBus = conf
					}
				}

				// system bus proxy is optional
				if flagDBusConfigSystem != "nil" {
					if conf, err := dbus.NewConfigFromFile(flagDBusConfigSystem); err != nil {
						log.Fatalf("cannot load system bus proxy config from %q: %s", flagDBusConfigSystem, err)
					} else {
						config.SystemBus = conf
					}
				}

				// override log from configuration
				if flagDBusVerbose {
					if config.SessionBus != nil {
						config.SessionBus.Log = true
					}
					if config.SystemBus != nil {
						config.SystemBus.Log = true
					}
				}
			}

			app.Main(ctx, std, config)
			panic("unreachable")
		}).
			Flag(&flagDBusConfigSession, "dbus-config", command.StringFlag("builtin"),
				"Path to session bus proxy config file, or \"builtin\" for defaults").
			Flag(&flagDBusConfigSystem, "dbus-system", command.StringFlag("nil"),
				"Path to system bus proxy config file, or \"nil\" to disable").
			Flag(&flagDBusMpris, "mpris", command.BoolFlag(false),
				"Allow owning MPRIS D-Bus path, has no effect if custom config is available").
			Flag(&flagDBusVerbose, "dbus-log", command.BoolFlag(false),
				"Force buffered logging in the D-Bus proxy").
			Flag(&flagID, "id", command.StringFlag(""),
				"Reverse-DNS style Application identifier, leave empty to inherit instance identifier").
			Flag(&flagIdentity, "a", command.IntFlag(0),
				"Application identity").
			Flag(nil, "g", &flagGroups,
				"Groups inherited by all container processes").
			Flag(&flagHomeDir, "d", command.StringFlag("os"),
				"Container home directory").
			Flag(&flagUserName, "u", command.StringFlag("chronos"),
				"Passwd user name within sandbox").
			Flag(&flagWayland, "wayland", command.BoolFlag(false),
				"Enable connection to Wayland via security-context-v1").
			Flag(&flagX11, "X", command.BoolFlag(false),
				"Enable direct connection to X11").
			Flag(&flagDBus, "dbus", command.BoolFlag(false),
				"Enable proxied connection to D-Bus").
			Flag(&flagPulse, "pulse", command.BoolFlag(false),
				"Enable direct connection to PulseAudio")
	}

	{
		var flagShort bool
		c.NewCommand("show", "Show live or local app configuration", func(args []string) error {
			switch len(args) {
			case 0: // system
				printShowSystem(os.Stdout, flagShort, flagJSON)

			case 1: // instance
				name := args[0]
				config, entry := tryShort(name)
				if config == nil {
					config = tryPath(name)
				}
				printShowInstance(os.Stdout, time.Now().UTC(), entry, config, flagShort, flagJSON)

			default:
				log.Fatal("show requires 1 argument")
			}
			return errSuccess
		}).Flag(&flagShort, "short", command.BoolFlag(false), "Omit filesystem information")
	}

	{
		var flagShort bool
		c.NewCommand("ps", "List active instances", func(args []string) error {
			printPs(os.Stdout, time.Now().UTC(), state.NewMulti(std.Paths().RunDirPath.String()), flagShort, flagJSON)
			return errSuccess
		}).Flag(&flagShort, "short", command.BoolFlag(false), "Print instance id")
	}

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
