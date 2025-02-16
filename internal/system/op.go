package system

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

const (
	// User type is reverted at final launcher exit.
	User = Enablement(ELen)
	// Process type is unconditionally reverted on exit.
	Process = Enablement(ELen + 1)
)

type Criteria struct {
	*Enablements
}

func (ec *Criteria) hasType(o Op) bool {
	// nil criteria: revert everything except User
	if ec.Enablements == nil {
		return o.Type() != User
	}

	return ec.Has(o.Type())
}

// Op is a reversible system operation.
type Op interface {
	// Type returns Op's enablement type.
	Type() Enablement

	// apply the Op
	apply(sys *I) error
	// revert reverses the Op if criteria is met
	revert(sys *I, ec *Criteria) error

	Is(o Op) bool
	Path() string
	String() string
}

func TypeString(e Enablement) string {
	switch e {
	case User:
		return "User"
	case Process:
		return "Process"
	default:
		return e.String()
	}
}

type I struct {
	uid int
	ops []Op
	ctx context.Context
	// sync fd passed to bwrap
	sp *os.File

	// whether sys has been reverted
	state bool

	lock sync.Mutex
}

func (sys *I) UID() int {
	return sys.uid
}

func (sys *I) Sync() *os.File {
	return sys.sp
}

func (sys *I) Equal(v *I) bool {
	if v == nil || sys.uid != v.uid || len(sys.ops) != len(v.ops) {
		return false
	}

	for i, o := range sys.ops {
		if !o.Is(v.ops[i]) {
			return false
		}
	}

	return true
}

func (sys *I) Commit(ctx context.Context) error {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	if sys.ctx != nil {
		panic("sys instance committed twice")
	}
	sys.ctx = ctx

	sp := New(sys.uid)
	sp.ops = make([]Op, 0, len(sys.ops)) // prevent copies during commits
	defer func() {
		// sp is set to nil when all ops are applied
		if sp != nil {
			// rollback partial commit
			fmsg.Verbosef("commit faulted after %d ops, rolling back partial commit", len(sp.ops))
			if err := sp.Revert(&Criteria{nil}); err != nil {
				log.Println("errors returned reverting partial commit:", err)
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

func (sys *I) Revert(ec *Criteria) error {
	sys.lock.Lock()
	defer sys.lock.Unlock()

	if sys.state {
		panic("sys instance reverted twice")
	}
	sys.state = true

	// collect errors
	errs := make([]error, len(sys.ops))

	for i := range sys.ops {
		errs[i] = sys.ops[len(sys.ops)-i-1].revert(sys, ec)
	}

	// errors.Join filters nils
	return errors.Join(errs...)
}

func New(uid int) *I {
	return &I{uid: uid}
}
