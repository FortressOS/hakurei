package app

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/helper/bwrap"
	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	LaunchMethodSudo uint8 = iota
	LaunchMethodMachineCtl
)

var (
	ErrConfig = errors.New("no configuration to seal")
	ErrUser   = errors.New("unknown user")
	ErrLaunch = errors.New("invalid launch method")

	ErrSudo       = errors.New("sudo not available")
	ErrSystemd    = errors.New("systemd not available")
	ErrMachineCtl = errors.New("machinectl not available")
)

type (
	SealConfigError     BaseError
	LauncherLookupError BaseError
	SecurityError       BaseError
)

// Seal seals the app launch context
func (a *app) Seal(config *Config) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.seal != nil {
		panic("app sealed twice")
	}

	if config == nil {
		return (*SealConfigError)(wrapError(ErrConfig, "attempted to seal app with nil config"))
	}

	// create seal
	seal := new(appSeal)

	// generate application ID
	if id, err := newAppID(); err != nil {
		return (*SecurityError)(wrapError(err, "cannot generate application ID:", err))
	} else {
		seal.id = id
	}

	// fetch system constants
	seal.SystemConstants = internal.GetSC()

	// pass through config values
	seal.fid = config.ID
	seal.command = config.Command

	// parses launch method text and looks up tool path
	switch config.Method {
	case "sudo":
		seal.launchOption = LaunchMethodSudo
		if sudoPath, err := exec.LookPath("sudo"); err != nil {
			return (*LauncherLookupError)(wrapError(ErrSudo, "sudo not found"))
		} else {
			seal.toolPath = sudoPath
		}
	case "systemd":
		seal.launchOption = LaunchMethodMachineCtl
		if !internal.SdBootedV {
			return (*LauncherLookupError)(wrapError(ErrSystemd,
				"system has not been booted with systemd as init system"))
		}

		if machineCtlPath, err := exec.LookPath("machinectl"); err != nil {
			return (*LauncherLookupError)(wrapError(ErrMachineCtl, "machinectl not found"))
		} else {
			seal.toolPath = machineCtlPath
		}
	default:
		return (*SealConfigError)(wrapError(ErrLaunch, "invalid launch method"))
	}

	// create seal system component
	seal.sys = new(appSealTx)

	// look up fortify executable path
	if p, err := os.Executable(); err != nil {
		return (*LauncherLookupError)(wrapError(err, "cannot look up fortify executable path:", err))
	} else {
		seal.sys.executable = p
	}

	// look up user from system
	if u, err := user.Lookup(config.User); err != nil {
		if errors.As(err, new(user.UnknownUserError)) {
			return (*SealConfigError)(wrapError(ErrUser, "unknown user", config.User))
		} else {
			// unreachable
			panic(err)
		}
	} else {
		seal.sys.User = u
		seal.sys.runtime = path.Join("/run/user", u.Uid)
	}

	// map sandbox config to bwrap
	if config.Confinement.Sandbox == nil {
		verbose.Println("sandbox configuration not supplied, PROCEED WITH CAUTION")

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
			conf.Tmpfs = append(conf.Tmpfs, bwrap.TmpfsConfig{Size: 8 * 1024, Dir: nscd})
		}
		// bind GPU stuff
		if config.Confinement.Enablements.Has(state.EnableX) || config.Confinement.Enablements.Has(state.EnableWayland) {
			conf.Filesystem = append(conf.Filesystem, &FilesystemConfig{Src: "/dev/dri", Device: true})
		}
		config.Confinement.Sandbox = conf
	}
	seal.sys.bwrap = config.Confinement.Sandbox.Bwrap()
	if seal.sys.bwrap.SetEnv == nil {
		seal.sys.bwrap.SetEnv = make(map[string]string)
	}

	// create wayland client wait channel if mediated wayland is enabled
	// this channel being set enables mediated wayland setup later on
	if config.Confinement.Sandbox.Wayland {
		seal.wlDone = make(chan struct{})
	}

	// open process state store
	// the simple store only starts holding an open file after first action
	// store activity begins after Start is called and must end before Wait
	seal.store = state.NewSimple(seal.SystemConstants.RunDirPath, seal.sys.Uid)

	// parse string UID
	if u, err := strconv.Atoi(seal.sys.Uid); err != nil {
		// unreachable unless kernel bug
		panic("uid parse")
	} else {
		seal.sys.uid = u
	}

	// pass through enablements
	seal.et = config.Confinement.Enablements

	// this method calls all share methods in sequence
	if err := seal.shareAll([2]*dbus.Config{config.Confinement.SessionBus, config.Confinement.SystemBus}); err != nil {
		return err
	}

	// verbose log seal information
	verbose.Println("created application seal as user",
		seal.sys.Username, "("+seal.sys.Uid+"),",
		"method:", config.Method+",",
		"launcher:", seal.toolPath+",",
		"command:", config.Command)

	// seal app and release lock
	a.seal = seal
	return nil
}
