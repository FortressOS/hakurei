package container

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"hakurei.app/container/seccomp"
	"hakurei.app/container/stub"
)

type opValidTestCase struct {
	name string
	op   Op
	want bool
}

func checkOpsValid(t *testing.T, testCases []opValidTestCase) {
	t.Helper()

	t.Run("valid", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				if got := tc.op.Valid(); got != tc.want {
					t.Errorf("Valid: %v, want %v", got, tc.want)
				}
			})
		}
	})
}

type opsBuilderTestCase struct {
	name string
	ops  *Ops
	want Ops
}

func checkOpsBuilder(t *testing.T, testCases []opsBuilderTestCase) {
	t.Helper()

	t.Run("build", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				if !slices.EqualFunc(*tc.ops, tc.want, func(op Op, v Op) bool { return op.Is(v) }) {
					t.Errorf("Ops: %#v, want %#v", tc.ops, tc.want)
				}
			})
		}
	})
}

type opIsTestCase struct {
	name  string
	op, v Op
	want  bool
}

func checkOpIs(t *testing.T, testCases []opIsTestCase) {
	t.Helper()

	t.Run("is", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				if got := tc.op.Is(tc.v); got != tc.want {
					t.Errorf("Is: %v, want %v", got, tc.want)
				}
			})
		}
	})
}

type opMetaTestCase struct {
	name string
	op   Op

	wantPrefix string
	wantString string
}

func checkOpMeta(t *testing.T, testCases []opMetaTestCase) {
	t.Helper()

	t.Run("meta", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				t.Run("prefix", func(t *testing.T) {
					t.Helper()

					if got, _ := tc.op.prefix(); got != tc.wantPrefix {
						t.Errorf("prefix: %q, want %q", got, tc.wantPrefix)
					}
				})

				t.Run("string", func(t *testing.T) {
					t.Helper()

					if got := tc.op.String(); got != tc.wantString {
						t.Errorf("String: %s, want %s", got, tc.wantString)
					}
				})
			})
		}
	})
}

// call initialises a [stub.Call].
// This keeps composites analysis happy without making the test cases too bloated.
func call(name string, args stub.ExpectArgs, ret any, err error) stub.Call {
	return stub.Call{Name: name, Args: args, Ret: ret, Err: err}
}

type simpleTestCase struct {
	name    string
	f       func(k syscallDispatcher) error
	want    stub.Expect
	wantErr error
}

func checkSimple(t *testing.T, fname string, testCases []simpleTestCase) {
	t.Helper()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			wait4signal := make(chan struct{})
			k := &kstub{wait4signal, stub.New(t, func(s *stub.Stub[syscallDispatcher]) syscallDispatcher { return &kstub{wait4signal, s} }, tc.want)}
			defer stub.HandleExit(t)
			if err := tc.f(k); !reflect.DeepEqual(err, tc.wantErr) {
				t.Errorf("%s: error = %v, want %v", fname, err, tc.wantErr)
			}
			k.VisitIncomplete(func(s *stub.Stub[syscallDispatcher]) {
				t.Helper()

				t.Errorf("%s: %d calls, want %d", fname, s.Pos(), s.Len())
			})
		})
	}
}

type opBehaviourTestCase struct {
	name   string
	params *Params
	op     Op

	early        []stub.Call
	wantErrEarly error

	apply        []stub.Call
	wantErrApply error
}

