package pipewire_test

import (
	"bytes"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"hakurei.app/internal/pipewire"
)

type encodingTestCases[V any, S interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	*V
}] []struct {
	// Uninterpreted name of subtest.
	name string
	// Encoded data.
	wantData []byte
	// Value corresponding to wantData.
	value V
	// Expected decoding error. Skips encoding check if non-nil.
	wantErr error
}

// run runs all test cases as subtests of [testing.T].
func (testCases encodingTestCases[V, S]) run(t *testing.T) {
	t.Helper()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("decode", func(t *testing.T) {
				t.Parallel()

				var value V
				if err := S(&value).UnmarshalBinary(tc.wantData); err != nil {
					t.Fatalf("UnmarshalBinary: error = %v", err)
				}
				if !reflect.DeepEqual(&value, &tc.value) {
					t.Fatalf("UnmarshalBinary:\n%s\nwant\n%s", mustMarshalJSON(value), mustMarshalJSON(tc.value))
				}
			})

			t.Run("encode", func(t *testing.T) {
				t.Parallel()

				if gotData, err := S(&tc.value).MarshalBinary(); err != nil {
					t.Fatalf("MarshalBinary: error = %v", err)
				} else if string(gotData) != string(tc.wantData) {
					t.Fatalf("MarshalBinary: %#v, want %#v", gotData, tc.wantData)
				}
			})

			if s, ok := any(&tc.value).(pipewire.KnownSize); ok {
				t.Run("size", func(t *testing.T) {
					if got := int(s.Size()); got != len(tc.wantData) {
						t.Errorf("Size: %d, want %d", got, len(tc.wantData))
					}
				})
			}
		})
	}
}

func TestPODErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want string
	}{
		{"UnsupportedTypeError", &pipewire.UnsupportedTypeError{
			Type: reflect.TypeFor[any](),
		}, "unsupported type interface {}"},

		{"UnsupportedSizeError", pipewire.UnsupportedSizeError(pipewire.SizeMax + 1), "size 16777216 out of range"},

		{"InvalidUnmarshalError untyped nil", new(pipewire.InvalidUnmarshalError), "attempting to unmarshal to nil"},
		{"InvalidUnmarshalError non-pointer", &pipewire.InvalidUnmarshalError{
			Type: reflect.TypeFor[uintptr](),
		}, "attempting to unmarshal to non-pointer type uintptr"},
		{"InvalidUnmarshalError nil", &pipewire.InvalidUnmarshalError{
			Type: reflect.TypeFor[*uintptr](),
		}, "attempting to unmarshal to nil *uintptr"},

		{"UnexpectedEOFError ErrEOFPrefix", pipewire.ErrEOFPrefix, "unexpected EOF decoding fixed-size POD prefix"},
		{"UnexpectedEOFError ErrEOFData", pipewire.ErrEOFData, "unexpected EOF establishing POD data bounds"},
		{"UnexpectedEOFError ErrEOFDataString", pipewire.ErrEOFDataString, "unexpected EOF establishing POD String bounds"},
		{"UnexpectedEOFError invalid", pipewire.UnexpectedEOFError(0xbad), "unexpected EOF"},

		{"UnmarshalSetError", &pipewire.UnmarshalSetError{
			Type: reflect.TypeFor[*uintptr](),
		}, "cannot set *uintptr"},

		{"TrailingGarbageError short", make(pipewire.TrailingGarbageError, 1<<3-1), "got 7 bytes of trailing garbage"},
		{"TrailingGarbageError String", pipewire.TrailingGarbageError{
			/* size: */ 0, 0, 0, 0,
			/* type: */ byte(pipewire.SPA_TYPE_String), 0, 0, 0,
		}, "data has extra values starting with String"},
		{"TrailingGarbageError invalid", pipewire.TrailingGarbageError{
			/* size:    */ 0, 0, 0, 0,
			/* type:    */ 0xff, 0xff, 0xff, 0xff,
			/* garbage: */ 0,
		}, "data has extra values starting with invalid type field 0xffffffff"},

		{"StringTerminationError", pipewire.StringTerminationError(0xff), "got byte 255 instead of NUL"},

		{"InconsistentSizeError", pipewire.InconsistentSizeError{
			Prefix: 0xbad,
			Expect: 0xff,
		}, "prefix claims size 2989 for a 255-byte long segment"},

		{"UnexpectedTypeError zero", pipewire.UnexpectedTypeError{}, "received invalid type field 0x0 for a value of type invalid type field 0x0"},
		{"UnexpectedTypeError", pipewire.UnexpectedTypeError{
			Type:   pipewire.SPA_TYPE_String,
			Expect: pipewire.SPA_TYPE_Array,
		}, "received String for a value of type Array"},
		{"UnexpectedTypeError invalid", pipewire.UnexpectedTypeError{
			Type:   0xdeadbeef,
			Expect: pipewire.SPA_TYPE_Long,
		}, "received invalid type field 0xdeadbeef for a value of type Long"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error: %q, want %q", got, tc.want)
			}
		})
	}
}

var benchmarkSample = func() (sample pipewire.CoreInfo) {
	if err := sample.UnmarshalBinary(samplePWContainer[1][0][1]); err != nil {
		panic(err)
	}
	return
}()

func BenchmarkMarshal(b *testing.B) {
	for b.Loop() {
		if _, err := benchmarkSample.MarshalBinary(); err != nil {
			b.Fatalf("MarshalBinary: error = %v", err)
		}
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	for b.Loop() {
		if _, err := json.Marshal(benchmarkSample); err != nil {
			b.Fatalf("json.Marshal: error = %v", err)
		}
	}
}

func BenchmarkGobEncode(b *testing.B) {
	e := gob.NewEncoder(io.Discard)
	type sampleRaw pipewire.CoreInfo

	for b.Loop() {
		if err := e.Encode((*sampleRaw)(&benchmarkSample)); err != nil {
			b.Fatalf("(*gob.Encoder).Encode: error = %v", err)
		}
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	var got pipewire.CoreInfo

	for b.Loop() {
		if err := got.UnmarshalBinary(samplePWContainer[1][0][1]); err != nil {
			b.Fatalf("UnmarshalBinary: error = %v", err)
		}
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	var got pipewire.CoreInfo
	data, err := json.Marshal(benchmarkSample)
	if err != nil {
		b.Fatalf("json.Marshal: error = %v", err)
	}

	for b.Loop() {
		if err = json.Unmarshal(data, &got); err != nil {
			b.Fatalf("json.Unmarshal: error = %v", err)
		}
	}
}

func BenchmarkGobDecode(b *testing.B) {
	type sampleRaw pipewire.CoreInfo
	var buf bytes.Buffer
	e := gob.NewEncoder(&buf)
	d := gob.NewDecoder(&buf)

	for b.Loop() {
		b.StopTimer()
		if err := e.Encode((*sampleRaw)(&benchmarkSample)); err != nil {
			b.Fatalf("(*gob.Encoder).Encode: error = %v", err)
		}
		b.StartTimer()

		if err := d.Decode(new(sampleRaw)); err != nil {
			b.Fatalf("(*gob.Encoder).Decode: error = %v", err)
		}
	}
}

// mustMarshalJSON calls [json.Marshal] and returns the result.
func mustMarshalJSON(v any) string {
	if data, err := json.Marshal(v); err != nil {
		panic(err)
	} else {
		return string(data)
	}
}
