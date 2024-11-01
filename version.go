package main

import (
	"flag"
	"fmt"

	"git.ophivana.moe/security/fortify/internal"
)

var (
	printVersion bool
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Print version")
}

func tryVersion() {
	if printVersion {
		if v, ok := internal.Check(internal.Version); ok {
			fmt.Println(v)
		} else {
			fmt.Println("impure")
		}
		os.Exit(0)
	}
}
