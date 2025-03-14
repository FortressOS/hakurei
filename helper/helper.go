// Package helper runs external helpers with optional sandboxing.
package helper

import (
	"fmt"
	"os/exec"
	"time"
)

var (
	WaitDelay = 2 * time.Second

	commandContext = exec.CommandContext
)

const (
	// FortifyHelper is set to 1 when args fd is enabled and 0 otherwise.
	FortifyHelper = "FORTIFY_HELPER"
	// FortifyStatus is set to 1 when stat fd is enabled and 0 otherwise.
	FortifyStatus = "FORTIFY_STATUS"
)

type Helper interface {
	// Start starts the helper process.
	Start() error
	// Wait blocks until Helper exits.
	Wait() error

	fmt.Stringer
}
