package setuid

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"

	"git.gensokyo.uk/security/hakurei/acl"
	"git.gensokyo.uk/security/hakurei/dbus"
	"git.gensokyo.uk/security/hakurei/hst"
	"git.gensokyo.uk/security/hakurei/internal"
	. "git.gensokyo.uk/security/hakurei/internal/app"
	"git.gensokyo.uk/security/hakurei/internal/app/instance/common"
	"git.gensokyo.uk/security/hakurei/internal/hlog"
	"git.gensokyo.uk/security/hakurei/internal/sys"
	"git.gensokyo.uk/security/hakurei/sandbox"
	"git.gensokyo.uk/security/hakurei/sandbox/wl"
	"git.gensokyo.uk/security/hakurei/system"
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

// outcome stores copies of various parts of [hst.Config]
type outcome struct {
	// copied from initialising [app]
	id *stringPair[ID]
	// copied from [sys.State] response
	runDirPath string

	// initial [hst.Config] gob stream for state data;
	// this is prepared ahead of time as config is clobbered during seal creation
	ct io.WriterTo
	// dump dbus proxy message buffer
	dbusMsg func()

	user hsuUser
	sys  *system.I
	ctx  context.Context

	container *sandbox.Params
	env       map[string]string
	sync      *os.File

	f atomic.Bool
}

// shareHost holds optional share directory state that must not be accessed directly
type shareHost struct {
	// whether XDG_RUNTIME_DIR is used post hsu
	useRuntimeDir bool
	// process-specific directory in tmpdir, empty if unused
	sharePath string
	// process-specific directory in XDG_RUNTIME_DIR, empty if unused
	runtimeSharePath string

	seal *outcome
	sc   Paths
}

// ensureRuntimeDir must be called if direct access to paths within XDG_RUNTIME_DIR is required
func (share *shareHost) ensureRuntimeDir() {
	if share.useRuntimeDir {
		return
	}
	share.useRuntimeDir = true
	share.seal.sys.Ensure(share.sc.RunDirPath, 0700)
	share.seal.sys.UpdatePermType(system.User, share.sc.RunDirPath, acl.Execute)
	share.seal.sys.Ensure(share.sc.RuntimePath, 0700) // ensure this dir in case XDG_RUNTIME_DIR is unset
	share.seal.sys.UpdatePermType(system.User, share.sc.RuntimePath, acl.Execute)
}

// instance returns a process-specific share path within tmpdir
func (share *shareHost) instance() string {
	if share.sharePath != "" {
		return share.sharePath
	}
	share.sharePath = path.Join(share.sc.SharePath, share.seal.id.String())
	share.seal.sys.Ephemeral(system.Process, share.sharePath, 0711)
	return share.sharePath
}

// runtime returns a process-specific share path within XDG_RUNTIME_DIR
func (share *shareHost) runtime() string {
	if share.runtimeSharePath != "" {
		return share.runtimeSharePath
	}
	share.ensureRuntimeDir()
	share.runtimeSharePath = path.Join(share.sc.RunDirPath, share.seal.id.String())
	share.seal.sys.Ephemeral(system.Process, share.runtimeSharePath, 0700)
	share.seal.sys.UpdatePerm(share.runtimeSharePath, acl.Execute)
	return share.runtimeSharePath
}

