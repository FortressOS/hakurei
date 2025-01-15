package fst

import (
	"errors"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal/linux"
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
func (s *SandboxConfig) Bwrap(os linux.System) (*bwrap.Config, error) {
	if s == nil {
		return nil, errors.New("nil sandbox config")
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

		NewSession:    !s.NoNewSession,
		DieWithParent: true,
		AsInit:        true,

		// initialise map
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

	for _, c := range s.Filesystem {
		if c == nil {
			continue
		}
		src := c.Src
		dest := c.Dst
		if c.Dst == "" {
			dest = c.Src
		}
		conf.Bind(src, dest, !c.Must, c.Write, c.Device)
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
