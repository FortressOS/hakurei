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
	"hakurei.app/system/dbus"
)

func buildCommand(ctx context.Context, msg container.Msg, early *earlyHardeningErrs, out io.Writer) command.Command {
	var (
		flagVerbose bool
		flagJSON    bool
	)
	c := command.New(out, log.Printf, "hakurei", func([]string) error {
		msg.SwapVerbose(flagVerbose)

		if early.yamaLSM != nil {
			msg.Verbosef("cannot enable ptrace protection via Yama LSM: %v", early.yamaLSM)
			// not fatal
		}

		if early.dumpable != nil {
			log.Printf("cannot set SUID_DUMP_DISABLE: %s", early.dumpable)
			// not fatal
		}

		return nil
	}).
		Flag(&flagVerbose, "v", command.BoolFlag(false), "Increase log verbosity").
		Flag(&flagJSON, "json", command.BoolFlag(false), "Serialise output in JSON when applicable")

	c.Command("shim", command.UsageInternal, func([]string) error { app.ShimMain(); return errSuccess })

	c.Command("app", "Load app from configuration file", func(args []string) error {
		if len(args) < 1 {
			log.Fatal("app requires at least 1 argument")
		}

		// config extraArgs...
		config := tryPath(msg, args[0])
		config.Args = append(config.Args, args[1:]...)

		app.Main(ctx, msg, config)
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
					us := strconv.Itoa(app.HsuUid(new(app.Hsu).MustIDMsg(msg), flagIdentity))
					if u, err := user.LookupId(us); err != nil {
						msg.Verbosef("cannot look up uid %s", us)
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

			var e hst.Enablement
			if flagWayland {
				e |= hst.EWayland
			}
			if flagX11 {
				e |= hst.EX11
			}
			if flagDBus {
				e |= hst.EDBus
			}
			if flagPulse {
				e |= hst.EPulse
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

			app.Main(ctx, msg, config)
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
				config, entry := tryShort(msg, name)
				if config == nil {
					config = tryPath(msg, name)
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
			var sc hst.Paths
			app.CopyPaths().Copy(&sc, new(app.Hsu).MustID())
			printPs(os.Stdout, time.Now().UTC(), state.NewMulti(msg, sc.RunDirPath.String()), flagShort, flagJSON)
			return errSuccess
		}).Flag(&flagShort, "short", command.BoolFlag(false), "Print instance id")
	}

	c.Command("version", "Display version information", func(args []string) error { fmt.Println(internal.Version()); return errSuccess })
	c.Command("license", "Show full license text", func(args []string) error { fmt.Println(license); return errSuccess })
	c.Command("template", "Produce a config template", func(args []string) error { printJSON(os.Stdout, false, hst.Template()); return errSuccess })
	c.Command("help", "Show this help message", func([]string) error { c.PrintHelp(); return errSuccess })

	return c
}
