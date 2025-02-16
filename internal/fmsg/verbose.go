package fmsg

import (
	"log"
	"sync/atomic"
)

var verbose = new(atomic.Bool)

func Load() bool   { return verbose.Load() }
func Store(v bool) { verbose.Store(v) }

func Verbosef(format string, v ...any) {
	if verbose.Load() {
		log.Printf(format, v...)
	}
}

func Verbose(v ...any) {
	if verbose.Load() {
		log.Println(v...)
	}
}
