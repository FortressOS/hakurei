package hst

import (
	"time"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
)

const (
	// TmpfsPerm is the permission bits for tmpfs mount points
	// configured through [FilesystemConfig].
	TmpfsPerm = 0755

	// TmpfsSize is the size for tmpfs mount points
	// configured through [FilesystemConfig].
	TmpfsSize = 0
)

type (
	// ContainerConfig describes the container configuration baseline to which the app implementation adds upon.
	ContainerConfig struct {
		// container hostname
		Hostname string `json:"hostname,omitempty"`

		// duration to wait for after interrupting a container's initial process in nanoseconds;
		// a negative value causes the container to be terminated immediately on cancellation
		WaitDelay time.Duration `json:"wait_delay,omitempty"`

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
		Filesystem []FilesystemConfig `json:"filesystem"`
		// create symlinks inside container filesystem
		Link []LinkConfig `json:"symlink"`

		// automatically bind mount top-level directories to container root;
		// the zero value disables this behaviour
		AutoRoot *container.Absolute `json:"auto_root,omitempty"`
		// extra flags for AutoRoot
		RootFlags int `json:"root_flags,omitempty"`

		// read-only /etc directory
		Etc *container.Absolute `json:"etc,omitempty"`
		// automatically set up /etc symlinks
		AutoEtc bool `json:"auto_etc"`
	}

	// FilesystemConfig is an abstract representation of a bind mount.
	FilesystemConfig struct {
		// mount point in container, same as src if empty
		Dst *container.Absolute `json:"dst,omitempty"`
		// host filesystem path to make available to the container
		Src *container.Absolute `json:"src"`
		// do not mount filesystem read-only
		Write bool `json:"write,omitempty"`
		// do not disable device files
		Device bool `json:"dev,omitempty"`
		// fail if the bind mount cannot be established for any reason
		Must bool `json:"require,omitempty"`
	}

	LinkConfig struct {
		// symlink target in container
		Target *container.Absolute `json:"target"`
		// linkname the symlink points to;
		// prepend '*' to dereference an absolute pathname on host
		Linkname string `json:"linkname"`
	}
)
