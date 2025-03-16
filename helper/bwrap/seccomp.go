package bwrap

import (
	"fmt"
	"strconv"

	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/sandbox/seccomp"
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

func (c *Config) seccompArgs() FDBuilder {
	// explicitly disable syscall filter
	if c.Syscall == nil {
		// nil File skips builder
		return new(seccompBuilder)
	}

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
	scmpPrintln := seccomp.GetOutput()
	if scmpPrintln != nil {
		optd = make([]string, 1, len(optCond)+1)
		optd[0] = "common"
	}
	for _, opt := range optCond {
		if opt.v {
			opts |= opt.o
			if scmpPrintln != nil {
				optd = append(optd, opt.d)
			}
		}
	}
	if scmpPrintln != nil {
		scmpPrintln(fmt.Sprintf("seccomp flags: %s", optd))
	}

	return &seccompBuilder{seccomp.NewFile(opts)}
}

type seccompBuilder struct{ proc.File }

func (s *seccompBuilder) Len() int {
	if s == nil || s.File == nil {
		return 0
	}
	return 2
}

func (s *seccompBuilder) Append(args *[]string) {
	if s == nil || s.File == nil {
		return
	}

	*args = append(*args, Seccomp.String(), strconv.Itoa(int(s.Fd())))
}
