package ldd

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"git.gensokyo.uk/security/fortify/internal/sandbox"
)

const lddTimeout = 2 * time.Second

var (
	msgStatic      = []byte("Not a valid dynamic program")
	msgStaticGlibc = []byte("not a dynamic executable")
)

func Exec(ctx context.Context, p string) ([]*Entry, error) { return ExecFilter(ctx, nil, nil, p) }

func ExecFilter(ctx context.Context,
	commandContext func(context.Context) *exec.Cmd,
	f func([]byte) []byte,
	p string) ([]*Entry, error) {
	c, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()
	container := sandbox.New(c, "ldd", p)
	container.CommandContext = commandContext
	container.Hostname = "fortify-ldd"
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	container.Stdout = stdout
	container.Stderr = stderr
	container.Bind("/", "/", 0).Proc("/proc").Dev("/dev")

	if err := container.Start(); err != nil {
		return nil, err
	}
	defer func() { _, _ = io.Copy(os.Stderr, stderr) }()
	if err := container.Serve(); err != nil {
		return nil, err
	}
	if err := container.Wait(); err != nil {
		m := stderr.Bytes()
		if bytes.Contains(m, append([]byte(p+": "), msgStatic...)) ||
			bytes.Contains(m, msgStaticGlibc) {
			return nil, nil
		}
		return nil, err
	}

	v := stdout.Bytes()
	if f != nil {
		v = f(v)
	}
	return Parse(v)
}
