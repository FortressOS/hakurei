package system

import (
	"io"
	"io/fs"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"
	"unsafe"

	"hakurei.app/container/stub"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
	"hakurei.app/system/internal/xcb"
)

// call initialises a [stub.Call].
// This keeps composites analysis happy without making the test cases too bloated.
func call(name string, args stub.ExpectArgs, ret any, err error) stub.Call {
	return stub.Call{Name: name, Args: args, Ret: ret, Err: err}
}

type opBehaviourTestCase struct {
	name string
	uid  int
	ec   Enablement
	op   Op

	apply        []stub.Call
	wantErrApply error

	revert        []stub.Call
	wantErrRevert error
}

func checkOpBehaviour(t *testing.T, testCases []opBehaviourTestCase) {
	t.Helper()

	t.Run("behaviour", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				var ec *Criteria
				if tc.ec != 0xff {
					ec = (*Criteria)(&tc.ec)
				}

				sys, s := InternalNew(t, stub.Expect{Calls: slices.Concat(tc.apply, []stub.Call{{Name: stub.CallSeparator}}, tc.revert)}, tc.uid)
				defer stub.HandleExit(t)
				errApply := tc.op.apply(sys)
				s.Expects(stub.CallSeparator)
				if !reflect.DeepEqual(errApply, tc.wantErrApply) {
					t.Errorf("apply: error = %v, want %v", errApply, tc.wantErrApply)
				}
				if errApply != nil {
					goto out
				}

				if err := tc.op.revert(sys, ec); !reflect.DeepEqual(err, tc.wantErrRevert) {
					t.Errorf("revert: error = %v, want %v", err, tc.wantErrRevert)
				}

			out:
				s.VisitIncomplete(func(s *stub.Stub[syscallDispatcher]) {
					count := s.Pos() - 1 // separator
					if count < len(tc.apply) {
						t.Errorf("apply: %d calls, want %d", count, len(tc.apply))
					} else {
						t.Errorf("revert: %d calls, want %d", count-len(tc.apply), len(tc.revert))
					}
				})
			})
		}
	})
}

type opsBuilderTestCase struct {
	name string
	uid  int
	f    func(t *testing.T, sys *I)
	want []Op
	exp  stub.Expect
}

