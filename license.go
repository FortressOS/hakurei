package main

import (
	_ "embed"
	"flag"
	"fmt"
)

var (
	//go:embed LICENSE
	license string

	printLicense bool
)

func init() {
	flag.BoolVar(&printLicense, "license", false, "Print license")
}

func tryLicense() {
	if printLicense {
		fmt.Println(license)
		os.Exit(0)
	}
}