func checkOpBehaviour(t *testing.T, testCases []opBehaviourTestCase) {
	t.Helper()

	t.Run("behaviour", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				state := &setupState{Params: tc.params}
				k := &kstub{nil, stub.New(t,
					func(s *stub.Stub[syscallDispatcher]) syscallDispatcher { return &kstub{nil, s} },
					stub.Expect{Calls: slices.Concat(tc.early, []stub.Call{{Name: stub.CallSeparator}}, tc.apply)},
				)}
				defer stub.HandleExit(t)
				errEarly := tc.op.early(state, k)
				k.Expects(stub.CallSeparator)
				if !reflect.DeepEqual(errEarly, tc.wantErrEarly) {
					t.Errorf("early: error = %v, want %v", errEarly, tc.wantErrEarly)
				}
				if errEarly != nil {
					goto out
				}

				if err := tc.op.apply(state, k); !reflect.DeepEqual(err, tc.wantErrApply) {
					t.Errorf("apply: error = %v, want %v", err, tc.wantErrApply)
				}

			out:
				k.VisitIncomplete(func(s *stub.Stub[syscallDispatcher]) {
					count := k.Pos() - 1 // separator
					if count < len(tc.early) {
						t.Errorf("early: %d calls, want %d", count, len(tc.early))
					} else {
						t.Errorf("apply: %d calls, want %d", count-len(tc.early), len(tc.apply))
					}
				})
			})
		}
	})
}

func sliceAddr[S any](s []S) *[]S { return &s }

func newCheckedFile(t *testing.T, name, wantData string, closeErr error) osFile {
	f := &checkedOsFile{t: t, name: name, want: wantData, closeErr: closeErr}
	// check happens in Close, and cleanup is not guaranteed to run, so relying on it for sloppy implementations will cause sporadic test results
	f.cleanup = runtime.AddCleanup(f, func(name string) { f.t.Fatalf("checkedOsFile %s became unreachable without a call to Close", name) }, f.name)
	return f
}

type checkedOsFile struct {
	t        *testing.T
	name     string
	want     string
	closeErr error
	cleanup  runtime.Cleanup
	bytes.Buffer
}

func (f *checkedOsFile) Name() string               { return f.name }
func (f *checkedOsFile) Stat() (fs.FileInfo, error) { panic("unreachable") }
func (f *checkedOsFile) Close() error {
	defer f.cleanup.Stop()
	if f.String() != f.want {
		f.t.Errorf("checkedOsFile:\n%s\nwant\n%s", f.String(), f.want)
		return syscall.ENOTRECOVERABLE
	}
	return f.closeErr
}

func newConstFile(s string) osFile { return &readerOsFile{Reader: strings.NewReader(s)} }

type readerOsFile struct {
	closed bool
	io.Reader
}

func (*readerOsFile) Name() string               { panic("unreachable") }
func (*readerOsFile) Write([]byte) (int, error)  { panic("unreachable") }
func (*readerOsFile) Stat() (fs.FileInfo, error) { panic("unreachable") }
func (r *readerOsFile) Close() error {
	if r.closed {
		return os.ErrClosed
	}
	r.closed = true
	return nil
}

type writeErrOsFile struct{ err error }

func (writeErrOsFile) Name() string                { panic("unreachable") }
func (f writeErrOsFile) Write([]byte) (int, error) { return 0, f.err }
func (writeErrOsFile) Stat() (fs.FileInfo, error)  { panic("unreachable") }
func (writeErrOsFile) Read([]byte) (int, error)    { panic("unreachable") }
func (writeErrOsFile) Close() error                { panic("unreachable") }

type isDirFi bool

func (isDirFi) Name() string       { panic("unreachable") }
func (isDirFi) Size() int64        { panic("unreachable") }
func (isDirFi) Mode() fs.FileMode  { panic("unreachable") }
func (isDirFi) ModTime() time.Time { panic("unreachable") }
func (fi isDirFi) IsDir() bool     { return bool(fi) }
func (isDirFi) Sys() any           { panic("unreachable") }

func stubDir(names ...string) []os.DirEntry {
	d := make([]os.DirEntry, len(names))
	for i, name := range names {
		d[i] = nameDentry(name)
	}
	return d
}

type nameDentry string

func (e nameDentry) Name() string             { return string(e) }
func (nameDentry) IsDir() bool                { panic("unreachable") }
func (nameDentry) Type() fs.FileMode          { panic("unreachable") }
func (nameDentry) Info() (fs.FileInfo, error) { panic("unreachable") }

