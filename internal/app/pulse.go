package app

import (
	"errors"
	"fmt"
	"git.ophivana.moe/cat/fortify/internal/final"
	"io/fs"
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/internal/acl"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/util"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

func (a *App) SharePulse() {
	a.setEnablement(state.EnablePulse)

	// ensure PulseAudio directory ACL (e.g. `/run/user/%d/pulse`)
	pulse := path.Join(a.runtimePath, "pulse")
	pulseS := path.Join(pulse, "native")
	if s, err := os.Stat(pulse); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			final.Fatal("Error accessing PulseAudio directory:", err)
		}
		final.Fatal(fmt.Sprintf("PulseAudio dir '%s' not found", pulse))
	} else {
		// add environment variable for new process
		a.AppendEnv(util.PulseServer, "unix:"+pulseS)
		if err = acl.UpdatePerm(pulse, a.UID(), acl.Execute); err != nil {
			final.Fatal("Error preparing PulseAudio:", err)
		} else {
			final.RegisterRevertPath(pulse)
		}

		// ensure PulseAudio socket permission (e.g. `/run/user/%d/pulse/native`)
		if s, err = os.Stat(pulseS); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				final.Fatal("PulseAudio directory found but socket does not exist")
			}
			final.Fatal("Error accessing PulseAudio socket:", err)
		} else {
			if m := s.Mode(); m&0o006 != 0o006 {
				final.Fatal(fmt.Sprintf("Unexpected permissions on '%s':", pulseS), m)
			}
		}

		// Publish current user's pulse-cookie for target user
		pulseCookieSource := util.DiscoverPulseCookie()
		pulseCookieFinal := path.Join(a.sharePath, "pulse-cookie")
		a.AppendEnv(util.PulseCookie, pulseCookieFinal)
		verbose.Printf("Publishing PulseAudio cookie '%s' to '%s'\n", pulseCookieSource, pulseCookieFinal)
		if err = util.CopyFile(pulseCookieFinal, pulseCookieSource); err != nil {
			final.Fatal("Error copying PulseAudio cookie:", err)
		}
		if err = acl.UpdatePerm(pulseCookieFinal, a.UID(), acl.Read); err != nil {
			final.Fatal("Error publishing PulseAudio cookie:", err)
		} else {
			final.RegisterRevertPath(pulseCookieFinal)
		}

		verbose.Printf("PulseAudio dir '%s' configured\n", pulse)
	}
}
