package main

import (
	"flag"

	"git.ophivana.moe/cat/fortify/internal"
)

var (
	userName string
	confPath string

	dbusConfigSession string
	dbusConfigSystem  string
	dbusVerbose       bool
	dbusID            string
	mpris             bool

	mustWayland bool
	mustX       bool
	mustDBus    bool
	mustPulse   bool

	flagVerbose  bool
	printVersion bool

	launchMethodText string
)

func init() {
	flag.StringVar(&userName, "u", "chronos", "Passwd name of user to run as")
	flag.StringVar(&confPath, "c", "nil", "Path to full app configuration, or \"nil\" to configure from flags")

	flag.StringVar(&dbusConfigSession, "dbus-config", "builtin", "Path to D-Bus proxy config file, or \"builtin\" for defaults")
	flag.StringVar(&dbusConfigSystem, "dbus-system", "nil", "Path to system D-Bus proxy config file, or \"nil\" to disable")
	flag.BoolVar(&dbusVerbose, "dbus-log", false, "Enable logging in the D-Bus proxy")
	flag.StringVar(&dbusID, "dbus-id", "", "D-Bus ID of application, leave empty to disable own paths, has no effect if custom config is available")
	flag.BoolVar(&mpris, "mpris", false, "Allow owning MPRIS D-Bus path, has no effect if custom config is available")

	flag.BoolVar(&mustWayland, "wayland", false, "Share Wayland socket")
	flag.BoolVar(&mustX, "X", false, "Share X11 socket and allow connection")
	flag.BoolVar(&mustDBus, "dbus", false, "Proxy D-Bus connection")
	flag.BoolVar(&mustPulse, "pulse", false, "Share PulseAudio socket and cookie")

	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
	flag.BoolVar(&printVersion, "V", false, "Print version")
}

func init() {
	methodHelpString := "Method of launching the child process, can be one of \"sudo\""
	if internal.SdBootedV {
		methodHelpString += ", \"systemd\""
	}

	flag.StringVar(&launchMethodText, "method", "sudo", methodHelpString)
}