const (
	// magicWait4Signal must be used in a single pair of signal and wait4 calls across two goroutines
	// originating from the same toplevel kstub.
	// To enable this behaviour this value must be the last element of the args field in the wait4 call
	// and the ret value of the signal call.
	magicWait4Signal = 0xdef
)

type kstub struct {
	wait4signal chan struct{}
	*stub.Stub[syscallDispatcher]
}

func (k *kstub) new(f func(k syscallDispatcher)) { k.Helper(); k.New(f) }

func (k *kstub) lockOSThread() { k.Helper(); k.Expects("lockOSThread") }

func (k *kstub) setPtracer(pid uintptr) error {
	k.Helper()
	return k.Expects("setPtracer").Error(
		stub.CheckArg(k.Stub, "pid", pid, 0))
}

func (k *kstub) setDumpable(dumpable uintptr) error {
	k.Helper()
	return k.Expects("setDumpable").Error(
		stub.CheckArg(k.Stub, "dumpable", dumpable, 0))
}

func (k *kstub) setNoNewPrivs() error { k.Helper(); return k.Expects("setNoNewPrivs").Err }
func (k *kstub) lastcap() uintptr     { k.Helper(); return k.Expects("lastcap").Ret.(uintptr) }

func (k *kstub) capset(hdrp *capHeader, datap *[2]capData) error {
	k.Helper()
	return k.Expects("capset").Error(
		stub.CheckArgReflect(k.Stub, "hdrp", hdrp, 0),
		stub.CheckArgReflect(k.Stub, "datap", datap, 1))
}

func (k *kstub) capBoundingSetDrop(cap uintptr) error {
	k.Helper()
	return k.Expects("capBoundingSetDrop").Error(
		stub.CheckArg(k.Stub, "cap", cap, 0))
}

func (k *kstub) capAmbientClearAll() error { k.Helper(); return k.Expects("capAmbientClearAll").Err }

func (k *kstub) capAmbientRaise(cap uintptr) error {
	k.Helper()
	return k.Expects("capAmbientRaise").Error(
		stub.CheckArg(k.Stub, "cap", cap, 0))
}

func (k *kstub) isatty(fd int) bool {
	k.Helper()
	expect := k.Expects("isatty")
	if !stub.CheckArg(k.Stub, "fd", fd, 0) {
		k.FailNow()
	}
	return expect.Ret.(bool)
}

func (k *kstub) receive(key string, e any, fdp *uintptr) (closeFunc func() error, err error) {
	k.Helper()
	expect := k.Expects("receive")

	var closed bool
	closeFunc = func() error {
		if closed {
			k.Error("closeFunc called more than once")
			return os.ErrClosed
		}
		closed = true

		if expect.Ret != nil {
			// use return stored in kexpect for closeFunc instead
			return expect.Ret.(error)
		}
		return nil
	}
	err = expect.Error(
		stub.CheckArg(k.Stub, "key", key, 0),
		stub.CheckArgReflect(k.Stub, "e", e, 1),
		stub.CheckArgReflect(k.Stub, "fdp", fdp, 2))

	// 3 is unused so stores params
	if expect.Args[3] != nil {
		if v, ok := expect.Args[3].(*initParams); ok && v != nil {
			if p, ok0 := e.(*initParams); ok0 && p != nil {
				*p = *v
			}
		}
	}

	// 4 is unused so stores fd
	if expect.Args[4] != nil {
		if v, ok := expect.Args[4].(uintptr); ok && v >= 3 {
			if fdp != nil {
				*fdp = v
			}
		}
	}

	return
}

func (k *kstub) bindMount(source, target string, flags uintptr) error {
	k.Helper()
	return k.Expects("bindMount").Error(
		stub.CheckArg(k.Stub, "source", source, 0),
		stub.CheckArg(k.Stub, "target", target, 1),
		stub.CheckArg(k.Stub, "flags", flags, 2))
}

func (k *kstub) remount(target string, flags uintptr) error {
	k.Helper()
	return k.Expects("remount").Error(
		stub.CheckArg(k.Stub, "target", target, 0),
		stub.CheckArg(k.Stub, "flags", flags, 1))
}

