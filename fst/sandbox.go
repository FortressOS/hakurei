package fst

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/sys"
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
	// direct access to wayland socket
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
	// paths to override by mounting tmpfs over them
	Override []string `json:"override"`
}

// Bwrap returns the address of the corresponding bwrap.Config to s.
// Note that remaining tmpfs entries must be queued by the caller prior to launch.
func (s *SandboxConfig) Bwrap(os sys.State) (*bwrap.Config, error) {
	if s == nil {
		return nil, errors.New("nil sandbox config")
	}

	if s.Syscall == nil {
		fmsg.Verbose("syscall filter not configured, PROCEED WITH CAUTION")
	}

	var uid int
	if !s.MapRealUID {
		uid = 65534
	} else {
		uid = os.Geteuid()
	}

	conf := (&bwrap.Config{
		Net:      s.Net,
		UserNS:   s.UserNS,
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
		SetUID(uid).SetGID(uid).
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
	sc := os.Paths()
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
							fmsg.Verbosef("dbus socket %q is in an unusual location", pair[1])
						}
						hidePaths = append(hidePaths, dir)
					} else {
						fmsg.Verbosef("dbus socket %q is not absolute", pair[1])
					}
				}
			}
		}
	}
	hidePathMatch := make([]bool, len(hidePaths))
	for i := range hidePaths {
		if err := evalSymlinks(os, &hidePaths[i]); err != nil {
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
		if err := evalSymlinks(os, &srcH); err != nil {
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
				fmsg.Verbosef("hiding paths from %q", c.Src)
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
		if d, err := os.ReadDir(etc); err != nil {
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

func evalSymlinks(os sys.State, v *string) error {
	if p, err := os.EvalSymlinks(*v); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		fmsg.Verbosef("path %q does not yet exist", *v)
	} else {
		*v = p
	}
	return nil
}