// hsuUser stores post-hsu credentials and metadata
type hsuUser struct {
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

func (seal *outcome) finalise(ctx context.Context, sys sys.State, config *hst.Config) error {
	if seal.ctx != nil {
		panic("finalise called twice")
	}
	seal.ctx = ctx

	{
		// encode initial configuration for state tracking
		ct := new(bytes.Buffer)
		if err := gob.NewEncoder(ct).Encode(config); err != nil {
			return hlog.WrapErrSuffix(err,
				"cannot encode initial config:")
		}
		seal.ct = ct
	}

	// allowed aid range 0 to 9999, this is checked again in hsu
	if config.Identity < 0 || config.Identity > 9999 {
		return hlog.WrapErr(ErrUser,
			fmt.Sprintf("identity %d out of range", config.Identity))
	}

	seal.user = hsuUser{
		aid:      newInt(config.Identity),
		data:     config.Data,
		home:     config.Dir,
		username: config.Username,
	}
	if seal.user.username == "" {
		seal.user.username = "chronos"
	} else if !posixUsername.MatchString(seal.user.username) ||
		len(seal.user.username) >= internal.Sysconf_SC_LOGIN_NAME_MAX() {
		return hlog.WrapErr(ErrName,
			fmt.Sprintf("invalid user name %q", seal.user.username))
	}
	if seal.user.data == "" || !path.IsAbs(seal.user.data) {
		return hlog.WrapErr(ErrHome,
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
	seal.user.supp = make([]string, len(config.Groups))
	for i, name := range config.Groups {
		if g, err := sys.LookupGroup(name); err != nil {
			return hlog.WrapErr(err,
				fmt.Sprintf("unknown group %q", name))
		} else {
			seal.user.supp[i] = g.Gid
		}
	}

	// this also falls back to host path if encountering an invalid path
	if !path.IsAbs(config.Shell) {
		config.Shell = "/bin/sh"
		if s, ok := sys.LookupEnv(shell); ok && path.IsAbs(s) {
			config.Shell = s
		}
	}
	// do not use the value of shell before this point

	// permissive defaults
	if config.Container == nil {
		hlog.Verbose("container configuration not supplied, PROCEED WITH CAUTION")

		// hsu clears the environment so resolve paths early
		if !path.IsAbs(config.Path) {
			if len(config.Args) > 0 {
				if p, err := sys.LookPath(config.Args[0]); err != nil {
					return hlog.WrapErr(err, err.Error())
				} else {
					config.Path = p
				}
			} else {
				config.Path = config.Shell
			}
		}

		conf := &hst.ContainerConfig{
			Userns:  true,
			Net:     true,
			Tty:     true,
			AutoEtc: true,
		}
		// bind entries in /
		if d, err := sys.ReadDir("/"); err != nil {
			return err
		} else {
			b := make([]*hst.FilesystemConfig, 0, len(d))
			for _, ent := range d {
				p := "/" + ent.Name()
				switch p {
				case "/proc":
				case "/dev":
				case "/tmp":
				case "/mnt":
				case "/etc":

				default:
					b = append(b, &hst.FilesystemConfig{Src: p, Write: true, Must: true})
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
		if config.Enablements&(system.EX11|system.EWayland) != 0 {
			conf.Filesystem = append(conf.Filesystem, &hst.FilesystemConfig{Src: "/dev/dri", Device: true})
		}
		// opportunistically bind kvm
		conf.Filesystem = append(conf.Filesystem, &hst.FilesystemConfig{Src: "/dev/kvm", Device: true})

		config.Container = conf
	}

	var mapuid, mapgid *stringPair[int]
	{
		var uid, gid int
		var err error
		seal.container, seal.env, err = common.NewContainer(config.Container, sys, &uid, &gid)
		if err != nil {
			return hlog.WrapErrSuffix(err,
				"cannot initialise container configuration:")
		}
		if !path.IsAbs(config.Path) {
			return hlog.WrapErr(syscall.EINVAL,
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
			seal.env = make(map[string]string, 1<<6)
		}
	}

	if !config.Container.AutoEtc {
		if config.Container.Etc != "" {
			seal.container.Bind(config.Container.Etc, "/etc", 0)
		}
	} else {
		etcPath := config.Container.Etc
		if etcPath == "" {
			etcPath = "/etc"
		}
		seal.container.Etc(etcPath, seal.id.String())
	}

	// inner XDG_RUNTIME_DIR default formatting of `/run/user/%d` as mapped uid
	innerRuntimeDir := path.Join("/run/user", mapuid.String())
	seal.env[xdgRuntimeDir] = innerRuntimeDir
	seal.env[xdgSessionClass] = "user"
	seal.env[xdgSessionType] = "tty"

	share := &shareHost{seal: seal, sc: sys.Paths()}
	seal.runDirPath = share.sc.RunDirPath
	seal.sys = system.New(seal.user.uid.unwrap())
	seal.sys.Ensure(share.sc.SharePath, 0711)

	{
		runtimeDir := path.Join(share.sc.SharePath, "runtime")
		seal.sys.Ensure(runtimeDir, 0700)
		seal.sys.UpdatePermType(system.User, runtimeDir, acl.Execute)
		runtimeDirInst := path.Join(runtimeDir, seal.user.aid.String())
		seal.sys.Ensure(runtimeDirInst, 0700)
		seal.sys.UpdatePermType(system.User, runtimeDirInst, acl.Read, acl.Write, acl.Execute)
		seal.container.Tmpfs("/run/user", 1<<12, 0755)
		seal.container.Bind(runtimeDirInst, innerRuntimeDir, sandbox.BindWritable)
	}

	{
		tmpdir := path.Join(share.sc.SharePath, "tmpdir")
		seal.sys.Ensure(tmpdir, 0700)
		seal.sys.UpdatePermType(system.User, tmpdir, acl.Execute)
		tmpdirInst := path.Join(tmpdir, seal.user.aid.String())
		seal.sys.Ensure(tmpdirInst, 01700)
		seal.sys.UpdatePermType(system.User, tmpdirInst, acl.Read, acl.Write, acl.Execute)
		// mount inner /tmp from share so it shares persistence and storage behaviour of host /tmp
		seal.container.Bind(tmpdirInst, "/tmp", sandbox.BindWritable)
	}

	{
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
		seal.env[shell] = config.Shell

		seal.container.Place("/etc/passwd",
			[]byte(username+":x:"+mapuid.String()+":"+mapgid.String()+":Hakurei:"+homeDir+":"+config.Shell+"\n"))
		seal.container.Place("/etc/group",
			[]byte("hakurei:x:"+mapgid.String()+":\n"))
	}

	// pass TERM for proper terminal I/O in initial process
	if t, ok := sys.LookupEnv(term); ok {
		seal.env[term] = t
	}

	if config.Enablements&system.EWayland != 0 {
		// outer wayland socket (usually `/run/user/%d/wayland-%d`)
		var socketPath string
		if name, ok := sys.LookupEnv(wl.WaylandDisplay); !ok {
			hlog.Verbose(wl.WaylandDisplay + " is not set, assuming " + wl.FallbackName)
			socketPath = path.Join(share.sc.RuntimePath, wl.FallbackName)
		} else if !path.IsAbs(name) {
			socketPath = path.Join(share.sc.RuntimePath, name)
		} else {
			socketPath = name
		}

		innerPath := path.Join(innerRuntimeDir, wl.FallbackName)
		seal.env[wl.WaylandDisplay] = wl.FallbackName

		if !config.DirectWayland { // set up security-context-v1
			appID := config.ID
			if appID == "" {
				// use instance ID in case app id is not set
				appID = "app.hakurei." + seal.id.String()
			}
			// downstream socket paths
			outerPath := path.Join(share.instance(), "wayland")
			seal.sys.Wayland(&seal.sync, outerPath, socketPath, appID, seal.id.String())
			seal.container.Bind(outerPath, innerPath, 0)
		} else { // bind mount wayland socket (insecure)
			hlog.Verbose("direct wayland access, PROCEED WITH CAUTION")
			share.ensureRuntimeDir()
			seal.container.Bind(socketPath, innerPath, 0)
			seal.sys.UpdatePermType(system.EWayland, socketPath, acl.Read, acl.Write, acl.Execute)
		}
	}

	if config.Enablements&system.EX11 != 0 {
		if d, ok := sys.LookupEnv(display); !ok {
			return hlog.WrapErr(ErrXDisplay,
				"DISPLAY is not set")
		} else {
			seal.sys.ChangeHosts("#" + seal.user.uid.String())
			seal.env[display] = d
			seal.container.Bind("/tmp/.X11-unix", "/tmp/.X11-unix", 0)
		}
	}

	if config.Enablements&system.EPulse != 0 {
		// PulseAudio runtime directory (usually `/run/user/%d/pulse`)
		pulseRuntimeDir := path.Join(share.sc.RuntimePath, "pulse")
		// PulseAudio socket (usually `/run/user/%d/pulse/native`)
		pulseSocket := path.Join(pulseRuntimeDir, "native")

		if _, err := sys.Stat(pulseRuntimeDir); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return hlog.WrapErrSuffix(err,
					fmt.Sprintf("cannot access PulseAudio directory %q:", pulseRuntimeDir))
			}
			return hlog.WrapErr(ErrPulseSocket,
				fmt.Sprintf("PulseAudio directory %q not found", pulseRuntimeDir))
		}

		if s, err := sys.Stat(pulseSocket); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return hlog.WrapErrSuffix(err,
					fmt.Sprintf("cannot access PulseAudio socket %q:", pulseSocket))
			}
			return hlog.WrapErr(ErrPulseSocket,
				fmt.Sprintf("PulseAudio directory %q found but socket does not exist", pulseRuntimeDir))
		} else {
			if m := s.Mode(); m&0o006 != 0o006 {
				return hlog.WrapErr(ErrPulseMode,
					fmt.Sprintf("unexpected permissions on %q:", pulseSocket), m)
			}
		}

		// hard link pulse socket into target-executable share
		innerPulseRuntimeDir := path.Join(share.runtime(), "pulse")
		innerPulseSocket := path.Join(innerRuntimeDir, "pulse", "native")
		seal.sys.Link(pulseSocket, innerPulseRuntimeDir)
		seal.container.Bind(innerPulseRuntimeDir, innerPulseSocket, 0)
		seal.env[pulseServer] = "unix:" + innerPulseSocket

		// publish current user's pulse cookie for target user
		if src, err := discoverPulseCookie(sys); err != nil {
			// not fatal
			hlog.Verbose(strings.TrimSpace(err.(*hlog.BaseError).Message()))
		} else {
			innerDst := hst.Tmp + "/pulse-cookie"
			seal.env[pulseCookie] = innerDst
			var payload *[]byte
			seal.container.PlaceP(innerDst, &payload)
			seal.sys.CopyFile(payload, src, 256, 256)
		}
	}

	if config.Enablements&system.EDBus != 0 {
		// ensure dbus session bus defaults
		if config.SessionBus == nil {
			config.SessionBus = dbus.NewConfig(config.ID, true, true)
		}

		// downstream socket paths
		sharePath := share.instance()
		sessionPath, systemPath := path.Join(sharePath, "bus"), path.Join(sharePath, "system_bus_socket")

		// configure dbus proxy
		if f, err := seal.sys.ProxyDBus(
			config.SessionBus, config.SystemBus,
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
		if config.SystemBus != nil {
			systemInner := "/run/dbus/system_bus_socket"
			seal.env[dbusSystemBusAddress] = "unix:path=" + systemInner
			seal.container.Bind(systemPath, systemInner, 0)
			seal.sys.UpdatePerm(systemPath, acl.Read, acl.Write)
		}
	}

	for _, dest := range config.Container.Cover {
		seal.container.Tmpfs(dest, 1<<13, 0755)
	}

	// append ExtraPerms last
	for _, p := range config.ExtraPerms {
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
	for k, v := range seal.env {
		if strings.IndexByte(k, '=') != -1 {
			return hlog.WrapErr(syscall.EINVAL,
				fmt.Sprintf("invalid environment variable %s", k))
		}
		seal.container.Env = append(seal.container.Env, k+"="+v)
	}
	slices.Sort(seal.container.Env)

	if hlog.Load() {
		hlog.Verbosef("created application seal for uid %s (%s) groups: %v, argv: %s, ops: %d",
			seal.user.uid, seal.user.username, config.Groups, seal.container.Args, len(*seal.container.Ops))
	}

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
				return p, hlog.WrapErrSuffix(err,
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
				return p, hlog.WrapErrSuffix(err,
					fmt.Sprintf("cannot access PulseAudio cookie %q:", p))
			}
			// not found, try next method
		} else if !s.IsDir() {
			return p, nil
		}
	}

	return "", hlog.WrapErr(ErrPulseCookie,
		fmt.Sprintf("cannot locate PulseAudio cookie (tried $%s, $%s/pulse/cookie, $%s/.pulse-cookie)",
			pulseCookie, xdgConfigHome, home))
}
