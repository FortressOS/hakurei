package main

import (
	"flag"
	"fmt"
)

var (
	Version = "impure"

	printVersion bool
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Print version")
}

func tryVersion() {
	if printVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
}
