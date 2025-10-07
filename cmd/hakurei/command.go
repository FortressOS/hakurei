package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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

	c.Command("app", "Load and start container from configuration file", func(args []string) error {
		if len(args) < 1 {
			log.Fatal("app requires at least 1 argument")
		}

		// config extraArgs...
		config := tryPath(msg, args[0])
		if config != nil && config.Container != nil {
			config.Container.Args = append(config.Container.Args, args[1:]...)
		}

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

		c.NewCommand("run", "Configure and start a permissive container", func(args []string) error {
			if flagIdentity < hst.IdentityMin || flagIdentity > hst.IdentityMax {
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

			// paths are identical, resolve inner shell and program path
			shell := container.AbsFHSRoot.Append("bin", "sh")
			if a, err := container.NewAbs(os.Getenv("SHELL")); err == nil {
				shell = a
			}
			progPath := shell
			if len(args) > 0 {
				if p, err := exec.LookPath(args[0]); err != nil {
					log.Fatal(errors.Unwrap(err))
					return err
				} else if progPath, err = container.NewAbs(p); err != nil {
					log.Fatal(err.Error())
					return err
				}
			}

			var et hst.Enablement
			if flagWayland {
				et |= hst.EWayland
			}
			if flagX11 {
				et |= hst.EX11
			}
			if flagDBus {
				et |= hst.EDBus
			}
			if flagPulse {
				et |= hst.EPulse
			}

			config := &hst.Config{
				ID:          flagID,
				Identity:    flagIdentity,
				Groups:      flagGroups,
				Enablements: hst.NewEnablements(et),

				Container: &hst.ContainerConfig{
					Userns:       true,
					HostNet:      true,
					Tty:          true,
					HostAbstract: true,

					Filesystem: []hst.FilesystemConfigJSON{
						// autoroot, includes the home directory
						{FilesystemConfig: &hst.FSBind{
							Target:  container.AbsFHSRoot,
							Source:  container.AbsFHSRoot,
							Write:   true,
							Special: true,
						}},
					},

					Username: flagUserName,
					Shell:    shell,

					Path: progPath,
					Args: args,
				},
			}

			// bind GPU stuff
			if et&(hst.EX11|hst.EWayland) != 0 {
				config.Container.Filesystem = append(config.Container.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{
					Source:   container.AbsFHSDev.Append("dri"),
					Device:   true,
					Optional: true,
				}})
			}

			config.Container.Filesystem = append(config.Container.Filesystem,
				// opportunistically bind kvm
				hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{
					Source:   container.AbsFHSDev.Append("kvm"),
					Device:   true,
					Optional: true,
				}},

				// do autoetc last
				hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{
					Target:  container.AbsFHSEtc,
					Source:  container.AbsFHSEtc,
					Special: true,
				}},
			)

			if config.Container.Username == "chronos" {
				passwdOnce.Do(passwdFunc)
				config.Container.Username = passwd.Username
			}

			{
				homeDir := flagHomeDir
				if homeDir == "os" {
					passwdOnce.Do(passwdFunc)
					homeDir = passwd.HomeDir
				}
				if a, err := container.NewAbs(homeDir); err != nil {
					log.Fatal(err.Error())
					return err
				} else {
					config.Container.Home = a
				}
			}

			// parse D-Bus config file from flags if applicable
			if flagDBus {
				if flagDBusConfigSession == "builtin" {
					config.SessionBus = dbus.NewConfig(flagID, true, flagDBusMpris)
				} else {
					if f, err := os.Open(flagDBusConfigSession); err != nil {
						log.Fatal(err.Error())
					} else if err = json.NewDecoder(f).Decode(&config.SessionBus); err != nil {
						log.Fatalf("cannot load session bus proxy config from %q: %s", flagDBusConfigSession, err)
					}
				}

				// system bus proxy is optional
				if flagDBusConfigSystem != "nil" {
					if f, err := os.Open(flagDBusConfigSystem); err != nil {
						log.Fatal(err.Error())
					} else if err = json.NewDecoder(f).Decode(&config.SystemBus); err != nil {
						log.Fatalf("cannot load system bus proxy config from %q: %s", flagDBusConfigSystem, err)
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
				if !printShowInstance(os.Stdout, time.Now().UTC(), entry, config, flagShort, flagJSON) {
					os.Exit(1)
				}

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
