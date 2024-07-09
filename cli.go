package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/user"
)

var (
	ego     *user.User
	command []string
	verbose bool
	method  = machinectl

	userName     string
	methodFlags  [2]bool
	printVersion bool
)

const (
	machinectl uint8 = iota
	machinectlBare
	sudo
)

func init() {
	flag.StringVar(&userName, "u", "ego", "Specify a username")
	flag.BoolVar(&methodFlags[0], "sudo", false, "Use 'sudo' to change user")
	flag.BoolVar(&methodFlags[1], "bare", false, "Use 'machinectl' but skip xdg-desktop-portal setup")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&printVersion, "V", false, "Print version")
}

func copyArgs() {
	if printVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	command = flag.Args()

	switch { // zero value is machinectl
	case methodFlags[0]:
		method = sudo
	case methodFlags[1]:
		method = machinectlBare
	}

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
		fmt.Println("Running command", command, "as user", ego.Username, "("+ego.Uid+")")
	}
}
