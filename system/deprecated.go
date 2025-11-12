// Package system exposes the internal/system package.
//
// Deprecated: This package will be removed in 0.4.
package system

import (
	"context"
	_ "unsafe" // for go:linkname

	"hakurei.app/hst"
	"hakurei.app/internal/system"
	"hakurei.app/message"
)

// ErrDBusConfig is returned when a required hst.BusConfig argument is nil.
//
//go:linkname ErrDBusConfig hakurei.app/internal/system.ErrDBusConfig
var ErrDBusConfig error

// OpError is returned by [I.Commit] and [I.Revert].
type OpError = system.OpError

const (
	// User type is reverted at final instance exit.
	User = system.User
	// Process type is unconditionally reverted on exit.
	Process = system.Process

	CM = system.CM
)

// Criteria specifies types of Op to revert.
type Criteria = system.Criteria

// Op is a reversible system operation.
type Op = system.Op

// TypeString extends [Enablement.String] to support [User] and [Process].
//
//go:linkname TypeString hakurei.app/internal/system.TypeString
func TypeString(e hst.Enablement) string

// New returns the address of a new [I] targeting uid.
//
//go:linkname New hakurei.app/internal/system.New
func New(ctx context.Context, msg message.Msg, uid int) (sys *I)

// An I provides deferred operating system interaction. [I] must not be copied.
// Methods of [I] must not be used concurrently.
type I = system.I
