package state

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestEntryData(t *testing.T) {
	t.Parallel()

	mustEncodeGob := func(e any) string {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(e); err != nil {
			t.Fatalf("cannot encode invalid state: %v", err)
			return "\x00" // not reached
		} else {
			return buf.String()
		}
	}
	templateStateGob := mustEncodeGob(newTemplateState())

	testCases := []struct {
		name string
		data string
		s    *hst.State
		err  error
	}{
		{"invalid header", "\x00\xff\xca\xfe\xff\xff\xff\x00", nil, &hst.AppError{
			Step: "decode state header", Err: errors.New("unexpected revision ffff")}},

		{"invalid gob", "\x00\xff\xca\xfe\x00\x00\xff\x00", nil, &hst.AppError{
			Step: "decode state body", Err: io.EOF}},

		{"invalid config", "\x00\xff\xca\xfe\x00\x00\xff\x00" + mustEncodeGob(new(hst.State)), new(hst.State), &hst.AppError{
			Step: "validate configuration", Err: hst.ErrConfigNull,
			Msg: "invalid configuration"}},

		{"inconsistent enablement", "\x00\xff\xca\xfe\x00\x00\xff\x00" + templateStateGob, newTemplateState(), &hst.AppError{
			Step: "validate state enablement", Err: os.ErrInvalid,
			Msg: "state entry aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa has unexpected enablement byte 0xd, 0xff"}},

		{"template", "\x00\xff\xca\xfe\x00\x00\x0d\xf2" + templateStateGob, newTemplateState(), nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("encode", func(t *testing.T) {
				if tc.s == nil || tc.s.Config == nil {
					return
				}
				t.Parallel()

				var buf bytes.Buffer
				if err := entryEncode(&buf, tc.s); err != nil {
					t.Fatalf("entryEncode: error = %v", err)
				}

				if tc.err == nil {
					// Gob encoding is not guaranteed to be deterministic.
					// While the current implementation mostly is, it has randomised order
					// for iterating over maps, and hst.Config holds a map for environ.
					var got hst.State
					if et, err := entryDecode(&buf, &got); err != nil {
						t.Fatalf("entryDecode: error = %v", err)
					} else if stateEt := got.Enablements.Unwrap(); et != stateEt {
						t.Fatalf("entryDecode: et = %x, state %x", et, stateEt)
					}
					if !reflect.DeepEqual(&got, tc.s) {
						t.Errorf("entryEncode: %x", buf.Bytes())
					}
				} else if testing.Verbose() {
					t.Logf("%x", buf.String())
				}
			})

			t.Run("decode", func(t *testing.T) {
				t.Parallel()

				var got hst.State
				if et, err := entryDecode(strings.NewReader(tc.data), &got); !reflect.DeepEqual(err, tc.err) {
					t.Fatalf("entryDecode: error = %#v, want %#v", err, tc.err)
				} else if err != nil {
					return
				} else if stateEt := got.Enablements.Unwrap(); et != stateEt {
					t.Fatalf("entryDecode: et = %x, state %x", et, stateEt)
				}

				if !reflect.DeepEqual(&got, tc.s) {
					t.Errorf("entryDecode: %#v, want %#v", &got, tc.s)
				}
			})
		})
	}

	t.Run("encode fault", func(t *testing.T) {
		t.Parallel()
		s := newTemplateState()

		t.Run("gob", func(t *testing.T) {
			var want = &hst.AppError{Step: "encode state body", Err: stub.UniqueError(0xcafe)}
			if err := entryEncode(stubNErrorWriter(entryHeaderSize), s); !reflect.DeepEqual(err, want) {
				t.Errorf("entryEncode: error = %#v, want %#v", err, want)
			}
		})

		t.Run("header", func(t *testing.T) {
			var want = &hst.AppError{Step: "encode state header", Err: stub.UniqueError(0xcafe)}
			if err := entryEncode(stubNErrorWriter(entryHeaderSize-1), s); !reflect.DeepEqual(err, want) {
				t.Errorf("entryEncode: error = %#v, want %#v", err, want)
			}
		})
	})
}

// newTemplateState returns the address of a new template [hst.State] struct.
func newTemplateState() *hst.State {
	return &hst.State{
		ID:      hst.ID(bytes.Repeat([]byte{0xaa}, len(hst.ID{}))),
		PID:     0xcafebabe,
		ShimPID: 0xdeadbeef,
		Config:  hst.Template(),
		Time:    time.Unix(0, 0),
	}
}

// stubNErrorWriter returns an error for writes above a certain size.
type stubNErrorWriter int

func (w stubNErrorWriter) Write(p []byte) (n int, err error) {
	if len(p) > int(w) {
		return int(w), stub.UniqueError(0xcafe)
	}
	return io.Discard.Write(p)
}
