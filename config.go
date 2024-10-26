package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"git.ophivana.moe/security/fortify/dbus"
	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/app"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	"git.ophivana.moe/security/fortify/internal/system"
)

var (
	printTemplate bool

	confPath string

	dbusConfigSession string
	dbusConfigSystem  string
	dbusID            string
	mpris             bool
	dbusVerbose       bool

	userName    string
	enablements [system.ELen]bool

	launchMethodText string
)

func init() {
	flag.BoolVar(&printTemplate, "template", false, "Print a full config template and exit")

	// config file, disables every other flag here
	flag.StringVar(&confPath, "c", "nil", "Path to full app configuration, or \"nil\" to configure from flags")

	flag.StringVar(&dbusConfigSession, "dbus-config", "builtin", "Path to D-Bus proxy config file, or \"builtin\" for defaults")
	flag.StringVar(&dbusConfigSystem, "dbus-system", "nil", "Path to system D-Bus proxy config file, or \"nil\" to disable")
	flag.StringVar(&dbusID, "dbus-id", "", "D-Bus ID of application, leave empty to disable own paths, has no effect if custom config is available")
	flag.BoolVar(&mpris, "mpris", false, "Allow owning MPRIS D-Bus path, has no effect if custom config is available")
	flag.BoolVar(&dbusVerbose, "dbus-log", false, "Force logging in the D-Bus proxy")

	flag.StringVar(&userName, "u", "chronos", "Passwd name of user to run as")
	flag.BoolVar(&enablements[system.EWayland], "wayland", false, "Share Wayland socket")
	flag.BoolVar(&enablements[system.EX11], "X", false, "Share X11 socket and allow connection")
	flag.BoolVar(&enablements[system.EDBus], "dbus", false, "Proxy D-Bus connection")
	flag.BoolVar(&enablements[system.EPulse], "pulse", false, "Share PulseAudio socket and cookie")
}

func init() {
	methodHelpString := "Method of launching the child process, can be one of \"sudo\""
	if internal.SdBootedV {
		methodHelpString += ", \"systemd\""
	}

	flag.StringVar(&launchMethodText, "method", "sudo", methodHelpString)
}

func tryTemplate() {
	if printTemplate {
		if s, err := json.MarshalIndent(app.Template(), "", "  "); err != nil {
			fmsg.Fatalf("cannot generate template: %v", err)
			panic("unreachable")
		} else {
			fmt.Println(string(s))
		}
		fmsg.Exit(0)
	}
}

func loadConfig() *app.Config {
	if confPath == "nil" {
		// config from flags
		return configFromFlags()
	} else {
		// config from file
		c := new(app.Config)
		if f, err := os.Open(confPath); err != nil {
			fmsg.Fatalf("cannot access config file %q: %s", confPath, err)
			panic("unreachable")
		} else if err = json.NewDecoder(f).Decode(&c); err != nil {
			fmsg.Fatalf("cannot parse config file %q: %s", confPath, err)
			panic("unreachable")
		} else {
			return c
		}
	}
}

func configFromFlags() (config *app.Config) {
	// initialise config from flags
	config = &app.Config{
		ID:      dbusID,
		User:    userName,
		Command: flag.Args(),
		Method:  launchMethodText,
	}

	// enablements from flags
	for i := system.Enablement(0); i < system.Enablement(system.ELen); i++ {
		if enablements[i] {
			config.Confinement.Enablements.Set(i)
		}
	}

	// parse D-Bus config file from flags if applicable
	if enablements[system.EDBus] {
		if dbusConfigSession == "builtin" {
			config.Confinement.SessionBus = dbus.NewConfig(dbusID, true, mpris)
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

	return
}
