package util

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/internal/state"
)

const (
	systemdCheckPath = "/run/systemd/system"

	home          = "HOME"
	xdgConfigHome = "XDG_CONFIG_HOME"

	PulseServer = "PULSE_SERVER"
	PulseCookie = "PULSE_COOKIE"
)

// SdBooted implements https://www.freedesktop.org/software/systemd/man/sd_booted.html
func SdBooted() (bool, error) {
	_, err := os.Stat(systemdCheckPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return false, err
	}

	return true, nil
}

// DiscoverPulseCookie try various standard methods to discover the current user's PulseAudio authentication cookie
func DiscoverPulseCookie() string {
	if p, ok := os.LookupEnv(PulseCookie); ok {
		return p
	}

	if p, ok := os.LookupEnv(home); ok {
		p = path.Join(p, ".pulse-cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				state.Fatal("Error accessing PulseAudio cookie:", err)
				// unreachable
				return p
			}
		} else if !s.IsDir() {
			return p
		}
	}

	if p, ok := os.LookupEnv(xdgConfigHome); ok {
		p = path.Join(p, "pulse", "cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				state.Fatal("Error accessing PulseAudio cookie:", err)
				// unreachable
				return p
			}
		} else if !s.IsDir() {
			return p
		}
	}

	state.Fatal(fmt.Sprintf("Cannot locate PulseAudio cookie (tried $%s, $%s/pulse/cookie, $%s/.pulse-cookie)",
		PulseCookie, xdgConfigHome, home))
	return ""
}
