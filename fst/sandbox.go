package fst

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
)

// SandboxConfig describes resources made available to the sandbox.
type SandboxConfig struct {
	// unix hostname within sandbox
	Hostname string `json:"hostname,omitempty"`
	// allow userns within sandbox
	UserNS bool `json:"userns,omitempty"`
	// share net namespace
	Net bool `json:"net,omitempty"`
	// share all devices
	Dev bool `json:"dev,omitempty"`
	// seccomp syscall filter policy
	Syscall *bwrap.SyscallPolicy `json:"syscall"`
	// do not run in new session
	NoNewSession bool `json:"no_new_session,omitempty"`
	// map target user uid to privileged user uid in the user namespace
	MapRealUID bool `json:"map_real_uid"`
	// direct access to wayland socket; when this gets set no attempt is made to attach security-context-v1
	// and the bare socket is mounted to the sandbox
	DirectWayland bool `json:"direct_wayland,omitempty"`

	// final environment variables
	Env map[string]string `json:"env"`
	// sandbox host filesystem access
	Filesystem []*FilesystemConfig `json:"filesystem"`
	// symlinks created inside the sandbox
	Link [][2]string `json:"symlink"`
	// read-only /etc directory
	Etc string `json:"etc,omitempty"`
	// automatically set up /etc symlinks
	AutoEtc bool `json:"auto_etc"`
	// mount tmpfs over these paths,
	// runs right before [ConfinementConfig.ExtraPerms]
	Override []string `json:"override"`
}

// SandboxSys encapsulates system functions used during the creation of [bwrap.Config].
type SandboxSys interface {
	Getuid() int
	Paths() Paths
	ReadDir(name string) ([]fs.DirEntry, error)
	EvalSymlinks(path string) (string, error)

	Println(v ...any)
	Printf(format string, v ...any)
}

// Bwrap returns the address of the corresponding bwrap.Config to s.
// Note that remaining tmpfs entries must be queued by the caller prior to launch.
func (s *SandboxConfig) Bwrap(sys SandboxSys, uid *int) (*bwrap.Config, error) {
	if s == nil {
		return nil, errors.New("nil sandbox config")
	}

	if s.Syscall == nil {
		sys.Println("syscall filter not configured, PROCEED WITH CAUTION")
	}

	if !s.MapRealUID {
		// mapped uid defaults to 65534 to work around file ownership checks due to a bwrap limitation
		*uid = 65534
	} else {
		// some programs fail to connect to dbus session running as a different uid, so a separate workaround
		// is introduced to map priv-side caller uid in namespace
		*uid = sys.Getuid()
	}

	conf := (&bwrap.Config{
		Net:      s.Net,
		UserNS:   s.UserNS,
		UID:      uid,
		GID:      uid,
		Hostname: s.Hostname,
		Clearenv: true,
		SetEnv:   s.Env,

		/* this is only 4 KiB of memory on a 64-bit system,
		permissive defaults on NixOS results in around 100 entries
		so this capacity should eliminate copies for most setups */
		Filesystem: make([]bwrap.FSBuilder, 0, 256),

		Syscall:       s.Syscall,
		NewSession:    !s.NoNewSession,
		DieWithParent: true,
		AsInit:        true,

		// initialise unconditionally as Once cannot be justified
		// for saving such a miniscule amount of memory
		Chmod: make(bwrap.ChmodConfig),
	}).
		Procfs("/proc").
		Tmpfs(Tmp, 4*1024)

	if !s.Dev {
		conf.DevTmpfs("/dev").Mqueue("/dev/mqueue")
	} else {
		conf.Bind("/dev", "/dev", false, true, true)
	}

	if !s.AutoEtc {
		if s.Etc == "" {
			conf.Dir("/etc")
		} else {
			conf.Bind(s.Etc, "/etc")
		}
	}

	// retrieve paths and hide them if they're made available in the sandbox
	var hidePaths []string
	sc := sys.Paths()
	hidePaths = append(hidePaths, sc.RuntimePath, sc.SharePath)
	_, systemBusAddr := dbus.Address()
	if entries, err := dbus.Parse([]byte(systemBusAddr)); err != nil {
		return nil, err
	} else {
		// there is usually only one, do not preallocate
		for _, entry := range entries {
			if entry.Method != "unix" {
				continue
			}
			for _, pair := range entry.Values {
				if pair[0] == "path" {
					if path.IsAbs(pair[1]) {
						// get parent dir of socket
						dir := path.Dir(pair[1])
						if dir == "." || dir == "/" {
							sys.Printf("dbus socket %q is in an unusual location", pair[1])
						}
						hidePaths = append(hidePaths, dir)
					} else {
						sys.Printf("dbus socket %q is not absolute", pair[1])
					}
				}
			}
		}
	}
	hidePathMatch := make([]bool, len(hidePaths))
	for i := range hidePaths {
		if err := evalSymlinks(sys, &hidePaths[i]); err != nil {
			return nil, err
		}
	}

	for _, c := range s.Filesystem {
		if c == nil {
			continue
		}

		if !path.IsAbs(c.Src) {
			return nil, fmt.Errorf("src path %q is not absolute", c.Src)
		}

		dest := c.Dst
		if c.Dst == "" {
			dest = c.Src
		} else if !path.IsAbs(dest) {
			return nil, fmt.Errorf("dst path %q is not absolute", dest)
		}

		srcH := c.Src
		if err := evalSymlinks(sys, &srcH); err != nil {
			return nil, err
		}

		for i := range hidePaths {
			// skip matched entries
			if hidePathMatch[i] {
				continue
			}

			if ok, err := deepContainsH(srcH, hidePaths[i]); err != nil {
				return nil, err
			} else if ok {
				hidePathMatch[i] = true
				sys.Printf("hiding paths from %q", c.Src)
			}
		}

		conf.Bind(c.Src, dest, !c.Must, c.Write, c.Device)
	}

	// hide marked paths before setting up shares
	for i, ok := range hidePathMatch {
		if ok {
			conf.Tmpfs(hidePaths[i], 8192)
		}
	}

	for _, l := range s.Link {
		conf.Symlink(l[0], l[1])
	}

	if s.AutoEtc {
		etc := s.Etc
		if etc == "" {
			etc = "/etc"
		}
		conf.Bind(etc, Tmp+"/etc")

		// link host /etc contents to prevent passwd/group from being overwritten
		if d, err := sys.ReadDir(etc); err != nil {
			return nil, err
		} else {
			for _, ent := range d {
				name := ent.Name()
				switch name {
				case "passwd":
				case "group":

				case "mtab":
					conf.Symlink("/proc/mounts", "/etc/"+name)
				default:
					conf.Symlink(Tmp+"/etc/"+name, "/etc/"+name)
				}
			}
		}
	}

	return conf, nil
}

func evalSymlinks(sys SandboxSys, v *string) error {
	if p, err := sys.EvalSymlinks(*v); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		sys.Printf("path %q does not yet exist", *v)
	} else {
		*v = p
	}
	return nil
}
