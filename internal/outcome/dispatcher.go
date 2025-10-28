package outcome

import (
	"context"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/seccomp"
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
	new(f func(k syscallDispatcher, msg message.Msg))

	// getpid provides [os.Getpid].
	getpid() int
	// getuid provides [os.Getuid].
	getuid() int
	// getgid provides [os.Getgid].
	getgid() int
	// lookupEnv provides [os.LookupEnv].
	lookupEnv(key string) (string, bool)
	// pipe provides os.Pipe.
	pipe() (r, w *os.File, err error)
	// stat provides [os.Stat].
	stat(name string) (os.FileInfo, error)
	// open provides [os.Open].
	open(name string) (osFile, error)
	// readdir provides [os.ReadDir].
	readdir(name string) ([]os.DirEntry, error)
	// tempdir provides [os.TempDir].
	tempdir() string
	// exit provides [os.Exit].
	exit(code int)

	// evalSymlinks provides [filepath.EvalSymlinks].
	evalSymlinks(path string) (string, error)

	// lookupGroupId calls [user.LookupGroup] and returns the Gid field of the resulting [user.Group] struct.
	lookupGroupId(name string) (string, error)

	// cmdOutput provides the Output method of [exec.Cmd].
	cmdOutput(cmd *exec.Cmd) ([]byte, error)

	// notifyContext provides [signal.NotifyContext].
	notifyContext(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc)

	// prctl provides [container.Prctl].
	prctl(op, arg2, arg3 uintptr) error
	// overflowUid provides [container.OverflowUid].
	overflowUid(msg message.Msg) int
	// overflowGid provides [container.OverflowGid].
	overflowGid(msg message.Msg) int
	// setDumpable provides [container.SetDumpable].
	setDumpable(dumpable uintptr) error
	// receive provides [container.Receive].
	receive(key string, e any, fdp *uintptr) (closeFunc func() error, err error)

	// containerStart provides the Start method of [container.Container].
	containerStart(z *container.Container) error
	// containerStart provides the Serve method of [container.Container].
	containerServe(z *container.Container) error
	// containerStart provides the Wait method of [container.Container].
	containerWait(z *container.Container) error

	// seccompLoad provides [seccomp.Load].
	seccompLoad(rules []seccomp.NativeRule, flags seccomp.ExportFlag) error

	// mustHsuPath provides [internal.MustHsuPath].
	mustHsuPath() *check.Absolute

	// dbusAddress provides [dbus.Address].
	dbusAddress() (session, system string)

	// setupContSignal provides setupContSignal.
	setupContSignal(pid int) (io.ReadCloser, func(), error)

	// getMsg returns the [message.Msg] held by syscallDispatcher.
	getMsg() message.Msg
	// fatal provides [log.Fatal].
	fatal(v ...any)
	// fatalf provides [log.Fatalf].
	fatalf(format string, v ...any)
}

// direct implements syscallDispatcher on the current kernel.
type direct struct{ msg message.Msg }

func (k direct) new(f func(k syscallDispatcher, msg message.Msg)) { go f(k, k.msg) }

func (direct) getpid() int                                { return os.Getpid() }
func (direct) getuid() int                                { return os.Getuid() }
func (direct) getgid() int                                { return os.Getgid() }
func (direct) lookupEnv(key string) (string, bool)        { return os.LookupEnv(key) }
func (direct) pipe() (r, w *os.File, err error)           { return os.Pipe() }
func (direct) stat(name string) (os.FileInfo, error)      { return os.Stat(name) }
func (direct) open(name string) (osFile, error)           { return os.Open(name) }
func (direct) readdir(name string) ([]os.DirEntry, error) { return os.ReadDir(name) }
func (direct) tempdir() string                            { return os.TempDir() }
func (direct) exit(code int)                              { os.Exit(code) }

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

func (direct) notifyContext(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc) {
	return signal.NotifyContext(parent, signals...)
}

func (direct) prctl(op, arg2, arg3 uintptr) error { return container.Prctl(op, arg2, arg3) }
func (direct) overflowUid(msg message.Msg) int    { return container.OverflowUid(msg) }
func (direct) overflowGid(msg message.Msg) int    { return container.OverflowGid(msg) }
func (direct) setDumpable(dumpable uintptr) error { return container.SetDumpable(dumpable) }
func (direct) receive(key string, e any, fdp *uintptr) (func() error, error) {
	return container.Receive(key, e, fdp)
}

func (direct) containerStart(z *container.Container) error { return z.Start() }
func (direct) containerServe(z *container.Container) error { return z.Serve() }
func (direct) containerWait(z *container.Container) error  { return z.Wait() }

func (direct) seccompLoad(rules []seccomp.NativeRule, flags seccomp.ExportFlag) error {
	return seccomp.Load(rules, flags)
}

func (direct) mustHsuPath() *check.Absolute { return internal.MustHsuPath() }

func (direct) dbusAddress() (session, system string) { return dbus.Address() }

func (direct) setupContSignal(pid int) (io.ReadCloser, func(), error) { return setupContSignal(pid) }

func (k direct) getMsg() message.Msg            { return k.msg }
func (k direct) fatal(v ...any)                 { k.msg.GetLogger().Fatal(v...) }
func (k direct) fatalf(format string, v ...any) { k.msg.GetLogger().Fatalf(format, v...) }
