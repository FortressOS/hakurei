package container

import (
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"hakurei.app/container/seccomp"
)

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

	// lockOSThread provides [runtime.LockOSThread].
	lockOSThread()

	// setPtracer provides [SetPtracer].
	setPtracer(pid uintptr) error
	// setDumpable provides [SetDumpable].
	setDumpable(dumpable uintptr) error
	// setNoNewPrivs provides [SetNoNewPrivs].
	setNoNewPrivs() error

	// lastcap provides [LastCap].
	lastcap() uintptr
	// capset provides capset.
	capset(hdrp *capHeader, datap *[2]capData) error
	// capBoundingSetDrop provides capBoundingSetDrop.
	capBoundingSetDrop(cap uintptr) error
	// capAmbientClearAll provides capAmbientClearAll.
	capAmbientClearAll() error
	// capAmbientRaise provides capAmbientRaise.
	capAmbientRaise(cap uintptr) error
	// isatty provides [Isatty].
	isatty(fd int) bool
	// receive provides [Receive].
	receive(key string, e any, fdp *uintptr) (closeFunc func() error, err error)

	// bindMount provides procPaths.bindMount.
	bindMount(source, target string, flags uintptr, eq bool) error
	// remount provides procPaths.remount.
	remount(target string, flags uintptr) error
	// mountTmpfs provides mountTmpfs.
	mountTmpfs(fsname, target string, flags uintptr, size int, perm os.FileMode) error
	// ensureFile provides ensureFile.
	ensureFile(name string, perm, pperm os.FileMode) error

	// seccompLoad provides [seccomp.Load].
	seccompLoad(rules []seccomp.NativeRule, flags seccomp.ExportFlag) error
	// notify provides [signal.Notify].
	notify(c chan<- os.Signal, sig ...os.Signal)
	// start starts [os/exec.Cmd].
	start(c *exec.Cmd) error
	// signal signals the underlying process of [os/exec.Cmd].
	signal(c *exec.Cmd, sig os.Signal) error
	// evalSymlinks provides [filepath.EvalSymlinks].
	evalSymlinks(path string) (string, error)

	// exit provides [os.Exit].
	exit(code int)
	// getpid provides [os.Getpid].
	getpid() int
	// stat provides [os.Stat].
	stat(name string) (os.FileInfo, error)
	// mkdir provides [os.Mkdir].
	mkdir(name string, perm os.FileMode) error
	// mkdirTemp provides [os.MkdirTemp].
	mkdirTemp(dir, pattern string) (string, error)
	// mkdirAll provides [os.MkdirAll].
	mkdirAll(path string, perm os.FileMode) error
	// readdir provides [os.ReadDir].
	readdir(name string) ([]os.DirEntry, error)
	// openNew provides [os.Open].
	openNew(name string) (osFile, error)
	// writeFile provides [os.WriteFile].
	writeFile(name string, data []byte, perm os.FileMode) error
	// createTemp provides [os.CreateTemp].
	createTemp(dir, pattern string) (osFile, error)
	// remove provides os.Remove.
	remove(name string) error
	// newFile provides os.NewFile.
	newFile(fd uintptr, name string) *os.File
	// symlink provides os.Symlink.
	symlink(oldname, newname string) error
	// readlink provides [os.Readlink].
	readlink(name string) (string, error)

	// umask provides syscall.Umask.
	umask(mask int) (oldmask int)
	// sethostname provides syscall.Sethostname
	sethostname(p []byte) (err error)
	// chdir provides syscall.Chdir
	chdir(path string) (err error)
	// fchdir provides syscall.Fchdir
	fchdir(fd int) (err error)
	// open provides syscall.Open
	open(path string, mode int, perm uint32) (fd int, err error)
	// close provides syscall.Close
	close(fd int) (err error)
	// pivotRoot provides syscall.PivotRoot
	pivotRoot(newroot, putold string) (err error)
	// mount provides syscall.Mount
	mount(source, target, fstype string, flags uintptr, data string) (err error)
	// unmount provides syscall.Unmount
	unmount(target string, flags int) (err error)
	// wait4 provides syscall.Wait4
	wait4(pid int, wstatus *syscall.WaitStatus, options int, rusage *syscall.Rusage) (wpid int, err error)

	// printf provides [log.Printf].
	printf(format string, v ...any)
	// fatal provides [log.Fatal]
	fatal(v ...any)
	// fatalf provides [log.Fatalf]
	fatalf(format string, v ...any)
	// verbose provides [Msg.Verbose].
	verbose(v ...any)
	// verbosef provides [Msg.Verbosef].
	verbosef(format string, v ...any)
	// suspend provides [Msg.Suspend].
	suspend()
	// resume provides [Msg.Resume].
	resume() bool
	// beforeExit provides [Msg.BeforeExit].
	beforeExit()
	// printBaseErr provides [Msg.PrintBaseErr].
	printBaseErr(err error, fallback string)
}

// direct implements syscallDispatcher on the current kernel.
type direct struct{}

func (k direct) new(f func(k syscallDispatcher)) { go f(k) }

