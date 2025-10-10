package app

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"testing"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/message"
	"hakurei.app/system"
)

// call initialises a [stub.Call].
// This keeps composites analysis happy without making the test cases too bloated.
func call(name string, args stub.ExpectArgs, ret any, err error) stub.Call {
	return stub.Call{Name: name, Args: args, Ret: ret, Err: err}
}

// checkExpectUid is the uid value used by checkOpBehaviour to initialise [system.I].
const checkExpectUid = 0xcafebabe

// checkExpectInstanceId is the [state.ID] value used by checkOpBehaviour to initialise outcomeState.
var checkExpectInstanceId = *(*state.ID)(bytes.Repeat([]byte{0xaa}, len(state.ID{})))

type opBehaviourTestCase struct {
	name      string
	newOp     func(isShim, clearUnexported bool) outcomeOp
	newConfig func() *hst.Config

	pStateSys     func(state *outcomeStateSys)
	toSystem      []stub.Call
	wantSys       *system.I
	extraCheckSys func(t *testing.T, state *outcomeStateSys)
	wantErrSystem error

	pStateContainer  func(state *outcomeStateParams)
	toContainer      []stub.Call
	wantParams       *container.Params
	extraCheckParams func(t *testing.T, state *outcomeStateParams)
	wantErrContainer error
}

func checkOpBehaviour(t *testing.T, testCases []opBehaviourTestCase) {
	t.Helper()

	wantNewState := []stub.Call{
		// newOutcomeState
		call("getpid", stub.ExpectArgs{}, 0xdead, nil),
		call("isVerbose", stub.ExpectArgs{}, true, nil),
		call("mustHsuPath", stub.ExpectArgs{}, m(container.Nonexistent), nil),
		call("cmdOutput", stub.ExpectArgs{container.Nonexistent, os.Stderr, []string{}, "/"}, []byte("0"), nil),
		call("tempdir", stub.ExpectArgs{}, container.Nonexistent+"/tmp", nil),
		call("lookupEnv", stub.ExpectArgs{"XDG_RUNTIME_DIR"}, container.Nonexistent+"/xdg_runtime_dir", nil),
		call("getuid", stub.ExpectArgs{}, 1000, nil),
		call("getgid", stub.ExpectArgs{}, 100, nil),

		// populateLocal
		call("verbosef", stub.ExpectArgs{"process share directory at %q, runtime directory at %q", []any{
			m(container.Nonexistent + "/tmp/hakurei.0"),
			m(container.Nonexistent + "/xdg_runtime_dir/hakurei"),
		}}, nil, nil),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			wantCallsFull := slices.Concat(wantNewState, tc.toSystem, []stub.Call{{Name: stub.CallSeparator}})
			if tc.wantErrSystem == nil {
				wantCallsFull = append(wantCallsFull, slices.Concat(wantNewState, tc.toContainer)...)
			}

			wantConfig := tc.newConfig()
			k := &kstub{panicDispatcher{}, stub.New(t,
				func(s *stub.Stub[syscallDispatcher]) syscallDispatcher { return &kstub{panicDispatcher{}, s} },
				stub.Expect{Calls: wantCallsFull},
			)}
			defer stub.HandleExit(t)

			{
				config := tc.newConfig()
				s := newOutcomeState(k, k, &checkExpectInstanceId, config, &Hsu{k: k})
				if err := s.populateLocal(k, k); err != nil {
					t.Fatalf("populateLocal: error = %v", err)
				}
				stateSys := s.newSys(config, newI())
				if tc.pStateSys != nil {
					tc.pStateSys(stateSys)
				}
				op := tc.newOp(false, true)

				if err := op.toSystem(stateSys); !reflect.DeepEqual(err, tc.wantErrSystem) {
					t.Errorf("toSystem: error = %v, want %v", err, tc.wantErrSystem)
				}
				k.Expects(stub.CallSeparator)
				if !reflect.DeepEqual(config, wantConfig) {
					t.Errorf("toSystem clobbered config: %#v, want %#v", config, wantConfig)
				}

				if tc.wantErrSystem != nil {
					goto out
				}

				if !stateSys.sys.Equal(tc.wantSys) {
					t.Errorf("toSystem: %#v, want %#v", stateSys.sys, tc.wantSys)
				}
				if tc.extraCheckSys != nil {
					tc.extraCheckSys(t, stateSys)
				}
				if wantOpSys := tc.newOp(true, false); !reflect.DeepEqual(op, wantOpSys) {
					t.Errorf("toSystem: op = %#v, want %#v", op, wantOpSys)
				}
			}

			{
				config := tc.newConfig()
				s := newOutcomeState(k, k, &checkExpectInstanceId, config, &Hsu{k: k})
				stateParams := s.newParams()
				if err := s.populateLocal(k, k); err != nil {
					t.Fatalf("populateLocal: error = %v", err)
				}
				if tc.pStateContainer != nil {
					tc.pStateContainer(stateParams)
				}
				op := tc.newOp(true, true)

				if err := op.toContainer(stateParams); !reflect.DeepEqual(err, tc.wantErrContainer) {
					t.Errorf("toContainer: error = %v, want %v", err, tc.wantErrContainer)
				}

				if tc.wantErrContainer != nil {
					goto out
				}

				if !reflect.DeepEqual(stateParams.params, tc.wantParams) {
					t.Errorf("toContainer:\n%s\nwant\n%s", mustMarshal(stateParams.params), mustMarshal(tc.wantParams))
				}
				if tc.extraCheckParams != nil {
					tc.extraCheckParams(t, stateParams)
				}
			}

		out:
			k.VisitIncomplete(func(s *stub.Stub[syscallDispatcher]) {
				count := k.Pos() - 1 // separator
				if count-len(wantNewState) < len(tc.toSystem) {
					t.Errorf("toSystem: %d calls, want %d", count-len(wantNewState), len(tc.toSystem))
				} else {
					t.Errorf("toContainer: %d calls, want %d", count-len(tc.toSystem)-2*len(wantNewState), len(tc.toContainer))
				}
			})
		})
	}
}

