package app

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"sync/atomic"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/sys"
	"git.gensokyo.uk/security/fortify/system"
	"git.gensokyo.uk/security/fortify/wl"
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

	pulseServer = "PULSE_SERVER"
	pulseCookie = "PULSE_COOKIE"

	dbusSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"
	dbusSystemBusAddress  = "DBUS_SYSTEM_BUS_ADDRESS"
)

var (
	ErrConfig = errors.New("no configuration to seal")
	ErrUser   = errors.New("invalid aid")
	ErrHome   = errors.New("invalid home directory")
	ErrName   = errors.New("invalid username")

	ErrXDisplay = errors.New(display + " unset")

	ErrPulseCookie = errors.New("pulse cookie not present")
	ErrPulseSocket = errors.New("pulse socket not present")
	ErrPulseMode   = errors.New("unexpected pulse socket mode")
)

var posixUsername = regexp.MustCompilePOSIX("^[a-z_]([A-Za-z0-9_-]{0,31}|[A-Za-z0-9_-]{0,30}\\$)$")

// outcome stores copies of various parts of [fst.Config]
type outcome struct {
	// copied from initialising [app]
	id *stringPair[fst.ID]
	// copied from [sys.State] response
	runDirPath string

	// passed through from [fst.Config]
	command []string

	// initial [fst.Config] gob stream for state data;
	// this is prepared ahead of time as config is mutated during seal creation
	ct io.WriterTo
	// dump dbus proxy message buffer
	dbusMsg func()

	user      fsuUser
	sys       *system.I
	container *bwrap.Config
	bwrapSync *os.File

	f atomic.Bool
}

// fsuUser stores post-fsu credentials and metadata
type fsuUser struct {
	// application id
	aid *stringPair[int]
	// target uid resolved by fid:aid
	uid *stringPair[int]

	// supplementary group ids
	supp []string

	// home directory host path
	data string
	// app user home directory
	home string
	// passwd database username
	username string
}

