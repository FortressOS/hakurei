package system

import (
	"io"
	"io/fs"
	"log"
	"os"

	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/dbus"
	"hakurei.app/internal/xcb"
)

type osFile interface {
	Name() string
	io.Writer
	fs.File
}

// syscallDispatcher provides methods that make state-dependent system calls as part of their behaviour.
// syscallDispatcher is embedded in [I], so all methods must be unexported.
type syscallDispatcher interface {
	// new starts a goroutine with a new instance of syscallDispatcher.
	// A syscallDispatcher must never be used in any goroutine other than the one owning it,
	// just synchronising access is not enough, as this is for test instrumentation.
	new(f func(k syscallDispatcher))

	// stat provides os.Stat.
	stat(name string) (os.FileInfo, error)
	// open provides [os.Open].
	open(name string) (osFile, error)
	// mkdir provides os.Mkdir.
	mkdir(name string, perm os.FileMode) error
	// chmod provides os.Chmod.
	chmod(name string, mode os.FileMode) error
	// link provides os.Link.
	link(oldname, newname string) error
	// remove provides os.Remove.
	remove(name string) error

	// println provides [log.Println].
	println(v ...any)

	// aclUpdate provides [acl.Update].
	aclUpdate(name string, uid int, perms ...acl.Perm) error

	// xcbChangeHosts provides [xcb.ChangeHosts].
	xcbChangeHosts(mode xcb.HostMode, family xcb.Family, address string) error

	// dbusFinalise provides [dbus.Finalise].
	dbusFinalise(sessionBus, systemBus dbus.ProxyPair, session, system *hst.BusConfig) (final *dbus.Final, err error)
	// dbusProxyStart provides the Start method of [dbus.Proxy].
	dbusProxyStart(proxy *dbus.Proxy) error
	// dbusProxyClose provides the Close method of [dbus.Proxy].
	dbusProxyClose(proxy *dbus.Proxy)
	// dbusProxyWait provides the Wait method of [dbus.Proxy].
	dbusProxyWait(proxy *dbus.Proxy) error
}

// direct implements syscallDispatcher on the current kernel.
type direct struct{}

func (k direct) new(f func(k syscallDispatcher)) { go f(k) }

func (k direct) stat(name string) (os.FileInfo, error)     { return os.Stat(name) }
func (k direct) open(name string) (osFile, error)          { return os.Open(name) }
func (k direct) mkdir(name string, perm os.FileMode) error { return os.Mkdir(name, perm) }
func (k direct) chmod(name string, mode os.FileMode) error { return os.Chmod(name, mode) }
func (k direct) link(oldname, newname string) error        { return os.Link(oldname, newname) }
func (k direct) remove(name string) error                  { return os.Remove(name) }

func (k direct) println(v ...any) { log.Println(v...) }

func (k direct) aclUpdate(name string, uid int, perms ...acl.Perm) error {
	return acl.Update(name, uid, perms...)
}

func (k direct) xcbChangeHosts(mode xcb.HostMode, family xcb.Family, address string) error {
	return xcb.ChangeHosts(mode, family, address)
}

func (k direct) dbusFinalise(sessionBus, systemBus dbus.ProxyPair, session, system *hst.BusConfig) (final *dbus.Final, err error) {
	return dbus.Finalise(sessionBus, systemBus, session, system)
}

func (k direct) dbusProxyStart(proxy *dbus.Proxy) error { return proxy.Start() }
func (k direct) dbusProxyClose(proxy *dbus.Proxy)       { proxy.Close() }
func (k direct) dbusProxyWait(proxy *dbus.Proxy) error  { return proxy.Wait() }
