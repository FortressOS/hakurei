package app

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/sys"
	"git.gensokyo.uk/security/fortify/sandbox"
	"git.gensokyo.uk/security/fortify/sandbox/wl"
	"git.gensokyo.uk/security/fortify/system"
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

	// initial [fst.Config] gob stream for state data;
	// this is prepared ahead of time as config is clobbered during seal creation
	ct io.WriterTo
	// dump dbus proxy message buffer
	dbusMsg func()

	user fsuUser
	sys  *system.I
	ctx  context.Context

	container *sandbox.Params
	env       map[string]string
	sync      *os.File

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

func (seal *outcome) finalise(ctx context.Context, sys sys.State, config *fst.Config) error {
	if seal.ctx != nil {
		panic("finalise called twice")
	}
	seal.ctx = ctx

	shellPath := "/bin/sh"
	if s, ok := sys.LookupEnv(shell); ok && path.IsAbs(s) {
		shellPath = s
	}

	{
		// encode initial configuration for state tracking
		ct := new(bytes.Buffer)
		if err := gob.NewEncoder(ct).Encode(config); err != nil {
			return fmsg.WrapErrorSuffix(err,
				"cannot encode initial config:")
		}
		seal.ct = ct
	}

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

		// fsu clears the environment so resolve paths early
		if !path.IsAbs(config.Path) {
			if len(config.Args) > 0 {
				if p, err := sys.LookPath(config.Args[0]); err != nil {
					return fmsg.WrapError(err, err.Error())
				} else {
					config.Path = p
				}
			} else {
				config.Path = shellPath
			}
		}

		conf := &fst.SandboxConfig{
			Userns:  true,
			Net:     true,
			Tty:     true,
			AutoEtc: true,
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
			conf.Cover = append(conf.Cover, nscd)
		}
		// bind GPU stuff
		if config.Confinement.Enablements&(system.EX11|system.EWayland) != 0 {
			conf.Filesystem = append(conf.Filesystem, &fst.FilesystemConfig{Src: "/dev/dri", Device: true})
		}
		// opportunistically bind kvm
		conf.Filesystem = append(conf.Filesystem, &fst.FilesystemConfig{Src: "/dev/kvm", Device: true})

		config.Confinement.Sandbox = conf
	}

	var mapuid, mapgid *stringPair[int]
	{
		var uid, gid int
		var err error
		seal.container, seal.env, err = config.Confinement.Sandbox.ToContainer(sys, &uid, &gid)
		if err != nil {
			return fmsg.WrapErrorSuffix(err,
				"cannot initialise container configuration:")
		}
		if !path.IsAbs(config.Path) {
			return fmsg.WrapError(syscall.EINVAL,
				"invalid program path")
		}
		if len(config.Args) == 0 {
			config.Args = []string{config.Path}
		}
		seal.container.Path = config.Path
		seal.container.Args = config.Args

		mapuid = newInt(uid)
		mapgid = newInt(gid)
		if seal.env == nil {
			seal.env = make(map[string]string)
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
	seal.container.Tmpfs("/run/user", 1<<12, 0755)
	seal.container.Tmpfs(innerRuntimeDir, 1<<23, 0700)
	seal.env[xdgRuntimeDir] = innerRuntimeDir
	seal.env[xdgSessionClass] = "user"
	seal.env[xdgSessionType] = "tty"

	// outer path for inner /tmp
	{
		tmpdir := path.Join(sc.SharePath, "tmpdir")
		seal.sys.Ensure(tmpdir, 0700)
		seal.sys.UpdatePermType(system.User, tmpdir, acl.Execute)
		tmpdirInst := path.Join(tmpdir, seal.user.aid.String())
		seal.sys.Ensure(tmpdirInst, 01700)
		seal.sys.UpdatePermType(system.User, tmpdirInst, acl.Read, acl.Write, acl.Execute)
		seal.container.Bind(tmpdirInst, "/tmp", sandbox.BindWritable)
	}

	/*
		Passwd database
	*/

	homeDir := "/var/empty"
	if seal.user.home != "" {
		homeDir = seal.user.home
	}
	username := "chronos"
	if seal.user.username != "" {
		username = seal.user.username
	}
	seal.container.Bind(seal.user.data, homeDir, sandbox.BindWritable)
	seal.container.Dir = homeDir
	seal.env["HOME"] = homeDir
	seal.env["USER"] = username

	seal.container.Place("/etc/passwd",
		[]byte(username+":x:"+mapuid.String()+":"+mapgid.String()+":Fortify:"+homeDir+":"+shellPath+"\n"))
	seal.container.Place("/etc/group",
		[]byte("fortify:x:"+mapgid.String()+":\n"))

	/*
		Display servers
	*/

	// pass $TERM for proper terminal I/O in shell
	if t, ok := sys.LookupEnv(term); ok {
		seal.env[term] = t
	}

	if config.Confinement.Enablements&system.EWayland != 0 {
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
		seal.env[wl.WaylandDisplay] = wl.FallbackName

		if !config.Confinement.Sandbox.DirectWayland { // set up security-context-v1
			socketDir := path.Join(sc.SharePath, "wayland")
			outerPath := path.Join(socketDir, seal.id.String())
			seal.sys.Ensure(socketDir, 0711)
			appID := config.ID
			if appID == "" {
				// use instance ID in case app id is not set
				appID = "uk.gensokyo.fortify." + seal.id.String()
			}
			seal.sys.Wayland(&seal.sync, outerPath, socketPath, appID, seal.id.String())
			seal.container.Bind(outerPath, innerPath, 0)
		} else { // bind mount wayland socket (insecure)
			fmsg.Verbose("direct wayland access, PROCEED WITH CAUTION")
			seal.container.Bind(socketPath, innerPath, 0)
			seal.sys.UpdatePermType(system.EWayland, socketPath, acl.Read, acl.Write, acl.Execute)
		}
	}

	if config.Confinement.Enablements&system.EX11 != 0 {
		if d, ok := sys.LookupEnv(display); !ok {
			return fmsg.WrapError(ErrXDisplay,
				"DISPLAY is not set")
		} else {
			seal.sys.ChangeHosts("#" + seal.user.uid.String())
			seal.env[display] = d
			seal.container.Bind("/tmp/.X11-unix", "/tmp/.X11-unix", 0)
		}
	}

	/*
		PulseAudio server and authentication
	*/

	if config.Confinement.Enablements&system.EPulse != 0 {
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
		seal.container.Bind(innerPulseRuntimeDir, innerPulseSocket, 0)
		seal.env[pulseServer] = "unix:" + innerPulseSocket

		// publish current user's pulse cookie for target user
		if src, err := discoverPulseCookie(sys); err != nil {
			// not fatal
			fmsg.Verbose(strings.TrimSpace(err.(*fmsg.BaseError).Message()))
		} else {
			innerDst := fst.Tmp + "/pulse-cookie"
			seal.env[pulseCookie] = innerDst
			var payload *[]byte
			seal.container.PlaceP(innerDst, &payload)
			seal.sys.CopyFile(payload, src, 256, 256)
		}
	}

	/*
		D-Bus proxy
	*/

	if config.Confinement.Enablements&system.EDBus != 0 {
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
		seal.env[dbusSessionBusAddress] = "unix:path=" + sessionInner
		seal.container.Bind(sessionPath, sessionInner, 0)
		seal.sys.UpdatePerm(sessionPath, acl.Read, acl.Write)
		if config.Confinement.SystemBus != nil {
			systemInner := "/run/dbus/system_bus_socket"
			seal.env[dbusSystemBusAddress] = "unix:path=" + systemInner
			seal.container.Bind(systemPath, systemInner, 0)
			seal.sys.UpdatePerm(systemPath, acl.Read, acl.Write)
		}
	}

	/*
		Miscellaneous
	*/

	for _, dest := range config.Confinement.Sandbox.Cover {
		seal.container.Tmpfs(dest, 1<<13, 0755)
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

	// flatten and sort env for deterministic behaviour
	seal.container.Env = make([]string, 0, len(seal.env))
	maps.All(seal.env)(func(k string, v string) bool { seal.container.Env = append(seal.container.Env, k+"="+v); return true })
	slices.Sort(seal.container.Env)

	fmsg.Verbosef("created application seal for uid %s (%s) groups: %v, argv: %s",
		seal.user.uid, seal.user.username, config.Confinement.Groups, seal.container.Args)

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
