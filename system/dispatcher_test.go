package system

import (
	"reflect"
	"slices"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/system/acl"
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

				defer stub.HandleExit()
				sys, s := InternalNew(t, stub.Expect{Calls: slices.Concat(tc.apply, []stub.Call{{Name: stub.CallSeparator}}, tc.revert)}, tc.uid)
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
	f    func(sys *I)
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

				defer stub.HandleExit()
				sys, s := InternalNew(t, tc.exp, tc.uid)
				tc.f(sys)
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

// InternalNew initialises [I] with a stub syscallDispatcher.
func InternalNew(t *testing.T, want stub.Expect, uid int) (*I, *stub.Stub[syscallDispatcher]) {
	k := stub.New(t, func(s *stub.Stub[syscallDispatcher]) syscallDispatcher { return &kstub{s} }, want)
	sys := New(t.Context(), uid)
	sys.syscallDispatcher = &kstub{k}
	return sys, k
}

type kstub struct{ *stub.Stub[syscallDispatcher] }

func (k *kstub) new(f func(k syscallDispatcher)) { k.Helper(); k.New(f) }

func (k *kstub) aclUpdate(name string, uid int, perms ...acl.Perm) error {
	k.Helper()
	return k.Expects("aclUpdate").Error(
		stub.CheckArg(k.Stub, "name", name, 0),
		stub.CheckArg(k.Stub, "uid", uid, 1),
		stub.CheckArgReflect(k.Stub, "perms", perms, 2))
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
