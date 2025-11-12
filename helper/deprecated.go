// Package helper exposes the internal/helper package.
//
// Deprecated: This package will be removed in 0.4.
package helper

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"
	_ "unsafe" // for go:linkname

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/internal/helper"
	"hakurei.app/message"
)

//go:linkname WaitDelay hakurei.app/internal/helper.WaitDelay
var WaitDelay time.Duration

const (
	// HakureiHelper is set to 1 when args fd is enabled and 0 otherwise.
	HakureiHelper = helper.HakureiHelper
	// HakureiStatus is set to 1 when stat fd is enabled and 0 otherwise.
	HakureiStatus = helper.HakureiStatus
)

type Helper = helper.Helper

// NewCheckedArgs returns a checked null-terminated argument writer for a copy of args.
//
//go:linkname NewCheckedArgs hakurei.app/internal/helper.NewCheckedArgs
func NewCheckedArgs(args ...string) (wt io.WriterTo, err error)

// MustNewCheckedArgs returns a checked null-terminated argument writer for a copy of args.
// If s contains a NUL byte this function panics instead of returning an error.
//
//go:linkname MustNewCheckedArgs hakurei.app/internal/helper.MustNewCheckedArgs
func MustNewCheckedArgs(args ...string) io.WriterTo

// NewDirect initialises a new direct Helper instance with wt as the null-terminated argument writer.
// Function argF returns an array of arguments passed directly to the child process.
//
//go:linkname NewDirect hakurei.app/internal/helper.NewDirect
func NewDirect(
	ctx context.Context,
	name string,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	cmdF func(cmd *exec.Cmd),
	extraFiles []*os.File,
) Helper

// New initialises a Helper instance with wt as the null-terminated argument writer.
//
//go:linkname New hakurei.app/internal/helper.New
func New(
	ctx context.Context,
	msg message.Msg,
	pathname *check.Absolute, name string,
	wt io.WriterTo,
	stat bool,
	argF func(argsFd, statFd int) []string,
	cmdF func(z *container.Container),
	extraFiles []*os.File,
) Helper

// InternalHelperStub is an internal function but exported because it is cross-package;
// it is part of the implementation of the helper stub.
func InternalHelperStub() { helper.InternalHelperStub() }
