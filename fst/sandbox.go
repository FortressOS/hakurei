package fst

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"syscall"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/sandbox"
	"git.gensokyo.uk/security/fortify/sandbox/seccomp"
)

// SandboxConfig describes resources made available to the sandbox.
type (
	SandboxConfig struct {
		// container hostname
		Hostname string `json:"hostname,omitempty"`

		// extra seccomp flags
		Seccomp seccomp.SyscallOpts `json:"seccomp"`
		// allow ptrace and friends
		Devel bool `json:"devel,omitempty"`
		// allow userns creation in container
		Userns bool `json:"userns,omitempty"`
		// share host net namespace
		Net bool `json:"net,omitempty"`
		// expose main process tty
		Tty bool `json:"tty,omitempty"`
		// allow multiarch
		Multiarch bool `json:"multiarch,omitempty"`

		// initial process environment variables
		Env map[string]string `json:"env"`
		// map target user uid to privileged user uid in the user namespace
		MapRealUID bool `json:"map_real_uid"`

		// expose all devices
		Dev bool `json:"dev,omitempty"`
		// container host filesystem bind mounts
		Filesystem []*FilesystemConfig `json:"filesystem"`
		// create symlinks inside container filesystem
		Link [][2]string `json:"symlink"`

		// direct access to wayland socket; when this gets set no attempt is made to attach security-context-v1
		// and the bare socket is mounted to the sandbox
		DirectWayland bool `json:"direct_wayland,omitempty"`

		// read-only /etc directory
		Etc string `json:"etc,omitempty"`
		// automatically set up /etc symlinks
		AutoEtc bool `json:"auto_etc"`
		// cover these paths or create them if they do not already exist
		Cover []string `json:"cover"`
	}

	// SandboxSys encapsulates system functions used during [sandbox.Container] initialisation.
	SandboxSys interface {
		Getuid() int
		Getgid() int
		Paths() Paths
		ReadDir(name string) ([]fs.DirEntry, error)
		EvalSymlinks(path string) (string, error)

		Println(v ...any)
		Printf(format string, v ...any)
	}

	// FilesystemConfig is a representation of [sandbox.BindMount].
	FilesystemConfig struct {
		// mount point in container, same as src if empty
		Dst string `json:"dst,omitempty"`
		// host filesystem path to make available to the container
		Src string `json:"src"`
		// do not mount filesystem read-only
		Write bool `json:"write,omitempty"`
		// do not disable device files
		Device bool `json:"dev,omitempty"`
		// fail if the bind mount cannot be established for any reason
		Must bool `json:"require,omitempty"`
	}
)

// ToContainer initialises [sandbox.Params] via [SandboxConfig].
// Note that remaining container setup must be queued by the [App] implementation.
func (s *SandboxConfig) ToContainer(sys SandboxSys, uid, gid *int) (*sandbox.Params, map[string]string, error) {
	if s == nil {
		return nil, nil, syscall.EBADE
	}

	container := &sandbox.Params{
		Hostname: s.Hostname,
		Ops:      new(sandbox.Ops),
		Seccomp:  s.Seccomp,
	}

	if s.Multiarch {
		container.Seccomp |= seccomp.FlagMultiarch
	}

	/* this is only 4 KiB of memory on a 64-bit system,
	permissive defaults on NixOS results in around 100 entries
	so this capacity should eliminate copies for most setups */
	*container.Ops = slices.Grow(*container.Ops, 1<<8)

	if s.Devel {
		container.Flags |= sandbox.FAllowDevel
	}
	if s.Userns {
		container.Flags |= sandbox.FAllowUserns
	}
	if s.Net {
		container.Flags |= sandbox.FAllowNet
	}
	if s.Tty {
		container.Flags |= sandbox.FAllowTTY
	}

	if s.MapRealUID {
		/* some programs fail to connect to dbus session running as a different uid
		so this workaround is introduced to map priv-side caller uid in container */
		container.Uid = sys.Getuid()
		*uid = container.Uid
		container.Gid = sys.Getgid()
		*gid = container.Gid
	} else {
		*uid = sandbox.OverflowUid()
		*gid = sandbox.OverflowGid()
	}

	container.
		Proc("/proc").
		Tmpfs(Tmp, 1<<12, 0755)

	if !s.Dev {
		container.Dev("/dev").Mqueue("/dev/mqueue")
	} else {
		container.Bind("/dev", "/dev", sandbox.BindDevice)
	}

	/* retrieve paths and hide them if they're made available in the sandbox;
	this feature tries to improve user experience of permissive defaults, and
	to warn about issues in custom configuration; it is NOT a security feature
	and should not be treated as such, ALWAYS be careful with what you bind */
	var hidePaths []string
	sc := sys.Paths()
	hidePaths = append(hidePaths, sc.RuntimePath, sc.SharePath)
	_, systemBusAddr := dbus.Address()
	if entries, err := dbus.Parse([]byte(systemBusAddr)); err != nil {
		return nil, nil, err
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
			return nil, nil, err
		}
	}

	for _, c := range s.Filesystem {
		if c == nil {
			continue
		}

		if !path.IsAbs(c.Src) {
			return nil, nil, fmt.Errorf("src path %q is not absolute", c.Src)
		}

		dest := c.Dst
		if c.Dst == "" {
			dest = c.Src
		} else if !path.IsAbs(dest) {
			return nil, nil, fmt.Errorf("dst path %q is not absolute", dest)
		}

		srcH := c.Src
		if err := evalSymlinks(sys, &srcH); err != nil {
			return nil, nil, err
		}

		for i := range hidePaths {
			// skip matched entries
			if hidePathMatch[i] {
				continue
			}

			if ok, err := deepContainsH(srcH, hidePaths[i]); err != nil {
				return nil, nil, err
			} else if ok {
				hidePathMatch[i] = true
				sys.Printf("hiding paths from %q", c.Src)
			}
		}

		var flags int
		if c.Write {
			flags |= sandbox.BindWritable
		}
		if c.Device {
			flags |= sandbox.BindDevice | sandbox.BindWritable
		}
		if !c.Must {
			flags |= sandbox.BindOptional
		}
		container.Bind(c.Src, dest, flags)
	}

	// cover matched paths
	for i, ok := range hidePathMatch {
		if ok {
			container.Tmpfs(hidePaths[i], 1<<13, 0755)
		}
	}

	for _, l := range s.Link {
		container.Link(l[0], l[1])
	}

	// perf: this might work better if implemented as a setup op in container init
	if !s.AutoEtc {
		if s.Etc != "" {
			container.Bind(s.Etc, "/etc", 0)
		}
	} else {
		etcPath := s.Etc
		if etcPath == "" {
			etcPath = "/etc"
		}
		container.Bind(etcPath, Tmp+"/etc", 0)

		// link host /etc contents to prevent dropping passwd/group bind mounts
		if d, err := sys.ReadDir(etcPath); err != nil {
			return nil, nil, err
		} else {
			for _, ent := range d {
				n := ent.Name()
				switch n {
				case "passwd":
				case "group":

				case "mtab":
					container.Link("/proc/mounts", "/etc/"+n)
				default:
					container.Link(Tmp+"/etc/"+n, "/etc/"+n)
				}
			}
		}
	}

	return container, maps.Clone(s.Env), nil
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