func (k *kstub) mountTmpfs(fsname, target string, flags uintptr, size int, perm os.FileMode) error {
	k.Helper()
	return k.Expects("mountTmpfs").Error(
		stub.CheckArg(k.Stub, "fsname", fsname, 0),
		stub.CheckArg(k.Stub, "target", target, 1),
		stub.CheckArg(k.Stub, "flags", flags, 2),
		stub.CheckArg(k.Stub, "size", size, 3),
		stub.CheckArg(k.Stub, "perm", perm, 4))
}

func (k *kstub) ensureFile(name string, perm, pperm os.FileMode) error {
	k.Helper()
	return k.Expects("ensureFile").Error(
		stub.CheckArg(k.Stub, "name", name, 0),
		stub.CheckArg(k.Stub, "perm", perm, 1),
		stub.CheckArg(k.Stub, "pperm", pperm, 2))
}

func (k *kstub) seccompLoad(rules []seccomp.NativeRule, flags seccomp.ExportFlag) error {
	k.Helper()
	return k.Expects("seccompLoad").Error(
		stub.CheckArgReflect(k.Stub, "rules", rules, 0),
		stub.CheckArg(k.Stub, "flags", flags, 1))
}

func (k *kstub) notify(c chan<- os.Signal, sig ...os.Signal) {
	k.Helper()
	expect := k.Expects("notify")
	if c == nil || expect.Error(
		stub.CheckArgReflect(k.Stub, "sig", sig, 1)) != nil {
		k.FailNow()
	}

	// export channel for external instrumentation
	if chanf, ok := expect.Args[0].(func(c chan<- os.Signal)); ok && chanf != nil {
		chanf(c)
	}
}

func (k *kstub) start(c *exec.Cmd) error {
	k.Helper()
	expect := k.Expects("start")
	err := expect.Error(
		stub.CheckArg(k.Stub, "c.Path", c.Path, 0),
		stub.CheckArgReflect(k.Stub, "c.Args", c.Args, 1),
		stub.CheckArgReflect(k.Stub, "c.Env", c.Env, 2),
		stub.CheckArg(k.Stub, "c.Dir", c.Dir, 3))

	if process, ok := expect.Ret.(*os.Process); ok && process != nil {
		c.Process = process
	}
	return err
}

func (k *kstub) signal(c *exec.Cmd, sig os.Signal) error {
	k.Helper()
	expect := k.Expects("signal")
	if v, ok := expect.Ret.(int); ok && v == magicWait4Signal {
		if k.wait4signal == nil {
			panic("kstub not initialised for wait4 simulation")
		}
		defer func() { close(k.wait4signal) }()
	}
	return expect.Error(
		stub.CheckArg(k.Stub, "c.Path", c.Path, 0),
		stub.CheckArgReflect(k.Stub, "c.Args", c.Args, 1),
		stub.CheckArgReflect(k.Stub, "c.Env", c.Env, 2),
		stub.CheckArg(k.Stub, "c.Dir", c.Dir, 3),
		stub.CheckArg(k.Stub, "sig", sig, 4))
}

func (k *kstub) evalSymlinks(path string) (string, error) {
	k.Helper()
	expect := k.Expects("evalSymlinks")
	return expect.Ret.(string), expect.Error(
		stub.CheckArg(k.Stub, "path", path, 0))
}

func (k *kstub) exit(code int) {
	k.Helper()
	k.Expects("exit")
	if !stub.CheckArg(k.Stub, "code", code, 0) {
		k.FailNow()
	}
	panic(stub.PanicExit)
}

func (k *kstub) getpid() int { k.Helper(); return k.Expects("getpid").Ret.(int) }

func (k *kstub) stat(name string) (os.FileInfo, error) {
	k.Helper()
	expect := k.Expects("stat")
	return expect.Ret.(os.FileInfo), expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}