func newI() *system.I { return system.New(panicMsgContext{}, panicMsgContext{}, checkExpectUid) }

type kstub struct {
	panicDispatcher
	*stub.Stub[syscallDispatcher]
}

func (k *kstub) getpid() int { k.Helper(); return k.Expects("getpid").Ret.(int) }
func (k *kstub) getuid() int { k.Helper(); return k.Expects("getuid").Ret.(int) }
func (k *kstub) getgid() int { k.Helper(); return k.Expects("getgid").Ret.(int) }
func (k *kstub) lookupEnv(key string) (string, bool) {
	k.Helper()
	expect := k.Expects("lookupEnv")
	if expect.Error(
		stub.CheckArg(k.Stub, "key", key, 0)) != nil {
		k.FailNow()
	}
	if expect.Ret == nil {
		return "\x00", false
	}
	return expect.Ret.(string), true
}
func (k *kstub) tempdir() string { k.Helper(); return k.Expects("tempdir").Ret.(string) }

func (k *kstub) cmdOutput(cmd *exec.Cmd) ([]byte, error) {
	k.Helper()
	expect := k.Expects("cmdOutput")
	return expect.Ret.([]byte), expect.Error(
		stub.CheckArg(k.Stub, "cmd.Path", cmd.Path, 0),
		stub.CheckArgReflect(k.Stub, "cmd.Stderr", cmd.Stderr, 1),
		stub.CheckArgReflect(k.Stub, "cmd.Env", cmd.Env, 2),
		stub.CheckArg(k.Stub, "cmd.Dir", cmd.Dir, 3))
}

func (k *kstub) mustHsuPath() *check.Absolute {
	k.Helper()
	return k.Expects("mustHsuPath").Ret.(*check.Absolute)
}

