package container

import (
	"strings"
	"syscall"
	"unsafe"

	"hakurei.app/container/seccomp"
)

// include/uapi/linux/landlock.h

const (
	LANDLOCK_CREATE_RULESET_VERSION = 1 << iota
)

type LandlockAccessFS uintptr

const (
	LANDLOCK_ACCESS_FS_EXECUTE LandlockAccessFS = 1 << iota
	LANDLOCK_ACCESS_FS_WRITE_FILE
	LANDLOCK_ACCESS_FS_READ_FILE
	LANDLOCK_ACCESS_FS_READ_DIR
	LANDLOCK_ACCESS_FS_REMOVE_DIR
	LANDLOCK_ACCESS_FS_REMOVE_FILE
	LANDLOCK_ACCESS_FS_MAKE_CHAR
	LANDLOCK_ACCESS_FS_MAKE_DIR
	LANDLOCK_ACCESS_FS_MAKE_REG
	LANDLOCK_ACCESS_FS_MAKE_SOCK
	LANDLOCK_ACCESS_FS_MAKE_FIFO
	LANDLOCK_ACCESS_FS_MAKE_BLOCK
	LANDLOCK_ACCESS_FS_MAKE_SYM
	LANDLOCK_ACCESS_FS_REFER
	LANDLOCK_ACCESS_FS_TRUNCATE
	LANDLOCK_ACCESS_FS_IOCTL_DEV

	_LANDLOCK_ACCESS_FS_DELIM
)

func (f LandlockAccessFS) String() string {
	switch f {
	case LANDLOCK_ACCESS_FS_EXECUTE:
		return "execute"

	case LANDLOCK_ACCESS_FS_WRITE_FILE:
		return "write_file"

	case LANDLOCK_ACCESS_FS_READ_FILE:
		return "read_file"

	case LANDLOCK_ACCESS_FS_READ_DIR:
		return "read_dir"

	case LANDLOCK_ACCESS_FS_REMOVE_DIR:
		return "remove_dir"

	case LANDLOCK_ACCESS_FS_REMOVE_FILE:
		return "remove_file"

	case LANDLOCK_ACCESS_FS_MAKE_CHAR:
		return "make_char"

	case LANDLOCK_ACCESS_FS_MAKE_DIR:
		return "make_dir"

	case LANDLOCK_ACCESS_FS_MAKE_REG:
		return "make_reg"

	case LANDLOCK_ACCESS_FS_MAKE_SOCK:
		return "make_sock"

	case LANDLOCK_ACCESS_FS_MAKE_FIFO:
		return "make_fifo"

	case LANDLOCK_ACCESS_FS_MAKE_BLOCK:
		return "make_block"

	case LANDLOCK_ACCESS_FS_MAKE_SYM:
		return "make_sym"

	case LANDLOCK_ACCESS_FS_REFER:
		return "fs_refer"

	case LANDLOCK_ACCESS_FS_TRUNCATE:
		return "fs_truncate"

	case LANDLOCK_ACCESS_FS_IOCTL_DEV:
		return "fs_ioctl_dev"

	default:
		var c []LandlockAccessFS
		for i := LandlockAccessFS(1); i < _LANDLOCK_ACCESS_FS_DELIM; i <<= 1 {
			if f&i != 0 {
				c = append(c, i)
			}
		}
		if len(c) == 0 {
			return "NULL"
		}
		s := make([]string, len(c))
		for i, v := range c {
			s[i] = v.String()
		}
		return strings.Join(s, " ")
	}
}

type LandlockAccessNet uintptr

const (
	LANDLOCK_ACCESS_NET_BIND_TCP LandlockAccessNet = 1 << iota
	LANDLOCK_ACCESS_NET_CONNECT_TCP

	_LANDLOCK_ACCESS_NET_DELIM
)

func (f LandlockAccessNet) String() string {
	switch f {
	case LANDLOCK_ACCESS_NET_BIND_TCP:
		return "bind_tcp"

	case LANDLOCK_ACCESS_NET_CONNECT_TCP:
		return "connect_tcp"

	default:
		var c []LandlockAccessNet
		for i := LandlockAccessNet(1); i < _LANDLOCK_ACCESS_NET_DELIM; i <<= 1 {
			if f&i != 0 {
				c = append(c, i)
			}
		}
		if len(c) == 0 {
			return "NULL"
		}
		s := make([]string, len(c))
		for i, v := range c {
			s[i] = v.String()
		}
		return strings.Join(s, " ")
	}
}

type LandlockScope uintptr

const (
	LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET LandlockScope = 1 << iota
	LANDLOCK_SCOPE_SIGNAL

	_LANDLOCK_SCOPE_DELIM
)

func (f LandlockScope) String() string {
	switch f {
	case LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET:
		return "abstract_unix_socket"

	case LANDLOCK_SCOPE_SIGNAL:
		return "signal"

	default:
		var c []LandlockScope
		for i := LandlockScope(1); i < _LANDLOCK_SCOPE_DELIM; i <<= 1 {
			if f&i != 0 {
				c = append(c, i)
			}
		}
		if len(c) == 0 {
			return "NULL"
		}
		s := make([]string, len(c))
		for i, v := range c {
			s[i] = v.String()
		}
		return strings.Join(s, " ")
	}
}

type RulesetAttr struct {
	// Bitmask of handled filesystem actions.
	HandledAccessFS LandlockAccessFS
	// Bitmask of handled network actions.
	HandledAccessNet LandlockAccessNet
	// Bitmask of scopes restricting a Landlock domain from accessing outside resources (e.g. IPCs).
	Scoped LandlockScope
}

func (rulesetAttr *RulesetAttr) String() string {
	if rulesetAttr == nil {
		return "NULL"
	}
	elems := make([]string, 0, 3)
	if rulesetAttr.HandledAccessFS > 0 {
		elems = append(elems, "fs: "+rulesetAttr.HandledAccessFS.String())
	}
	if rulesetAttr.HandledAccessNet > 0 {
		elems = append(elems, "net: "+rulesetAttr.HandledAccessNet.String())
	}
	if rulesetAttr.Scoped > 0 {
		elems = append(elems, "scoped: "+rulesetAttr.Scoped.String())
	}
	if len(elems) == 0 {
		return "0"
	}
	return strings.Join(elems, ", ")
}

func (rulesetAttr *RulesetAttr) Create(flags uintptr) (fd int, err error) {
	var pointer, size uintptr
	// NULL needed for abi version
	if rulesetAttr != nil {
		pointer = uintptr(unsafe.Pointer(rulesetAttr))
		size = unsafe.Sizeof(*rulesetAttr)
	}

	rulesetFd, _, errno := syscall.Syscall(seccomp.SYS_LANDLOCK_CREATE_RULESET, pointer, size, flags)
	fd = int(rulesetFd)
	err = errno

	if fd < 0 {
		return
	}

	if rulesetAttr != nil { // not a fd otherwise
		syscall.CloseOnExec(fd)
	}
	return fd, nil
}

func LandlockGetABI() (int, error) {
	return (*RulesetAttr)(nil).Create(LANDLOCK_CREATE_RULESET_VERSION)
}

func LandlockRestrictSelf(rulesetFd int, flags uintptr) error {
	r, _, errno := syscall.Syscall(seccomp.SYS_LANDLOCK_RESTRICT_SELF, uintptr(rulesetFd), flags, 0)
	if r != 0 {
		return errno
	}
	return nil
}
