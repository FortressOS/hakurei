package app

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
	"git.gensokyo.uk/security/fortify/system"
)

var (
	ErrConfig = errors.New("no configuration to seal")
	ErrUser   = errors.New("invalid aid")
	ErrHome   = errors.New("invalid home directory")
	ErrName   = errors.New("invalid username")
)

var posixUsername = regexp.MustCompilePOSIX("^[a-z_]([A-Za-z0-9_-]{0,31}|[A-Za-z0-9_-]{0,30}\\$)$")

// appSeal stores copies of various parts of [fst.Config]
type appSeal struct {
	// string representation of [fst.ID]
	id string
	// dump dbus proxy message buffer
	dbusMsg func()

	// reverse-DNS style arbitrary identifier string from config;
	// passed to wayland security-context-v1 as application ID
	// and used as part of defaults in dbus session proxy
	appID string
	// final argv, passed to init
	command []string
	// state instance initialised during seal and used on process lifecycle events
	store state.Store

	// process-specific share directory path ([os.TempDir])
	share string
	// process-specific share directory path ([fst.Paths] XDG_RUNTIME_DIR)
	shareLocal string

	// initial [fst.Config] gob stream for state data;
	// this is prepared ahead of time as config is mutated during seal creation
	ct io.WriterTo
	// passed through from [fst.SandboxConfig];
	// when this gets set no attempt is made to attach security-context-v1
	// and the bare socket is mounted to the sandbox
	directWayland bool
	// extra [acl.Update] ops, appended at the end of [system.I]
	extraPerms []*sealedExtraPerm

	// prevents sharing from happening twice
	shared bool
	// seal system-level component
	sys *appSealSys

	system.Enablements
	fst.Paths

	// protected by upstream mutex
}

type sealedExtraPerm struct {
	name   string
	perms  acl.Perms
	ensure bool
}

