package outcome

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
	"unsafe"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/message"
	"hakurei.app/system"
)

// call initialises a [stub.Call].
// This keeps composites analysis happy without making the test cases too bloated.
func call(name string, args stub.ExpectArgs, ret any, err error) stub.Call {
	return stub.Call{Name: name, Args: args, Ret: ret, Err: err}
}

const (
	// checkExpectUid is the uid value used by checkOpBehaviour to initialise [system.I].
	checkExpectUid = 0xcafebabe
	// wantAutoEtcPrefix is the autoetc prefix corresponding to checkExpectInstanceId.
	wantAutoEtcPrefix = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	// wantInstancePrefix is the SharePath corresponding to checkExpectInstanceId.
	wantInstancePrefix = container.Nonexistent + "/tmp/hakurei.0/" + wantAutoEtcPrefix

	// wantRuntimePath is the XDG_RUNTIME_DIR value returned during testing.
	wantRuntimePath = "/proc/nonexistent/xdg_runtime_dir"
	// wantRunDirPath is the RunDirPath value resolved during testing.
	wantRunDirPath = wantRuntimePath + "/hakurei"
	// wantRuntimeSharePath is the runtimeSharePath value resolved during testing.
	wantRuntimeSharePath = wantRunDirPath + "/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// checkExpectInstanceId is the [hst.ID] value used by checkOpBehaviour to initialise outcomeState.
var checkExpectInstanceId = *(*hst.ID)(bytes.Repeat([]byte{0xaa}, len(hst.ID{})))

type (
	// pStateSysFunc is called before each test case is run to prepare outcomeStateSys.
	pStateSysFunc = func(state *outcomeStateSys)
	// pStateContainerFunc is called before each test case is run to prepare outcomeStateParams.
	pStateContainerFunc = func(state *outcomeStateParams)

	// extraCheckSysFunc is called to check outcomeStateSys and must not have side effects.
	extraCheckSysFunc = func(t *testing.T, state *outcomeStateSys)
	// extraCheckParamsFunc is called to check outcomeStateParams and must not have side effects.
	extraCheckParamsFunc = func(t *testing.T, state *outcomeStateParams)
)

// insertsOps prepares outcomeStateParams to allow [container.Op] to be inserted.
func insertsOps(next pStateContainerFunc) pStateContainerFunc {
	return func(state *outcomeStateParams) {
		state.params.Ops = new(container.Ops)

		if next != nil {
			next(state)
		}
	}
}

// afterSpRuntimeOp prepares outcomeStateParams for an outcomeOp meant to run after spRuntimeOp.
func afterSpRuntimeOp(next pStateContainerFunc) pStateContainerFunc {
	return func(state *outcomeStateParams) {
		// emulates spRuntimeOp
		state.runtimeDir = m("/run/user/1000")

		if next != nil {
			next(state)
		}
	}
}

// sysUsesInstance checks for use of the outcomeStateSys.instance method.
func sysUsesInstance(next extraCheckSysFunc) extraCheckSysFunc {
	return func(t *testing.T, state *outcomeStateSys) {
		if want := m(wantInstancePrefix); !reflect.DeepEqual(state.sharePath, want) {
			t.Errorf("outcomeStateSys: sharePath = %v, want %v", state.sharePath, want)
		}

		if next != nil {
			next(t, state)
		}
	}
}

// sysUsesRuntime checks for use of the outcomeStateSys.runtime method.
func sysUsesRuntime(next extraCheckSysFunc) extraCheckSysFunc {
	return func(t *testing.T, state *outcomeStateSys) {
		if want := m(wantRuntimeSharePath); !reflect.DeepEqual(state.runtimeSharePath, want) {
			t.Errorf("outcomeStateSys: runtimeSharePath = %v, want %v", state.runtimeSharePath, want)
		}

		if next != nil {
			next(t, state)
		}
	}
}

// paramsWantEnv checks outcomeStateParams.env for inserted entries on top of [hst.Config].
func paramsWantEnv(config *hst.Config, wantEnv map[string]string, next extraCheckParamsFunc) extraCheckParamsFunc {
	want := make(map[string]string, len(wantEnv)+len(config.Container.Env))
	maps.Copy(want, wantEnv)
	maps.Copy(want, config.Container.Env)
	return func(t *testing.T, state *outcomeStateParams) {
		if !maps.Equal(state.env, want) {
			t.Errorf("toContainer: env = %#v, want %#v", state.env, want)
		}

		if next != nil {
			next(t, state)
		}
	}
}

// opBehaviourTestCase checks outcomeOp behaviour against outcomeStateSys and outcomeStateParams.
type opBehaviourTestCase struct {
	name string
	// newOp returns a new instance of outcomeOp under testing that is safe to clobber.
	newOp func(isShim, clearUnexported bool) outcomeOp
	// newConfig returns a new instance of [hst.Config] that is checked not to be clobbered by outcomeOp.
	newConfig func() *hst.Config

	// pStateSys is called before outcomeOp.toSystem to prepare outcomeStateSys.
	pStateSys pStateSysFunc
	// toSystem are expected syscallDispatcher calls during outcomeOp.toSystem.
	toSystem []stub.Call
	// wantSys is the expected [system.I] state after outcomeOp.toSystem.
	wantSys *system.I
	// extraCheckSys is called after outcomeOp.toSystem to check the state of outcomeStateSys.
	extraCheckSys extraCheckSysFunc
	// wantErrSystem is the expected error value returned by outcomeOp.toSystem.
	// Further testing is skipped if not nil.
	wantErrSystem error

	// pStateContainer is called before outcomeOp.toContainer to prepare outcomeStateParams.
	pStateContainer pStateContainerFunc
	// toContainer are expected syscallDispatcher calls during outcomeOp.toContainer.
	toContainer []stub.Call
	// wantParams is the expected [container.Params] after outcomeOp.toContainer.
	wantParams *container.Params
	// extraCheckParams is called after outcomeOp.toContainer to check the state of outcomeStateParams.
	extraCheckParams extraCheckParamsFunc
	// wantErrContainer is the expected error value returned by outcomeOp.toContainer.
	wantErrContainer error
}

// checkOpBehaviour runs a slice of opBehaviourTestCase.
func checkOpBehaviour(t *testing.T, testCases []opBehaviourTestCase) {
	t.Helper()

	wantNewState := []stub.Call{
		// newOutcomeState
		call("getpid", stub.ExpectArgs{}, 0xdead, nil),
		call("isVerbose", stub.ExpectArgs{}, true, nil),
		call("mustHsuPath", stub.ExpectArgs{}, m(container.Nonexistent), nil),
		call("cmdOutput", stub.ExpectArgs{container.Nonexistent, os.Stderr, []string{}, "/"}, []byte("0"), nil),
		call("tempdir", stub.ExpectArgs{}, container.Nonexistent+"/tmp", nil),
		call("lookupEnv", stub.ExpectArgs{"XDG_RUNTIME_DIR"}, wantRuntimePath, nil),
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
			t.Parallel()

			wantCallsFull := slices.Concat(wantNewState, tc.toSystem, []stub.Call{{Name: stub.CallSeparator}})
			if tc.wantErrSystem == nil {
				wantCallsFull = append(wantCallsFull, slices.Concat(wantNewState, tc.toContainer)...)
			}

			wantConfig := tc.newConfig()
			k := &kstub{nil, nil, panicDispatcher{}, stub.New(t,
				func(s *stub.Stub[syscallDispatcher]) syscallDispatcher { return &kstub{nil, nil, panicDispatcher{}, s} },
				stub.Expect{Calls: wantCallsFull},
			)}
			defer stub.HandleExit(t)

			{
				config := tc.newConfig()
				s := newOutcomeState(k, k, &checkExpectInstanceId, config, &Hsu{k: k})
				if err := s.populateLocal(k, k); err != nil {
					t.Fatalf("populateLocal: error = %v", err)
				}
				stateSys := s.newSys(config, system.New(panicMsgContext{}, k, checkExpectUid))
				if tc.pStateSys != nil {
					tc.pStateSys(stateSys)
				}
				op := tc.newOp(false, true)

				if err := op.toSystem(stateSys); !reflect.DeepEqual(err, tc.wantErrSystem) {
					t.Fatalf("toSystem: error = %#v, want %#v", err, tc.wantErrSystem)
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
					t.Fatalf("toContainer: error = %#v, want %#v", err, tc.wantErrContainer)
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

// simpleTestCase is a simple freeform test case utilising kstub.
type simpleTestCase struct {
	name string
	f    func(k *kstub) error
	// want are expected syscallDispatcher calls during f.
	want stub.Expect
	// wantErr is the expected error value returned by f.
	wantErr error
}

// checkSimple runs a slice of simpleTestCase.
func checkSimple(t *testing.T, fname string, testCases []simpleTestCase) {
	t.Helper()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()
			t.Parallel()

			defer stub.HandleExit(t)

			uNotifyContext, uShimReader := make(chan struct{}), make(chan struct{})
			k := &kstub{uNotifyContext, uShimReader, panicDispatcher{},
				stub.New(t, func(s *stub.Stub[syscallDispatcher]) syscallDispatcher {
					return &kstub{uNotifyContext, uShimReader, panicDispatcher{}, s}
				}, tc.want)}
			if err := tc.f(k); !reflect.DeepEqual(err, tc.wantErr) {
				t.Errorf("%s: error = %#v, want %#v", fname, err, tc.wantErr)
			}
			k.VisitIncomplete(func(s *stub.Stub[syscallDispatcher]) {
				t.Helper()

				t.Errorf("%s: %d calls, want %d", fname, s.Pos(), s.Len())
			})
		})
	}
}

// kstub partially implements syscallDispatcher via [stub.Stub].
type kstub struct {
	// notifyContext blocks on unblockNotifyContext if provided with a kstub.Stub track.
	unblockNotifyContext chan struct{}
	// stubTrackReader blocks on unblockShimReader if unblocking notifyContext.
	unblockShimReader chan struct{}

	panicDispatcher
	*stub.Stub[syscallDispatcher]
}

func (k *kstub) new(f func(k syscallDispatcher, msg message.Msg)) {
	k.New(func(k syscallDispatcher) { f(k, k.(*kstub)) })
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
func (k *kstub) stat(name string) (os.FileInfo, error) {
	k.Helper()
	expect := k.Expects("stat")
	return expect.Ret.(os.FileInfo), expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}
func (k *kstub) open(name string) (osFile, error) {
	k.Helper()
	expect := k.Expects("open")
	return expect.Ret.(osFile), expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}
func (k *kstub) readdir(name string) ([]os.DirEntry, error) {
	k.Helper()
	expect := k.Expects("readdir")
	return expect.Ret.([]os.DirEntry), expect.Error(
		stub.CheckArg(k.Stub, "name", name, 0))
}
func (k *kstub) tempdir() string { k.Helper(); return k.Expects("tempdir").Ret.(string) }
func (k *kstub) exit(code int) {
	k.Helper()
	expect := k.Expects("exit")

	if errors.Is(expect.Err, unblockNotifyContext) {
		close(k.unblockNotifyContext)
	}

	if !stub.CheckArg(k.Stub, "code", code, 0) {
		k.FailNow()
	}
	panic(expect.Ret.(int))
}
func (k *kstub) evalSymlinks(path string) (string, error) {
	k.Helper()
	expect := k.Expects("evalSymlinks")
	return expect.Ret.(string), expect.Error(
		stub.CheckArg(k.Stub, "path", path, 0))
}

func (k *kstub) prctl(op, arg2, arg3 uintptr) error {
	k.Helper()
	return k.Expects("prctl").Error(
		stub.CheckArg(k.Stub, "op", op, 0),
		stub.CheckArg(k.Stub, "arg2", arg2, 1),
		stub.CheckArg(k.Stub, "arg3", arg3, 2))
}

func (k *kstub) setDumpable(dumpable uintptr) error {
	k.Helper()
	return k.Expects("setDumpable").Error(
		stub.CheckArg(k.Stub, "dumpable", dumpable, 0))
}

func (k *kstub) receive(key string, e any, fdp *uintptr) (closeFunc func() error, err error) {
	k.Helper()
	expect := k.Expects("receive")
	reflect.ValueOf(e).Elem().Set(reflect.ValueOf(expect.Args[1]))
	if expect.Args[2] != nil {
		*fdp = expect.Args[2].(uintptr)
	}
	return func() error { return k.Expects("closeReceive").Err }, expect.Error(
		stub.CheckArg(k.Stub, "key", key, 0))
}

func (k *kstub) expectCheckContainer(expect *stub.Call, z *container.Container) error {
	k.Helper()
	if !stub.CheckArgReflect(k.Stub, "params", &z.Params, 0) {
		k.Errorf("params:\n%s\n%s", mustMarshal(&z.Params), mustMarshal(expect.Args[0]))
	}
	return expect.Err
}

func (k *kstub) containerStart(z *container.Container) error {
	k.Helper()
	if k.unblockShimReader != nil {
		close(k.unblockShimReader)
	}
	return k.expectCheckContainer(k.Expects("containerStart"), z)
}
func (k *kstub) containerServe(z *container.Container) error {
	k.Helper()
	return k.expectCheckContainer(k.Expects("containerServe"), z)
}
func (k *kstub) containerWait(z *container.Container) error {
	k.Helper()
	return k.expectCheckContainer(k.Expects("containerWait"), z)
}

func (k *kstub) seccompLoad(rules []seccomp.NativeRule, flags seccomp.ExportFlag) error {
	k.Helper()
	return k.Expects("seccompLoad").Error(
		stub.CheckArgReflect(k.Stub, "rules", rules, 0),
		stub.CheckArg(k.Stub, "flags", flags, 1))
}

func (k *kstub) cmdOutput(cmd *exec.Cmd) ([]byte, error) {
	k.Helper()
	expect := k.Expects("cmdOutput")
	return expect.Ret.([]byte), expect.Error(
		stub.CheckArg(k.Stub, "cmd.Path", cmd.Path, 0),
		stub.CheckArgReflect(k.Stub, "cmd.Stderr", cmd.Stderr, 1),
		stub.CheckArgReflect(k.Stub, "cmd.Env", cmd.Env, 2),
		stub.CheckArg(k.Stub, "cmd.Dir", cmd.Dir, 3))
}

func (k *kstub) notifyContext(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc) {
	k.Helper()
	expect := k.Expects("notifyContext")

	if expect.Error(
		stub.CheckArgReflect(k.Stub, "parent", parent, 0),
		stub.CheckArgReflect(k.Stub, "signals", signals, 1)) != nil {
		k.FailNow()
	}

	if sub, ok := expect.Ret.(int); ok && sub >= 0 {
		subVal := reflect.ValueOf(k.Stub).Elem().FieldByName("sub")
		ks := &kstub{nil, nil, panicDispatcher{}, reflect.
			NewAt(subVal.Type(), unsafe.Pointer(subVal.UnsafeAddr())).Elem().
			Interface().([]*stub.Stub[syscallDispatcher])[sub]}

		<-k.unblockNotifyContext
		return k.Context(), func() { k.Helper(); ks.Expects("notifyContextStop") }
	}

	return k.Context(), func() { panic("unexpected call to stop") }
}

func (k *kstub) mustHsuPath() *check.Absolute {
	k.Helper()
	return k.Expects("mustHsuPath").Ret.(*check.Absolute)
}

func (k *kstub) dbusAddress() (session, system string) {
	k.Helper()
	ret := k.Expects("dbusAddress").Ret.([2]string)
	return ret[0], ret[1]
}

// stubTrackReader embeds kstub but switches the underlying [stub.Stub] index to sub on its first Read.
// The resulting kstub does not share any state with the instance passed to the instrumented goroutine.
// Therefore, any method making use of such must not be called.
type stubTrackReader struct {
	sub     int
	subOnce sync.Once

	*kstub
}

// unblockNotifyContext is passed via call and must be handled by stubTrackReader.Read
var unblockNotifyContext = errors.New("this error unblocks notifyContext and must not be returned")

func (r *stubTrackReader) Read(p []byte) (n int, err error) {
	r.subOnce.Do(func() {
		subVal := reflect.ValueOf(r.kstub.Stub).Elem().FieldByName("sub")
		r.kstub = &kstub{r.kstub.unblockNotifyContext, r.kstub.unblockShimReader, panicDispatcher{}, reflect.
			NewAt(subVal.Type(), unsafe.Pointer(subVal.UnsafeAddr())).Elem().
			Interface().([]*stub.Stub[syscallDispatcher])[r.sub]}
	})

	n, err = r.kstub.Read(p)
	if errors.Is(err, unblockNotifyContext) {
		err = nil
		close(r.unblockNotifyContext)
		<-r.unblockShimReader
	}
	return n, err
}

func (k *kstub) setupContSignal(pid int) (io.ReadCloser, func(), error) {
	k.Helper()
	expect := k.Expects("setupContSignal")
	return &stubTrackReader{sub: expect.Ret.(int), kstub: k}, func() { k.Helper(); k.Expects("wKeepAlive") }, expect.Error(
		stub.CheckArg(k.Stub, "pid", pid, 0))
}

func (k *kstub) getMsg() message.Msg { k.Helper(); k.Expects("getMsg"); return k }

func (k *kstub) fatal(v ...any) {
	if k.Expects("fatal").Error(
		stub.CheckArgReflect(k.Stub, "v", v, 0)) != nil {
		k.FailNow()
	}
	panic(stub.PanicExit)
}
func (k *kstub) fatalf(format string, v ...any) {
	if k.Expects("fatalf").Error(
		stub.CheckArg(k.Stub, "format", format, 0),
		stub.CheckArgReflect(k.Stub, "v", v, 1)) != nil {
		k.FailNow()
	}
	panic(stub.PanicExit)
}

func (k *kstub) Close() error { k.Helper(); return k.Expects("rcClose").Err }
func (k *kstub) Read(p []byte) (n int, err error) {
	k.Helper()
	expect := k.Expects("rcRead")

	// special case to terminate exit outcomes goroutine
	// to proceed with further testing of the entrypoint
	if expect.Ret == nil {
		panic(stub.PanicExit)
	}

	return copy(p, expect.Ret.([]byte)), expect.Err
}

func (k *kstub) GetLogger() *log.Logger { k.Helper(); return k.Expects("getLogger").Ret.(*log.Logger) }
func (k *kstub) IsVerbose() bool        { k.Helper(); return k.Expects("isVerbose").Ret.(bool) }
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

// stubOsFile partially implements osFile.
type stubOsFile struct {
	closeErr error

	io.Reader
	io.Writer
}

func (f *stubOsFile) Close() error               { return f.closeErr }
func (f *stubOsFile) Name() string               { panic("unreachable") }
func (f *stubOsFile) Stat() (fs.FileInfo, error) { panic("unreachable") }

// stubFi partially implements [os.FileInfo]. Can be passed as nil to assert all methods unreachable.
type stubFi struct {
	size  int64
	mode  os.FileMode
	isDir bool
}

func (fi *stubFi) Name() string       { panic("unreachable") }
func (fi *stubFi) ModTime() time.Time { panic("unreachable") }
func (fi *stubFi) Sys() any           { panic("unreachable") }
func (fi *stubFi) Size() int64        { return fi.size }
func (fi *stubFi) Mode() os.FileMode  { return fi.mode }
func (fi *stubFi) IsDir() bool        { return fi.isDir }

// stubDir returns a slice of [os.DirEntry] with only their Name method implemented.
func stubDir(names ...string) []os.DirEntry {
	d := make([]os.DirEntry, len(names))
	for i, name := range names {
		d[i] = nameDentry(name)
	}
	return d
}

// nameDentry implements the Name method on [os.DirEntry].
type nameDentry string

func (e nameDentry) Name() string             { return string(e) }
func (nameDentry) IsDir() bool                { panic("unreachable") }
func (nameDentry) Type() fs.FileMode          { panic("unreachable") }
func (nameDentry) Info() (fs.FileInfo, error) { panic("unreachable") }

// errorReader implements [io.Reader] that unconditionally returns -1, val.
type errorReader struct{ val error }

func (r errorReader) Read([]byte) (int, error) { return -1, r.val }

// mustMarshal returns the result of [json.Marshal] as a string and panics on error.
func mustMarshal(v any) string {
	if b, err := json.Marshal(v); err != nil {
		panic(err.Error())
	} else {
		return string(b)
	}
}

// m is a shortcut for [check.MustAbs].
func m(pathname string) *check.Absolute { return check.MustAbs(pathname) }

// f returns [hst.FilesystemConfig] wrapped in its [json] adapter.
func f(c hst.FilesystemConfig) hst.FilesystemConfigJSON {
	return hst.FilesystemConfigJSON{FilesystemConfig: c}
}

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

func (panicDispatcher) new(func(k syscallDispatcher, msg message.Msg))      { panic("unreachable") }
func (panicDispatcher) getpid() int                                         { panic("unreachable") }
func (panicDispatcher) getuid() int                                         { panic("unreachable") }
func (panicDispatcher) getgid() int                                         { panic("unreachable") }
func (panicDispatcher) lookupEnv(string) (string, bool)                     { panic("unreachable") }
func (panicDispatcher) pipe() (*os.File, *os.File, error)                   { panic("unreachable") }
func (panicDispatcher) stat(string) (os.FileInfo, error)                    { panic("unreachable") }
func (panicDispatcher) open(string) (osFile, error)                         { panic("unreachable") }
func (panicDispatcher) readdir(string) ([]os.DirEntry, error)               { panic("unreachable") }
func (panicDispatcher) tempdir() string                                     { panic("unreachable") }
func (panicDispatcher) exit(int)                                            { panic("unreachable") }
func (panicDispatcher) evalSymlinks(string) (string, error)                 { panic("unreachable") }
func (panicDispatcher) prctl(uintptr, uintptr, uintptr) error               { panic("unreachable") }
func (panicDispatcher) lookupGroupId(string) (string, error)                { panic("unreachable") }
func (panicDispatcher) cmdOutput(*exec.Cmd) ([]byte, error)                 { panic("unreachable") }
func (panicDispatcher) overflowUid(message.Msg) int                         { panic("unreachable") }
func (panicDispatcher) overflowGid(message.Msg) int                         { panic("unreachable") }
func (panicDispatcher) setDumpable(uintptr) error                           { panic("unreachable") }
func (panicDispatcher) receive(string, any, *uintptr) (func() error, error) { panic("unreachable") }
func (panicDispatcher) containerStart(*container.Container) error           { panic("unreachable") }
func (panicDispatcher) containerServe(*container.Container) error           { panic("unreachable") }
func (panicDispatcher) containerWait(*container.Container) error            { panic("unreachable") }
func (panicDispatcher) mustHsuPath() *check.Absolute                        { panic("unreachable") }
func (panicDispatcher) dbusAddress() (string, string)                       { panic("unreachable") }
func (panicDispatcher) setupContSignal(int) (io.ReadCloser, func(), error)  { panic("unreachable") }
func (panicDispatcher) getMsg() message.Msg                                 { panic("unreachable") }
func (panicDispatcher) fatal(...any)                                        { panic("unreachable") }
func (panicDispatcher) fatalf(string, ...any)                               { panic("unreachable") }

func (panicDispatcher) notifyContext(context.Context, ...os.Signal) (context.Context, context.CancelFunc) {
	panic("unreachable")
}
func (panicDispatcher) seccompLoad([]seccomp.NativeRule, seccomp.ExportFlag) error {
	panic("unreachable")
}
