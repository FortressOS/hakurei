package main

import (
	"flag"

	"git.ophivana.moe/cat/fortify/internal/app"
)

var (
	userName   string
	dbusConfig string
	dbusID     string
	mpris      bool

	mustWayland bool
	mustX       bool
	mustDBus    bool
	mustPulse   bool

	flagVerbose  bool
	printVersion bool
)

func init() {
	flag.StringVar(&userName, "u", "chronos", "Passwd name of user to run as")
	flag.StringVar(&dbusConfig, "dbus-config", "builtin", "Path to D-Bus proxy config file, or \"builtin\" for defaults")
	flag.StringVar(&dbusID, "dbus-id", "", "D-Bus ID of application, leave empty to disable own paths, has no effect if custom config is available")
	flag.BoolVar(&mpris, "mpris", false, "Allow owning MPRIS D-Bus path, has no effect if custom config is available")

	flag.BoolVar(&mustWayland, "wayland", false, "Share Wayland socket")
	flag.BoolVar(&mustX, "X", false, "Share X11 socket and allow connection")
	flag.BoolVar(&mustDBus, "dbus", false, "Proxy D-Bus connection")
	flag.BoolVar(&mustPulse, "pulse", false, "Share PulseAudio socket and cookie")

	flag.BoolVar(&app.LaunchOptions[app.LaunchMethodSudo], "sudo", false, "Use 'sudo' to switch user")
	flag.BoolVar(&app.LaunchOptions[app.LaunchMethodMachineCtl], "machinectl", true, "Use 'machinectl' to switch user")

	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
	flag.BoolVar(&printVersion, "V", false, "Print version")
}
