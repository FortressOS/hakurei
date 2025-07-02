package hst

import (
	"git.gensokyo.uk/security/hakurei/seccomp"
)

type (
	// ContainerConfig describes the container configuration baseline to which the app implementation adds upon.
	ContainerConfig struct {
		// container hostname
		Hostname string `json:"hostname,omitempty"`

		// extra seccomp flags
		SeccompFlags seccomp.ExportFlag `json:"seccomp_flags"`
		// extra seccomp presets
		SeccompPresets seccomp.FilterPreset `json:"seccomp_presets"`
		// disable project-specific filter extensions
		SeccompCompat bool `json:"seccomp_compat,omitempty"`
		// allow ptrace and friends
		Devel bool `json:"devel,omitempty"`
		// allow userns creation in container
		Userns bool `json:"userns,omitempty"`
		// share host net namespace
		Net bool `json:"net,omitempty"`
		// allow dangerous terminal I/O
		Tty bool `json:"tty,omitempty"`
		// allow multiarch
		Multiarch bool `json:"multiarch,omitempty"`

		// initial process environment variables
		Env map[string]string `json:"env"`
		// map target user uid to privileged user uid in the user namespace
		MapRealUID bool `json:"map_real_uid"`

		// pass through all devices
		Device bool `json:"device,omitempty"`
		// container host filesystem bind mounts
		Filesystem []*FilesystemConfig `json:"filesystem"`
		// create symlinks inside container filesystem
		Link [][2]string `json:"symlink"`

		// read-only /etc directory
		Etc string `json:"etc,omitempty"`
		// automatically set up /etc symlinks
		AutoEtc bool `json:"auto_etc"`
		// cover these paths or create them if they do not already exist
		Cover []string `json:"cover"`
	}

	// FilesystemConfig is an abstract representation of a bind mount.
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
