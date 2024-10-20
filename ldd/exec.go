package ldd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.ophivana.moe/security/fortify/helper"
	"git.ophivana.moe/security/fortify/helper/bwrap"
)

func Exec(p string) ([]*Entry, error) {
	var (
		h   helper.Helper
		cmd *exec.Cmd
	)

	if b, err := helper.NewBwrap((&bwrap.Config{
		Hostname:      "fortify-ldd",
		Chdir:         "/",
		NewSession:    true,
		DieWithParent: true,
	}).Bind("/", "/").DevTmpfs("/dev"),
		nil, "ldd", func(_, _ int) []string { return []string{p} }); err != nil {
		return nil, err
	} else {
		cmd = b.Unwrap()
		h = b
	}

	cmd.Stdout, cmd.Stderr = new(strings.Builder), os.Stderr
	if err := h.Start(); err != nil {
		return nil, err
	}
	if err := h.Wait(); err != nil {
		return nil, err
	}

	return Parse(cmd.Stdout.(fmt.Stringer))
}
