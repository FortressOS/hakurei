package app

import (
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/internal"
	"hakurei.app/message"
	"hakurei.app/system/dbus"
)

// osFile represents [os.File].
type osFile interface {
	Name() string
	io.Writer
	fs.File
}

// syscallDispatcher provides methods that make state-dependent system calls as part of their behaviour.
type syscallDispatcher interface {
	// new starts a goroutine with a new instance of syscallDispatcher.
	// A syscallDispatcher must never be used in any goroutine other than the one owning it,
	// just synchronising access is not enough, as this is for test instrumentation.
	new(f func(k syscallDispatcher))

	// getpid provides [os.Getpid].
	getpid() int
	// getuid provides [os.Getuid].
	getuid() int
	// getgid provides [os.Getgid].
	getgid() int
	// lookupEnv provides [os.LookupEnv].
	lookupEnv(key string) (string, bool)
	// stat provides [os.Stat].
	stat(name string) (os.FileInfo, error)
	// open provides [os.Open].
	open(name string) (osFile, error)
	// readdir provides [os.ReadDir].
	readdir(name string) ([]os.DirEntry, error)
	// tempdir provides [os.TempDir].
	tempdir() string

	// evalSymlinks provides [filepath.EvalSymlinks].
	evalSymlinks(path string) (string, error)

	// lookupGroupId calls [user.LookupGroup] and returns the Gid field of the resulting [user.Group] struct.
	lookupGroupId(name string) (string, error)

	// cmdOutput provides the Output method of [exec.Cmd].
	cmdOutput(cmd *exec.Cmd) ([]byte, error)

	// overflowUid provides [container.OverflowUid].
	overflowUid(msg message.Msg) int
	// overflowGid provides [container.OverflowGid].
	overflowGid(msg message.Msg) int

	// mustHsuPath provides [internal.MustHsuPath].
	mustHsuPath() *check.Absolute

	// dbusAddress provides [dbus.Address].
	dbusAddress() (session, system string)

	// fatalf provides [log.Fatalf].
	fatalf(format string, v ...any)
}

// direct implements syscallDispatcher on the current kernel.
type direct struct{}

func (k direct) new(f func(k syscallDispatcher)) { go f(k) }

func (direct) getpid() int                                { return os.Getpid() }
func (direct) getuid() int                                { return os.Getuid() }
func (direct) getgid() int                                { return os.Getgid() }
func (direct) lookupEnv(key string) (string, bool)        { return os.LookupEnv(key) }
func (direct) stat(name string) (os.FileInfo, error)      { return os.Stat(name) }
func (direct) open(name string) (osFile, error)           { return os.Open(name) }
func (direct) readdir(name string) ([]os.DirEntry, error) { return os.ReadDir(name) }
func (direct) tempdir() string                            { return os.TempDir() }

func (direct) evalSymlinks(path string) (string, error) { return filepath.EvalSymlinks(path) }

func (direct) lookupGroupId(name string) (gid string, err error) {
	var group *user.Group
	group, err = user.LookupGroup(name)
	if group != nil {
		gid = group.Gid
	}
	return
}

func (direct) cmdOutput(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() }

func (direct) overflowUid(msg message.Msg) int { return container.OverflowUid(msg) }
func (direct) overflowGid(msg message.Msg) int { return container.OverflowGid(msg) }

func (direct) mustHsuPath() *check.Absolute { return internal.MustHsuPath() }

func (k direct) dbusAddress() (session, system string) { return dbus.Address() }

func (direct) fatalf(format string, v ...any) { log.Fatalf(format, v...) }
