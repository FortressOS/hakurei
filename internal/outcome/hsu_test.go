package outcome

import (
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"syscall"
	"testing"
	"unsafe"

	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestHsu(t *testing.T) {
	t.Parallel()

	t.Run("ensure dispatcher", func(t *testing.T) {
		hsu := new(Hsu)
		hsu.ensureDispatcher()

		k := direct{}
		if !reflect.DeepEqual(hsu.k, k) {
			t.Errorf("ensureDispatcher: k = %#v, want %#v", hsu.k, k)
		}
	})

	fCheckID := func(k *kstub) error {
		hsu := &Hsu{k: k}
		id, err := hsu.ID()
		k.Verbose(id)
		if id0, err0 := hsu.ID(); id0 != id || !reflect.DeepEqual(err0, err) {
			t.Fatalf("ID: id0 = %d, err0 = %#v, id = %d, err = %#v", id0, err0, id, err)
		}
		return err
	}

	checkSimple(t, "Hsu.ID", []simpleTestCase{
		{"hsu nonexistent", fCheckID, stub.Expect{Calls: []stub.Call{
			call("mustHsuPath", stub.ExpectArgs{}, m("/run/wrappers/bin/hsu"), nil),
			call("cmdOutput", stub.ExpectArgs{"/run/wrappers/bin/hsu", os.Stderr, []string{}, "/"}, ([]byte)(nil), os.ErrNotExist),
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, &hst.AppError{
			Step: "obtain uid from hsu",
			Err:  os.ErrNotExist,
			Msg:  "the setuid helper is missing: /run/wrappers/bin/hsu",
		}},

		{"access", fCheckID, stub.Expect{Calls: []stub.Call{
			call("mustHsuPath", stub.ExpectArgs{}, m("/run/wrappers/bin/hsu"), nil),
			call("cmdOutput", stub.ExpectArgs{"/run/wrappers/bin/hsu", os.Stderr, []string{}, "/"}, ([]byte)(nil), makeExitError(1<<8)),
			call("verbose", stub.ExpectArgs{[]any{-1}}, nil, nil),
		}}, &hst.AppError{
			Step: "obtain uid from hsu",
			Err:  ErrHsuAccess,
		}},

		{"invalid output", fCheckID, stub.Expect{Calls: []stub.Call{
			call("mustHsuPath", stub.ExpectArgs{}, m("/run/wrappers/bin/hsu"), nil),
			call("cmdOutput", stub.ExpectArgs{"/run/wrappers/bin/hsu", os.Stderr, []string{}, "/"}, []byte{0}, nil),
			call("verbose", stub.ExpectArgs{[]any{0}}, nil, nil),
		}}, &hst.AppError{
			Step: "obtain uid from hsu",
			Err:  &strconv.NumError{Func: "Atoi", Num: "\x00", Err: strconv.ErrSyntax},
			Msg:  "invalid uid string from hsu",
		}},

		{"success", fCheckID, stub.Expect{Calls: []stub.Call{
			call("mustHsuPath", stub.ExpectArgs{}, m("/run/wrappers/bin/hsu"), nil),
			call("cmdOutput", stub.ExpectArgs{"/run/wrappers/bin/hsu", os.Stderr, []string{}, "/"}, []byte{'0'}, nil),
			call("verbose", stub.ExpectArgs{[]any{0}}, nil, nil),
		}}, nil},
	})
}

// makeExitError populates syscall.WaitStatus in an [exec.ExitError].
// Do not reuse this function in a cross-platform package.
func makeExitError(status syscall.WaitStatus) error {
	ps := new(os.ProcessState)
	statusV := reflect.ValueOf(ps).Elem().FieldByName("status")
	*reflect.NewAt(statusV.Type(), unsafe.Pointer(statusV.UnsafeAddr())).Interface().(*syscall.WaitStatus) = status
	return &exec.ExitError{ProcessState: ps}
}
