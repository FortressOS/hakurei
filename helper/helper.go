// Package helper runs external helpers with optional sandboxing and manages their status/args pipes.
package helper

import (
	"errors"
	"os/exec"
)

var (
	ErrStatusFault = errors.New("generic status pipe fault")
	ErrStatusRead  = errors.New("unexpected status response")
)

const (
	// FortifyHelper is set for the process launched by Helper.
	FortifyHelper = "FORTIFY_HELPER"
	// FortifyStatus is 1 when sync fd is enabled and 0 otherwise.
	FortifyStatus = "FORTIFY_STATUS"
)

type Helper interface {
	// StartNotify starts the helper process.
	// A status pipe is passed to the helper if ready is not nil.
	StartNotify(ready chan error) error
	// Start starts the helper process.
	Start() error
	// Close closes the status pipe.
	// If helper is started without the status pipe, Close panics.
	Close() error
	// Wait calls wait on the child process and cleans up pipes.
	Wait() error
	// Unwrap returns the underlying exec.Cmd instance.
	Unwrap() *exec.Cmd
}

var execCommand = exec.Command
