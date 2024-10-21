// Package fmsg provides various functions for output messages.
package fmsg

import (
	"log"
	"os"
	"sync/atomic"
)

var (
	std  = log.New(os.Stdout, "fortify: ", 0)
	warn = log.New(os.Stderr, "fortify: ", 0)

	verbose = new(atomic.Bool)
)

func SetPrefix(prefix string) {
	prefix += ": "
	std.SetPrefix(prefix)
	warn.SetPrefix(prefix)
}

func Print(v ...any) {
	warn.Print(v...)
}

func Printf(format string, v ...any) {
	warn.Printf(format, v...)
}

func Println(v ...any) {
	warn.Println(v...)
}

func Fatal(v ...any) {
	warn.Fatal(v...)
}

func Fatalf(format string, v ...any) {
	warn.Fatalf(format, v...)
}
