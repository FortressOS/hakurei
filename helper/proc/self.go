package proc

import (
	"os"
	"sync"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

var (
	executable     string
	executableOnce sync.Once
)

func copyExecutable() {
	if name, err := os.Executable(); err != nil {
		fmsg.Fatalf("cannot read executable path: %v", err)
	} else {
		executable = name
	}
}

func MustExecutable() string {
	executableOnce.Do(copyExecutable)
	return executable
}
