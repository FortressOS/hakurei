package wayland

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

		{"wl_display_connect_to_fd", Error{
			Cause: RConnect,
			Errno: stub.UniqueError(1),
		}, "wl_display_connect_to_fd failed: unique error 1 injected by the test suite"},

		{"wl_registry_add_listener", Error{
			Cause: RListener,
			Errno: stub.UniqueError(2),
		}, "wl_registry_add_listener failed: unique error 2 injected by the test suite"},

		{"wl_display_roundtrip", Error{
			Cause: RRoundtrip,
			Errno: stub.UniqueError(3),
		}, "wl_display_roundtrip failed: unique error 3 injected by the test suite"},

		{"not available", Error{
			Cause: RNotAvail,
		}, "compositor does not implement security_context_v1"},

		{"not available errno", Error{
			Cause: RNotAvail,
			Errno: syscall.EAGAIN,
		}, "compositor does not implement security_context_v1"},

		{"socket", Error{
			Cause: RSocket,
			Errno: stub.UniqueError(4),
		}, "socket: unique error 4 injected by the test suite"},

		{"bind", Error{
			Cause: RBind,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland",
			Errno: stub.UniqueError(5),
		}, "cannot bind /tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland: unique error 5 injected by the test suite"},

		{"listen", Error{
			Cause: RListen,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland",
			Errno: stub.UniqueError(6),
		}, "cannot listen on /tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland: unique error 6 injected by the test suite"},

		{"socket invalid", Error{
			Cause: RSocket,
		}, "socket operation failed"},

		{"create", Error{
			Cause: RCreate,
		}, "cannot ensure wayland pathname socket"},

		{"create path", Error{
			Cause: RCreate,
			Errno: &os.PathError{Op: "create", Path: "/proc/nonexistent", Err: syscall.EEXIST},
		}, "create /proc/nonexistent: file exists"},

		{"host socket", Error{
			Cause: RHostSocket,
			Errno: stub.UniqueError(7),
		}, "socket: unique error 7 injected by the test suite"},

		{"host connect", Error{
			Cause: RHostConnect,
			Host:  "/run/user/1971/wayland-1",
			Errno: stub.UniqueError(8),
		}, "cannot connect to /run/user/1971/wayland-1: unique error 8 injected by the test suite"},

		{"cleanup", Error{
			Cause: RCleanup,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland",
		}, "cannot hang up wayland security_context"},

		{"cleanup PathError", Error{
			Cause: RCleanup,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland",
			Errno: errors.Join(syscall.EINVAL, &os.PathError{
				Op:   "remove",
				Path: "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland",
				Err:  stub.UniqueError(9),
			}),
		}, "remove /tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland: unique error 9 injected by the test suite"},

		{"cleanup errno", Error{
			Cause: RCleanup,
			Path:  "/tmp/hakurei.0/18783d07791f2460dbbcffb76c24c9e6/wayland",
			Errno: errors.Join(syscall.EINVAL),
		}, "cannot close wayland close_fd pipe: invalid argument"},

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
		if got := securityContextBind("\x00", -1, "\x00", "\x00", -1); !reflect.DeepEqual(got, want) {
			t.Fatalf("securityContextBind: error = %#v, want %#v", got, want)
		}
	})

	t.Run("long", func(t *testing.T) {
		t.Parallel()
		// 256 bytes
		const oversizedPath = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

		want := &Error{Cause: RBind, Path: oversizedPath, Errno: errors.New("socket pathname too long")}
		if got := securityContextBind(oversizedPath, -1, "", "", -1); !reflect.DeepEqual(got, want) {
			t.Fatalf("securityContextBind: error = %#v, want %#v", got, want)
		}
	})
}
