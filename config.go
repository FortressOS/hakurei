package main

import (
	"encoding/json"
	"flag"
	"os"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/app"
	"git.ophivana.moe/cat/fortify/internal/state"
)

var (
	confPath string

	dbusConfigSession string
	dbusConfigSystem  string
	dbusID            string
	mpris             bool
	dbusVerbose       bool

	userName    string
	enablements [state.EnableLength]bool

	launchMethodText string
)

func init() {
	// config file, disables every other flag here
	flag.StringVar(&confPath, "c", "nil", "Path to full app configuration, or \"nil\" to configure from flags")

	flag.StringVar(&dbusConfigSession, "dbus-config", "builtin", "Path to D-Bus proxy config file, or \"builtin\" for defaults")
	flag.StringVar(&dbusConfigSystem, "dbus-system", "nil", "Path to system D-Bus proxy config file, or \"nil\" to disable")
	flag.StringVar(&dbusID, "dbus-id", "", "D-Bus ID of application, leave empty to disable own paths, has no effect if custom config is available")
	flag.BoolVar(&mpris, "mpris", false, "Allow owning MPRIS D-Bus path, has no effect if custom config is available")
	flag.BoolVar(&dbusVerbose, "dbus-log", false, "Force logging in the D-Bus proxy")

	flag.StringVar(&userName, "u", "chronos", "Passwd name of user to run as")
	flag.BoolVar(&enablements[state.EnableWayland], "wayland", false, "Share Wayland socket")
	flag.BoolVar(&enablements[state.EnableX], "X", false, "Share X11 socket and allow connection")
	flag.BoolVar(&enablements[state.EnableDBus], "dbus", false, "Proxy D-Bus connection")
	flag.BoolVar(&enablements[state.EnablePulse], "pulse", false, "Share PulseAudio socket and cookie")
}

func init() {
	methodHelpString := "Method of launching the child process, can be one of \"sudo\""
	if internal.SdBootedV {
		methodHelpString += ", \"systemd\""
	}

	flag.StringVar(&launchMethodText, "method", "sudo", methodHelpString)
}

func loadConfig() *app.Config {
	if confPath == "nil" {
		// config from flags
		return configFromFlags()
	} else {
		// config from file
		c := new(app.Config)
		if f, err := os.Open(confPath); err != nil {
			fatalf("cannot access config file '%s': %s\n", confPath, err)
			panic("unreachable")
		} else if err = json.NewDecoder(f).Decode(&c); err != nil {
			fatalf("cannot parse config file '%s': %s\n", confPath, err)
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
	for i := state.Enablement(0); i < state.EnableLength; i++ {
		if enablements[i] {
			config.Confinement.Enablements.Set(i)
		}
	}

	// parse D-Bus config file from flags if applicable
	if enablements[state.EnableDBus] {
		if dbusConfigSession == "builtin" {
			config.Confinement.SessionBus = dbus.NewConfig(dbusID, true, mpris)
		} else {
			if c, err := dbus.NewConfigFromFile(dbusConfigSession); err != nil {
				fatalf("cannot load session bus proxy config from %q: %s\n", dbusConfigSession, err)
			} else {
				config.Confinement.SessionBus = c
			}
		}

		// system bus proxy is optional
		if dbusConfigSystem != "nil" {
			if c, err := dbus.NewConfigFromFile(dbusConfigSystem); err != nil {
				fatalf("cannot load system bus proxy config from %q: %s\n", dbusConfigSystem, err)
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
