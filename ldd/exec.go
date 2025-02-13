package ldd

import (
	"context"
	"os"
	"strings"
	"time"

	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

const lddTimeout = 2 * time.Second

func Exec(ctx context.Context, p string) ([]*Entry, error) {
	var h helper.Helper

	if b, err := helper.NewBwrap(
		(&bwrap.Config{
			Hostname:      "fortify-ldd",
			Chdir:         "/",
			Syscall:       &bwrap.SyscallPolicy{DenyDevel: true, Multiarch: true},
			NewSession:    true,
			DieWithParent: true,
		}).Bind("/", "/").DevTmpfs("/dev"), "ldd",
		nil, func(_, _ int) []string { return []string{p} },
		nil, nil,
	); err != nil {
		return nil, err
	} else {
		h = b
	}

	stdout := new(strings.Builder)
	h.Stdout(stdout).Stderr(os.Stderr)

	c, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()
	if err := h.Start(c, false); err != nil {
		return nil, err
	}
	if err := h.Wait(); err != nil {
		return nil, err
	}

	return Parse(stdout)
}
