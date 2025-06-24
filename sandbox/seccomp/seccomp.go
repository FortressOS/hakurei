// Package seccomp provides filter presets and high level wrappers around libseccomp.
package seccomp

/*
#cgo linux pkg-config: --static libseccomp

#include "seccomp-build.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
)

// LibraryError represents a libseccomp error.
type LibraryError struct {
	Prefix  string
	Seccomp syscall.Errno
	Errno   error
}

func (e *LibraryError) Error() string {
	if e.Seccomp == 0 {
		if e.Errno == nil {
			panic("invalid libseccomp error")
		}
		return fmt.Sprintf("%s: %s", e.Prefix, e.Errno)
	}
	if e.Errno == nil {
		return fmt.Sprintf("%s: %s", e.Prefix, e.Seccomp)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Prefix, e.Seccomp, e.Errno)
}

func (e *LibraryError) Is(err error) bool {
	if e == nil {
		return err == nil
	}
	if ef, ok := err.(*LibraryError); ok {
		return *e == *ef
	}
	return (e.Seccomp != 0 && errors.Is(err, e.Seccomp)) ||
		(e.Errno != nil && errors.Is(err, e.Errno))
}

var resPrefix = [...]string{
	0: "",
	1: "seccomp_init failed",
	2: "seccomp_arch_add failed",
	3: "seccomp_arch_add failed (multiarch)",
	4: "internal libseccomp failure",
	5: "seccomp_rule_add failed",
	6: "seccomp_export_bpf failed",
	7: "seccomp_load failed",
}

type FilterOpts = C.hakurei_filter_opts

const (
	filterVerbose FilterOpts = C.HAKUREI_VERBOSE
	// FilterExt are project-specific extensions.
	FilterExt FilterOpts = C.HAKUREI_EXT
	// FilterDenyNS denies namespace setup syscalls.
	FilterDenyNS FilterOpts = C.HAKUREI_DENY_NS
	// FilterDenyTTY denies faking input.
	FilterDenyTTY FilterOpts = C.HAKUREI_DENY_TTY
	// FilterDenyDevel denies development-related syscalls.
	FilterDenyDevel FilterOpts = C.HAKUREI_DENY_DEVEL
	// FilterMultiarch allows multiarch/emulation.
	FilterMultiarch FilterOpts = C.HAKUREI_MULTIARCH
	// FilterLinux32 sets PER_LINUX32.
	FilterLinux32 FilterOpts = C.HAKUREI_LINUX32
	// FilterCan allows AF_CAN.
	FilterCan FilterOpts = C.HAKUREI_CAN
	// FilterBluetooth allows AF_BLUETOOTH.
	FilterBluetooth FilterOpts = C.HAKUREI_BLUETOOTH
)

func buildFilter(fd int, opts FilterOpts) error {
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
	// when producing log output via hakurei_println and CPrintln is nil
	if fp := printlnP.Load(); fp != nil {
		opts |= filterVerbose
	}

	var ret C.int
	res, err := C.hakurei_build_filter(&ret, C.int(fd), arch, multiarch, opts)
	if prefix := resPrefix[res]; prefix != "" {
		return &LibraryError{
			prefix,
			-syscall.Errno(ret),
			err,
		}
	}
	return err
}
