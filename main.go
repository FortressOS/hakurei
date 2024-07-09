package main

import (
	"flag"
)

var Version = "impure"

func main() {
	flag.Parse()
	copyArgs()
}
