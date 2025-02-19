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

	"git.gensokyo.uk/security/fortify/acl"
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
	"git.gensokyo.uk/security/fortify/internal/sys"
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

	// state instance initialised during seal; used during process lifecycle events
	store state.Store
	// whether [system.I] was committed; used during process lifecycle events
	needRevert bool
	// whether state was inserted into [state.Store]; used during process lifecycle events
	stateInStore bool

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
	// mount tmpfs over these paths, runs right before extraPerms
	override []string
	// extra [acl.Update] ops, appended at the end of [system.I]
	extraPerms []*sealedExtraPerm

	// post fsu state
	user appUser
	// inner XDG_RUNTIME_DIR, default formatting via user
	innerRuntimeDir string
	// mapped uid and gid in user namespace
	mapuid *stringPair[int]

	sys       *system.I
	container *bwrap.Config
	bwrapSync *os.File

	// prevents sharing from happening twice
	shared bool

	system.Enablements
	fst.Paths

	// protected by upstream mutex
}

// appUser stores post-fsu credentials and metadata
type appUser struct {
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

type sealedExtraPerm struct {
	name   string
	perms  acl.Perms
	ensure bool
}

func (seal *appSeal) finalise(sys sys.State, config *fst.Config, id string) error {
	// encode initial configuration for state tracking
	ct := new(bytes.Buffer)
	if err := gob.NewEncoder(ct).Encode(config); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot encode initial config:")
	}
	seal.ct = ct

	seal.Paths = sys.Paths()

	// pass through config values
	seal.id = id
	seal.appID = config.ID
	seal.command = config.Command

	{
		// mapped uid defaults to 65534 to work around file ownership checks due to a bwrap limitation
		mapuid := 65534
		if config.Confinement.Sandbox != nil && config.Confinement.Sandbox.MapRealUID {
			// some programs fail to connect to dbus session running as a different uid, so a
			// separate workaround is introduced to map priv-side caller uid in namespace
			mapuid = sys.Geteuid()
		}
		seal.mapuid = newInt(mapuid)
		seal.innerRuntimeDir = path.Join("/run/user", seal.mapuid.String())
	}

	// validate uid and set user info
	if config.Confinement.AppID < 0 || config.Confinement.AppID > 9999 {
		return fmsg.WrapError(ErrUser,
			fmt.Sprintf("aid %d out of range", config.Confinement.AppID))
	}
	seal.user = appUser{
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

	// invoke fsu for full uid
	if u, err := sys.Uid(seal.user.aid.unwrap()); err != nil {
		return err
	} else {
		seal.user.uid = newInt(u)
	}

	// resolve supplementary group ids from names
	seal.user.supp = make([]string, len(config.Confinement.Groups))
	for i, name := range config.Confinement.Groups {
		if g, err := sys.LookupGroup(name); err != nil {
			return fmsg.WrapError(err,
				fmt.Sprintf("unknown group %q", name))
		} else {
			seal.user.supp[i] = g.Gid
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
	seal.directWayland = config.Confinement.Sandbox.DirectWayland
	if b, err := config.Confinement.Sandbox.Bwrap(sys); err != nil {
		return err
	} else {
		seal.container = b
	}
	seal.override = config.Confinement.Sandbox.Override
	if seal.container.SetEnv == nil {
		seal.container.SetEnv = make(map[string]string)
	}

	// open process state store
	// the simple store only starts holding an open file after first action
	// store activity begins after Start is called and must end before Wait
	seal.store = state.NewMulti(seal.RunDirPath)

	// initialise system interface with os uid
	seal.sys = system.New(seal.user.uid.unwrap())
	seal.sys.IsVerbose = fmsg.Load
	seal.sys.Verbose = fmsg.Verbose
	seal.sys.Verbosef = fmsg.Verbosef
	seal.sys.WrapErr = fmsg.WrapError

	// pass through enablements
	seal.Enablements = config.Confinement.Enablements

	// this method calls all share methods in sequence
	if err := seal.setupShares([2]*dbus.Config{config.Confinement.SessionBus, config.Confinement.SystemBus}, sys); err != nil {
		return err
	}

	// verbose log seal information
	fmsg.Verbosef("created application seal for uid %s (%s) groups: %v, command: %s",
		seal.user.uid, seal.user.username, config.Confinement.Groups, config.Command)

	return nil
}
