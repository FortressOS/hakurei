package ldd

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
)

const (
	lddName    = "ldd"
	lddTimeout = 2 * time.Second
)

var (
	msgStatic      = []byte("Not a valid dynamic program")
	msgStaticGlibc = []byte("not a dynamic executable")
)

func Exec(ctx context.Context, msg container.Msg, p string) ([]*Entry, error) {
	c, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()

	var toolPath *container.Absolute
	if s, err := exec.LookPath(lddName); err != nil {
		return nil, err
	} else if toolPath, err = container.NewAbs(s); err != nil {
		return nil, err
	}

	z := container.NewCommand(c, msg, toolPath, lddName, p)
	z.Hostname = "hakurei-" + lddName
	z.SeccompFlags |= seccomp.AllowMultiarch
	z.SeccompPresets |= seccomp.PresetStrict
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	z.Stdout = stdout
	z.Stderr = stderr
	z.
		Bind(container.AbsFHSRoot, container.AbsFHSRoot, 0).
		Proc(container.AbsFHSProc).
		Dev(container.AbsFHSDev, false)

	if err := z.Start(); err != nil {
		return nil, err
	}
	defer func() { _, _ = io.Copy(os.Stderr, stderr) }()
	if err := z.Serve(); err != nil {
		return nil, err
	}
	if err := z.Wait(); err != nil {
		m := stderr.Bytes()
		if bytes.Contains(m, append([]byte(p+": "), msgStatic...)) ||
			bytes.Contains(m, msgStaticGlibc) {
			return nil, nil
		}
		return nil, err
	}

	v := stdout.Bytes()
	return Parse(v)
}
