package app

import (
	"errors"
	"path"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	"git.ophivana.moe/security/fortify/internal/linux"
	"git.ophivana.moe/security/fortify/internal/system"
)

const (
	term    = "TERM"
	display = "DISPLAY"

	// https://manpages.debian.org/experimental/libwayland-doc/wl_display_connect.3.en.html
	waylandDisplay = "WAYLAND_DISPLAY"
)

var (
	ErrWayland  = errors.New(waylandDisplay + " unset")
	ErrXDisplay = errors.New(display + " unset")
)

func (seal *appSeal) shareDisplay(os linux.System) error {
	// pass $TERM to launcher
	if t, ok := os.LookupEnv(term); ok {
		seal.sys.bwrap.SetEnv[term] = t
	}

	// set up wayland
	if seal.et.Has(system.EWayland) {
		var wp string
		if wd, ok := os.LookupEnv(waylandDisplay); !ok {
			return fmsg.WrapError(ErrWayland,
				"WAYLAND_DISPLAY is not set")
		} else {
			wp = path.Join(seal.RuntimePath, wd)
		}

		w := path.Join(seal.sys.runtime, "wayland-0")
		seal.sys.bwrap.SetEnv[waylandDisplay] = w

		if seal.directWayland {
			// hardlink wayland socket
			wpi := path.Join(seal.shareLocal, "wayland")
			seal.sys.Link(wp, wpi)
			seal.sys.bwrap.Bind(wpi, w)

			// ensure Wayland socket ACL (e.g. `/run/user/%d/wayland-%d`)
			seal.sys.UpdatePermType(system.EWayland, wp, acl.Read, acl.Write, acl.Execute)
		} else {
			wc := path.Join(seal.SharePath, "wayland")
			wt := path.Join(wc, seal.id)
			seal.sys.Ensure(wc, 0711)
			appID := seal.fid
			if appID == "" {
				// use instance ID in case app id is not set
				appID = "moe.ophivana.fortify." + seal.id
			}
			seal.sys.Wayland(wt, wp, appID, seal.id)
			seal.sys.bwrap.Bind(wt, w)
		}
	}

	// set up X11
	if seal.et.Has(system.EX11) {
		// discover X11 and grant user permission via the `ChangeHosts` command
		if d, ok := os.LookupEnv(display); !ok {
			return fmsg.WrapError(ErrXDisplay,
				"DISPLAY is not set")
		} else {
			seal.sys.ChangeHosts("#" + seal.sys.user.us)
			seal.sys.bwrap.SetEnv[display] = d
			seal.sys.bwrap.Bind("/tmp/.X11-unix", "/tmp/.X11-unix")
		}
	}

	return nil
}
