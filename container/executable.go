package container

import (
	"fmt"
	"log"
	"os"
	"sync"

	"hakurei.app/message"
)

var (
	executable     string
	executableOnce sync.Once
)

func copyExecutable(msg message.Msg) {
	if name, err := os.Executable(); err != nil {
		m := fmt.Sprintf("cannot read executable path: %v", err)
		if msg != nil {
			msg.BeforeExit()
			msg.GetLogger().Fatal(m)
		} else {
			log.Fatal(m)
		}
	} else {
		executable = name
	}
}

func MustExecutable(msg message.Msg) string {
	executableOnce.Do(func() { copyExecutable(msg) })
	return executable
}
