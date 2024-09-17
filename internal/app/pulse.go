package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/util"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	pulseServer = "PULSE_SERVER"
	pulseCookie = "PULSE_COOKIE"

	home          = "HOME"
	xdgConfigHome = "XDG_CONFIG_HOME"
)

func (a *App) SharePulse() {
	a.setEnablement(internal.EnablePulse)

	// ensure PulseAudio directory ACL (e.g. `/run/user/%d/pulse`)
	pulse := path.Join(a.runtimePath, "pulse")
	pulseS := path.Join(pulse, "native")
	if s, err := os.Stat(pulse); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			internal.Fatal("Error accessing PulseAudio directory:", err)
		}
		internal.Fatal(fmt.Sprintf("PulseAudio dir '%s' not found", pulse))
	} else {
		// add environment variable for new process
		a.AppendEnv(pulseServer, "unix:"+pulseS)
		if err = acl.UpdatePerm(pulse, a.UID(), acl.Execute); err != nil {
			internal.Fatal("Error preparing PulseAudio:", err)
		} else {
			a.exit.RegisterRevertPath(pulse)
		}

		// ensure PulseAudio socket permission (e.g. `/run/user/%d/pulse/native`)
		if s, err = os.Stat(pulseS); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				internal.Fatal("PulseAudio directory found but socket does not exist")
			}
			internal.Fatal("Error accessing PulseAudio socket:", err)
		} else {
			if m := s.Mode(); m&0o006 != 0o006 {
				internal.Fatal(fmt.Sprintf("Unexpected permissions on '%s':", pulseS), m)
			}
		}

		// Publish current user's pulse-cookie for target user
		pulseCookieSource := discoverPulseCookie()
		pulseCookieFinal := path.Join(a.sharePath, "pulse-cookie")
		a.AppendEnv(pulseCookie, pulseCookieFinal)
		verbose.Printf("Publishing PulseAudio cookie '%s' to '%s'\n", pulseCookieSource, pulseCookieFinal)
		if err = util.CopyFile(pulseCookieFinal, pulseCookieSource); err != nil {
			internal.Fatal("Error copying PulseAudio cookie:", err)
		}
		if err = acl.UpdatePerm(pulseCookieFinal, a.UID(), acl.Read); err != nil {
			internal.Fatal("Error publishing PulseAudio cookie:", err)
		} else {
			a.exit.RegisterRevertPath(pulseCookieFinal)
		}

		verbose.Printf("PulseAudio dir '%s' configured\n", pulse)
	}
}

// discoverPulseCookie try various standard methods to discover the current user's PulseAudio authentication cookie
func discoverPulseCookie() string {
	if p, ok := os.LookupEnv(pulseCookie); ok {
		return p
	}

	if p, ok := os.LookupEnv(home); ok {
		p = path.Join(p, ".pulse-cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				internal.Fatal("Error accessing PulseAudio cookie:", err)
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
				internal.Fatal("Error accessing PulseAudio cookie:", err)
				// unreachable
				return p
			}
		} else if !s.IsDir() {
			return p
		}
	}

	internal.Fatal(fmt.Sprintf("Cannot locate PulseAudio cookie (tried $%s, $%s/pulse/cookie, $%s/.pulse-cookie)",
		pulseCookie, xdgConfigHome, home))
	return ""
}
