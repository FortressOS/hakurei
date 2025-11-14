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

	// lddName is the file name of ldd(1) passed to exec.LookPath.
	lddName = "ldd"
	// lddTimeout is the maximum duration ldd(1) is allowed to ran for before it is terminated.
	lddTimeout = 4 * time.Second
)

// Resolve runs ldd(1) in a strict sandbox and connects its stdout to a [Decoder].
//
// The returned error has concrete type
// [exec.Error] or [check.AbsoluteError] for fault during lookup of ldd(1),
// [os.SyscallError] for fault creating the stdout pipe,
// [container.StartError] for fault during either stage of container setup.
// Otherwise, it passes through the return values of [Decoder.Decode].
func Resolve(ctx context.Context, msg message.Msg, pathname *check.Absolute) ([]*Entry, error) {
	c, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()

	var toolPath *check.Absolute
	if s, err := exec.LookPath(lddName); err != nil {
		return nil, err
	} else if toolPath, err = check.NewAbs(s); err != nil {
		return nil, err
	}

	z := container.NewCommand(c, msg, toolPath, lddName, pathname.String())
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

// Exec runs ldd(1) in a restrictive [container] and connects it to a [Decoder], returning resulting entries.
//
// Deprecated: this function takes an unchecked pathname string.
// Relative pathnames do not work in the container as working directory information is not sent.
func Exec(ctx context.Context, msg message.Msg, pathname string) ([]*Entry, error) {
	if a, err := check.NewAbs(pathname); err != nil {
		return nil, err
	} else {
		return Resolve(ctx, msg, a)
	}
}
