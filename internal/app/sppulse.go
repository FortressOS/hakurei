package app

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"

	"hakurei.app/container/check"
	"hakurei.app/hst"
)

const pulseCookieSizeMax = 1 << 8

func init() { gob.Register(new(spPulseOp)) }

// spPulseOp exports the PulseAudio server to the container.
type spPulseOp struct {
	// PulseAudio cookie data, populated during toSystem if a cookie is present.
	Cookie *[pulseCookieSizeMax]byte
}

func (s *spPulseOp) toSystem(state *outcomeStateSys, _ *hst.Config) error {
	pulseRuntimeDir, pulseSocket := s.commonPaths(state.outcomeState)

	if _, err := state.k.stat(pulseRuntimeDir.String()); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return &hst.AppError{Step: fmt.Sprintf("access PulseAudio directory %q", pulseRuntimeDir), Err: err}
		}
		return newWithMessage(fmt.Sprintf("PulseAudio directory %q not found", pulseRuntimeDir))
	}

	if fi, err := state.k.stat(pulseSocket.String()); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return &hst.AppError{Step: fmt.Sprintf("access PulseAudio socket %q", pulseSocket), Err: err}
		}
		return newWithMessage(fmt.Sprintf("PulseAudio directory %q found but socket does not exist", pulseRuntimeDir))
	} else {
		if m := fi.Mode(); m&0o006 != 0o006 {
			return newWithMessage(fmt.Sprintf("unexpected permissions on %q: %s", pulseSocket, m))
		}
	}

	// hard link pulse socket into target-executable share
	state.sys.Link(pulseSocket, state.runtime().Append("pulse"))

	// publish current user's pulse cookie for target user
	var paCookiePath *check.Absolute
	{
		const paLocateStep = "locate PulseAudio cookie"

		// from environment
		if p, ok := state.k.lookupEnv("PULSE_COOKIE"); ok {
			if a, err := check.NewAbs(p); err != nil {
				return &hst.AppError{Step: paLocateStep, Err: err}
			} else {
				// this takes precedence, do not verify whether the file is accessible
				paCookiePath = a
				goto out
			}
		}

		// $HOME/.pulse-cookie
		if p, ok := state.k.lookupEnv("HOME"); ok {
			if a, err := check.NewAbs(p); err != nil {
				return &hst.AppError{Step: paLocateStep, Err: err}
			} else {
				paCookiePath = a.Append(".pulse-cookie")
			}

			if fi, err := state.k.stat(paCookiePath.String()); err != nil {
				paCookiePath = nil
				if !errors.Is(err, fs.ErrNotExist) {
					return &hst.AppError{Step: "access PulseAudio cookie", Err: err}
				}
				// fallthrough
			} else if fi.IsDir() {
				paCookiePath = nil
			} else {
				goto out
			}
		}

		// $XDG_CONFIG_HOME/pulse/cookie
		if p, ok := state.k.lookupEnv("XDG_CONFIG_HOME"); ok {
			if a, err := check.NewAbs(p); err != nil {
				return &hst.AppError{Step: paLocateStep, Err: err}
			} else {
				paCookiePath = a.Append("pulse", "cookie")
			}
			if fi, err := state.k.stat(paCookiePath.String()); err != nil {
				paCookiePath = nil
				if !errors.Is(err, fs.ErrNotExist) {
					return &hst.AppError{Step: "access PulseAudio cookie", Err: err}
				}
				// fallthrough
			} else if fi.IsDir() {
				paCookiePath = nil
			} else {
				goto out
			}
		}
	out:
	}

	if paCookiePath != nil {
		if b, err := state.k.stat(paCookiePath.String()); err != nil {
			return &hst.AppError{Step: "access PulseAudio cookie", Err: err}
		} else {
			if b.IsDir() {
				return &hst.AppError{Step: "read PulseAudio cookie", Err: &os.PathError{Op: "stat", Path: paCookiePath.String(), Err: syscall.EISDIR}}
			}
			if b.Size() > pulseCookieSizeMax {
				return newWithMessageError(
					fmt.Sprintf("PulseAudio cookie at %q exceeds maximum expected size", paCookiePath),
					&os.PathError{Op: "stat", Path: paCookiePath.String(), Err: syscall.ENOMEM},
				)
			}
		}

		var r io.ReadCloser
		if f, err := state.k.open(paCookiePath.String()); err != nil {
			return &hst.AppError{Step: "open PulseAudio cookie", Err: err}
		} else {
			r = f
		}

		s.Cookie = new([pulseCookieSizeMax]byte)
		if n, err := r.Read(s.Cookie[:]); err != nil {
			if !errors.Is(err, io.EOF) {
				_ = r.Close()
				return &hst.AppError{Step: "read PulseAudio cookie", Err: err}
			}
			state.msg.Verbosef("copied %d bytes from %q", n, paCookiePath)
		}

		if err := r.Close(); err != nil {
			return &hst.AppError{Step: "close PulseAudio cookie", Err: err}
		}
	} else {
		state.msg.Verbose("cannot locate PulseAudio cookie (tried " +
			"$PULSE_COOKIE, " +
			"$XDG_CONFIG_HOME/pulse/cookie, " +
			"$HOME/.pulse-cookie)")
	}

	return nil
}

func (s *spPulseOp) toContainer(state *outcomeStateParams) error {
	innerPulseSocket := state.runtimeDir.Append("pulse", "native")
	state.params.Bind(state.runtimePath().Append("pulse"), innerPulseSocket, 0)
	state.env["PULSE_SERVER"] = "unix:" + innerPulseSocket.String()

	if s.Cookie != nil {
		innerDst := hst.AbsTmp.Append("/pulse-cookie")
		state.env["PULSE_COOKIE"] = innerDst.String()
		state.params.Place(innerDst, s.Cookie[:])
	}

	return nil
}

func (s *spPulseOp) commonPaths(state *outcomeState) (pulseRuntimeDir, pulseSocket *check.Absolute) {
	// PulseAudio runtime directory (usually `/run/user/%d/pulse`)
	pulseRuntimeDir = state.sc.RuntimePath.Append("pulse")
	// PulseAudio socket (usually `/run/user/%d/pulse/native`)
	pulseSocket = pulseRuntimeDir.Append("native")
	return
}
