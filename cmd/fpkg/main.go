package main

import (
	"flag"
	"os"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

const shell = "/run/current-system/sw/bin/bash"

func init() {
	if err := os.Setenv("SHELL", shell); err != nil {
		fmsg.Fatalf("cannot set $SHELL: %v", err)
	}
}

var (
	flagVerbose bool
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
}

func main() {
	fmsg.SetPrefix("fpkg")

	flag.Parse()
	fmsg.SetVerbose(flagVerbose)

	args := flag.Args()
	if len(args) < 1 {
		fmsg.Fatal("invalid argument")
	}

	switch args[0] {
	case "install":
		actionInstall(args[1:])
	case "start":
		actionStart(args[1:])

	default:
		fmsg.Fatal("invalid argument")
	}

	fmsg.Exit(0)
}
