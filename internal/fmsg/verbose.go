package fmsg

import "sync/atomic"

var verbose = new(atomic.Bool)

func Verbose() bool {
	return verbose.Load()
}

func SetVerbose(v bool) {
	verbose.Store(v)
}

func VPrintf(format string, v ...any) {
	if verbose.Load() {
		Printf(format, v...)
	}
}

func VPrintln(v ...any) {
	if verbose.Load() {
		Println(v...)
	}
}
