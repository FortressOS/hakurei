package seccomp

import "C"
import "sync/atomic"

var printlnP atomic.Pointer[func(v ...any)]

func SetOutput(f func(v ...any)) {
	if f == nil {
		// avoid storing nil function
		printlnP.Store(nil)
	} else {
		printlnP.Store(&f)
	}
}

func GetOutput() func(v ...any) {
	if fp := printlnP.Load(); fp == nil {
		return nil
	} else {
		return *fp
	}
}

//export f_println
func f_println(v *C.char) {
	if fp := printlnP.Load(); fp != nil {
		(*fp)(C.GoString(v))
	}
}
