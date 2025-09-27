// Package hlog provides various functions for output messages.
package hlog

import (
	"log"
	"os"

	"hakurei.app/container"
)

var o = &container.Suspendable{Downstream: os.Stderr}

// Prepare configures the system logger for [Suspend] and [Resume] to take effect.
func Prepare(prefix string) { log.SetPrefix(prefix + ": "); log.SetFlags(0); log.SetOutput(o) }

func Suspend() bool { return o.Suspend() }
func Resume() bool {
	resumed, dropped, _, err := o.Resume()
	if err != nil {
		// probably going to result in an error as well,
		// so this call is as good as unreachable
		log.Printf("cannot dump buffer on resume: %v", err)
	}
	if resumed && dropped > 0 {
		log.Fatalf("dropped %d bytes while output is suspended", dropped)
	}
	return resumed
}

func BeforeExit() {
	if Resume() {
		log.Printf("beforeExit reached on suspended output")
	}
}
