package hst

import (
	"time"

	"hakurei.app/container"
	"hakurei.app/system/dbus"
)

const Tmp = "/.hakurei"

var AbsTmp = container.MustAbs(Tmp)

const (
	// DefaultWaitDelay is used when WaitDelay has its zero value.
	DefaultWaitDelay = 5 * time.Second
	// MaxWaitDelay is used if WaitDelay exceeds its value.
	MaxWaitDelay = 30 * time.Second
)

type (
	// Config configures an application container, implemented in internal/app.
	Config struct {
		// Reverse-DNS style configured arbitrary identifier string.
		// Passed to wayland security-context-v1 and used as part of defaults in dbus session proxy.
		ID string `json:"id"`

		// Pathname to executable file in the container filesystem.
		Path *container.Absolute `json:"path,omitempty"`
		// Final args passed to the initial program.
		Args []string `json:"args"`

		// System services to make available in the container.
		Enablements *Enablements `json:"enablements,omitempty"`

		// Session D-Bus proxy configuration.
		// If set to nil, session bus proxy assume built-in defaults.
		SessionBus *dbus.Config `json:"session_bus,omitempty"`
		// System D-Bus proxy configuration.
		// If set to nil, system bus proxy is disabled.
		SystemBus *dbus.Config `json:"system_bus,omitempty"`
		// Direct access to wayland socket, no attempt is made to attach security-context-v1
		// and the bare socket is made available to the container.
		DirectWayland bool `json:"direct_wayland,omitempty"`

		// String used as the username of the emulated user, validated against the default NAME_REGEX from adduser.
		// Defaults to passwd name of target uid or chronos.
		Username string `json:"username,omitempty"`
		// Pathname of shell in the container filesystem to use for the emulated user.
		Shell *container.Absolute `json:"shell"`
		// Directory in the container filesystem to enter and use as the home directory of the emulated user.
		Home *container.Absolute `json:"home"`

		// Extra acl update ops to perform before setuid.
		ExtraPerms []*ExtraPermConfig `json:"extra_perms,omitempty"`

		// Numerical application id, passed to hsu, used to derive init user namespace credentials.
		Identity int `json:"identity"`
		// Init user namespace supplementary groups inherited by all container processes.
		Groups []string `json:"groups"`

		// High level configuration applied to the underlying [container.Params].
		Container *ContainerConfig `json:"container"`
	}

	// ContainerConfig describes the container configuration to be applied to an underlying [container.Params].
	ContainerConfig struct {
		// Container UTS namespace hostname.
		Hostname string `json:"hostname,omitempty"`

		// Duration in nanoseconds to wait for after interrupting the initial process.
		// Defaults to [DefaultWaitDelay] if less than or equals to zero,
		// or [MaxWaitDelay] if greater than [MaxWaitDelay].
		WaitDelay time.Duration `json:"wait_delay,omitempty"`

		// Emit Flatpak-compatible seccomp filter programs.
		SeccompCompat bool `json:"seccomp_compat,omitempty"`
		// Allow ptrace and friends.
		Devel bool `json:"devel,omitempty"`
		// Allow userns creation and container setup syscalls.
		Userns bool `json:"userns,omitempty"`
		// Share host net namespace.
		HostNet bool `json:"host_net,omitempty"`
		// Share abstract unix socket scope.
		HostAbstract bool `json:"host_abstract,omitempty"`
		// Allow dangerous terminal I/O (faking input).
		Tty bool `json:"tty,omitempty"`
		// Allow multiarch.
		Multiarch bool `json:"multiarch,omitempty"`

		// Initial process environment variables.
		Env map[string]string `json:"env"`

		/* Map target user uid to privileged user uid in the container user namespace.

		Some programs fail to connect to dbus session running as a different uid,
		this option works around it by mapping priv-side caller uid in container. */
		MapRealUID bool `json:"map_real_uid"`

		// Mount /dev/ from the init mount namespace as-is in the container mount namespace.
		Device bool `json:"device,omitempty"`

		/* Container mount points.

		If the first element targets /, it is inserted early and excluded from path hiding. */
		Filesystem []FilesystemConfigJSON `json:"filesystem"`
	}
)

// ExtraPermConfig describes an acl update op.
type ExtraPermConfig struct {
	Ensure  bool                `json:"ensure,omitempty"`
	Path    *container.Absolute `json:"path"`
	Read    bool                `json:"r,omitempty"`
	Write   bool                `json:"w,omitempty"`
	Execute bool                `json:"x,omitempty"`
}

func (e *ExtraPermConfig) String() string {
	if e == nil || e.Path == nil {
		return "<invalid>"
	}
	buf := make([]byte, 0, 5+len(e.Path.String()))
	buf = append(buf, '-', '-', '-')
	if e.Ensure {
		buf = append(buf, '+')
	}
	buf = append(buf, ':')
	buf = append(buf, []byte(e.Path.String())...)
	if e.Read {
		buf[0] = 'r'
	}
	if e.Write {
		buf[1] = 'w'
	}
	if e.Execute {
		buf[2] = 'x'
	}
	return string(buf)
}
