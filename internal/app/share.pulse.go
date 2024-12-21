package app

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/linux"
	"git.gensokyo.uk/security/fortify/internal/system"
)

const (
	pulseServer = "PULSE_SERVER"
	pulseCookie = "PULSE_COOKIE"

	home          = "HOME"
	xdgConfigHome = "XDG_CONFIG_HOME"
)

var (
	ErrPulseCookie = errors.New("pulse cookie not present")
	ErrPulseSocket = errors.New("pulse socket not present")
	ErrPulseMode   = errors.New("unexpected pulse socket mode")
)

func (seal *appSeal) sharePulse(os linux.System) error {
	if !seal.et.Has(system.EPulse) {
		return nil
	}

	// check PulseAudio directory presence (e.g. `/run/user/%d/pulse`)
	pd := path.Join(seal.RuntimePath, "pulse")
	ps := path.Join(pd, "native")
	if _, err := os.Stat(pd); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmsg.WrapErrorSuffix(err,
				fmt.Sprintf("cannot access PulseAudio directory %q:", pd))
		}
		return fmsg.WrapError(ErrPulseSocket,
			fmt.Sprintf("PulseAudio directory %q not found", pd))
	}

	// check PulseAudio socket permission (e.g. `/run/user/%d/pulse/native`)
	if s, err := os.Stat(ps); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmsg.WrapErrorSuffix(err,
				fmt.Sprintf("cannot access PulseAudio socket %q:", ps))
		}
		return fmsg.WrapError(ErrPulseSocket,
			fmt.Sprintf("PulseAudio directory %q found but socket does not exist", pd))
	} else {
		if m := s.Mode(); m&0o006 != 0o006 {
			return fmsg.WrapError(ErrPulseMode,
				fmt.Sprintf("unexpected permissions on %q:", ps), m)
		}
	}

	// hard link pulse socket into target-executable share
	psi := path.Join(seal.shareLocal, "pulse")
	p := path.Join(seal.sys.runtime, "pulse", "native")
	seal.sys.Link(ps, psi)
	seal.sys.bwrap.Bind(psi, p)
	seal.sys.bwrap.SetEnv[pulseServer] = "unix:" + p

	// publish current user's pulse cookie for target user
	if src, err := discoverPulseCookie(os); err != nil {
		fmsg.VPrintln(err.(*fmsg.BaseError).Message())
	} else {
		dst := path.Join(seal.share, "pulse-cookie")
		seal.sys.bwrap.SetEnv[pulseCookie] = dst
		seal.sys.CopyFile(dst, src)
		seal.sys.bwrap.Bind(dst, dst)
	}

	return nil
}

// discoverPulseCookie attempts various standard methods to discover the current user's PulseAudio authentication cookie
func discoverPulseCookie(os linux.System) (string, error) {
	if p, ok := os.LookupEnv(pulseCookie); ok {
		return p, nil
	}

	// dotfile $HOME/.pulse-cookie
	if p, ok := os.LookupEnv(home); ok {
		p = path.Join(p, ".pulse-cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return p, fmsg.WrapErrorSuffix(err,
					fmt.Sprintf("cannot access PulseAudio cookie %q:", p))
			}
			// not found, try next method
		} else if !s.IsDir() {
			return p, nil
		}
	}

	// $XDG_CONFIG_HOME/pulse/cookie
	if p, ok := os.LookupEnv(xdgConfigHome); ok {
		p = path.Join(p, "pulse", "cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return p, fmsg.WrapErrorSuffix(err,
					fmt.Sprintf("cannot access PulseAudio cookie %q:", p))
			}
			// not found, try next method
		} else if !s.IsDir() {
			return p, nil
		}
	}

	return "", fmsg.WrapError(ErrPulseCookie,
		fmt.Sprintf("cannot locate PulseAudio cookie (tried $%s, $%s/pulse/cookie, $%s/.pulse-cookie)",
			pulseCookie, xdgConfigHome, home))
}
