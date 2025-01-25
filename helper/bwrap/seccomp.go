package bwrap

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

type (
	syscallOpts = C.f_syscall_opts
)

const (
	flagExt       syscallOpts = C.F_EXT
	flagDenyNS    syscallOpts = C.F_DENY_NS
	flagDenyTTY   syscallOpts = C.F_DENY_TTY
	flagDenyDevel syscallOpts = C.F_DENY_DEVEL
	flagMultiarch syscallOpts = C.F_MULTIARCH
	flagLinux32   syscallOpts = C.F_LINUX32
	flagCan       syscallOpts = C.F_CAN
	flagBluetooth syscallOpts = C.F_BLUETOOTH
)

func tmpfile() (*os.File, error) {
	fd, err := C.f_tmpfile_fd()
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "tmpfile"), err
}

func exportFilter(fd uintptr, opts syscallOpts) error {
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