func (seal *outcome) finalise(sys sys.State, config *fst.Config) error {
	{
		// encode initial configuration for state tracking
		ct := new(bytes.Buffer)
		if err := gob.NewEncoder(ct).Encode(config); err != nil {
			return fmsg.WrapErrorSuffix(err,
				"cannot encode initial config:")
		}
		seal.ct = ct
	}

	// pass through command slice; this value is never touched in the main process
	seal.command = config.Command

	// allowed aid range 0 to 9999, this is checked again in fsu
	if config.Confinement.AppID < 0 || config.Confinement.AppID > 9999 {
		return fmsg.WrapError(ErrUser,
			fmt.Sprintf("aid %d out of range", config.Confinement.AppID))
	}

	/*
		Resolve post-fsu user state
	*/

	seal.user = fsuUser{
		aid:      newInt(config.Confinement.AppID),
		data:     config.Confinement.Outer,
		home:     config.Confinement.Inner,
		username: config.Confinement.Username,
	}
	if seal.user.username == "" {
		seal.user.username = "chronos"
	} else if !posixUsername.MatchString(seal.user.username) ||
		len(seal.user.username) >= internal.Sysconf_SC_LOGIN_NAME_MAX() {
		return fmsg.WrapError(ErrName,
			fmt.Sprintf("invalid user name %q", seal.user.username))
	}
	if seal.user.data == "" || !path.IsAbs(seal.user.data) {
		return fmsg.WrapError(ErrHome,
			fmt.Sprintf("invalid home directory %q", seal.user.data))
	}
	if seal.user.home == "" {
		seal.user.home = seal.user.data
	}
	if u, err := sys.Uid(seal.user.aid.unwrap()); err != nil {
		return err
	} else {
		seal.user.uid = newInt(u)
	}
	seal.user.supp = make([]string, len(config.Confinement.Groups))
	for i, name := range config.Confinement.Groups {
		if g, err := sys.LookupGroup(name); err != nil {
			return fmsg.WrapError(err,
				fmt.Sprintf("unknown group %q", name))
		} else {
			seal.user.supp[i] = g.Gid
		}
	}

	/*
		Resolve initial container state
	*/

	// permissive defaults
	if config.Confinement.Sandbox == nil {
		fmsg.Verbose("sandbox configuration not supplied, PROCEED WITH CAUTION")

		conf := &fst.SandboxConfig{
			UserNS:       true,
			Net:          true,
			Syscall:      new(bwrap.SyscallPolicy),
			NoNewSession: true,
			AutoEtc:      true,
		}
		// bind entries in /
		if d, err := sys.ReadDir("/"); err != nil {
			return err
		} else {
			b := make([]*fst.FilesystemConfig, 0, len(d))
			for _, ent := range d {
				p := "/" + ent.Name()
				switch p {
				case "/proc":
				case "/dev":
				case "/tmp":
				case "/mnt":
				case "/etc":

				default:
					b = append(b, &fst.FilesystemConfig{Src: p, Write: true, Must: true})
				}
			}
			conf.Filesystem = append(conf.Filesystem, b...)
		}

		// hide nscd from sandbox if present
		nscd := "/var/run/nscd"
		if _, err := sys.Stat(nscd); !errors.Is(err, fs.ErrNotExist) {
			conf.Override = append(conf.Override, nscd)
		}
		// bind GPU stuff
		if config.Confinement.Enablements.Has(system.EX11) || config.Confinement.Enablements.Has(system.EWayland) {
			conf.Filesystem = append(conf.Filesystem, &fst.FilesystemConfig{Src: "/dev/dri", Device: true})
		}
		// opportunistically bind kvm
		conf.Filesystem = append(conf.Filesystem, &fst.FilesystemConfig{Src: "/dev/kvm", Device: true})

		config.Confinement.Sandbox = conf
	}

	var mapuid *stringPair[int]
	{
		var uid int
		var err error
		seal.container, err = config.Confinement.Sandbox.Bwrap(sys, &uid)
		if err != nil {
			return err
		}
		mapuid = newInt(uid)
		if seal.container.SetEnv == nil {
			seal.container.SetEnv = make(map[string]string)
		}
	}

	/*
		Initialise externals
	*/

	sc := sys.Paths()
	seal.runDirPath = sc.RunDirPath
	seal.sys = system.New(seal.user.uid.unwrap())

	/*
		Work directories
	*/

	// base fortify share path
	seal.sys.Ensure(sc.SharePath, 0711)

	// outer paths used by the main process
	seal.sys.Ensure(sc.RunDirPath, 0700)
	seal.sys.UpdatePermType(system.User, sc.RunDirPath, acl.Execute)
	seal.sys.Ensure(sc.RuntimePath, 0700) // ensure this dir in case XDG_RUNTIME_DIR is unset
	seal.sys.UpdatePermType(system.User, sc.RuntimePath, acl.Execute)

	// outer process-specific share directory
	sharePath := path.Join(sc.SharePath, seal.id.String())
	seal.sys.Ephemeral(system.Process, sharePath, 0711)
	// similar to share but within XDG_RUNTIME_DIR
	sharePathLocal := path.Join(sc.RunDirPath, seal.id.String())
	seal.sys.Ephemeral(system.Process, sharePathLocal, 0700)
	seal.sys.UpdatePerm(sharePathLocal, acl.Execute)

	// inner XDG_RUNTIME_DIR default formatting of `/run/user/%d` as post-fsu user
	innerRuntimeDir := path.Join("/run/user", mapuid.String())
	seal.container.Tmpfs("/run/user", 1*1024*1024)
	seal.container.Tmpfs(innerRuntimeDir, 8*1024*1024)
	seal.container.SetEnv[xdgRuntimeDir] = innerRuntimeDir
	seal.container.SetEnv[xdgSessionClass] = "user"
	seal.container.SetEnv[xdgSessionType] = "tty"

	// outer path for inner /tmp
	{
		tmpdir := path.Join(sc.SharePath, "tmpdir")
		seal.sys.Ensure(tmpdir, 0700)
		seal.sys.UpdatePermType(system.User, tmpdir, acl.Execute)
		tmpdirProc := path.Join(tmpdir, seal.user.aid.String())
		seal.sys.Ensure(tmpdirProc, 01700)
		seal.sys.UpdatePermType(system.User, tmpdirProc, acl.Read, acl.Write, acl.Execute)
		seal.container.Bind(tmpdirProc, "/tmp", false, true)
	}

	/*
		Passwd database
	*/

	// look up shell
	sh := "/bin/sh"
	if s, ok := sys.LookupEnv(shell); ok {
		seal.container.SetEnv[shell] = s
		sh = s
	}

	// bind home directory
	homeDir := "/var/empty"
	if seal.user.home != "" {
		homeDir = seal.user.home
	}
	username := "chronos"
	if seal.user.username != "" {
		username = seal.user.username
	}
	seal.container.Bind(seal.user.data, homeDir, false, true)
	seal.container.Chdir = homeDir
	seal.container.SetEnv["HOME"] = homeDir
	seal.container.SetEnv["USER"] = username

	// generate /etc/passwd and /etc/group
	seal.container.CopyBind("/etc/passwd",
		[]byte(username+":x:"+mapuid.String()+":"+mapuid.String()+":Fortify:"+homeDir+":"+sh+"\n"))
	seal.container.CopyBind("/etc/group",
		[]byte("fortify:x:"+mapuid.String()+":\n"))

	/*
		Display servers
	*/

	// pass $TERM to launcher
	if t, ok := sys.LookupEnv(term); ok {
		seal.container.SetEnv[term] = t
	}

	// set up wayland
	if config.Confinement.Enablements.Has(system.EWayland) {
		// outer wayland socket (usually `/run/user/%d/wayland-%d`)
		var socketPath string
		if name, ok := sys.LookupEnv(wl.WaylandDisplay); !ok {
			fmsg.Verbose(wl.WaylandDisplay + " is not set, assuming " + wl.FallbackName)
			socketPath = path.Join(sc.RuntimePath, wl.FallbackName)
		} else if !path.IsAbs(name) {
			socketPath = path.Join(sc.RuntimePath, name)
		} else {
			socketPath = name
		}

		innerPath := path.Join(innerRuntimeDir, wl.FallbackName)
		seal.container.SetEnv[wl.WaylandDisplay] = wl.FallbackName

		if !config.Confinement.Sandbox.DirectWayland { // set up security-context-v1
			socketDir := path.Join(sc.SharePath, "wayland")
			outerPath := path.Join(socketDir, seal.id.String())
			seal.sys.Ensure(socketDir, 0711)
			appID := config.ID
			if appID == "" {
				// use instance ID in case app id is not set
				appID = "uk.gensokyo.fortify." + seal.id.String()
			}
			seal.sys.Wayland(&seal.bwrapSync, outerPath, socketPath, appID, seal.id.String())
			seal.container.Bind(outerPath, innerPath)
		} else { // bind mount wayland socket (insecure)
			fmsg.Verbose("direct wayland access, PROCEED WITH CAUTION")
			seal.container.Bind(socketPath, innerPath)
			seal.sys.UpdatePermType(system.EWayland, socketPath, acl.Read, acl.Write, acl.Execute)
		}
	}

	// set up X11
	if config.Confinement.Enablements.Has(system.EX11) {
		// discover X11 and grant user permission via the `ChangeHosts` command
		if d, ok := sys.LookupEnv(display); !ok {
			return fmsg.WrapError(ErrXDisplay,
				"DISPLAY is not set")
		} else {
			seal.sys.ChangeHosts("#" + seal.user.uid.String())
			seal.container.SetEnv[display] = d
			seal.container.Bind("/tmp/.X11-unix", "/tmp/.X11-unix")
		}
	}

	/*
		PulseAudio server and authentication
	*/

	if config.Confinement.Enablements.Has(system.EPulse) {
		// PulseAudio runtime directory (usually `/run/user/%d/pulse`)
		pulseRuntimeDir := path.Join(sc.RuntimePath, "pulse")
		// PulseAudio socket (usually `/run/user/%d/pulse/native`)
		pulseSocket := path.Join(pulseRuntimeDir, "native")

		if _, err := sys.Stat(pulseRuntimeDir); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmsg.WrapErrorSuffix(err,
					fmt.Sprintf("cannot access PulseAudio directory %q:", pulseRuntimeDir))
			}
			return fmsg.WrapError(ErrPulseSocket,
				fmt.Sprintf("PulseAudio directory %q not found", pulseRuntimeDir))
		}

		if s, err := sys.Stat(pulseSocket); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmsg.WrapErrorSuffix(err,
					fmt.Sprintf("cannot access PulseAudio socket %q:", pulseSocket))
			}
			return fmsg.WrapError(ErrPulseSocket,
				fmt.Sprintf("PulseAudio directory %q found but socket does not exist", pulseRuntimeDir))
		} else {
			if m := s.Mode(); m&0o006 != 0o006 {
				return fmsg.WrapError(ErrPulseMode,
					fmt.Sprintf("unexpected permissions on %q:", pulseSocket), m)
			}
		}

		// hard link pulse socket into target-executable share
		innerPulseRuntimeDir := path.Join(sharePathLocal, "pulse")
		innerPulseSocket := path.Join(innerRuntimeDir, "pulse", "native")
		seal.sys.Link(pulseSocket, innerPulseRuntimeDir)
		seal.container.Bind(innerPulseRuntimeDir, innerPulseSocket)
		seal.container.SetEnv[pulseServer] = "unix:" + innerPulseSocket

		// publish current user's pulse cookie for target user
		if src, err := discoverPulseCookie(sys); err != nil {
			// not fatal
			fmsg.Verbose(strings.TrimSpace(err.(*fmsg.BaseError).Message()))
		} else {
			innerDst := fst.Tmp + "/pulse-cookie"
			seal.container.SetEnv[pulseCookie] = innerDst
			payload := new([]byte)
			seal.container.CopyBindRef(innerDst, &payload)
			seal.sys.CopyFile(payload, src, 256, 256)
		}
	}

	/*
		D-Bus proxy
	*/

	if config.Confinement.Enablements.Has(system.EDBus) {
		// ensure dbus session bus defaults
		if config.Confinement.SessionBus == nil {
			config.Confinement.SessionBus = dbus.NewConfig(config.ID, true, true)
		}

		// downstream socket paths
		sessionPath, systemPath := path.Join(sharePath, "bus"), path.Join(sharePath, "system_bus_socket")

		// configure dbus proxy
		if f, err := seal.sys.ProxyDBus(
			config.Confinement.SessionBus, config.Confinement.SystemBus,
			sessionPath, systemPath,
		); err != nil {
			return err
		} else {
			seal.dbusMsg = f
		}

		// share proxy sockets
		sessionInner := path.Join(innerRuntimeDir, "bus")
		seal.container.SetEnv[dbusSessionBusAddress] = "unix:path=" + sessionInner
		seal.container.Bind(sessionPath, sessionInner)
		seal.sys.UpdatePerm(sessionPath, acl.Read, acl.Write)
		if config.Confinement.SystemBus != nil {
			systemInner := "/run/dbus/system_bus_socket"
			seal.container.SetEnv[dbusSystemBusAddress] = "unix:path=" + systemInner
			seal.container.Bind(systemPath, systemInner)
			seal.sys.UpdatePerm(systemPath, acl.Read, acl.Write)
		}
	}

	/*
		Miscellaneous
	*/

	// queue overriding tmpfs at the end of seal.container.Filesystem
	for _, dest := range config.Confinement.Sandbox.Override {
		seal.container.Tmpfs(dest, 8*1024)
	}

	// append ExtraPerms last
	for _, p := range config.Confinement.ExtraPerms {
		if p == nil {
			continue
		}

		if p.Ensure {
			seal.sys.Ensure(p.Path, 0700)
		}

		perms := make(acl.Perms, 0, 3)
		if p.Read {
			perms = append(perms, acl.Read)
		}
		if p.Write {
			perms = append(perms, acl.Write)
		}
		if p.Execute {
			perms = append(perms, acl.Execute)
		}
		seal.sys.UpdatePermType(system.User, p.Path, perms...)
	}

	// mount fortify in sandbox for init
	seal.container.Bind(sys.MustExecutable(), path.Join(fst.Tmp, "sbin/fortify"))
	seal.container.Symlink("fortify", path.Join(fst.Tmp, "sbin/init0"))

	fmsg.Verbosef("created application seal for uid %s (%s) groups: %v, command: %s",
		seal.user.uid, seal.user.username, config.Confinement.Groups, config.Command)

	return nil
}

// discoverPulseCookie attempts various standard methods to discover the current user's PulseAudio authentication cookie
func discoverPulseCookie(sys sys.State) (string, error) {
	if p, ok := sys.LookupEnv(pulseCookie); ok {
		return p, nil
	}

	// dotfile $HOME/.pulse-cookie
	if p, ok := sys.LookupEnv(home); ok {
		p = path.Join(p, ".pulse-cookie")
		if s, err := sys.Stat(p); err != nil {
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
	if p, ok := sys.LookupEnv(xdgConfigHome); ok {
		p = path.Join(p, "pulse", "cookie")
		if s, err := sys.Stat(p); err != nil {
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
