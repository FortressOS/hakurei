package main

import (
	"flag"

	"git.ophivana.moe/cat/fortify/internal/system"
)

var (
	userName     string
	printVersion bool
	mustPulse    bool
	flagVerbose  bool
)

func init() {
	flag.StringVar(&userName, "u", "chronos", "Specify a username")
	flag.BoolVar(&system.MethodFlags[0], "sudo", false, "Use 'sudo' to change user")
	flag.BoolVar(&system.MethodFlags[1], "bare", false, "Use 'machinectl' but skip xdg-desktop-portal setup")
	flag.BoolVar(&mustPulse, "pulse", false, "Treat unavailable PulseAudio as fatal")
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
	flag.BoolVar(&printVersion, "V", false, "Print version")
}