// Seal seals the app launch context
func (a *app) Seal(config *fst.Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.appSeal != nil {
		panic("app sealed twice")
	}

	if config == nil {
		return fmsg.WrapError(ErrConfig,
			"attempted to seal app with nil config")
	}

	// create seal
	seal := new(appSeal)

	// encode initial configuration for state tracking
	ct := new(bytes.Buffer)
	if err := gob.NewEncoder(ct).Encode(config); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot encode initial config:")
	}
	seal.ct = ct

	// fetch system constants
	seal.Paths = a.sys.Paths()

	// pass through config values
	seal.id = a.id.String()
	seal.appID = config.ID
	seal.command = config.Command

	// create seal system component
	seal.sys = new(appSealSys)

	{
		// mapped uid defaults to 65534 to work around file ownership checks due to a bwrap limitation
		mapuid := 65534
		if config.Confinement.Sandbox != nil && config.Confinement.Sandbox.MapRealUID {
			// some programs fail to connect to dbus session running as a different uid, so a
			// separate workaround is introduced to map priv-side caller uid in namespace
			mapuid = a.sys.Geteuid()
		}
		seal.sys.mapuid = newInt(mapuid)
		seal.sys.runtime = path.Join("/run/user", seal.sys.mapuid.String())
	}

	// validate uid and set user info
	if config.Confinement.AppID < 0 || config.Confinement.AppID > 9999 {
		return fmsg.WrapError(ErrUser,
			fmt.Sprintf("aid %d out of range", config.Confinement.AppID))
	}
	seal.sys.user = appUser{
		aid:      newInt(config.Confinement.AppID),
		data:     config.Confinement.Outer,
		home:     config.Confinement.Inner,
		username: config.Confinement.Username,
	}
	if seal.sys.user.username == "" {
		seal.sys.user.username = "chronos"
	} else if !posixUsername.MatchString(seal.sys.user.username) ||
		len(seal.sys.user.username) >= internal.Sysconf_SC_LOGIN_NAME_MAX() {
		return fmsg.WrapError(ErrName,
			fmt.Sprintf("invalid user name %q", seal.sys.user.username))
	}
	if seal.sys.user.data == "" || !path.IsAbs(seal.sys.user.data) {
		return fmsg.WrapError(ErrHome,
			fmt.Sprintf("invalid home directory %q", seal.sys.user.data))
	}
	if seal.sys.user.home == "" {
		seal.sys.user.home = seal.sys.user.data
	}

	// invoke fsu for full uid
	if u, err := a.sys.Uid(seal.sys.user.aid.unwrap()); err != nil {
		return err
	} else {
		seal.sys.user.uid = newInt(u)
	}

	// resolve supplementary group ids from names
	seal.sys.user.supp = make([]string, len(config.Confinement.Groups))
	for i, name := range config.Confinement.Groups {
		if g, err := a.sys.LookupGroup(name); err != nil {
			return fmsg.WrapError(err,
				fmt.Sprintf("unknown group %q", name))
		} else {
			seal.sys.user.supp[i] = g.Gid
		}
	}

	// build extra perms
	seal.extraPerms = make([]*sealedExtraPerm, len(config.Confinement.ExtraPerms))
	for i, p := range config.Confinement.ExtraPerms {
		if p == nil {
			continue
		}

		seal.extraPerms[i] = new(sealedExtraPerm)
		seal.extraPerms[i].name = p.Path
		seal.extraPerms[i].perms = make(acl.Perms, 0, 3)
		if p.Read {
			seal.extraPerms[i].perms = append(seal.extraPerms[i].perms, acl.Read)
		}
		if p.Write {
			seal.extraPerms[i].perms = append(seal.extraPerms[i].perms, acl.Write)
		}
		if p.Execute {
			seal.extraPerms[i].perms = append(seal.extraPerms[i].perms, acl.Execute)
		}
		seal.extraPerms[i].ensure = p.Ensure
	}

	// map sandbox config to bwrap
	if config.Confinement.Sandbox == nil {
		fmsg.Verbose("sandbox configuration not supplied, PROCEED WITH CAUTION")

		// permissive defaults
		conf := &fst.SandboxConfig{
			UserNS:       true,
			Net:          true,
			Syscall:      new(bwrap.SyscallPolicy),
			NoNewSession: true,
			AutoEtc:      true,
		}
		// bind entries in /
		if d, err := a.sys.ReadDir("/"); err != nil {
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
		if _, err := a.sys.Stat(nscd); !errors.Is(err, fs.ErrNotExist) {
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
	seal.directWayland = config.Confinement.Sandbox.DirectWayland
	if b, err := config.Confinement.Sandbox.Bwrap(a.sys); err != nil {
		return err
	} else {
		seal.sys.bwrap = b
	}
	seal.sys.override = config.Confinement.Sandbox.Override
	if seal.sys.bwrap.SetEnv == nil {
		seal.sys.bwrap.SetEnv = make(map[string]string)
	}

	// open process state store
	// the simple store only starts holding an open file after first action
	// store activity begins after Start is called and must end before Wait
	seal.store = state.NewMulti(seal.RunDirPath)

	// initialise system interface with os uid
	seal.sys.I = system.New(seal.sys.user.uid.unwrap())
	seal.sys.I.IsVerbose = fmsg.Load
	seal.sys.I.Verbose = fmsg.Verbose
	seal.sys.I.Verbosef = fmsg.Verbosef
	seal.sys.I.WrapErr = fmsg.WrapError

	// pass through enablements
	seal.Enablements = config.Confinement.Enablements

	// this method calls all share methods in sequence
	if err := seal.setupShares([2]*dbus.Config{config.Confinement.SessionBus, config.Confinement.SystemBus}, a.sys); err != nil {
		return err
	}

	// verbose log seal information
	fmsg.Verbosef("created application seal for uid %s (%s) groups: %v, command: %s",
		seal.sys.user.uid, seal.sys.user.username, config.Confinement.Groups, config.Command)

	// seal app and release lock
	a.appSeal = seal
	return nil
}
