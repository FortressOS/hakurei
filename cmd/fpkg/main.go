package main

import (
	"flag"
	"log"
	"os"

	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

const shellPath = "/run/current-system/sw/bin/bash"

func init() {
	if err := os.Setenv("SHELL", shellPath); err != nil {
		log.Fatalf("cannot set $SHELL: %v", err)
	}
}

var (
	flagVerbose bool
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
}

func main() {
	fmsg.Prepare("fpkg")

	flag.Parse()
	fmsg.Store(flagVerbose)

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("invalid argument")
	}

	switch args[0] {
	case "install":
		actionInstall(args[1:])
	case "start":
		actionStart(args[1:])

	default:
		log.Fatal("invalid argument")
	}

	internal.Exit(0)
}
