package ldd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/std"
	"hakurei.app/message"
)

const (
	// msgStaticSuffix is the suffix of message printed to stderr by musl on a statically linked program.
	msgStaticSuffix = ": Not a valid dynamic program"
	// msgStaticGlibc is a substring of the message printed to stderr by glibc on a statically linked program.
	msgStaticGlibc = "not a dynamic executable"
)

// Exec runs ldd(1) in a restrictive [container] and connects it to a [Decoder], returning resulting entries.
func Exec(ctx context.Context, msg message.Msg, p string) ([]*Entry, error) {
	const (
		lddName    = "ldd"
		lddTimeout = 4 * time.Second
	)

	c, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()

	var toolPath *check.Absolute
	if s, err := exec.LookPath(lddName); err != nil {
		return nil, err
	} else if toolPath, err = check.NewAbs(s); err != nil {
		return nil, err
	}

	z := container.NewCommand(c, msg, toolPath, lddName, p)
	z.Hostname = "hakurei-" + lddName
	z.SeccompFlags |= seccomp.AllowMultiarch
	z.SeccompPresets |= std.PresetStrict
	stderr := new(bytes.Buffer)
	z.Stderr = stderr
	z.
		Bind(fhs.AbsRoot, fhs.AbsRoot, 0).
		Proc(fhs.AbsProc).
		Dev(fhs.AbsDev, false)

	var d *Decoder
	if r, err := z.StdoutPipe(); err != nil {
		return nil, err
	} else {
		d = NewDecoder(r)
	}

	if err := z.Start(); err != nil {
		return nil, err
	}
	defer func() { _, _ = io.Copy(os.Stderr, stderr) }()
	if err := z.Serve(); err != nil {
		return nil, err
	}

	entries, decodeErr := d.Decode()
	if decodeErr != nil {
		// do not cancel on successful decode to avoid racing with ldd(1) termination
		cancel()
	}

	if err := z.Wait(); err != nil {
		m := stderr.Bytes()
		if bytes.Contains(m, []byte(msgStaticSuffix)) || bytes.Contains(m, []byte(msgStaticGlibc)) {
			return nil, nil
		}

		if decodeErr != nil {
			return nil, errors.Join(decodeErr, err)
		}
		return nil, err
	}
	return entries, decodeErr
}
