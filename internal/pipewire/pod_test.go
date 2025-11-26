package pipewire_test

import (
	"encoding"
	"encoding/json"
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

// mustMarshalJSON calls [json.Marshal] and returns the result.
func mustMarshalJSON(v any) string {
	if data, err := json.Marshal(v); err != nil {
		panic(err)
	} else {
		return string(data)
	}
}