func (k *kstub) mkdir(name string, perm os.FileMode) error {
	k.Helper()
	return k.Expects("mkdir").Error(
		stub.CheckArg(k.Stub, "name", name, 0),
		stub.CheckArg(k.Stub, "perm", perm, 1))
}

func (k *kstub) mkdirTemp(dir, pattern string) (string, error) {
	k.Helper()
	expect := k.Expects("mkdirTemp")
	return expect.Ret.(string), expect.Error(
		stub.CheckArg(k.Stub, "dir", dir, 0),
		stub.CheckArg(k.Stub, "pattern", pattern, 1))
}

func (k *kstub) mkdirAll(path string, perm os.FileMode) error {
	k.Helper()
	return k.Expects("mkdirAll").Error(
		stub.CheckArg(k.Stub, "path", path, 0),
		stub.CheckArg(k.Stub, "perm", perm, 1))
}

func (k *kstub) readdir(name string) ([]os.DirEntry, error) {
	k.Helper()
	expect := k.Expects("readdir")
	return expect.Ret.([]os.DirEntry), expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}

func (k *kstub) openNew(name string) (osFile, error) {
	k.Helper()
	expect := k.Expects("openNew")
	return expect.Ret.(osFile), expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}

func (k *kstub) writeFile(name string, data []byte, perm os.FileMode) error {
	k.Helper()
	return k.Expects("writeFile").Error(
		stub.CheckArg(k.Stub, "name", name, 0),
		stub.CheckArgReflect(k.Stub, "data", data, 1),
		stub.CheckArg(k.Stub, "perm", perm, 2))
}

func (k *kstub) createTemp(dir, pattern string) (osFile, error) {
	k.Helper()
	expect := k.Expects("createTemp")
	return expect.Ret.(osFile), expect.Error(
		stub.CheckArg(k.Stub, "dir", dir, 0),
		stub.CheckArg(k.Stub, "pattern", pattern, 1))
}

func (k *kstub) remove(name string) error {
	k.Helper()
	return k.Expects("remove").Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}

func (k *kstub) newFile(fd uintptr, name string) *os.File {
	k.Helper()
	expect := k.Expects("newFile")
	if expect.Error(
		stub.CheckArg(k.Stub, "fd", fd, 0),
		stub.CheckArg(k.Stub, "name", name, 1)) != nil {
		k.FailNow()
	}
	return expect.Ret.(*os.File)
}

func (k *kstub) symlink(oldname, newname string) error {
	k.Helper()
	return k.Expects("symlink").Error(
		stub.CheckArg(k.Stub, "oldname", oldname, 0),
		stub.CheckArg(k.Stub, "newname", newname, 1))
}

func (k *kstub) readlink(name string) (string, error) {
	k.Helper()
	expect := k.Expects("readlink")
	return expect.Ret.(string), expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}

func (k *kstub) umask(mask int) (oldmask int) {
	k.Helper()
	expect := k.Expects("umask")
	if !stub.CheckArg(k.Stub, "mask", mask, 0) {
		k.FailNow()
	}
	return expect.Ret.(int)
}

func (k *kstub) sethostname(p []byte) (err error) {
	k.Helper()
	return k.Expects("sethostname").Error(
		stub.CheckArgReflect(k.Stub, "p", p, 0))
}

func (k *kstub) chdir(path string) (err error) {
	k.Helper()
	return k.Expects("chdir").Error(
		stub.CheckArg(k.Stub, "path", path, 0))
}

func (k *kstub) fchdir(fd int) (err error) {
	k.Helper()
	return k.Expects("fchdir").Error(
		stub.CheckArg(k.Stub, "fd", fd, 0))
}

func (k *kstub) open(path string, mode int, perm uint32) (fd int, err error) {
	k.Helper()
	expect := k.Expects("open")
	return expect.Ret.(int), expect.Error(
		stub.CheckArg(k.Stub, "path", path, 0),
		stub.CheckArg(k.Stub, "mode", mode, 1),
		stub.CheckArg(k.Stub, "perm", perm, 2))
}

