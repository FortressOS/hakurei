package container_test

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
)

func TestSuspendable(t *testing.T) {
	// copied from output.go
	const suspendBufMax = 1 << 24

	const (
		// equivalent to len(want.pt)
		nSpecialPtEquiv = -iota - 1
		// equivalent to len(want.w)
		nSpecialWEquiv
		// suspends writer before executing test case, implies nSpecialWEquiv
		nSpecialSuspend
		// offset: resume writer and measure against dump instead, implies nSpecialPtEquiv
		nSpecialDump
	)

	// shares the same writer
	testCases := []struct {
		name    string
		w, pt   []byte
		err     error
		wantErr error
		n       int
	}{
		{"simple", []byte{0xde, 0xad, 0xbe, 0xef}, []byte{0xde, 0xad, 0xbe, 0xef},
			nil, nil, nSpecialPtEquiv},

		{"error", []byte{0xb, 0xad}, []byte{0xb, 0xad},
			stub.UniqueError(0), stub.UniqueError(0), nSpecialPtEquiv},

		{"suspend short", []byte{0}, nil,
			nil, nil, nSpecialSuspend},
		{"sw short 0", []byte{0xca, 0xfe, 0xba, 0xbe}, nil,
			nil, nil, nSpecialWEquiv},
		{"sw short 1", []byte{0xff}, nil,
			nil, nil, nSpecialWEquiv},
		{"resume short", nil, []byte{0, 0xca, 0xfe, 0xba, 0xbe, 0xff}, nil, nil,
			nSpecialDump},

		{"long pt", bytes.Repeat([]byte{0xff}, suspendBufMax+1), bytes.Repeat([]byte{0xff}, suspendBufMax+1),
			nil, nil, nSpecialPtEquiv},

		{"suspend fill", bytes.Repeat([]byte{0xfe}, suspendBufMax), nil,
			nil, nil, nSpecialSuspend},
		{"drop", []byte{0}, nil,
			nil, syscall.ENOMEM, 0},
		{"drop error", []byte{0}, nil,
			stub.UniqueError(1), syscall.ENOMEM, 0},
		{"resume fill", nil, bytes.Repeat([]byte{0xfe}, suspendBufMax),
			nil, nil, nSpecialDump - 2},

		{"suspend fill partial", bytes.Repeat([]byte{0xfd}, suspendBufMax-0xf), nil,
			nil, nil, nSpecialSuspend},
		{"partial write", bytes.Repeat([]byte{0xad}, 0x1f), nil,
			nil, syscall.ENOMEM, 0xf},
		{"full drop", []byte{0}, nil,
			nil, syscall.ENOMEM, 0},
		{"resume fill partial", nil, append(bytes.Repeat([]byte{0xfd}, suspendBufMax-0xf), bytes.Repeat([]byte{0xad}, 0xf)...),
			nil, nil, nSpecialDump - 0x10 - 1},
	}

	var dw expectWriter

	w := container.Suspendable{Downstream: &dw}
	for _, tc := range testCases {
		// these share the same writer, so cannot be subtests
		t.Logf("writing step %q", tc.name)
		dw.expect, dw.err = tc.pt, tc.err

		var (
			gotN   int
			gotErr error
		)

		wantN := tc.n
		switch wantN {
		case nSpecialPtEquiv:
			wantN = len(tc.pt)
			gotN, gotErr = w.Write(tc.w)

		case nSpecialWEquiv:
			wantN = len(tc.w)
			gotN, gotErr = w.Write(tc.w)

		case nSpecialSuspend:
			s := w.IsSuspended()
			if ok := w.Suspend(); s && ok {
				t.Fatal("Suspend: unexpected success")
			}

			wantN = len(tc.w)
			gotN, gotErr = w.Write(tc.w)

		default:
			if wantN <= nSpecialDump {
				if !w.IsSuspended() {
					t.Fatal("IsSuspended unexpected false")
				}

				resumed, dropped, n, err := w.Resume()
				if !resumed {
					t.Fatal("Resume: resumed = false")
				}
				if wantDropped := nSpecialDump - wantN; int(dropped) != wantDropped {
					t.Errorf("Resume: dropped = %d, want %d", dropped, wantDropped)
				}

				wantN = len(tc.pt)
				gotN, gotErr = int(n), err
			} else {
				gotN, gotErr = w.Write(tc.w)
			}
		}

		if gotN != wantN {
			t.Errorf("Write: n = %d, want %d", gotN, wantN)
		}

		if !reflect.DeepEqual(gotErr, tc.wantErr) {
			t.Errorf("Write: %v", gotErr)
		}
	}
}

// expectWriter compares Write calls to expect.
type expectWriter struct {
	expect []byte
	err    error
}

func (w *expectWriter) Write(p []byte) (n int, err error) {
	defer func() { w.expect = nil }()

	n, err = len(p), w.err
	if w.expect == nil {
		return 0, errors.New("unexpected call to Write: " + strconv.Quote(string(p)))
	}
	if string(p) != string(w.expect) {
		return 0, errors.New("p = " + strconv.Quote(string(p)) + ", want " + strconv.Quote(string(w.expect)))
	}
	return
}
