package app

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"

	"git.ophivana.moe/security/fortify/dbus"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	"git.ophivana.moe/security/fortify/internal/linux"
	"git.ophivana.moe/security/fortify/internal/state"
	"git.ophivana.moe/security/fortify/internal/system"
)

var (
	ErrConfig = errors.New("no configuration to seal")
	ErrUser   = errors.New("invalid aid")
	ErrHome   = errors.New("invalid home directory")
	ErrName   = errors.New("invalid username")
)

var posixUsername = regexp.MustCompilePOSIX("^[a-z_]([A-Za-z0-9_-]{0,31}|[A-Za-z0-9_-]{0,30}\\$)$")

// appSeal seals the application with child-related information
type appSeal struct {
	// app unique ID string representation
	id string
	// dbus proxy message buffer retriever
	dbusMsg func(f func(msgbuf []string))

	// freedesktop application ID
	fid string
	// argv to start process with in the final confined environment
	command []string
	// persistent process state store
	store state.Store

	// process-specific share directory path
	share string
	// process-specific share directory path local to XDG_RUNTIME_DIR
	shareLocal string

	// pass-through enablement tracking from config
	et system.Enablements
	// wayland socket direct access
	directWayland bool

	// prevents sharing from happening twice
	shared bool
	// seal system-level component
	sys *appSealSys

	linux.Paths

	// protected by upstream mutex
}

// Seal seals the app launch context
func (a *app) Seal(config *Config) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.seal != nil {
		panic("app sealed twice")
	}

	if config == nil {
		return fmsg.WrapError(ErrConfig,
			"attempted to seal app with nil config")
	}

	// create seal
	seal := new(appSeal)

	// fetch system constants
	seal.Paths = a.os.Paths()

	// pass through config values
	seal.id = a.id.String()
	seal.fid = config.ID
	seal.command = config.Command

	// create seal system component
	seal.sys = new(appSealSys)

	// mapped uid
	if config.Confinement.Sandbox != nil && config.Confinement.Sandbox.MapRealUID {
		seal.sys.mappedID = a.os.Geteuid()
	} else {
		seal.sys.mappedID = 65534
	}
	seal.sys.mappedIDString = strconv.Itoa(seal.sys.mappedID)
	seal.sys.runtime = path.Join("/run/user", seal.sys.mappedIDString)

	// validate uid and set user info
	if config.Confinement.AppID < 0 || config.Confinement.AppID > 9999 {
		return fmsg.WrapError(ErrUser,
			fmt.Sprintf("aid %d out of range", config.Confinement.AppID))
	} else {
		seal.sys.user = appUser{
			aid:      config.Confinement.AppID,
			as:       strconv.Itoa(config.Confinement.AppID),
			data:     config.Confinement.Outer,
			home:     config.Confinement.Inner,
			username: config.Confinement.Username,
		}
		if seal.sys.user.username == "" {
			seal.sys.user.username = "chronos"
		} else if !posixUsername.MatchString(seal.sys.user.username) {
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
		if u, err := a.os.Uid(seal.sys.user.aid); err != nil {
			return fmsg.WrapErrorSuffix(err,
				"cannot obtain uid from fsu:")
		} else {
			seal.sys.user.uid = u
			seal.sys.user.us = strconv.Itoa(u)
		}

		// resolve supplementary group ids from names
		seal.sys.user.supp = make([]string, len(config.Confinement.Groups))
		for i, name := range config.Confinement.Groups {
			if g, err := a.os.LookupGroup(name); err != nil {
				return fmsg.WrapError(err,
					fmt.Sprintf("unknown group %q", name))
			} else {
				seal.sys.user.supp[i] = g.Gid
			}
		}
	}

	// map sandbox config to bwrap
	if config.Confinement.Sandbox == nil {
		fmsg.VPrintln("sandbox configuration not supplied, PROCEED WITH CAUTION")

		// permissive defaults
		conf := &SandboxConfig{
			UserNS:       true,
			Net:          true,
			NoNewSession: true,
			AutoEtc:      true,
		}
		// bind entries in /
		if d, err := a.os.ReadDir("/"); err != nil {
			return err
		} else {
			b := make([]*FilesystemConfig, 0, len(d))
			for _, ent := range d {
				p := "/" + ent.Name()
				switch p {
				case "/proc":
				case "/dev":
				case "/run":
				case "/tmp":
				case "/mnt":
				case "/etc":

				default:
					b = append(b, &FilesystemConfig{Src: p, Write: true, Must: true})
				}
			}
			conf.Filesystem = append(conf.Filesystem, b...)
		}
		// bind entries in /run
		if d, err := a.os.ReadDir("/run"); err != nil {
			return err
		} else {
			b := make([]*FilesystemConfig, 0, len(d))
			for _, ent := range d {
				name := ent.Name()
				switch name {
				case "user":
				case "dbus":
				default:
					p := "/run/" + name
					b = append(b, &FilesystemConfig{Src: p, Write: true, Must: true})
				}
			}
			conf.Filesystem = append(conf.Filesystem, b...)
		}
		// hide nscd from sandbox if present
		nscd := "/var/run/nscd"
		if _, err := a.os.Stat(nscd); !errors.Is(err, fs.ErrNotExist) {
			conf.Override = append(conf.Override, nscd)
		}
		// bind GPU stuff
		if config.Confinement.Enablements.Has(system.EX11) || config.Confinement.Enablements.Has(system.EWayland) {
			conf.Filesystem = append(conf.Filesystem, &FilesystemConfig{Src: "/dev/dri", Device: true})
		}

		config.Confinement.Sandbox = conf
	}
	seal.directWayland = config.Confinement.Sandbox.DirectWayland
	if b, err := config.Confinement.Sandbox.Bwrap(a.os); err != nil {
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
	seal.store = state.NewSimple(seal.RunDirPath, seal.sys.user.as)

	// initialise system interface with full uid
	seal.sys.I = system.New(seal.sys.user.uid)

	// pass through enablements
	seal.et = config.Confinement.Enablements

	// this method calls all share methods in sequence
	if err := seal.shareAll([2]*dbus.Config{config.Confinement.SessionBus, config.Confinement.SystemBus}, a.os); err != nil {
		return err
	}

	// verbose log seal information
	fmsg.VPrintf("created application seal for uid %s (%s) groups: %v, command: %s",
		seal.sys.user.us, seal.sys.user.username, config.Confinement.Groups, config.Command)

	// seal app and release lock
	a.seal = seal
	return nil
}
