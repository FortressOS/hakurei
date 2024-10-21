package app

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"

	"git.ophivana.moe/security/fortify/dbus"
	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	"git.ophivana.moe/security/fortify/internal/shim"
	"git.ophivana.moe/security/fortify/internal/state"
	"git.ophivana.moe/security/fortify/internal/system"
)

const (
	LaunchMethodSudo uint8 = iota
	LaunchMethodMachineCtl
)

var method = [...]string{
	LaunchMethodSudo:       "sudo",
	LaunchMethodMachineCtl: "systemd",
}

var (
	ErrConfig = errors.New("no configuration to seal")
	ErrUser   = errors.New("unknown user")
	ErrLaunch = errors.New("invalid launch method")

	ErrSudo       = errors.New("sudo not available")
	ErrSystemd    = errors.New("systemd not available")
	ErrMachineCtl = errors.New("machinectl not available")
)

// appSeal seals the application with child-related information
type appSeal struct {
	// app unique ID string representation
	id string
	// wayland mediation, disabled if nil
	wl *shim.Wayland

	// freedesktop application ID
	fid string
	// argv to start process with in the final confined environment
	command []string
	// persistent process state store
	store state.Store

	// uint8 representation of launch method sealed from config
	launchOption uint8
	// process-specific share directory path
	share string
	// process-specific share directory path local to XDG_RUNTIME_DIR
	shareLocal string

	// path to launcher program
	toolPath string
	// pass-through enablement tracking from config
	et system.Enablements

	// prevents sharing from happening twice
	shared bool
	// seal system-level component
	sys *appSealSys

	// used in various sealing operations
	internal.SystemConstants

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
	seal.SystemConstants = internal.GetSC()

	// pass through config values
	seal.id = a.id.String()
	seal.fid = config.ID
	seal.command = config.Command

	// parses launch method text and looks up tool path
	switch config.Method {
	case method[LaunchMethodSudo]:
		seal.launchOption = LaunchMethodSudo
		if sudoPath, err := exec.LookPath("sudo"); err != nil {
			return fmsg.WrapError(ErrSudo,
				"sudo not found")
		} else {
			seal.toolPath = sudoPath
		}
	case method[LaunchMethodMachineCtl]:
		seal.launchOption = LaunchMethodMachineCtl
		if !internal.SdBootedV {
			return fmsg.WrapError(ErrSystemd,
				"system has not been booted with systemd as init system")
		}

		if machineCtlPath, err := exec.LookPath("machinectl"); err != nil {
			return fmsg.WrapError(ErrMachineCtl,
				"machinectl not found")
		} else {
			seal.toolPath = machineCtlPath
		}
	default:
		return fmsg.WrapError(ErrLaunch,
			"invalid launch method")
	}

	// create seal system component
	seal.sys = new(appSealSys)

	// look up fortify executable path
	if p, err := os.Executable(); err != nil {
		return fmsg.WrapErrorSuffix(err, "cannot look up fortify executable path:")
	} else {
		seal.sys.executable = p
	}

	// look up user from system
	if u, err := user.Lookup(config.User); err != nil {
		if errors.As(err, new(user.UnknownUserError)) {
			return fmsg.WrapError(ErrUser, "unknown user", config.User)
		} else {
			// unreachable
			panic(err)
		}
	} else {
		seal.sys.user = u
		seal.sys.runtime = path.Join("/run/user", u.Uid)
	}

	// map sandbox config to bwrap
	if config.Confinement.Sandbox == nil {
		fmsg.VPrintln("sandbox configuration not supplied, PROCEED WITH CAUTION")

		// permissive defaults
		conf := &SandboxConfig{
			UserNS:       true,
			Net:          true,
			NoNewSession: true,
		}
		// bind entries in /
		if d, err := os.ReadDir("/"); err != nil {
			return err
		} else {
			b := make([]*FilesystemConfig, 0, len(d))
			for _, ent := range d {
				name := ent.Name()
				switch name {
				case "proc":
				case "dev":
				case "run":
				case "tmp":
				case "mnt":
				default:
					p := "/" + name
					b = append(b, &FilesystemConfig{Src: p, Write: true, Must: true})
				}
			}
			conf.Filesystem = append(conf.Filesystem, b...)
		}
		// bind entries in /run
		if d, err := os.ReadDir("/run"); err != nil {
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
		if _, err := os.Stat(nscd); !errors.Is(err, os.ErrNotExist) {
			conf.Override = append(conf.Override, nscd)
		}
		// bind GPU stuff
		if config.Confinement.Enablements.Has(system.EX11) || config.Confinement.Enablements.Has(system.EWayland) {
			conf.Filesystem = append(conf.Filesystem, &FilesystemConfig{Src: "/dev/dri", Device: true})
		}
		config.Confinement.Sandbox = conf
	}
	seal.sys.bwrap = config.Confinement.Sandbox.Bwrap()
	seal.sys.override = config.Confinement.Sandbox.Override
	if seal.sys.bwrap.SetEnv == nil {
		seal.sys.bwrap.SetEnv = make(map[string]string)
	}

	// create wayland struct and client wait channel if mediated wayland is enabled
	// this field being set enables mediated wayland setup later on
	if config.Confinement.Sandbox.Wayland {
		seal.wl = shim.NewWayland()
	}

	// open process state store
	// the simple store only starts holding an open file after first action
	// store activity begins after Start is called and must end before Wait
	seal.store = state.NewSimple(seal.SystemConstants.RunDirPath, seal.sys.user.Uid)

	// parse string UID
	if u, err := strconv.Atoi(seal.sys.user.Uid); err != nil {
		// unreachable unless kernel bug
		panic("uid parse")
	} else {
		seal.sys.I = system.New(u)
	}

	// pass through enablements
	seal.et = config.Confinement.Enablements

	// this method calls all share methods in sequence
	if err := seal.shareAll([2]*dbus.Config{config.Confinement.SessionBus, config.Confinement.SystemBus}); err != nil {
		return err
	}

	// verbose log seal information
	fmsg.VPrintln("created application seal as user",
		seal.sys.user.Username, "("+seal.sys.user.Uid+"),",
		"method:", config.Method+",",
		"launcher:", seal.toolPath+",",
		"command:", config.Command)

	// seal app and release lock
	a.seal = seal
	return nil
}
