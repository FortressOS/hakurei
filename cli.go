package main

import (
	"flag"

	"git.ophivana.moe/cat/fortify/internal/app"
)

var (
	userName string

	mustWayland bool
	mustX       bool
	mustDBus    bool
	mustPulse   bool

	flagVerbose  bool
	printVersion bool
)

func init() {
	flag.StringVar(&userName, "u", "chronos", "Specify a username")

	flag.BoolVar(&mustWayland, "wayland", false, "Share Wayland socket")
	flag.BoolVar(&mustX, "X", false, "Share X11 socket and allow connection")
	flag.BoolVar(&mustDBus, "dbus", false, "Proxy D-Bus connection")
	flag.BoolVar(&mustPulse, "pulse", false, "Share PulseAudio socket and cookie")

	flag.BoolVar(&app.LaunchOptions[app.LaunchMethodSudo], "sudo", false, "Use 'sudo' to switch user")
	flag.BoolVar(&app.LaunchOptions[app.LaunchMethodMachineCtl], "machinectl", true, "Use 'machinectl' to switch user")

	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
	flag.BoolVar(&printVersion, "V", false, "Print version")
}
