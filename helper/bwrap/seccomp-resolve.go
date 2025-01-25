package bwrap

import (
	"fmt"
	"io"
	"os"

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
		opts    syscallOpts
		optd    []string
		optCond = [...]struct {
			v bool
			o syscallOpts
			d string
		}{
			{!c.Syscall.Compat, flagExt, "fortify"},
			{!c.UserNS, flagDenyNS, "denyns"},
			{c.NewSession, flagDenyTTY, "denytty"},
			{c.Syscall.DenyDevel, flagDenyDevel, "denydevel"},
			{c.Syscall.Multiarch, flagMultiarch, "multiarch"},
			{c.Syscall.Linux32, flagLinux32, "linux32"},
			{c.Syscall.Can, flagCan, "can"},
			{c.Syscall.Bluetooth, flagBluetooth, "bluetooth"},
		}
	)
	if CPrintln != nil {
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
	if CPrintln != nil {
		CPrintln(fmt.Sprintf("seccomp flags: %s", optd))
	}

	// export seccomp filter to tmpfile
	if f, err := tmpfile(); err != nil {
		return nil, err
	} else {
		return f, exportAndSeek(f, opts)
	}
}

func exportAndSeek(f *os.File, opts syscallOpts) error {
	if err := exportFilter(f.Fd(), opts); err != nil {
		return err
	}
	_, err := f.Seek(0, io.SeekStart)
	return err
}
