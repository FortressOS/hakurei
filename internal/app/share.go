package app

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/linux"
	"git.gensokyo.uk/security/fortify/internal/system"
)

const (
	home  = "HOME"
	shell = "SHELL"

	xdgConfigHome   = "XDG_CONFIG_HOME"
	xdgRuntimeDir   = "XDG_RUNTIME_DIR"
	xdgSessionClass = "XDG_SESSION_CLASS"
	xdgSessionType  = "XDG_SESSION_TYPE"

	term    = "TERM"
	display = "DISPLAY"

	// https://manpages.debian.org/experimental/libwayland-doc/wl_display_connect.3.en.html
	waylandDisplay = "WAYLAND_DISPLAY"

	pulseServer = "PULSE_SERVER"
	pulseCookie = "PULSE_COOKIE"

	dbusSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"
	dbusSystemBusAddress  = "DBUS_SYSTEM_BUS_ADDRESS"
)

var (
	ErrWayland  = errors.New(waylandDisplay + " unset")
	ErrXDisplay = errors.New(display + " unset")

	ErrPulseCookie = errors.New("pulse cookie not present")
	ErrPulseSocket = errors.New("pulse socket not present")
	ErrPulseMode   = errors.New("unexpected pulse socket mode")
)

func (seal *appSeal) setupShares(bus [2]*dbus.Config, os linux.System) error {
	if seal.shared {
		panic("seal shared twice")
	}
	seal.shared = true

	/*
		Tmpdir-based share directory
	*/

	// ensure Share (e.g. `/tmp/fortify.%d`)
	// acl is unnecessary as this directory is world executable
	seal.sys.Ensure(seal.SharePath, 0711)

	// ensure process-specific share (e.g. `/tmp/fortify.%d/%s`)
	// acl is unnecessary as this directory is world executable
	seal.share = path.Join(seal.SharePath, seal.id)
	seal.sys.Ephemeral(system.Process, seal.share, 0711)

	// ensure child tmpdir parent directory (e.g. `/tmp/fortify.%d/tmpdir`)
	targetTmpdirParent := path.Join(seal.SharePath, "tmpdir")
	seal.sys.Ensure(targetTmpdirParent, 0700)
	seal.sys.UpdatePermType(system.User, targetTmpdirParent, acl.Execute)

	// ensure child tmpdir (e.g. `/tmp/fortify.%d/tmpdir/%d`)
	targetTmpdir := path.Join(targetTmpdirParent, seal.sys.user.as)
	seal.sys.Ensure(targetTmpdir, 01700)
	seal.sys.UpdatePermType(system.User, targetTmpdir, acl.Read, acl.Write, acl.Execute)
	seal.sys.bwrap.Bind(targetTmpdir, "/tmp", false, true)

	/*
		XDG runtime directory
	*/

	// mount tmpfs on inner runtime (e.g. `/run/user/%d`)
	seal.sys.bwrap.Tmpfs("/run/user", 1*1024*1024)
	seal.sys.bwrap.Tmpfs(seal.sys.runtime, 8*1024*1024)

	// point to inner runtime path `/run/user/%d`
	seal.sys.bwrap.SetEnv[xdgRuntimeDir] = seal.sys.runtime
	seal.sys.bwrap.SetEnv[xdgSessionClass] = "user"
	seal.sys.bwrap.SetEnv[xdgSessionType] = "tty"

	// ensure RunDir (e.g. `/run/user/%d/fortify`)
	seal.sys.Ensure(seal.RunDirPath, 0700)
	seal.sys.UpdatePermType(system.User, seal.RunDirPath, acl.Execute)

	// ensure runtime directory ACL (e.g. `/run/user/%d`)
	seal.sys.Ensure(seal.RuntimePath, 0700) // ensure this dir in case XDG_RUNTIME_DIR is unset
	seal.sys.UpdatePermType(system.User, seal.RuntimePath, acl.Execute)

	// ensure process-specific share local to XDG_RUNTIME_DIR (e.g. `/run/user/%d/fortify/%s`)
	seal.shareLocal = path.Join(seal.RunDirPath, seal.id)
	seal.sys.Ephemeral(system.Process, seal.shareLocal, 0700)
	seal.sys.UpdatePerm(seal.shareLocal, acl.Execute)

	/*
		Inner passwd database
	*/

	// look up shell
	sh := "/bin/sh"
	if s, ok := os.LookupEnv(shell); ok {
		seal.sys.bwrap.SetEnv[shell] = s
		sh = s
	}

	// generate /etc/passwd
	passwdPath := path.Join(seal.share, "passwd")
	username := "chronos"
	if seal.sys.user.username != "" {
		username = seal.sys.user.username
	}
	homeDir := "/var/empty"
	if seal.sys.user.home != "" {
		homeDir = seal.sys.user.home
	}

	// bind home directory
	seal.sys.bwrap.Bind(seal.sys.user.data, homeDir, false, true)
	seal.sys.bwrap.Chdir = homeDir

	seal.sys.bwrap.SetEnv["USER"] = username
	seal.sys.bwrap.SetEnv["HOME"] = homeDir

	passwd := username + ":x:" + seal.sys.mappedIDString + ":" + seal.sys.mappedIDString + ":Fortify:" + homeDir + ":" + sh + "\n"
	seal.sys.Write(passwdPath, passwd)

	// write /etc/group
	groupPath := path.Join(seal.share, "group")
	seal.sys.Write(groupPath, "fortify:x:"+seal.sys.mappedIDString+":\n")

	// bind /etc/passwd and /etc/group
	seal.sys.bwrap.Bind(passwdPath, "/etc/passwd")
	seal.sys.bwrap.Bind(groupPath, "/etc/group")

	/*
		Display servers
	*/

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

		if !seal.directWayland { // set up security-context-v1
			wc := path.Join(seal.SharePath, "wayland")
			wt := path.Join(wc, seal.id)
			seal.sys.Ensure(wc, 0711)
			appID := seal.fid
			if appID == "" {
				// use instance ID in case app id is not set
				appID = "uk.gensokyo.fortify." + seal.id
			}
			seal.sys.Wayland(wt, wp, appID, seal.id)
			seal.sys.bwrap.Bind(wt, w)
		} else { // bind mount wayland socket (insecure)
			// hardlink wayland socket
			wpi := path.Join(seal.shareLocal, "wayland")
			seal.sys.Link(wp, wpi)
			seal.sys.bwrap.Bind(wpi, w)

			// ensure Wayland socket ACL (e.g. `/run/user/%d/wayland-%d`)
			seal.sys.UpdatePermType(system.EWayland, wp, acl.Read, acl.Write, acl.Execute)
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

	/*
		PulseAudio server and authentication
	*/

	if seal.et.Has(system.EPulse) {
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
			// not fatal
			fmsg.VPrintln(strings.TrimSpace(err.(*fmsg.BaseError).Message()))
		} else {
			dst := path.Join(seal.share, "pulse-cookie")
			innerDst := fst.Tmp + "/pulse-cookie"
			seal.sys.bwrap.SetEnv[pulseCookie] = innerDst
			seal.sys.CopyFile(dst, src)
			seal.sys.bwrap.Bind(dst, innerDst)
		}
	}

	/*
		D-Bus proxy
	*/

	if seal.et.Has(system.EDBus) {
		// ensure dbus session bus defaults
		if bus[0] == nil {
			bus[0] = dbus.NewConfig(seal.fid, true, true)
		}

		// downstream socket paths
		sessionPath, systemPath := path.Join(seal.share, "bus"), path.Join(seal.share, "system_bus_socket")

		// configure dbus proxy
		if f, err := seal.sys.ProxyDBus(bus[0], bus[1], sessionPath, systemPath); err != nil {
			return err
		} else {
			seal.dbusMsg = f
		}

		// share proxy sockets
		sessionInner := path.Join(seal.sys.runtime, "bus")
		seal.sys.bwrap.SetEnv[dbusSessionBusAddress] = "unix:path=" + sessionInner
		seal.sys.bwrap.Bind(sessionPath, sessionInner)
		seal.sys.UpdatePerm(sessionPath, acl.Read, acl.Write)
		if bus[1] != nil {
			systemInner := "/run/dbus/system_bus_socket"
			seal.sys.bwrap.SetEnv[dbusSystemBusAddress] = "unix:path=" + systemInner
			seal.sys.bwrap.Bind(systemPath, systemInner)
			seal.sys.UpdatePerm(systemPath, acl.Read, acl.Write)
		}
	}

	/*
		Miscellaneous
	*/

	// queue overriding tmpfs at the end of seal.sys.bwrap.Filesystem
	for _, dest := range seal.sys.override {
		seal.sys.bwrap.Tmpfs(dest, 8*1024)
	}

	// append extra perms
	for _, p := range seal.extraPerms {
		if p == nil {
			continue
		}
		if p.ensure {
			seal.sys.Ensure(p.name, 0700)
		}
		seal.sys.UpdatePermType(system.User, p.name, p.perms...)
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
