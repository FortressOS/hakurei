package ldd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"

	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

const lddTimeout = 2 * time.Second

var (
	msgStaticGlibc = []byte("not a dynamic executable")
)

func Exec(ctx context.Context, p string) ([]*Entry, error) {
	var h helper.Helper

	if toolPath, err := exec.LookPath("ldd"); err != nil {
		return nil, err
	} else if h, err = helper.NewBwrap(
		(&bwrap.Config{
			Hostname:      "fortify-ldd",
			Chdir:         "/",
			Syscall:       &bwrap.SyscallPolicy{DenyDevel: true, Multiarch: true},
			NewSession:    true,
			DieWithParent: true,
		}).Bind("/", "/").DevTmpfs("/dev"), toolPath,
		nil, func(_, _ int) []string { return []string{p} },
		nil, nil,
	); err != nil {
		return nil, err
	}

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	h.Stdout(stdout).Stderr(stderr)

	c, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()
	if err := h.Start(c, false); err != nil {
		return nil, err
	}
	if err := h.Wait(); err != nil {
		m := stderr.Bytes()
		if bytes.Contains(m, msgStaticGlibc) {
			return nil, nil
		}

		_, _ = os.Stderr.Write(m)
		return nil, err
	}

	return Parse(stdout)
}
