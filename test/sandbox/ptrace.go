//go:build testtool

package sandbox

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	NULL = 0

	PTRACE_ATTACH             = 16
	PTRACE_DETACH             = 17
	PTRACE_SECCOMP_GET_FILTER = 0x420c
)

type ptraceError struct {
	op    string
	errno syscall.Errno
}

func (p *ptraceError) Error() string { return fmt.Sprintf("%s: %v", p.op, p.errno) }

func (p *ptraceError) Unwrap() error {
	if p.errno == 0 {
		return nil
	}
	return p.errno
}

func ptrace(op uintptr, pid, addr int, data unsafe.Pointer) (r uintptr, errno syscall.Errno) {
	r, _, errno = syscall.Syscall6(syscall.SYS_PTRACE, op, uintptr(pid), uintptr(addr), uintptr(data), NULL, NULL)
	return
}

func ptraceAttach(pid int) error {
	const (
		statePrefix = "State:"
		stateSuffix = "t (tracing stop)"
	)

	var r io.ReadSeekCloser
	if f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid)); err != nil {
		return err
	} else {
		r = f
	}

	if _, errno := ptrace(PTRACE_ATTACH, pid, 0, nil); errno != 0 {
		return &ptraceError{"PTRACE_ATTACH", errno}
	}

	// ugly! but there does not appear to be another way
	for {
		time.Sleep(10 * time.Millisecond)

		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return err
		}
		s := bufio.NewScanner(r)

		var found bool
		for s.Scan() {
			found = strings.HasPrefix(s.Text(), statePrefix)
			if found {
				break
			}
		}
		if err := s.Err(); err != nil {
			return err
		}

		if !found {
			return syscall.EBADE
		}

		if strings.HasSuffix(s.Text(), stateSuffix) {
			break
		}
	}

	return nil
}

func ptraceDetach(pid int) error {
	if _, errno := ptrace(PTRACE_DETACH, pid, 0, nil); errno != 0 {
		return &ptraceError{"PTRACE_DETACH", errno}
	}
	return nil
}

type sockFilter struct { /* Filter block */
	code uint16 /* Actual filter code */
	jt   uint8  /* Jump true */
	jf   uint8  /* Jump false */
	k    uint32 /* Generic multiuse field */
}

func getFilter[T comparable](pid, index int) ([]T, error) {
	if s := unsafe.Sizeof(*new(T)); s != 8 {
		panic(fmt.Sprintf("invalid filter block size %d", s))
	}

	var buf []T
	if n, errno := ptrace(PTRACE_SECCOMP_GET_FILTER, pid, index, nil); errno != 0 {
		return nil, &ptraceError{"PTRACE_SECCOMP_GET_FILTER", errno}
	} else {
		buf = make([]T, n)
	}
	if _, errno := ptrace(PTRACE_SECCOMP_GET_FILTER, pid, index, unsafe.Pointer(&buf[0])); errno != 0 {
		return nil, &ptraceError{"PTRACE_SECCOMP_GET_FILTER", errno}
	}
	return buf, nil
}
