package app

import "testing"

const (
	_POSIX_LOGIN_NAME_MAX = 9
)

func TestSysconf(t *testing.T) {
	t.Run("LOGIN_NAME_MAX", func(t *testing.T) {
		if got := sysconf(_SC_LOGIN_NAME_MAX); got < _POSIX_LOGIN_NAME_MAX {
			t.Errorf("sysconf(_SC_LOGIN_NAME_MAX): %d < _POSIX_LOGIN_NAME_MAX", got)
		}
	})
}