func (direct) lockOSThread() { runtime.LockOSThread() }

func (direct) setPtracer(pid uintptr) error       { return SetPtracer(pid) }
func (direct) setDumpable(dumpable uintptr) error { return SetDumpable(dumpable) }
func (direct) setNoNewPrivs() error               { return SetNoNewPrivs() }

func (direct) lastcap() uintptr                                { return LastCap() }
func (direct) capset(hdrp *capHeader, datap *[2]capData) error { return capset(hdrp, datap) }
func (direct) capBoundingSetDrop(cap uintptr) error            { return capBoundingSetDrop(cap) }
func (direct) capAmbientClearAll() error                       { return capAmbientClearAll() }
func (direct) capAmbientRaise(cap uintptr) error               { return capAmbientRaise(cap) }
func (direct) isatty(fd int) bool                              { return Isatty(fd) }
func (direct) receive(key string, e any, fdp *uintptr) (func() error, error) {
	return Receive(key, e, fdp)
}

func (direct) bindMount(source, target string, flags uintptr, eq bool) error {
	return hostProc.bindMount(source, target, flags, eq)
}
func (direct) remount(target string, flags uintptr) error {
	return hostProc.remount(target, flags)
}
func (k direct) mountTmpfs(fsname, target string, flags uintptr, size int, perm os.FileMode) error {
	return mountTmpfs(k, fsname, target, flags, size, perm)
}
func (direct) ensureFile(name string, perm, pperm os.FileMode) error {
	return ensureFile(name, perm, pperm)
}

func (direct) seccompLoad(rules []seccomp.NativeRule, flags seccomp.ExportFlag) error {
	return seccomp.Load(rules, flags)
}
func (direct) notify(c chan<- os.Signal, sig ...os.Signal) { signal.Notify(c, sig...) }
func (direct) start(c *exec.Cmd) error                     { return c.Start() }
func (direct) signal(c *exec.Cmd, sig os.Signal) error     { return c.Process.Signal(sig) }
func (direct) evalSymlinks(path string) (string, error)    { return filepath.EvalSymlinks(path) }

func (direct) exit(code int)                                 { os.Exit(code) }
func (direct) getpid() int                                   { return os.Getpid() }
func (direct) stat(name string) (os.FileInfo, error)         { return os.Stat(name) }
func (direct) mkdir(name string, perm os.FileMode) error     { return os.Mkdir(name, perm) }
func (direct) mkdirTemp(dir, pattern string) (string, error) { return os.MkdirTemp(dir, pattern) }
func (direct) mkdirAll(path string, perm os.FileMode) error  { return os.MkdirAll(path, perm) }
func (direct) readdir(name string) ([]os.DirEntry, error)    { return os.ReadDir(name) }
func (direct) openNew(name string) (osFile, error)           { return os.Open(name) }
func (direct) writeFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (direct) createTemp(dir, pattern string) (osFile, error) {
	return os.CreateTemp(dir, pattern)
}
func (direct) remove(name string) error {
	return os.Remove(name)
}
func (direct) newFile(fd uintptr, name string) *os.File {
	return os.NewFile(fd, name)
}
func (direct) symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
func (direct) readlink(name string) (string, error) {
	return os.Readlink(name)
}

func (direct) umask(mask int) (oldmask int)     { return syscall.Umask(mask) }
func (direct) sethostname(p []byte) (err error) { return syscall.Sethostname(p) }
func (direct) chdir(path string) (err error)    { return syscall.Chdir(path) }
func (direct) fchdir(fd int) (err error)        { return syscall.Fchdir(fd) }
func (direct) open(path string, mode int, perm uint32) (fd int, err error) {
	return syscall.Open(path, mode, perm)
}
func (direct) close(fd int) (err error) {
	return syscall.Close(fd)
}
func (direct) pivotRoot(newroot, putold string) (err error) {
	return syscall.PivotRoot(newroot, putold)
}
func (direct) mount(source, target, fstype string, flags uintptr, data string) (err error) {
	return syscall.Mount(source, target, fstype, flags, data)
}
func (direct) unmount(target string, flags int) (err error) {
	return syscall.Unmount(target, flags)
}
func (direct) wait4(pid int, wstatus *syscall.WaitStatus, options int, rusage *syscall.Rusage) (wpid int, err error) {
	return syscall.Wait4(pid, wstatus, options, rusage)
}

func (direct) printf(format string, v ...any)          { log.Printf(format, v...) }
func (direct) fatal(v ...any)                          { log.Fatal(v...) }
func (direct) fatalf(format string, v ...any)          { log.Fatalf(format, v...) }
func (direct) verbose(v ...any)                        { msg.Verbose(v...) }
func (direct) verbosef(format string, v ...any)        { msg.Verbosef(format, v...) }
func (direct) suspend()                                { msg.Suspend() }
func (direct) resume() bool                            { return msg.Resume() }
func (direct) beforeExit()                             { msg.BeforeExit() }
func (direct) printBaseErr(err error, fallback string) { msg.PrintBaseErr(err, fallback) }
