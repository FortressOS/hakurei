package container_test

import (
	"testing"
	"unsafe"

	"hakurei.app/container"
)

func TestLandlockString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		rulesetAttr *container.RulesetAttr
		want        string
	}{
		{"nil", nil, "NULL"},
		{"zero", new(container.RulesetAttr), "0"},
		{"some", &container.RulesetAttr{Scoped: container.LANDLOCK_SCOPE_SIGNAL}, "scoped: signal"},
		{"set", &container.RulesetAttr{
			HandledAccessFS:  container.LANDLOCK_ACCESS_FS_MAKE_SYM | container.LANDLOCK_ACCESS_FS_IOCTL_DEV | container.LANDLOCK_ACCESS_FS_WRITE_FILE,
			HandledAccessNet: container.LANDLOCK_ACCESS_NET_BIND_TCP,
			Scoped:           container.LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET | container.LANDLOCK_SCOPE_SIGNAL,
		}, "fs: write_file make_sym fs_ioctl_dev, net: bind_tcp, scoped: abstract_unix_socket signal"},
		{"all", &container.RulesetAttr{
			HandledAccessFS: container.LANDLOCK_ACCESS_FS_EXECUTE |
				container.LANDLOCK_ACCESS_FS_WRITE_FILE |
				container.LANDLOCK_ACCESS_FS_READ_FILE |
				container.LANDLOCK_ACCESS_FS_READ_DIR |
				container.LANDLOCK_ACCESS_FS_REMOVE_DIR |
				container.LANDLOCK_ACCESS_FS_REMOVE_FILE |
				container.LANDLOCK_ACCESS_FS_MAKE_CHAR |
				container.LANDLOCK_ACCESS_FS_MAKE_DIR |
				container.LANDLOCK_ACCESS_FS_MAKE_REG |
				container.LANDLOCK_ACCESS_FS_MAKE_SOCK |
				container.LANDLOCK_ACCESS_FS_MAKE_FIFO |
				container.LANDLOCK_ACCESS_FS_MAKE_BLOCK |
				container.LANDLOCK_ACCESS_FS_MAKE_SYM |
				container.LANDLOCK_ACCESS_FS_REFER |
				container.LANDLOCK_ACCESS_FS_TRUNCATE |
				container.LANDLOCK_ACCESS_FS_IOCTL_DEV,
			HandledAccessNet: container.LANDLOCK_ACCESS_NET_BIND_TCP |
				container.LANDLOCK_ACCESS_NET_CONNECT_TCP,
			Scoped: container.LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET |
				container.LANDLOCK_SCOPE_SIGNAL,
		}, "fs: execute write_file read_file read_dir remove_dir remove_file make_char make_dir make_reg make_sock make_fifo make_block make_sym fs_refer fs_truncate fs_ioctl_dev, net: bind_tcp connect_tcp, scoped: abstract_unix_socket signal"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.rulesetAttr.String(); got != tc.want {
				t.Errorf("String: %s, want %s", got, tc.want)
			}
		})
	}
}

func TestLandlockAttrSize(t *testing.T) {
	t.Parallel()
	want := 24
	if got := unsafe.Sizeof(container.RulesetAttr{}); got != uintptr(want) {
		t.Errorf("Sizeof: %d, want %d", got, want)
	}
}
