package main_test

import (
	"io"
	"reflect"
	"strings"
	"testing"
	_ "unsafe"

	"hakurei.app/container/stub"
)

//go:linkname decodeJSON hakurei.app/cmd/hakurei.decodeJSON
func decodeJSON(fatal func(v ...any), op string, r io.Reader, v any)

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		t    reflect.Type
		data string
		want any
		msg  string
	}{
		{"success", reflect.TypeFor[uintptr](), "3735928559\n", uintptr(0xdeadbeef), ""},

		{"syntax", reflect.TypeFor[*int](), "\x00", nil,
			`cannot load sample: invalid character '\x00' looking for beginning of value at byte 1`},
		{"type", reflect.TypeFor[uintptr](), "-1", nil,
			`cannot load sample: inappropriate number -1 at byte 2`},
		{"default", reflect.TypeFor[*int](), "{", nil,
			"cannot load sample: unexpected EOF"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				gotP   = reflect.New(tc.t)
				gotMsg *string
			)
			decodeJSON(func(v ...any) {
				if gotMsg != nil {
					t.Fatal("fatal called twice")
				}
				msg := v[0].(string)
				gotMsg = &msg
			}, "load sample", strings.NewReader(tc.data), gotP.Interface())
			if tc.msg != "" {
				if gotMsg == nil {
					t.Errorf("decodeJSON: success, want fatal %q", tc.msg)
				} else if *gotMsg != tc.msg {
					t.Errorf("decodeJSON: fatal = %q, want %q", *gotMsg, tc.msg)
				}
			} else if gotMsg != nil {
				t.Errorf("decodeJSON: fatal = %q", *gotMsg)
			} else if !reflect.DeepEqual(gotP.Elem().Interface(), tc.want) {
				t.Errorf("decodeJSON: %#v, want %#v", gotP.Elem().Interface(), tc.want)
			}
		})
	}
}

//go:linkname encodeJSON hakurei.app/cmd/hakurei.encodeJSON
func encodeJSON(fatal func(v ...any), output io.Writer, short bool, v any)

func TestEncodeJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		v    any
		want string
	}{
		{"marshaler", errorJSONMarshaler{},
			`cannot encode json for main_test.errorJSONMarshaler: unique error 3735928559 injected by the test suite`},
		{"default", func() {},
			`cannot write json: json: unsupported type: func()`},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var called bool
			encodeJSON(func(v ...any) {
				if called {
					t.Fatal("fatal called twice")
				}
				called = true

				if v[0].(string) != tc.want {
					t.Errorf("encodeJSON: fatal = %q, want %q", v[0].(string), tc.want)
				}
			}, nil, false, tc.v)

			if !called {
				t.Errorf("encodeJSON: success, want fatal %q", tc.want)
			}
		})
	}
}

// errorJSONMarshaler implements json.Marshaler.
type errorJSONMarshaler struct{}

func (errorJSONMarshaler) MarshalJSON() ([]byte, error) { return nil, stub.UniqueError(0xdeadbeef) }
