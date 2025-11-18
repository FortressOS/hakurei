package pipewire

import (
	"errors"
	"os"
	"reflect"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
)

func TestError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  Error
		want string
	}{
		{"success", Error{
			Cause: RSuccess,
		}, "success"},

		{"success errno", Error{
			Cause: RSuccess,
			Errno: stub.UniqueError(0),
		}, "unique error 0 injected by the test suite"},

		{"pw_main_loop_new", Error{
			Cause: RMainloop,
			Errno: stub.UniqueError(1),
		}, "pw_main_loop_new failed: unique error 1 injected by the test suite"},

		{"pw_context_new", Error{
			Cause: RContext,
			Errno: stub.UniqueError(2),
		}, "pw_context_new failed: unique error 2 injected by the test suite"},

		{"pw_context_connect", Error{
			Cause: RConnect,
			Errno: stub.UniqueError(3),
		}, "pw_context_connect failed: unique error 3 injected by the test suite"},

		{"pw_core_get_registry", Error{
			Cause: RRegistry,
			Errno: stub.UniqueError(4),
		}, "pw_core_get_registry failed: unique error 4 injected by the test suite"},

		{"not available", Error{
			Cause: RNotAvail,
		}, "no security context object found"},

		{"not available errno", Error{
			Cause: RNotAvail,
			Errno: syscall.EAGAIN,
		}, "no security context object found"},

		{"socket", Error{
			Cause: RSocket,
			Errno: stub.UniqueError(5),
		}, "socket: unique error 5 injected by the test suite"},

		{"bind", Error{
			Cause: RBind,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire",
			Errno: stub.UniqueError(6),
		}, "cannot bind /tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire: unique error 6 injected by the test suite"},

		{"listen", Error{
			Cause: RListen,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire",
			Errno: stub.UniqueError(7),
		}, "cannot listen on /tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire: unique error 7 injected by the test suite"},

		{"socket invalid", Error{
			Cause: RSocket,
		}, "socket operation failed"},

		{"pw_security_context_create", Error{
			Cause: RAttach,
			Errno: stub.UniqueError(8),
		}, "pw_security_context_create failed: unique error 8 injected by the test suite"},

		{"create", Error{
			Cause: RCreate,
		}, "cannot ensure pipewire pathname socket"},

		{"create path", Error{
			Cause: RCreate,
			Errno: &os.PathError{Op: "create", Path: "/proc/nonexistent", Err: syscall.EEXIST},
		}, "create /proc/nonexistent: file exists"},

		{"cleanup", Error{
			Cause: RCleanup,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire",
		}, "cannot hang up pipewire security context"},

		{"cleanup PathError", Error{
			Cause: RCleanup,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire",
			Errno: errors.Join(syscall.EINVAL, &os.PathError{
				Op:   "remove",
				Path: "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire",
				Err:  stub.UniqueError(9),
			}),
		}, "remove /tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire: unique error 9 injected by the test suite"},

		{"cleanup errno", Error{
			Cause: RCleanup,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/pipewire",
			Errno: errors.Join(syscall.EINVAL),
		}, "cannot close pipewire close_fd pipe: invalid argument"},

		{"invalid", Error{
			Cause: 0xbad,
		}, "impossible outcome"},

		{"invalid errno", Error{
			Cause: 0xbad,
			Errno: stub.UniqueError(9),
		}, "impossible outcome: unique error 9 injected by the test suite"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Message(); got != tc.want {
				t.Errorf("Message: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSecurityContextBindValidate(t *testing.T) {
	t.Parallel()

	t.Run("NUL", func(t *testing.T) {
		t.Parallel()

		want := &Error{Cause: RBind, Path: "\x00", Errno: errors.New("argument contains NUL character")}
		if got := securityContextBind("\x00", "\x00", -1); !reflect.DeepEqual(got, want) {
			t.Fatalf("securityContextBind: error = %#v, want %#v", got, want)
		}
	})

	t.Run("long", func(t *testing.T) {
		t.Parallel()
		// 256 bytes
		const oversizedPath = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

		want := &Error{Cause: RBind, Path: oversizedPath, Errno: errors.New("socket pathname too long")}
		if got := securityContextBind(oversizedPath, "", -1); !reflect.DeepEqual(got, want) {
			t.Fatalf("securityContextBind: error = %#v, want %#v", got, want)
		}
	})
}