func checkOpsBuilder(t *testing.T, fname string, testCases []opsBuilderTestCase) {
	t.Helper()

	t.Run("build", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				sys, s := InternalNew(t, tc.exp, tc.uid)
				defer stub.HandleExit(t)
				tc.f(t, sys)
				s.VisitIncomplete(func(s *stub.Stub[syscallDispatcher]) {
					t.Helper()

					t.Errorf("%s: %d calls, want %d", fname, s.Pos(), s.Len())
				})
				if !slices.EqualFunc(sys.ops, tc.want, func(op Op, v Op) bool { return op.Is(v) }) {
					t.Errorf("ops: %#v, want %#v", sys.ops, tc.want)
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

	wantType   Enablement
	wantPath   string
	wantString string
}

func checkOpMeta(t *testing.T, testCases []opMetaTestCase) {
	t.Helper()

	t.Run("meta", func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Helper()

				t.Run("type", func(t *testing.T) {
					t.Helper()

					if got := tc.op.Type(); got != tc.wantType {
						t.Errorf("Type: %q, want %q", got, tc.wantType)
					}
				})

				t.Run("path", func(t *testing.T) {
					t.Helper()

					if got := tc.op.Path(); got != tc.wantPath {
						t.Errorf("Path: %q, want %q", got, tc.wantPath)
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

type stubFi struct {
	size  int64
	isDir bool
}

func (stubFi) Name() string       { panic("unreachable") }
func (fi stubFi) Size() int64     { return fi.size }
func (stubFi) Mode() fs.FileMode  { panic("unreachable") }
func (stubFi) ModTime() time.Time { panic("unreachable") }
func (fi stubFi) IsDir() bool     { return fi.isDir }
func (stubFi) Sys() any           { panic("unreachable") }

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

// InternalNew initialises [I] with a stub syscallDispatcher.
func InternalNew(t *testing.T, want stub.Expect, uid int) (*I, *stub.Stub[syscallDispatcher]) {
	k := stub.New(t, func(s *stub.Stub[syscallDispatcher]) syscallDispatcher { return &kstub{s} }, want)
	sys := New(t.Context(), uid)
	sys.syscallDispatcher = &kstub{k}
	return sys, k
}

type kstub struct{ *stub.Stub[syscallDispatcher] }

func (k *kstub) new(f func(k syscallDispatcher)) { k.Helper(); k.New(f) }

func (k *kstub) stat(name string) (fi os.FileInfo, err error) {
	k.Helper()
	expect := k.Expects("stat")
	err = expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
	if err == nil {
		fi = expect.Ret.(os.FileInfo)
	}
	return
}

func (k *kstub) open(name string) (f osFile, err error) {
	k.Helper()
	expect := k.Expects("open")
	err = expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
	if err == nil {
		f = expect.Ret.(osFile)
	}
	return
}

func (k *kstub) mkdir(name string, perm os.FileMode) error {
	k.Helper()
	return k.Expects("mkdir").Error(
		stub.CheckArg(k.Stub, "name", name, 0),
		stub.CheckArg(k.Stub, "perm", perm, 1))
}

func (k *kstub) chmod(name string, mode os.FileMode) error {
	k.Helper()
	return k.Expects("chmod").Error(
		stub.CheckArg(k.Stub, "name", name, 0),
		stub.CheckArg(k.Stub, "mode", mode, 1))
}

func (k *kstub) link(oldname, newname string) error {
	k.Helper()
	return k.Expects("link").Error(
		stub.CheckArg(k.Stub, "oldname", oldname, 0),
		stub.CheckArg(k.Stub, "newname", newname, 1))
}

func (k *kstub) remove(name string) error {
	k.Helper()
	return k.Expects("remove").Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}

func (k *kstub) aclUpdate(name string, uid int, perms ...acl.Perm) error {
	k.Helper()
	return k.Expects("aclUpdate").Error(
		stub.CheckArg(k.Stub, "name", name, 0),
		stub.CheckArg(k.Stub, "uid", uid, 1),
		stub.CheckArgReflect(k.Stub, "perms", perms, 2))
}

func (k *kstub) xcbChangeHosts(mode xcb.HostMode, family xcb.Family, address string) error {
	k.Helper()
	return k.Expects("xcbChangeHosts").Error(
		stub.CheckArg(k.Stub, "mode", mode, 0),
		stub.CheckArg(k.Stub, "family", family, 1),
		stub.CheckArg(k.Stub, "address", address, 2))
}

func (k *kstub) dbusAddress() (session, system string) {
	k.Helper()
	ret := k.Expects("dbusAddress").Ret.([2]string)
	return ret[0], ret[1]
}

func (k *kstub) dbusFinalise(sessionBus, systemBus dbus.ProxyPair, session, system *dbus.Config) (final *dbus.Final, err error) {
	k.Helper()
	expect := k.Expects("dbusFinalise")

	final = expect.Ret.(*dbus.Final)
	err = expect.Error(
		stub.CheckArg(k.Stub, "sessionBus", sessionBus, 0),
		stub.CheckArg(k.Stub, "systemBus", systemBus, 1),
		stub.CheckArgReflect(k.Stub, "session", session, 2),
		stub.CheckArgReflect(k.Stub, "system", system, 3))
	if err != nil {
		final = nil
	}
	return
}

func (k *kstub) dbusProxyStart(proxy *dbus.Proxy) error {
	k.Helper()
	return k.dbusProxySCW(k.Expects("dbusProxyStart"), proxy)
}
func (k *kstub) dbusProxyClose(proxy *dbus.Proxy) {
	k.Helper()
	if k.dbusProxySCW(k.Expects("dbusProxyClose"), proxy) != nil {
		k.Fail()
	}
}
func (k *kstub) dbusProxyWait(proxy *dbus.Proxy) error {
	k.Helper()
	return k.dbusProxySCW(k.Expects("dbusProxyWait"), proxy)
}
func (k *kstub) dbusProxySCW(expect *stub.Call, proxy *dbus.Proxy) error {
	k.Helper()
	v := reflect.ValueOf(proxy).Elem()

	if ctxV := v.FieldByName("ctx"); ctxV.IsNil() {
		k.Errorf("proxy: ctx = %s", ctxV.String())
		return os.ErrInvalid
	}

	finalV := v.FieldByName("final")
	if gotFinal := reflect.NewAt(finalV.Type(), unsafe.Pointer(finalV.UnsafeAddr())).Elem().Interface().(*dbus.Final); !reflect.DeepEqual(gotFinal, expect.Args[0]) {
		k.Errorf("proxy: final = %#v, want %#v", gotFinal, expect.Args[0])
		return os.ErrInvalid
	}

	outputV := v.FieldByName("output")
	if _, ok := reflect.NewAt(outputV.Type(), unsafe.Pointer(outputV.UnsafeAddr())).Elem().Interface().(*linePrefixWriter); !ok {
		k.Errorf("proxy: output = %s", outputV.String())
		return os.ErrInvalid
	}

	return expect.Err
}

func (k *kstub) isVerbose() bool { k.Helper(); return k.Expects("isVerbose").Ret.(bool) }

// ignoreValue marks a value to be ignored by the test suite.
type ignoreValue struct{}

func (k *kstub) verbose(v ...any) {
	k.Helper()
	expect := k.Expects("verbose")

	// translate ignores in v
	if want, ok := expect.Args[0].([]any); ok && len(v) == len(want) {
		for i, a := range want {
			if _, ok = a.(ignoreValue); ok {
				v[i] = ignoreValue{}
			}
		}
	}

	if expect.Error(
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
