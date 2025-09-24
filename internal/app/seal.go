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
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/hlog"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
	"hakurei.app/system/wayland"
)

func newWithMessage(msg string) error { return newWithMessageError(msg, os.ErrInvalid) }
func newWithMessageError(msg string, err error) error {
	return &hst.AppError{Step: "finalise", Err: err, Msg: msg}
}

// An Outcome is the runnable state of a hakurei container via [hst.Config].
type Outcome struct {
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

	seal *Outcome
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
	identity *stringPair[int]
	// target uid resolved by hid:aid
	uid *stringPair[int]

	// supplementary group ids
	supp []string

	// app user home directory
	home *container.Absolute
	// passwd database username
	username string
}

func (seal *Outcome) finalise(ctx context.Context, k sys.State, config *hst.Config) error {
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

	if ctx == nil {
		// unreachable
		panic("invalid call to finalise")
	}
	if seal.ctx != nil {
		// unreachable
		panic("attempting to finalise twice")
	}
	seal.ctx = ctx

	if config == nil {
		// unreachable
		return newWithMessage("invalid configuration")
	}
	if config.Home == nil {
		return newWithMessage("invalid path to home directory")
	}

	{
		// encode initial configuration for state tracking
		ct := new(bytes.Buffer)
		if err := gob.NewEncoder(ct).Encode(config); err != nil {
			return &hst.AppError{Step: "encode initial config", Err: err}
		}
		seal.ct = ct
	}

	// allowed identity range 0 to 9999, this is checked again in hsu
	if config.Identity < 0 || config.Identity > 9999 {
		return newWithMessage(fmt.Sprintf("identity %d out of range", config.Identity))
	}

	seal.user = hsuUser{
		identity: newInt(config.Identity),
		home:     config.Home,
		username: config.Username,
	}

	if seal.user.username == "" {
		seal.user.username = "chronos"
	} else if !isValidUsername(seal.user.username) {
		return newWithMessage(fmt.Sprintf("invalid user name %q", seal.user.username))
	}
	seal.user.uid = newInt(sys.MustUid(k, seal.user.identity.unwrap()))

	seal.user.supp = make([]string, len(config.Groups))
	for i, name := range config.Groups {
		if g, err := k.LookupGroup(name); err != nil {
			return newWithMessageError(fmt.Sprintf("unknown group %q", name), err)
		} else {
			seal.user.supp[i] = g.Gid
		}
	}

	// permissive defaults
	if config.Container == nil {
		hlog.Verbose("container configuration not supplied, PROCEED WITH CAUTION")

		if config.Shell == nil {
			config.Shell = container.AbsFHSRoot.Append("bin", "sh")
			s, _ := k.LookupEnv(shell)
			if a, err := container.NewAbs(s); err == nil {
				config.Shell = a
			}
		}

		// hsu clears the environment so resolve paths early
		if config.Path == nil {
			if len(config.Args) > 0 {
				if p, err := k.LookPath(config.Args[0]); err != nil {
					return &hst.AppError{Step: "look up executable file", Err: err}
				} else if config.Path, err = container.NewAbs(p); err != nil {
					return newWithMessageError(err.Error(), err)
				}
			} else {
				config.Path = config.Shell
			}
		}

		conf := &hst.ContainerConfig{
			Userns:       true,
			HostNet:      true,
			HostAbstract: true,
			Tty:          true,

			Filesystem: []hst.FilesystemConfigJSON{
				// autoroot, includes the home directory
				{FilesystemConfig: &hst.FSBind{
					Target:  container.AbsFHSRoot,
					Source:  container.AbsFHSRoot,
					Write:   true,
					Special: true,
				}},
			},
		}

		// bind GPU stuff
		if config.Enablements.Unwrap()&(system.EX11|system.EWayland) != 0 {
			conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("dri"), Device: true, Optional: true}})
		}
		// opportunistically bind kvm
		conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("kvm"), Device: true, Optional: true}})

		// hide nscd from container if present
		nscd := container.AbsFHSVar.Append("run/nscd")
		if _, err := k.Stat(nscd.String()); !errors.Is(err, fs.ErrNotExist) {
			conf.Filesystem = append(conf.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSEphemeral{Target: nscd}})
		}

		// do autoetc last
		conf.Filesystem = append(conf.Filesystem,
			hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{
				Target:  container.AbsFHSEtc,
				Source:  container.AbsFHSEtc,
				Special: true,
			}},
		)

		config.Container = conf
	}

	// late nil checks for pd behaviour
	if config.Shell == nil {
		return newWithMessage("invalid shell path")
	}
	if config.Path == nil {
		return newWithMessage("invalid program path")
	}

	var mapuid, mapgid *stringPair[int]
	{
		var uid, gid int
		var err error
		seal.container, seal.env, err = newContainer(config.Container, k, seal.id.String(), &uid, &gid)
		seal.waitDelay = config.Container.WaitDelay
		if err != nil {
			return &hst.AppError{Step: "initialise container configuration", Err: err}
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

	share := &shareHost{seal: seal, sc: k.Paths()}
	seal.runDirPath = share.sc.RunDirPath
	seal.sys = system.New(seal.ctx, seal.user.uid.unwrap())
	seal.sys.Ensure(share.sc.SharePath.String(), 0711)

	{
		runtimeDir := share.sc.SharePath.Append("runtime")
		seal.sys.Ensure(runtimeDir.String(), 0700)
		seal.sys.UpdatePermType(system.User, runtimeDir.String(), acl.Execute)
		runtimeDirInst := runtimeDir.Append(seal.user.identity.String())
		seal.sys.Ensure(runtimeDirInst.String(), 0700)
		seal.sys.UpdatePermType(system.User, runtimeDirInst.String(), acl.Read, acl.Write, acl.Execute)
		seal.container.Tmpfs(container.AbsFHSRunUser, 1<<12, 0755)
		seal.container.Bind(runtimeDirInst, innerRuntimeDir, container.BindWritable)
	}

	{
		tmpdir := share.sc.SharePath.Append("tmpdir")
		seal.sys.Ensure(tmpdir.String(), 0700)
		seal.sys.UpdatePermType(system.User, tmpdir.String(), acl.Execute)
		tmpdirInst := tmpdir.Append(seal.user.identity.String())
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
	if t, ok := k.LookupEnv(term); ok {
		seal.env[term] = t
	}

	if config.Enablements.Unwrap()&system.EWayland != 0 {
		// outer wayland socket (usually `/run/user/%d/wayland-%d`)
		var socketPath *container.Absolute
		if name, ok := k.LookupEnv(wayland.WaylandDisplay); !ok {
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
		if d, ok := k.LookupEnv(display); !ok {
			return newWithMessage("DISPLAY is not set")
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
				if _, err := k.Stat(socketPath.String()); err != nil {
					if !errors.Is(err, fs.ErrNotExist) {
						return &hst.AppError{Step: fmt.Sprintf("access X11 socket %q", socketPath), Err: err}
					}
				} else {
					seal.sys.UpdatePermType(system.EX11, socketPath.String(), acl.Read, acl.Write, acl.Execute)
					if !config.Container.HostAbstract {
						d = "unix:" + socketPath.String()
					}
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

		if _, err := k.Stat(pulseRuntimeDir.String()); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return &hst.AppError{Step: fmt.Sprintf("access PulseAudio directory %q", pulseRuntimeDir), Err: err}
			}
			return newWithMessage(fmt.Sprintf("PulseAudio directory %q not found", pulseRuntimeDir))
		}

		if s, err := k.Stat(pulseSocket.String()); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return &hst.AppError{Step: fmt.Sprintf("access PulseAudio socket %q", pulseSocket), Err: err}
			}
			return newWithMessage(fmt.Sprintf("PulseAudio directory %q found but socket does not exist", pulseRuntimeDir))
		} else {
			if m := s.Mode(); m&0o006 != 0o006 {
				return newWithMessage(fmt.Sprintf("unexpected permissions on %q: %s", pulseSocket, m))
			}
		}

		// hard link pulse socket into target-executable share
		innerPulseRuntimeDir := share.runtime().Append("pulse")
		innerPulseSocket := innerRuntimeDir.Append("pulse", "native")
		seal.sys.Link(pulseSocket.String(), innerPulseRuntimeDir.String())
		seal.container.Bind(innerPulseRuntimeDir, innerPulseSocket, 0)
		seal.env[pulseServer] = "unix:" + innerPulseSocket.String()

		// publish current user's pulse cookie for target user
		var paCookiePath *container.Absolute
		{
			const paLocateStep = "locate PulseAudio cookie"

			// from environment
			if p, ok := k.LookupEnv(pulseCookie); ok {
				if a, err := container.NewAbs(p); err != nil {
					return &hst.AppError{Step: paLocateStep, Err: err}
				} else {
					// this takes precedence, do not verify whether the file is accessible
					paCookiePath = a
					goto out
				}
			}

			// $HOME/.pulse-cookie
			if p, ok := k.LookupEnv(home); ok {
				if a, err := container.NewAbs(p); err != nil {
					return &hst.AppError{Step: paLocateStep, Err: err}
				} else {
					paCookiePath = a.Append(".pulse-cookie")
				}

				if s, err := k.Stat(paCookiePath.String()); err != nil {
					paCookiePath = nil
					if !errors.Is(err, fs.ErrNotExist) {
						return &hst.AppError{Step: "access PulseAudio cookie", Err: err}
					}
					// fallthrough
				} else if s.IsDir() {
					paCookiePath = nil
				} else {
					goto out
				}
			}

			// $XDG_CONFIG_HOME/pulse/cookie
			if p, ok := k.LookupEnv(xdgConfigHome); ok {
				if a, err := container.NewAbs(p); err != nil {
					return &hst.AppError{Step: paLocateStep, Err: err}
				} else {
					paCookiePath = a.Append("pulse", "cookie")
				}
				if s, err := k.Stat(paCookiePath.String()); err != nil {
					paCookiePath = nil
					if !errors.Is(err, fs.ErrNotExist) {
						return &hst.AppError{Step: "access PulseAudio cookie", Err: err}
					}
					// fallthrough
				} else if s.IsDir() {
					paCookiePath = nil
				} else {
					goto out
				}
			}
		out:
		}

		if paCookiePath != nil {
			innerDst := hst.AbsTmp.Append("/pulse-cookie")
			seal.env[pulseCookie] = innerDst.String()
			var payload *[]byte
			seal.container.PlaceP(innerDst, &payload)
			seal.sys.CopyFile(payload, paCookiePath.String(), 256, 256)
		} else {
			hlog.Verbose("cannot locate PulseAudio cookie (tried " +
				"$PULSE_COOKIE, " +
				"$XDG_CONFIG_HOME/pulse/cookie, " +
				"$HOME/.pulse-cookie)")
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
			return &hst.AppError{Step: "flatten environment", Err: syscall.EINVAL,
				Msg: fmt.Sprintf("invalid environment variable %s", k)}
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
