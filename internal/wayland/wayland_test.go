package wayland_test

import (
	"os"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/internal/wayland"
)

func TestError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  wayland.Error
		want string
	}{
		{"success", wayland.Error{
			Cause: wayland.RSuccess,
		}, "success"},

		{"success errno", wayland.Error{
			Cause: wayland.RSuccess,
			Errno: stub.UniqueError(0),
		}, "unique error 0 injected by the test suite"},

		{"wl_display_connect_to_fd", wayland.Error{
			Cause: wayland.RConnect,
			Errno: stub.UniqueError(1),
		}, "wl_display_connect_to_fd failed: unique error 1 injected by the test suite"},

		{"wl_registry_add_listener", wayland.Error{
			Cause: wayland.RListener,
			Errno: stub.UniqueError(2),
		}, "wl_registry_add_listener failed: unique error 2 injected by the test suite"},

		{"wl_display_roundtrip", wayland.Error{
			Cause: wayland.RRoundtrip,
			Errno: stub.UniqueError(3),
		}, "wl_display_roundtrip failed: unique error 3 injected by the test suite"},

		{"not available", wayland.Error{
			Cause: wayland.RNotAvail,
		}, "compositor does not implement security_context_v1"},

		{"not available errno", wayland.Error{
			Cause: wayland.RNotAvail,
			Errno: syscall.EAGAIN,
		}, "compositor does not implement security_context_v1"},

		{"socket", wayland.Error{
			Cause: wayland.RSocket,
			Errno: stub.UniqueError(4),
		}, "unique error 4 injected by the test suite"},

		{"socket invalid", wayland.Error{
			Cause: wayland.RSocket,
		}, "socket operation failed"},

		{"host create", wayland.Error{
			Cause: wayland.RHostCreate,
		}, "cannot ensure wayland pathname socket"},

		{"host create path", wayland.Error{
			Cause: wayland.RHostCreate,
			Errno: &os.PathError{Op: "create", Path: "/proc/nonexistent", Err: syscall.EEXIST},
		}, "create /proc/nonexistent: file exists"},

		{"host socket", wayland.Error{
			Cause: wayland.RHostSocket,
			Errno: stub.UniqueError(5),
		}, "socket for host wayland server: unique error 5 injected by the test suite"},

		{"host connect", wayland.Error{
			Cause: wayland.RHostConnect,
			Errno: stub.UniqueError(6),
		}, "connect to host wayland server: unique error 6 injected by the test suite"},

		{"invalid", wayland.Error{
			Cause: 0xbad,
		}, "impossible outcome"},

		{"invalid errno", wayland.Error{
			Cause: 0xbad,
			Errno: stub.UniqueError(5),
		}, "impossible outcome: unique error 5 injected by the test suite"},
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
