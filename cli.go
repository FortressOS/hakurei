package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/user"
)

var (
	userName     string
	methodFlags  [2]bool
	printVersion bool
	mustPulse    bool
)

func init() {
	flag.StringVar(&userName, "u", "ego", "Specify a username")
	flag.BoolVar(&methodFlags[0], "sudo", false, "Use 'sudo' to change user")
	flag.BoolVar(&methodFlags[1], "bare", false, "Use 'machinectl' but skip xdg-desktop-portal setup")
	flag.BoolVar(&mustPulse, "pulse", false, "Treat unavailable PulseAudio as fatal")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&printVersion, "V", false, "Print version")
}

func copyArgs() {
	tryLauncher()
	tryVersion()
	tryLicense()

	command = flag.Args()

	if u, err := user.Lookup(userName); err != nil {
		if errors.As(err, new(user.UnknownUserError)) {
			fmt.Println("unknown user", userName)
		} else {
			// unreachable
			panic(err)
		}

		os.Exit(1)
	} else {
		ego = u
	}

	if verbose {
		fmt.Println("Running as user", ego.Username, "("+ego.Uid+"),", "command:", command)
	}
}
