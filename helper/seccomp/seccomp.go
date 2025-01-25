package seccomp

/*
#cgo linux pkg-config: --static libseccomp

#include "seccomp-export.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"runtime"
)

var CPrintln func(v ...any)

var resErr = [...]error{
	0: nil,
	1: errors.New("seccomp_init failed"),
	2: errors.New("seccomp_arch_add failed"),
	3: errors.New("seccomp_arch_add failed (multiarch)"),
	4: errors.New("internal libseccomp failure"),
	5: errors.New("seccomp_rule_add failed"),
	6: errors.New("seccomp_export_bpf failed"),
}

type SyscallOpts = C.f_syscall_opts

const (
	flagVerbose   SyscallOpts = C.F_VERBOSE
	FlagExt       SyscallOpts = C.F_EXT
	FlagDenyNS    SyscallOpts = C.F_DENY_NS
	FlagDenyTTY   SyscallOpts = C.F_DENY_TTY
	FlagDenyDevel SyscallOpts = C.F_DENY_DEVEL
	FlagMultiarch SyscallOpts = C.F_MULTIARCH
	FlagLinux32   SyscallOpts = C.F_LINUX32
	FlagCan       SyscallOpts = C.F_CAN
	FlagBluetooth SyscallOpts = C.F_BLUETOOTH
)

func tmpfile() (*os.File, error) {
	fd, err := C.f_tmpfile_fd()
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "tmpfile"), err
}

func exportFilter(fd uintptr, opts SyscallOpts) error {
	var (
		arch      C.uint32_t = 0
		multiarch C.uint32_t = 0
	)
	switch runtime.GOARCH {
	case "386":
		arch = C.SCMP_ARCH_X86
	case "amd64":
		arch = C.SCMP_ARCH_X86_64
		multiarch = C.SCMP_ARCH_X86
	case "arm":
		arch = C.SCMP_ARCH_ARM
	case "arm64":
		arch = C.SCMP_ARCH_AARCH64
		multiarch = C.SCMP_ARCH_ARM
	}

	// this removes repeated transitions between C and Go execution
	// when producing log output via F_println and CPrintln is nil
	if CPrintln != nil {
		opts |= flagVerbose
	}

	res, err := C.f_export_bpf(C.int(fd), arch, multiarch, opts)
	if re := resErr[res]; re != nil {
		if err == nil {
			return re
		}
		return fmt.Errorf("%s: %v", re.Error(), err)
	}
	return err
}

//export F_println
func F_println(v *C.char) {
	if CPrintln != nil {
		CPrintln(C.GoString(v))
	}
}
