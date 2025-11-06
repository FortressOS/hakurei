package std_test

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	"hakurei.app/container/std"
)

func TestScmpSyscall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data string
		want std.ScmpSyscall
		err  error
	}{
		{"epoll_create1", `"epoll_create1"`, std.SNR_EPOLL_CREATE1, nil},
		{"clone3", `"clone3"`, std.SNR_CLONE3, nil},

		{"oob", `-2147483647`, -math.MaxInt32,
			&json.UnmarshalTypeError{Value: "number", Type: reflect.TypeFor[string](), Offset: 11}},
		{"name", `"nonexistent_syscall"`, -math.MaxInt32,
			std.SyscallNameError("nonexistent_syscall")},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("decode", func(t *testing.T) {
				var got std.ScmpSyscall
				if err := json.Unmarshal([]byte(tc.data), &got); !reflect.DeepEqual(err, tc.err) {
					t.Fatalf("Unmarshal: error = %#v, want %#v", err, tc.err)
				} else if err == nil && got != tc.want {
					t.Errorf("Unmarshal: %v, want %v", got, tc.want)
				}
			})
			if errors.As(tc.err, new(std.SyscallNameError)) {
				return
			}

			t.Run("encode", func(t *testing.T) {
				if got, err := json.Marshal(&tc.want); err != nil {
					t.Fatalf("Marshal: error = %v", err)
				} else if string(got) != tc.data {
					t.Errorf("Marshal: %s, want %s", string(got), tc.data)
				}
			})
		})
	}

	t.Run("error", func(t *testing.T) {
		const want = `invalid syscall name "\x00"`
		if got := std.SyscallNameError("\x00").Error(); got != want {
			t.Fatalf("Error: %q, want %q", got, want)
		}
	})
}
