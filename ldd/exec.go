package ldd

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
)

const lddTimeout = 2 * time.Second

var (
	msgStatic      = []byte("Not a valid dynamic program")
	msgStaticGlibc = []byte("not a dynamic executable")
)

func Exec(ctx context.Context, p string) ([]*Entry, error) {
	c, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()
	z := container.New(c, "ldd", p)
	z.Hostname = "hakurei-ldd"
	z.SeccompFlags |= seccomp.AllowMultiarch
	z.SeccompPresets |= seccomp.PresetStrict
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	z.Stdout = stdout
	z.Stderr = stderr
	z.Bind("/", "/", 0).Proc("/proc").Dev("/dev", false)

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
