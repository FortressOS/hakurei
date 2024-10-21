package fmsg

func Verbose() bool {
	return verbose.Load()
}

func SetVerbose(v bool) {
	verbose.Store(v)
}

func VPrintf(format string, v ...any) {
	if verbose.Load() {
		std.Printf(format, v...)
	}
}

func VPrintln(v ...any) {
	if verbose.Load() {
		std.Println(v...)
	}
}
