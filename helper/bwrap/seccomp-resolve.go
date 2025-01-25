package bwrap

import (
	"fmt"
	"os"

	"git.gensokyo.uk/security/fortify/helper/seccomp"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

type SyscallPolicy struct {
	// disable fortify extensions
	Compat bool `json:"compat"`
	// deny development syscalls
	DenyDevel bool `json:"deny_devel"`
	// deny multiarch/emulation syscalls
	Multiarch bool `json:"multiarch"`
	// allow PER_LINUX32
	Linux32 bool `json:"linux32"`
	// allow AF_CAN
	Can bool `json:"can"`
	// allow AF_BLUETOOTH
	Bluetooth bool `json:"bluetooth"`
}

type seccompBuilder struct {
	config *Config
}

func (s *seccompBuilder) Len() int {
	if s == nil {
		return 0
	}
	return 2
}

func (s *seccompBuilder) Append(args *[]string, extraFiles *[]*os.File) error {
	if s == nil {
		return nil
	}
	if f, err := s.config.resolveSeccomp(); err != nil {
		return err
	} else {
		extraFile(args, extraFiles, positionalArgs[Seccomp], f)
		return nil
	}
}

func (c *Config) resolveSeccomp() (*os.File, error) {
	if c.Syscall == nil {
		return nil, nil
	}

	// resolve seccomp filter opts
	var (
		opts    seccomp.SyscallOpts
		optd    []string
		optCond = [...]struct {
			v bool
			o seccomp.SyscallOpts
			d string
		}{
			{!c.Syscall.Compat, seccomp.FlagExt, "fortify"},
			{!c.UserNS, seccomp.FlagDenyNS, "denyns"},
			{c.NewSession, seccomp.FlagDenyTTY, "denytty"},
			{c.Syscall.DenyDevel, seccomp.FlagDenyDevel, "denydevel"},
			{c.Syscall.Multiarch, seccomp.FlagMultiarch, "multiarch"},
			{c.Syscall.Linux32, seccomp.FlagLinux32, "linux32"},
			{c.Syscall.Can, seccomp.FlagCan, "can"},
			{c.Syscall.Bluetooth, seccomp.FlagBluetooth, "bluetooth"},
		}
	)
	if seccomp.CPrintln != nil {
		optd = make([]string, 1, len(optCond)+1)
		optd[0] = "common"
	}
	for _, opt := range optCond {
		if opt.v {
			opts |= opt.o
			if fmsg.Verbose() {
				optd = append(optd, opt.d)
			}
		}
	}
	if seccomp.CPrintln != nil {
		seccomp.CPrintln(fmt.Sprintf("seccomp flags: %s", optd))
	}

	return seccomp.Export(opts)
}
