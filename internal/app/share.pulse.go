package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/internal/state"
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

type (
	PulseCookieAccessError BaseError
	PulseSocketAccessError BaseError
)

func (seal *appSeal) sharePulse() error {
	if !seal.et.Has(state.EnablePulse) {
		return nil
	}

	// check PulseAudio directory presence (e.g. `/run/user/%d/pulse`)
	pd := path.Join(seal.RuntimePath, "pulse")
	ps := path.Join(pd, "native")
	if _, err := os.Stat(pd); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return (*PulseSocketAccessError)(wrapError(err,
				fmt.Sprintf("cannot access PulseAudio directory '%s':", pd), err))
		}
		return (*PulseSocketAccessError)(wrapError(ErrPulseSocket,
			fmt.Sprintf("PulseAudio directory '%s' not found", pd)))
	}

	// check PulseAudio socket permission (e.g. `/run/user/%d/pulse/native`)
	if s, err := os.Stat(ps); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return (*PulseSocketAccessError)(wrapError(err,
				fmt.Sprintf("cannot access PulseAudio socket '%s':", ps), err))
		}
		return (*PulseSocketAccessError)(wrapError(ErrPulseSocket,
			fmt.Sprintf("PulseAudio directory '%s' found but socket does not exist", pd)))
	} else {
		if m := s.Mode(); m&0o006 != 0o006 {
			return (*PulseSocketAccessError)(wrapError(ErrPulseMode,
				fmt.Sprintf("unexpected permissions on '%s':", ps), m))
		}
	}

	// hard link pulse socket into target-executable share
	psi := path.Join(seal.shareLocal, "pulse")
	p := path.Join(seal.sys.runtime, "pulse", "native")
	seal.sys.link(ps, psi)
	seal.sys.bwrap.Bind(psi, p)
	seal.sys.setEnv(pulseServer, "unix:"+p)

	// publish current user's pulse cookie for target user
	if src, err := discoverPulseCookie(); err != nil {
		return err
	} else {
		dst := path.Join(seal.share, "pulse-cookie")
		seal.sys.setEnv(pulseCookie, dst)
		seal.sys.copyFile(dst, src)
	}

	return nil
}

// discoverPulseCookie attempts various standard methods to discover the current user's PulseAudio authentication cookie
func discoverPulseCookie() (string, error) {
	if p, ok := os.LookupEnv(pulseCookie); ok {
		return p, nil
	}

	// dotfile $HOME/.pulse-cookie
	if p, ok := os.LookupEnv(home); ok {
		p = path.Join(p, ".pulse-cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return p, (*PulseCookieAccessError)(wrapError(err,
					fmt.Sprintf("cannot access PulseAudio cookie '%s':", p), err))
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
				return p, (*PulseCookieAccessError)(wrapError(err, "cannot access PulseAudio cookie", p+":", err))
			}
			// not found, try next method
		} else if !s.IsDir() {
			return p, nil
		}
	}

	return "", (*PulseCookieAccessError)(wrapError(ErrPulseCookie,
		fmt.Sprintf("cannot locate PulseAudio cookie (tried $%s, $%s/pulse/cookie, $%s/.pulse-cookie)",
			pulseCookie, xdgConfigHome, home)))
}
