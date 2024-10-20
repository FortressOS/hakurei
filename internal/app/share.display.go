package app

import (
	"errors"
	"os"
	"path"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/fmsg"
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

func (seal *appSeal) shareDisplay() error {
	// pass $TERM to launcher
	if t, ok := os.LookupEnv(term); ok {
		seal.sys.bwrap.SetEnv[term] = t
	}

	// set up wayland
	if seal.et.Has(system.EWayland) {
		if wd, ok := os.LookupEnv(waylandDisplay); !ok {
			return fmsg.WrapError(ErrWayland,
				"WAYLAND_DISPLAY is not set")
		} else if seal.wlDone == nil {
			// hardlink wayland socket
			wp := path.Join(seal.RuntimePath, wd)
			wpi := path.Join(seal.shareLocal, "wayland")
			w := path.Join(seal.sys.runtime, "wayland-0")
			seal.sys.Link(wp, wpi)
			seal.sys.bwrap.SetEnv[waylandDisplay] = w
			seal.sys.bwrap.Bind(wpi, w)

			// ensure Wayland socket ACL (e.g. `/run/user/%d/wayland-%d`)
			seal.sys.UpdatePermType(system.EWayland, wp, acl.Read, acl.Write, acl.Execute)
		} else {
			// set wayland socket path (e.g. `/run/user/%d/wayland-%d`)
			seal.wl = path.Join(seal.RuntimePath, wd)
		}
	}

	// set up X11
	if seal.et.Has(system.EX11) {
		// discover X11 and grant user permission via the `ChangeHosts` command
		if d, ok := os.LookupEnv(display); !ok {
			return fmsg.WrapError(ErrXDisplay,
				"DISPLAY is not set")
		} else {
			seal.sys.ChangeHosts(seal.sys.user.Username)
			seal.sys.bwrap.SetEnv[display] = d
			seal.sys.bwrap.Bind("/tmp/.X11-unix", "/tmp/.X11-unix")
		}
	}

	return nil
}
