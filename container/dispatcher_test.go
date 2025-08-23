package container

import (
	"bytes"
	"errors"
	"fmt"
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
)

var errUnique = errors.New("unique error injected by the test suite")

type opValidTestCase struct {
	name string
	op   Op
	want bool
}

func checkOpsValid(t *testing.T, testCases []opValidTestCase) {
	t.Run("valid", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
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
	t.Run("build", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
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
	t.Run("is", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
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
	t.Run("meta", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Run("prefix", func(t *testing.T) {
					if got := tc.op.prefix(); got != tc.wantPrefix {
						t.Errorf("prefix: %q, want %q", got, tc.wantPrefix)
					}
				})

				t.Run("string", func(t *testing.T) {
					if got := tc.op.String(); got != tc.wantString {
						t.Errorf("String: %s, want %s", got, tc.wantString)
					}
				})
			})
		}
	})
}

type simpleTestCase struct {
	name    string
	f       func(k syscallDispatcher) error
	want    [][]kexpect
	wantErr error
}

func checkSimple(t *testing.T, fname string, testCases []simpleTestCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k := &kstub{t: t, want: tc.want}
			if err := tc.f(k); !errors.Is(err, tc.wantErr) {
				t.Errorf("%s: error = %v, want %v", fname, err, tc.wantErr)
			}
			k.handleIncomplete(func(k *kstub) {
				t.Errorf("%s: %d calls, want %d (track %d)", fname, k.pos, len(k.want[k.track]), k.track)
			})
		})
	}
}

type opBehaviourTestCase struct {
	name   string
	params *Params
	op     Op

	early        []kexpect
	wantErrEarly error

	apply        []kexpect
	wantErrApply error
}

