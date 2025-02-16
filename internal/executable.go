package internal

import (
	"log"
	"os"
	"sync"
)

var (
	executable     string
	executableOnce sync.Once
)

func copyExecutable() {
	if name, err := os.Executable(); err != nil {
		log.Fatalf("cannot read executable path: %v", err)
	} else {
		executable = name
	}
}

func MustExecutable() string {
	executableOnce.Do(copyExecutable)
	return executable
}
