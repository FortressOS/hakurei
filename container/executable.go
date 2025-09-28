package container

import (
	"os"
	"sync"
)

var (
	executable     string
	executableOnce sync.Once
)

func copyExecutable(msg Msg) {
	if name, err := os.Executable(); err != nil {
		msg.BeforeExit()
		msg.GetLogger().Fatalf("cannot read executable path: %v", err)
	} else {
		executable = name
	}
}

func MustExecutable(msg Msg) string {
	executableOnce.Do(func() { copyExecutable(msg) })
	return executable
}
