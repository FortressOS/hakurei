// Package fmsg provides various functions for output messages.
package fmsg

import (
	"log"
	"os"
)

var std = log.New(os.Stderr, "fortify: ", 0)

func SetPrefix(prefix string) {
	prefix += ": "
	std.SetPrefix(prefix)
	std.SetPrefix(prefix)
}

func Print(v ...any) {
	dequeueOnce.Do(dequeue)
	queue(dPrint(v))
}

func Printf(format string, v ...any) {
	dequeueOnce.Do(dequeue)
	queue(&dPrintf{format, v})
}

func Println(v ...any) {
	dequeueOnce.Do(dequeue)
	queue(dPrintln(v))
}

func Fatal(v ...any) {
	Print(v...)
	Exit(1)
}

func Fatalf(format string, v ...any) {
	Printf(format, v...)
	Exit(1)
}