func (k *kstub) close(fd int) (err error) {
	k.Helper()
	return k.Expects("close").Error(
		stub.CheckArg(k.Stub, "fd", fd, 0))
}

func (k *kstub) pivotRoot(newroot, putold string) (err error) {
	k.Helper()
	return k.Expects("pivotRoot").Error(
		stub.CheckArg(k.Stub, "newroot", newroot, 0),
		stub.CheckArg(k.Stub, "putold", putold, 1))
}

func (k *kstub) mount(source, target, fstype string, flags uintptr, data string) (err error) {
	k.Helper()
	return k.Expects("mount").Error(
		stub.CheckArg(k.Stub, "source", source, 0),
		stub.CheckArg(k.Stub, "target", target, 1),
		stub.CheckArg(k.Stub, "fstype", fstype, 2),
		stub.CheckArg(k.Stub, "flags", flags, 3),
		stub.CheckArg(k.Stub, "data", data, 4))
}

func (k *kstub) unmount(target string, flags int) (err error) {
	k.Helper()
	return k.Expects("unmount").Error(
		stub.CheckArg(k.Stub, "target", target, 0),
		stub.CheckArg(k.Stub, "flags", flags, 1))
}

func (k *kstub) wait4(pid int, wstatus *syscall.WaitStatus, options int, rusage *syscall.Rusage) (wpid int, err error) {
	k.Helper()
	expect := k.Expects("wait4")
	if v, ok := expect.Args[4].(int); ok {
		switch v {
		case stub.PanicExit: // special case to prevent leaking the wait4 goroutine while testing initEntrypoint
			panic(stub.PanicExit)

		case magicWait4Signal: // block until corresponding signal call
			if k.wait4signal == nil {
				panic("kstub not initialised for wait4 simulation")
			}
			<-k.wait4signal
		}
	}

	wpid = expect.Ret.(int)
	err = expect.Error(
		stub.CheckArg(k.Stub, "pid", pid, 0),
		stub.CheckArg(k.Stub, "options", options, 2))

	if wstatusV, ok := expect.Args[1].(syscall.WaitStatus); wstatus != nil && ok {
		*wstatus = wstatusV
	}
	if rusageV, ok := expect.Args[3].(syscall.Rusage); rusage != nil && ok {
		*rusage = rusageV
	}

	return
}

func (k *kstub) printf(format string, v ...any) {
	k.Helper()
	if k.Expects("printf").Error(
		stub.CheckArg(k.Stub, "format", format, 0),
		stub.CheckArgReflect(k.Stub, "v", v, 1)) != nil {
		k.FailNow()
	}
}

func (k *kstub) fatal(v ...any) {
	k.Helper()
	if k.Expects("fatal").Error(
		stub.CheckArgReflect(k.Stub, "v", v, 0)) != nil {
		k.FailNow()
	}
	panic(stub.PanicExit)
}

func (k *kstub) fatalf(format string, v ...any) {
	k.Helper()
	if k.Expects("fatalf").Error(
		stub.CheckArg(k.Stub, "format", format, 0),
		stub.CheckArgReflect(k.Stub, "v", v, 1)) != nil {
		k.FailNow()
	}
	panic(stub.PanicExit)
}

func (k *kstub) verbose(v ...any) {
	k.Helper()
	if k.Expects("verbose").Error(
		stub.CheckArgReflect(k.Stub, "v", v, 0)) != nil {
		k.FailNow()
	}
}

func (k *kstub) verbosef(format string, v ...any) {
	k.Helper()
	if k.Expects("verbosef").Error(
		stub.CheckArg(k.Stub, "format", format, 0),
		stub.CheckArgReflect(k.Stub, "v", v, 1)) != nil {
		k.FailNow()
	}
}

func (k *kstub) suspend()     { k.Helper(); k.Expects("suspend") }
func (k *kstub) resume() bool { k.Helper(); return k.Expects("resume").Ret.(bool) }
func (k *kstub) beforeExit()  { k.Helper(); k.Expects("beforeExit") }
