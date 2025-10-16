package app

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"syscall"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/message"
)

const pulseCookieSizeMax = 1 << 8

func init() { gob.Register(new(spPulseOp)) }

// spPulseOp exports the PulseAudio server to the container.
// Runs after spRuntimeOp.
type spPulseOp struct {
	// PulseAudio cookie data, populated during toSystem if a cookie is present.
	Cookie *[pulseCookieSizeMax]byte
}

func (s *spPulseOp) toSystem(state *outcomeStateSys) error {
	if state.et&hst.EPulse == 0 {
		return errNotEnabled
	}

	pulseRuntimeDir, pulseSocket := s.commonPaths(state.outcomeState)

	if _, err := state.k.stat(pulseRuntimeDir.String()); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return &hst.AppError{Step: fmt.Sprintf("access PulseAudio directory %q", pulseRuntimeDir), Err: err}
		}
		return newWithMessageError(fmt.Sprintf("PulseAudio directory %q not found", pulseRuntimeDir), err)
	}

	if fi, err := state.k.stat(pulseSocket.String()); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return &hst.AppError{Step: fmt.Sprintf("access PulseAudio socket %q", pulseSocket), Err: err}
		}
		return newWithMessageError(fmt.Sprintf("PulseAudio directory %q found but socket does not exist", pulseRuntimeDir), err)
	} else {
		if m := fi.Mode(); m&0o006 != 0o006 {
			return newWithMessage(fmt.Sprintf("unexpected permissions on %q: %s", pulseSocket, m))
		}
	}

	// pulse socket is world writable and its parent directory DAC permissions prevents access;
	// hard link to target-executable share directory to grant access
	state.sys.Link(pulseSocket, state.runtime().Append("pulse"))

	// load up to pulseCookieSizeMax bytes of pulse cookie for transmission to shim
	if a, err := discoverPulseCookie(state.k); err != nil {
		return err
	} else if a != nil {
		s.Cookie = new([pulseCookieSizeMax]byte)
		if err = loadFile(state.msg, state.k, "PulseAudio cookie", a.String(), s.Cookie[:]); err != nil {
			return err
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
		innerDst := hst.AbsPrivateTmp.Append("/pulse-cookie")
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

// discoverPulseCookie attempts to discover the pathname of the PulseAudio cookie of the current user.
// If both returned pathname and error are nil, the cookie is likely unavailable and can be silently skipped.
func discoverPulseCookie(k syscallDispatcher) (*check.Absolute, error) {
	const paLocateStep = "locate PulseAudio cookie"

	// from environment
	if p, ok := k.lookupEnv("PULSE_COOKIE"); ok {
		if a, err := check.NewAbs(p); err != nil {
			return nil, &hst.AppError{Step: paLocateStep, Err: err}
		} else {
			// this takes precedence, do not verify whether the file is accessible
			return a, nil
		}
	}

	// $HOME/.pulse-cookie
	if p, ok := k.lookupEnv("HOME"); ok {
		var pulseCookiePath *check.Absolute
		if a, err := check.NewAbs(p); err != nil {
			return nil, &hst.AppError{Step: paLocateStep, Err: err}
		} else {
			pulseCookiePath = a.Append(".pulse-cookie")
		}

		if fi, err := k.stat(pulseCookiePath.String()); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, &hst.AppError{Step: "access PulseAudio cookie", Err: err}
			}
			// fallthrough
		} else if fi.IsDir() {
			// fallthrough
		} else {
			return pulseCookiePath, nil
		}
	}

	// $XDG_CONFIG_HOME/pulse/cookie
	if p, ok := k.lookupEnv("XDG_CONFIG_HOME"); ok {
		var pulseCookiePath *check.Absolute
		if a, err := check.NewAbs(p); err != nil {
			return nil, &hst.AppError{Step: paLocateStep, Err: err}
		} else {
			pulseCookiePath = a.Append("pulse", "cookie")
		}

		if fi, err := k.stat(pulseCookiePath.String()); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, &hst.AppError{Step: "access PulseAudio cookie", Err: err}
			}
			// fallthrough
		} else if fi.IsDir() {
			// fallthrough
		} else {
			return pulseCookiePath, nil
		}
	}

	// cookie not present
	// not fatal: authentication is disabled
	return nil, nil
}

// loadFile reads up to len(buf) bytes from the file at pathname.
func loadFile(
	msg message.Msg, k syscallDispatcher,
	description, pathname string, buf []byte,
) error {
	n := len(buf)
	if n == 0 {
		return errors.New("invalid buffer")
	}
	msg.Verbosef("loading up to %d bytes from %q", n, pathname)

	if fi, err := k.stat(pathname); err != nil {
		return &hst.AppError{Step: "access " + description, Err: err}
	} else {
		if fi.IsDir() {
			return &hst.AppError{Step: "read " + description,
				Err: &os.PathError{Op: "stat", Path: pathname, Err: syscall.EISDIR}}
		}
		if s := fi.Size(); s > int64(n) {
			return newWithMessageError(
				description+" at "+strconv.Quote(pathname)+" exceeds maximum expected size",
				&os.PathError{Op: "stat", Path: pathname, Err: syscall.ENOMEM},
			)
		} else if s < int64(n) {
			msg.Verbosef("%s at %q is %d bytes longer than expected", description, pathname, int64(n)-s)
		}
	}

	if f, err := k.open(pathname); err != nil {
		return &hst.AppError{Step: "open " + description, Err: err}
	} else {
		if n, err = f.Read(buf); err != nil {
			if !errors.Is(err, io.EOF) {
				_ = f.Close()
				return &hst.AppError{Step: "read " + description, Err: err}
			}
			msg.Verbosef("copied %d bytes from %q", n, pathname)
		} // nil error indicates a partial read, which is handled after stat

		if err = f.Close(); err != nil {
			return &hst.AppError{Step: "close " + description, Err: err}
		}
		return nil
	}
}
