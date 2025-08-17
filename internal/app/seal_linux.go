package app

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
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/hlog"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
	"hakurei.app/system/wayland"
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
	ErrIdent = errors.New("invalid identity")
	ErrName  = errors.New("invalid username")

	ErrXDisplay = errors.New(display + " unset")

	ErrPulseCookie = errors.New("pulse cookie not present")
	ErrPulseSocket = errors.New("pulse socket not present")
	ErrPulseMode   = errors.New("unexpected pulse socket mode")
)

var posixUsername = regexp.MustCompilePOSIX("^[a-z_]([A-Za-z0-9_-]{0,31}|[A-Za-z0-9_-]{0,30}\\$)$")

// outcome stores copies of various parts of [hst.Config]
type outcome struct {
	// copied from initialising [app]
	id *stringPair[state.ID]
	// copied from [sys.State]
	runDirPath *container.Absolute

	// initial [hst.Config] gob stream for state data;
	// this is prepared ahead of time as config is clobbered during seal creation
	ct io.WriterTo
	// dump dbus proxy message buffer
	dbusMsg func()

	user hsuUser
	sys  *system.I
	ctx  context.Context

	waitDelay time.Duration
	container *container.Params
	env       map[string]string
	sync      *os.File

	f atomic.Bool
}

// shareHost holds optional share directory state that must not be accessed directly
type shareHost struct {
	// whether XDG_RUNTIME_DIR is used post hsu
	useRuntimeDir bool
	// process-specific directory in tmpdir, empty if unused
	sharePath *container.Absolute
	// process-specific directory in XDG_RUNTIME_DIR, empty if unused
	runtimeSharePath *container.Absolute

	seal *outcome
	sc   hst.Paths
}

// ensureRuntimeDir must be called if direct access to paths within XDG_RUNTIME_DIR is required
func (share *shareHost) ensureRuntimeDir() {
	if share.useRuntimeDir {
		return
	}
	share.useRuntimeDir = true
	share.seal.sys.Ensure(share.sc.RunDirPath.String(), 0700)
	share.seal.sys.UpdatePermType(system.User, share.sc.RunDirPath.String(), acl.Execute)
	share.seal.sys.Ensure(share.sc.RuntimePath.String(), 0700) // ensure this dir in case XDG_RUNTIME_DIR is unset
	share.seal.sys.UpdatePermType(system.User, share.sc.RuntimePath.String(), acl.Execute)
}

// instance returns a process-specific share path within tmpdir
func (share *shareHost) instance() *container.Absolute {
	if share.sharePath != nil {
		return share.sharePath
	}
	share.sharePath = share.sc.SharePath.Append(share.seal.id.String())
	share.seal.sys.Ephemeral(system.Process, share.sharePath.String(), 0711)
	return share.sharePath
}

// runtime returns a process-specific share path within XDG_RUNTIME_DIR
func (share *shareHost) runtime() *container.Absolute {
	if share.runtimeSharePath != nil {
		return share.runtimeSharePath
	}
	share.ensureRuntimeDir()
	share.runtimeSharePath = share.sc.RunDirPath.Append(share.seal.id.String())
	share.seal.sys.Ephemeral(system.Process, share.runtimeSharePath.String(), 0700)
	share.seal.sys.UpdatePerm(share.runtimeSharePath.String(), acl.Execute)
	return share.runtimeSharePath
}

// hsuUser stores post-hsu credentials and metadata
type hsuUser struct {
	// identity
	aid *stringPair[int]
	// target uid resolved by hid:aid
	uid *stringPair[int]

	// supplementary group ids
	supp []string

	// home directory host path
	data *container.Absolute
	// app user home directory
	home *container.Absolute
	// passwd database username
	username string
}

