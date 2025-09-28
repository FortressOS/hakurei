// Package system provides helpers to apply and revert groups of operations to the system.
package system

import (
	"context"
	"errors"
	"strings"

	"hakurei.app/container"
	"hakurei.app/hst"
)

const (
	// User type is reverted at final instance exit.
	User = hst.EM << iota
	// Process type is unconditionally reverted on exit.
	Process

	CM
)

// Criteria specifies types of Op to revert.
type Criteria hst.Enablement

func (ec *Criteria) hasType(t hst.Enablement) bool {
	// nil criteria: revert everything except User
	if ec == nil {
		return t != User
	}

	return hst.Enablement(*ec)&t != 0
}

// Op is a reversible system operation.
type Op interface {
	// Type returns [Op]'s enablement type, for matching a revert criteria.
	Type() hst.Enablement

	apply(sys *I) error
	revert(sys *I, ec *Criteria) error

	Is(o Op) bool
	Path() string
	String() string
}

// TypeString extends [Enablement.String] to support [User] and [Process].
func TypeString(e hst.Enablement) string {
	switch e {
	case User:
		return "user"
	case Process:
		return "process"
	default:
		buf := new(strings.Builder)
		buf.Grow(48)
		if v := e &^ User &^ Process; v != 0 {
			buf.WriteString(v.String())
		}

		for i := User; i < CM; i <<= 1 {
			if e&i != 0 {
				buf.WriteString(", " + TypeString(i))
			}
		}
		return strings.TrimPrefix(buf.String(), ", ")
	}
}

// New returns the address of a new [I] targeting uid.
func New(ctx context.Context, msg container.Msg, uid int) (sys *I) {
	if ctx == nil || msg == nil || uid < 0 {
		panic("invalid call to New")
	}
	return &I{ctx: ctx, msg: msg, uid: uid, syscallDispatcher: direct{}}
}

// An I provides deferred operating system interaction. [I] must not be copied.
// Methods of [I] must not be used concurrently.
type I struct {
	_ noCopy

	uid int
	ops []Op
	ctx context.Context

	// the behaviour of Commit is only defined for up to one call
	committed bool
	// the behaviour of Revert is only defined for up to one call
	reverted bool

	msg container.Msg
	syscallDispatcher
}

func (sys *I) UID() int { return sys.uid }

// Equal returns whether all [Op] instances held by sys matches that of target.
func (sys *I) Equal(target *I) bool {
	if sys == nil || target == nil || sys.uid != target.uid || len(sys.ops) != len(target.ops) {
		return false
	}

	for i, o := range sys.ops {
		if !o.Is(target.ops[i]) {
			return false
		}
	}

	return true
}

// Commit applies all [Op] held by [I] and reverts all successful [Op] on first error encountered.
// Commit must not be called more than once.
func (sys *I) Commit() error {
	if sys.committed {
		panic("attempting to commit twice")
	}
	sys.committed = true

	sp := New(sys.ctx, sys.msg, sys.uid)
	sp.syscallDispatcher = sys.syscallDispatcher
	sp.ops = make([]Op, 0, len(sys.ops)) // prevent copies during commits
	defer func() {
		// sp is set to nil when all ops are applied
		if sp != nil {
			// rollback partial commit
			sys.msg.Verbosef("commit faulted after %d ops, rolling back partial commit", len(sp.ops))
			if err := sp.Revert(nil); err != nil {
				printJoinedError(sys.println, "cannot revert partial commit:", err)
			}
		}
	}()

	for _, o := range sys.ops {
		if err := o.apply(sys); err != nil {
			return err
		} else {
			// register partial commit
			sp.ops = append(sp.ops, o)
		}
	}

	// disarm partial commit rollback
	sp = nil
	return nil
}

// Revert reverts all [Op] meeting [Criteria] held by [I].
func (sys *I) Revert(ec *Criteria) error {
	if sys.reverted {
		panic("attempting to revert twice")
	}
	sys.reverted = true

	// collect errors
	errs := make([]error, len(sys.ops))
	for i := range sys.ops {
		errs[i] = sys.ops[len(sys.ops)-i-1].revert(sys, ec)
	}

	// errors.Join filters nils
	return errors.Join(errs...)
}

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