func checkOpBehaviour(t *testing.T, testCases []opBehaviourTestCase) {
	t.Run("behaviour", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				state := &setupState{Params: tc.params}
				k := &kstub{t: t, want: [][]kexpect{slices.Concat(tc.early, []kexpect{{name: "\x00"}}, tc.apply)}}
				errEarly := tc.op.early(state, k)
				k.expect("\x00")
				if !errors.Is(errEarly, tc.wantErrEarly) {
					t.Errorf("early: error = %v, want %v", errEarly, tc.wantErrEarly)
				}
				if errEarly != nil {
					goto out
				}

				if err := tc.op.apply(state, k); !errors.Is(err, tc.wantErrApply) {
					t.Errorf("apply: error = %v, want %v", err, tc.wantErrApply)
				}

			out:
				k.handleIncomplete(func(k *kstub) {
					count := k.pos - 1 // separator
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

type expectArgs = [5]any

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

type kexpect struct {
	name string
	args expectArgs
	ret  any
	err  error
}

func (k *kexpect) error(ok ...bool) error {
	if !slices.Contains(ok, false) {
		return k.err
	}
	return syscall.ENOTRECOVERABLE
}

type kstub struct {
	t *testing.T

	want [][]kexpect
	// pos is the current position in want[track].
	pos int
	// track is the current active want.
	track int
	// sub stores addresses of kstub created by new.
	sub []*kstub
}

// handleIncomplete calls f on an incomplete k and all its descendants.
func (k *kstub) handleIncomplete(f func(k *kstub)) {
	if k.want != nil && len(k.want[k.track]) != k.pos {
		f(k)
	}
	for _, sk := range k.sub {
		sk.handleIncomplete(f)
	}
}

// expect checks name and returns the current kexpect and advances pos.
func (k *kstub) expect(name string) (expect *kexpect) {
	if len(k.want[k.track]) == k.pos {
		k.t.Fatal("expect: want too short")
	}
	expect = &k.want[k.track][k.pos]
	if name != expect.name {
		if expect.name == "\x00" {
			k.t.Fatalf("expect: func = %s, separator overrun", name)
		}
		if name == "\x00" {
			k.t.Fatalf("expect: separator, want %s", expect.name)
		}
		k.t.Fatalf("expect: func = %s, want %s", name, expect.name)
	}
	k.pos++
	return
}

// checkArg checks an argument comparable with the == operator. Avoid using this with pointers.
func checkArg[T comparable](k *kstub, arg string, got T, n int) bool {
	if k.pos == 0 {
		panic("invalid call to checkArg")
	}
	expect := k.want[k.track][k.pos-1]
	want, ok := expect.args[n].(T)
	if !ok || got != want {
		k.t.Errorf("%s: %s = %#v, want %#v (%d)", expect.name, arg, got, want, k.pos-1)
		return false
	}
	return true
}

// checkArgReflect checks an argument of any type.
func checkArgReflect(k *kstub, arg string, got any, n int) bool {
	if k.pos == 0 {
		panic("invalid call to checkArgReflect")
	}
	expect := k.want[k.track][k.pos-1]
	want := expect.args[n]
	if !reflect.DeepEqual(got, want) {
		k.t.Errorf("%s: %s = %#v, want %#v (%d)", expect.name, arg, got, want, k.pos-1)
		return false
	}
	return true
}

func (k *kstub) new() syscallDispatcher {
	k.expect("new")
	if len(k.want) <= k.track+1 {
		k.t.Fatalf("new: track overrun")
	}
	k.sub = append(k.sub, &kstub{t: k.t, want: k.want, track: k.track + 1})
	return k.sub[len(k.sub)-1]
}

func (k *kstub) lockOSThread() { k.expect("lockOSThread") }

func (k *kstub) setPtracer(pid uintptr) error {
	return k.expect("setPtracer").error(
		checkArg(k, "pid", pid, 0))
}

func (k *kstub) setDumpable(dumpable uintptr) error {
	return k.expect("setDumpable").error(
		checkArg(k, "dumpable", dumpable, 0))
}

func (k *kstub) setNoNewPrivs() error { return k.expect("setNoNewPrivs").err }
func (k *kstub) lastcap() uintptr     { return k.expect("lastcap").ret.(uintptr) }

func (k *kstub) capset(hdrp *capHeader, datap *[2]capData) error {
	return k.expect("capset").error(
		checkArgReflect(k, "hdrp", hdrp, 0),
		checkArgReflect(k, "datap", datap, 1))
}

func (k *kstub) capBoundingSetDrop(cap uintptr) error {
	return k.expect("capBoundingSetDrop").error(
		checkArg(k, "cap", cap, 0))
}

func (k *kstub) capAmbientClearAll() error { return k.expect("capAmbientClearAll").err }

func (k *kstub) capAmbientRaise(cap uintptr) error {
	return k.expect("capAmbientRaise").error(
		checkArg(k, "cap", cap, 0))
}

func (k *kstub) isatty(fd int) bool {
	expect := k.expect("isatty")
	if !checkArg(k, "fd", fd, 0) {
		k.t.FailNow()
	}
	return expect.ret.(bool)
}

func (k *kstub) receive(key string, e any, fdp *uintptr) (closeFunc func() error, err error) {
	expect := k.expect("receive")
	return expect.ret.(func() error), expect.error(
		checkArg(k, "key", key, 0),
		checkArgReflect(k, "e", e, 1),
		checkArg(k, "fdp", fdp, 2))
}

func (k *kstub) bindMount(source, target string, flags uintptr, eq bool) error {
	return k.expect("bindMount").error(
		checkArg(k, "source", source, 0),
		checkArg(k, "target", target, 1),
		checkArg(k, "flags", flags, 2),
		checkArg(k, "eq", eq, 3))
}

func (k *kstub) remount(target string, flags uintptr) error {
	return k.expect("remount").error(
		checkArg(k, "target", target, 0),
		checkArg(k, "flags", flags, 1))
}

func (k *kstub) mountTmpfs(fsname, target string, flags uintptr, size int, perm os.FileMode) error {
	return k.expect("mountTmpfs").error(
		checkArg(k, "fsname", fsname, 0),
		checkArg(k, "target", target, 1),
		checkArg(k, "flags", flags, 2),
		checkArg(k, "size", size, 3),
		checkArg(k, "perm", perm, 4))
}

func (k *kstub) ensureFile(name string, perm, pperm os.FileMode) error {

	return k.expect("ensureFile").error(
		checkArg(k, "name", name, 0),
		checkArg(k, "perm", perm, 1),
		checkArg(k, "pperm", pperm, 2))
}

func (k *kstub) seccompLoad(rules []seccomp.NativeRule, flags seccomp.ExportFlag) error {
	return k.expect("seccompLoad").error(
		checkArgReflect(k, "rules", rules, 0),
		checkArg(k, "flags", flags, 1))
}

func (k *kstub) notify(c chan<- os.Signal, sig ...os.Signal) {
	expect := k.expect("notify")
	if c == nil || expect.error(
		checkArgReflect(k, "sig", sig, 1)) != nil {
		k.t.FailNow()
	}

	// export channel for external instrumentation
	if chanp, ok := expect.args[0].(*chan<- os.Signal); ok && chanp != nil {
		if *chanp != nil {
			panic(fmt.Sprintf("attempting reuse of %p", chanp))
		}
		*chanp = c
	}
}

func (k *kstub) start(c *exec.Cmd) error {
	return k.expect("start").error(
		checkArg(k, "c.Path", c.Path, 0),
		checkArgReflect(k, "c.Args", c.Args, 1),
		checkArgReflect(k, "c.Env", c.Env, 2),
		checkArg(k, "c.Dir", c.Dir, 3))
}

func (k *kstub) signal(c *exec.Cmd, sig os.Signal) error {
	return k.expect("signal").error(
		checkArg(k, "c.Path", c.Path, 0),
		checkArgReflect(k, "c.Args", c.Args, 1),
		checkArgReflect(k, "c.Env", c.Env, 2),
		checkArg(k, "c.Dir", c.Dir, 3),
		checkArg(k, "sig", sig, 4))
}

func (k *kstub) evalSymlinks(path string) (string, error) {
	expect := k.expect("evalSymlinks")
	return expect.ret.(string), expect.error(
		checkArg(k, "path", path, 0))
}

func (k *kstub) exit(code int) {
	k.expect("exit")
	if !checkArg(k, "code", code, 0) {
		k.t.FailNow()
	}
}

func (k *kstub) getpid() int { return k.expect("getpid").ret.(int) }

func (k *kstub) stat(name string) (os.FileInfo, error) {
	expect := k.expect("stat")
	return expect.ret.(os.FileInfo), expect.error(
		checkArg(k, "name", name, 0))
}

func (k *kstub) mkdir(name string, perm os.FileMode) error {
	return k.expect("mkdir").error(
		checkArg(k, "name", name, 0),
		checkArg(k, "perm", perm, 1))
}

func (k *kstub) mkdirTemp(dir, pattern string) (string, error) {
	expect := k.expect("mkdirTemp")
	return expect.ret.(string), expect.error(
		checkArg(k, "dir", dir, 0),
		checkArg(k, "pattern", pattern, 1))
}

func (k *kstub) mkdirAll(path string, perm os.FileMode) error {
	return k.expect("mkdirAll").error(
		checkArg(k, "path", path, 0),
		checkArg(k, "perm", perm, 1))
}

func (k *kstub) readdir(name string) ([]os.DirEntry, error) {
	expect := k.expect("readdir")
	return expect.ret.([]os.DirEntry), expect.error(
		checkArg(k, "name", name, 0))
}

func (k *kstub) openNew(name string) (osFile, error) {
	expect := k.expect("openNew")
	return expect.ret.(osFile), expect.error(
		checkArg(k, "name", name, 0))
}

func (k *kstub) writeFile(name string, data []byte, perm os.FileMode) error {
	return k.expect("writeFile").error(
		checkArg(k, "name", name, 0),
		checkArgReflect(k, "data", data, 1),
		checkArg(k, "perm", perm, 2))
}

func (k *kstub) createTemp(dir, pattern string) (osFile, error) {
	expect := k.expect("createTemp")
	return expect.ret.(osFile), expect.error(
		checkArg(k, "dir", dir, 0),
		checkArg(k, "pattern", pattern, 1))
}

func (k *kstub) remove(name string) error {
	return k.expect("remove").error(
		checkArg(k, "name", name, 0))
}

func (k *kstub) newFile(fd uintptr, name string) *os.File {
	expect := k.expect("newFile")
	if expect.error(
		checkArg(k, "fd", fd, 0),
		checkArg(k, "name", name, 1)) != nil {
		k.t.FailNow()
	}
	return expect.ret.(*os.File)
}

func (k *kstub) symlink(oldname, newname string) error {
	return k.expect("symlink").error(
		checkArg(k, "oldname", oldname, 0),
		checkArg(k, "newname", newname, 1))
}

func (k *kstub) readlink(name string) (string, error) {
	expect := k.expect("readlink")
	return expect.ret.(string), expect.error(
		checkArg(k, "name", name, 0))
}

func (k *kstub) umask(mask int) (oldmask int) {
	expect := k.expect("umask")
	if !checkArg(k, "mask", mask, 0) {
		k.t.FailNow()
	}
	return expect.ret.(int)
}

func (k *kstub) sethostname(p []byte) (err error) {
	return k.expect("sethostname").error(
		checkArgReflect(k, "p", p, 0))
}

func (k *kstub) chdir(path string) (err error) {
	return k.expect("chdir").error(
		checkArg(k, "path", path, 0))
}

func (k *kstub) fchdir(fd int) (err error) {
	return k.expect("fchdir").error(
		checkArg(k, "fd", fd, 0))
}

func (k *kstub) open(path string, mode int, perm uint32) (fd int, err error) {
	expect := k.expect("open")
	return expect.ret.(int), expect.error(
		checkArg(k, "path", path, 0),
		checkArg(k, "mode", mode, 1),
		checkArg(k, "perm", perm, 2))
}

func (k *kstub) close(fd int) (err error) {
	return k.expect("close").error(
		checkArg(k, "fd", fd, 0))
}

func (k *kstub) pivotRoot(newroot, putold string) (err error) {
	return k.expect("pivotRoot").error(
		checkArg(k, "newroot", newroot, 0),
		checkArg(k, "putold", putold, 1))
}

func (k *kstub) mount(source, target, fstype string, flags uintptr, data string) (err error) {
	return k.expect("mount").error(
		checkArg(k, "source", source, 0),
		checkArg(k, "target", target, 1),
		checkArg(k, "fstype", fstype, 2),
		checkArg(k, "flags", flags, 3),
		checkArg(k, "data", data, 4))
}

func (k *kstub) unmount(target string, flags int) (err error) {
	return k.expect("unmount").error(
		checkArg(k, "target", target, 0),
		checkArg(k, "flags", flags, 1))
}

func (k *kstub) wait4(pid int, wstatus *syscall.WaitStatus, options int, rusage *syscall.Rusage) (wpid int, err error) {
	expect := k.expect("wait4")
	return expect.ret.(int), expect.error(
		checkArg(k, "pid", pid, 0),
		checkArg(k, "wstatus", wstatus, 1),
		checkArg(k, "options", options, 2),
		checkArg(k, "rusage", rusage, 3))
}

func (k *kstub) printf(format string, v ...any) {
	if k.expect("printf").error(
		checkArg(k, "format", format, 0),
		checkArgReflect(k, "v", v, 1)) != nil {
		k.t.FailNow()
	}
}

func (k *kstub) fatal(v ...any) {
	if k.expect("fatal").error(
		checkArgReflect(k, "v", v, 0)) != nil {
		k.t.FailNow()
	}
}

func (k *kstub) fatalf(format string, v ...any) {
	if k.expect("fatalf").error(
		checkArg(k, "format", format, 0),
		checkArgReflect(k, "v", v, 1)) != nil {
		k.t.FailNow()
	}
}

func (k *kstub) verbose(v ...any) {
	if k.expect("verbose").error(
		checkArgReflect(k, "v", v, 0)) != nil {
		k.t.FailNow()
	}
}

func (k *kstub) verbosef(format string, v ...any) {
	if k.expect("verbosef").error(
		checkArg(k, "format", format, 0),
		checkArgReflect(k, "v", v, 1)) != nil {
		k.t.FailNow()
	}
}

func (k *kstub) suspend()     { k.expect("suspend") }
func (k *kstub) resume() bool { return k.expect("resume").ret.(bool) }
func (k *kstub) beforeExit()  { k.expect("beforeExit") }

func (k *kstub) printBaseErr(err error, fallback string) {
	if k.expect("printBaseErr").error(
		checkArgReflect(k, "err", err, 0),
		checkArg(k, "fallback", fallback, 1)) != nil {
		k.t.FailNow()
	}
}