func (k *kstub) GetLogger() *log.Logger { panic("unreachable") }

func (k *kstub) IsVerbose() bool { k.Helper(); return k.Expects("isVerbose").Ret.(bool) }
func (k *kstub) SwapVerbose(verbose bool) bool {
	k.Helper()
	expect := k.Expects("swapVerbose")
	if expect.Error(
		stub.CheckArg(k.Stub, "verbose", verbose, 0)) != nil {
		k.FailNow()
	}
	return expect.Ret.(bool)
}

// ignoreValue marks a value to be ignored by the test suite.
type ignoreValue struct{}

func (k *kstub) Verbose(v ...any) {
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

func (k *kstub) Verbosef(format string, v ...any) {
	k.Helper()
	if k.Expects("verbosef").Error(
		stub.CheckArg(k.Stub, "format", format, 0),
		stub.CheckArgReflect(k.Stub, "v", v, 1)) != nil {
		k.FailNow()
	}
}

func (k *kstub) Suspend() bool { k.Helper(); return k.Expects("suspend").Ret.(bool) }
func (k *kstub) Resume() bool  { k.Helper(); return k.Expects("resume").Ret.(bool) }
func (k *kstub) BeforeExit()   { k.Helper(); k.Expects("beforeExit") }

// panicMsgContext implements [message.Msg] and [context.Context] with methods wrapping panic.
// This should be assigned to test cases to be checked against.
type panicMsgContext struct{}

func (panicMsgContext) GetLogger() *log.Logger  { panic("unreachable") }
func (panicMsgContext) IsVerbose() bool         { panic("unreachable") }
func (panicMsgContext) SwapVerbose(bool) bool   { panic("unreachable") }
func (panicMsgContext) Verbose(...any)          { panic("unreachable") }
func (panicMsgContext) Verbosef(string, ...any) { panic("unreachable") }
func (panicMsgContext) Suspend() bool           { panic("unreachable") }
func (panicMsgContext) Resume() bool            { panic("unreachable") }
func (panicMsgContext) BeforeExit()             { panic("unreachable") }

func (panicMsgContext) Deadline() (time.Time, bool) { panic("unreachable") }
func (panicMsgContext) Done() <-chan struct{}       { panic("unreachable") }
func (panicMsgContext) Err() error                  { panic("unreachable") }
func (panicMsgContext) Value(any) any               { panic("unreachable") }

// panicDispatcher implements syscallDispatcher with methods wrapping panic.
// This type is meant to be embedded in partial syscallDispatcher implementations.
type panicDispatcher struct{}

func (panicDispatcher) new(func(k syscallDispatcher))         { panic("unreachable") }
func (panicDispatcher) getpid() int                           { panic("unreachable") }
func (panicDispatcher) getuid() int                           { panic("unreachable") }
func (panicDispatcher) getgid() int                           { panic("unreachable") }
func (panicDispatcher) lookupEnv(string) (string, bool)       { panic("unreachable") }
func (panicDispatcher) stat(string) (os.FileInfo, error)      { panic("unreachable") }
func (panicDispatcher) open(string) (osFile, error)           { panic("unreachable") }
func (panicDispatcher) readdir(string) ([]os.DirEntry, error) { panic("unreachable") }
func (panicDispatcher) tempdir() string                       { panic("unreachable") }
func (panicDispatcher) evalSymlinks(string) (string, error)   { panic("unreachable") }
func (panicDispatcher) lookupGroupId(string) (string, error)  { panic("unreachable") }
func (panicDispatcher) cmdOutput(*exec.Cmd) ([]byte, error)   { panic("unreachable") }
func (panicDispatcher) overflowUid(message.Msg) int           { panic("unreachable") }
func (panicDispatcher) overflowGid(message.Msg) int           { panic("unreachable") }
func (panicDispatcher) mustHsuPath() *check.Absolute          { panic("unreachable") }
func (panicDispatcher) fatalf(string, ...any)                 { panic("unreachable") }
