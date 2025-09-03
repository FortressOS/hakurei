package system

import "hakurei.app/system/acl"

// syscallDispatcher provides methods that make state-dependent system calls as part of their behaviour.
// syscallDispatcher is embedded in [I], so all methods must be unexported.
type syscallDispatcher interface {
	// new starts a goroutine with a new instance of syscallDispatcher.
	// A syscallDispatcher must never be used in any goroutine other than the one owning it,
	// just synchronising access is not enough, as this is for test instrumentation.
	new(f func(k syscallDispatcher))

	// aclUpdate provides [acl.Update].
	aclUpdate(name string, uid int, perms ...acl.Perm) error

	verbose(v ...any)
	verbosef(format string, v ...any)
}

// direct implements syscallDispatcher on the current kernel.
type direct struct{}

func (k direct) new(f func(k syscallDispatcher)) { go f(k) }

func (k direct) aclUpdate(name string, uid int, perms ...acl.Perm) error {
	return acl.Update(name, uid, perms...)
}

func (direct) verbose(v ...any)                 { msg.Verbose(v...) }
func (direct) verbosef(format string, v ...any) { msg.Verbosef(format, v...) }