func (seal *outcome) finalise(ctx context.Context, sys sys.State, config *hst.Config) error {
	if seal.ctx != nil {
		panic("finalise called twice")
	}
	seal.ctx = ctx

	if config == nil {
		return hlog.WrapErr(syscall.EINVAL, syscall.EINVAL.Error())
	}
	if config.Data == nil {
		return hlog.WrapErr(os.ErrInvalid, "invalid data directory")
	}

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
		return hlog.WrapErr(ErrIdent,
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
		len(seal.user.username) >= internal.Sysconf(internal.SC_LOGIN_NAME_MAX) {
		return hlog.WrapErr(ErrName,
			fmt.Sprintf("invalid user name %q", seal.user.username))
	}
	if seal.user.home == nil {
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

	// permissive defaults
	if config.Container == nil {
		hlog.Verbose("container configuration not supplied, PROCEED WITH CAUTION")

		if config.Shell == nil {
			config.Shell = container.AbsFHSRoot.Append("bin", "sh")
			s, _ := sys.LookupEnv(shell)
			if a, err := container.NewAbs(s); err == nil {
				config.Shell = a
			}
		}

		// hsu clears the environment so resolve paths early
		if config.Path == nil {
			if len(config.Args) > 0 {
				if p, err := sys.LookPath(config.Args[0]); err != nil {
					return hlog.WrapErr(err, err.Error())
				} else if config.Path, err = container.NewAbs(p); err != nil {
					return hlog.WrapErr(err, err.Error())
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

			AutoRoot:  container.AbsFHSRoot,
			RootFlags: container.BindWritable,
		}

		// bind GPU stuff
		if config.Enablements.Unwrap()&(system.EX11|system.EWayland) != 0 {
			conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("dri"), Device: true, Optional: true}})
		}
		// opportunistically bind kvm
		conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("kvm"), Device: true, Optional: true}})

		// hide nscd from container if present
		nscd := container.AbsFHSVar.Append("run/nscd")
		if _, err := sys.Stat(nscd.String()); !errors.Is(err, fs.ErrNotExist) {
			conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSEphemeral{Target: nscd}})
		}

		config.Container = conf
	}

	// late nil checks for pd behaviour
	if config.Shell == nil {
		return hlog.WrapErr(syscall.EINVAL, "invalid shell path")
	}
	if config.Path == nil {
		return hlog.WrapErr(syscall.EINVAL, "invalid program path")
	}

	var mapuid, mapgid *stringPair[int]
	{
		var uid, gid int
		var err error
		seal.container, seal.env, err = newContainer(config.Container, sys, seal.id.String(), &uid, &gid)
		seal.waitDelay = config.Container.WaitDelay
		if err != nil {
			return hlog.WrapErrSuffix(err,
				"cannot initialise container configuration:")
		}
		if len(config.Args) == 0 {
			config.Args = []string{config.Path.String()}
		}
		seal.container.Path = config.Path
		seal.container.Args = config.Args

		mapuid = newInt(uid)
		mapgid = newInt(gid)
		if seal.env == nil {
			seal.env = make(map[string]string, 1<<6)
		}
	}

	// inner XDG_RUNTIME_DIR default formatting of `/run/user/%d` as mapped uid
	innerRuntimeDir := container.AbsFHSRunUser.Append(mapuid.String())
	seal.env[xdgRuntimeDir] = innerRuntimeDir.String()
	seal.env[xdgSessionClass] = "user"
	seal.env[xdgSessionType] = "tty"

	share := &shareHost{seal: seal, sc: sys.Paths()}
	seal.runDirPath = share.sc.RunDirPath
	seal.sys = system.New(seal.user.uid.unwrap())
	seal.sys.Ensure(share.sc.SharePath.String(), 0711)

	{
		runtimeDir := share.sc.SharePath.Append("runtime")
		seal.sys.Ensure(runtimeDir.String(), 0700)
		seal.sys.UpdatePermType(system.User, runtimeDir.String(), acl.Execute)
		runtimeDirInst := runtimeDir.Append(seal.user.aid.String())
		seal.sys.Ensure(runtimeDirInst.String(), 0700)
		seal.sys.UpdatePermType(system.User, runtimeDirInst.String(), acl.Read, acl.Write, acl.Execute)
		seal.container.Tmpfs(container.AbsFHSRunUser, 1<<12, 0755)
		seal.container.Bind(runtimeDirInst, innerRuntimeDir, container.BindWritable)
	}

	{
		tmpdir := share.sc.SharePath.Append("tmpdir")
		seal.sys.Ensure(tmpdir.String(), 0700)
		seal.sys.UpdatePermType(system.User, tmpdir.String(), acl.Execute)
		tmpdirInst := tmpdir.Append(seal.user.aid.String())
		seal.sys.Ensure(tmpdirInst.String(), 01700)
		seal.sys.UpdatePermType(system.User, tmpdirInst.String(), acl.Read, acl.Write, acl.Execute)
		// mount inner /tmp from share so it shares persistence and storage behaviour of host /tmp
		seal.container.Bind(tmpdirInst, container.AbsFHSTmp, container.BindWritable)
	}

	{
		username := "chronos"
		if seal.user.username != "" {
			username = seal.user.username
		}
		seal.container.Bind(seal.user.data, seal.user.home, container.BindWritable)
		seal.container.Dir = seal.user.home
		seal.env["HOME"] = seal.user.home.String()
		seal.env["USER"] = username
		seal.env[shell] = config.Shell.String()

		seal.container.Place(container.AbsFHSEtc.Append("passwd"),
			[]byte(username+":x:"+mapuid.String()+":"+mapgid.String()+":Hakurei:"+seal.user.home.String()+":"+config.Shell.String()+"\n"))
		seal.container.Place(container.AbsFHSEtc.Append("group"),
			[]byte("hakurei:x:"+mapgid.String()+":\n"))
	}

	// pass TERM for proper terminal I/O in initial process
	if t, ok := sys.LookupEnv(term); ok {
		seal.env[term] = t
	}

	if config.Enablements.Unwrap()&system.EWayland != 0 {
		// outer wayland socket (usually `/run/user/%d/wayland-%d`)
		var socketPath *container.Absolute
		if name, ok := sys.LookupEnv(wayland.WaylandDisplay); !ok {
			hlog.Verbose(wayland.WaylandDisplay + " is not set, assuming " + wayland.FallbackName)
			socketPath = share.sc.RuntimePath.Append(wayland.FallbackName)
		} else if a, err := container.NewAbs(name); err != nil {
			socketPath = share.sc.RuntimePath.Append(name)
		} else {
			socketPath = a
		}

		innerPath := innerRuntimeDir.Append(wayland.FallbackName)
		seal.env[wayland.WaylandDisplay] = wayland.FallbackName

		if !config.DirectWayland { // set up security-context-v1
			appID := config.ID
			if appID == "" {
				// use instance ID in case app id is not set
				appID = "app.hakurei." + seal.id.String()
			}
			// downstream socket paths
			outerPath := share.instance().Append("wayland")
			seal.sys.Wayland(&seal.sync, outerPath.String(), socketPath.String(), appID, seal.id.String())
			seal.container.Bind(outerPath, innerPath, 0)
		} else { // bind mount wayland socket (insecure)
			hlog.Verbose("direct wayland access, PROCEED WITH CAUTION")
			share.ensureRuntimeDir()
			seal.container.Bind(socketPath, innerPath, 0)
			seal.sys.UpdatePermType(system.EWayland, socketPath.String(), acl.Read, acl.Write, acl.Execute)
		}
	}

	if config.Enablements.Unwrap()&system.EX11 != 0 {
		if d, ok := sys.LookupEnv(display); !ok {
			return hlog.WrapErr(ErrXDisplay,
				"DISPLAY is not set")
		} else {
			socketDir := container.AbsFHSTmp.Append(".X11-unix")

			// the socket file at `/tmp/.X11-unix/X%d` is typically owned by the priv user
			// and not accessible by the target user
			var socketPath *container.Absolute
			if len(d) > 1 && d[0] == ':' { // `:%d`
				if n, err := strconv.Atoi(d[1:]); err == nil && n >= 0 {
					socketPath = socketDir.Append("X" + strconv.Itoa(n))
				}
			} else if len(d) > 5 && strings.HasPrefix(d, "unix:") { // `unix:%s`
				if a, err := container.NewAbs(d[5:]); err == nil {
					socketPath = a
				}
			}
			if socketPath != nil {
				if _, err := sys.Stat(socketPath.String()); err != nil {
					if !errors.Is(err, fs.ErrNotExist) {
						return hlog.WrapErrSuffix(err,
							fmt.Sprintf("cannot access X11 socket %q:", socketPath))
					}
				} else {
					seal.sys.UpdatePermType(system.EX11, socketPath.String(), acl.Read, acl.Write, acl.Execute)
					d = "unix:" + socketPath.String()
				}
			}

			seal.sys.ChangeHosts("#" + seal.user.uid.String())
			seal.env[display] = d
			seal.container.Bind(socketDir, socketDir, 0)
		}
	}

	if config.Enablements.Unwrap()&system.EPulse != 0 {
		// PulseAudio runtime directory (usually `/run/user/%d/pulse`)
		pulseRuntimeDir := share.sc.RuntimePath.Append("pulse")
		// PulseAudio socket (usually `/run/user/%d/pulse/native`)
		pulseSocket := pulseRuntimeDir.Append("native")

		if _, err := sys.Stat(pulseRuntimeDir.String()); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return hlog.WrapErrSuffix(err,
					fmt.Sprintf("cannot access PulseAudio directory %q:", pulseRuntimeDir))
			}
			return hlog.WrapErr(ErrPulseSocket,
				fmt.Sprintf("PulseAudio directory %q not found", pulseRuntimeDir))
		}

		if s, err := sys.Stat(pulseSocket.String()); err != nil {
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
		innerPulseRuntimeDir := share.runtime().Append("pulse")
		innerPulseSocket := innerRuntimeDir.Append("pulse", "native")
		seal.sys.Link(pulseSocket.String(), innerPulseRuntimeDir.String())
		seal.container.Bind(innerPulseRuntimeDir, innerPulseSocket, 0)
		seal.env[pulseServer] = "unix:" + innerPulseSocket.String()

		// publish current user's pulse cookie for target user
		if src, err := discoverPulseCookie(sys); err != nil {
			// not fatal
			hlog.Verbose(strings.TrimSpace(err.(*hlog.BaseError).Message()))
		} else {
			innerDst := hst.AbsTmp.Append("/pulse-cookie")
			seal.env[pulseCookie] = innerDst.String()
			var payload *[]byte
			seal.container.PlaceP(innerDst, &payload)
			seal.sys.CopyFile(payload, src, 256, 256)
		}
	}

	if config.Enablements.Unwrap()&system.EDBus != 0 {
		// ensure dbus session bus defaults
		if config.SessionBus == nil {
			config.SessionBus = dbus.NewConfig(config.ID, true, true)
		}

		// downstream socket paths
		sessionPath, systemPath := share.instance().Append("bus"), share.instance().Append("system_bus_socket")

		// configure dbus proxy
		if f, err := seal.sys.ProxyDBus(
			config.SessionBus, config.SystemBus,
			sessionPath.String(), systemPath.String(),
		); err != nil {
			return err
		} else {
			seal.dbusMsg = f
		}

		// share proxy sockets
		sessionInner := innerRuntimeDir.Append("bus")
		seal.env[dbusSessionBusAddress] = "unix:path=" + sessionInner.String()
		seal.container.Bind(sessionPath, sessionInner, 0)
		seal.sys.UpdatePerm(sessionPath.String(), acl.Read, acl.Write)
		if config.SystemBus != nil {
			systemInner := container.AbsFHSRun.Append("dbus/system_bus_socket")
			seal.env[dbusSystemBusAddress] = "unix:path=" + systemInner.String()
			seal.container.Bind(systemPath, systemInner, 0)
			seal.sys.UpdatePerm(systemPath.String(), acl.Read, acl.Write)
		}
	}

	// mount root read-only as the final setup Op
	seal.container.Remount(container.AbsFHSRoot, syscall.MS_RDONLY)

	// append ExtraPerms last
	for _, p := range config.ExtraPerms {
		if p == nil || p.Path == nil {
			continue
		}

		if p.Ensure {
			seal.sys.Ensure(p.Path.String(), 0700)
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
		seal.sys.UpdatePermType(system.User, p.Path.String(), perms...)
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
