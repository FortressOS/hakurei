package hst

import (
	"time"

	"hakurei.app/container"
	"hakurei.app/system/dbus"
)

const Tmp = "/.hakurei"

var AbsTmp = container.MustAbs(Tmp)

// Config is used to seal an app implementation.
type (
	Config struct {
		// reverse-DNS style arbitrary identifier string from config;
		// passed to wayland security-context-v1 as application ID
		// and used as part of defaults in dbus session proxy
		ID string `json:"id"`

		// absolute path to executable file
		Path *container.Absolute `json:"path,omitempty"`
		// final args passed to container init
		Args []string `json:"args"`

		// system services to make available in the container
		Enablements *Enablements `json:"enablements,omitempty"`

		// session D-Bus proxy configuration;
		// nil makes session bus proxy assume built-in defaults
		SessionBus *dbus.Config `json:"session_bus,omitempty"`
		// system D-Bus proxy configuration;
		// nil disables system bus proxy
		SystemBus *dbus.Config `json:"system_bus,omitempty"`
		// direct access to wayland socket; when this gets set no attempt is made to attach security-context-v1
		// and the bare socket is mounted to the sandbox
		DirectWayland bool `json:"direct_wayland,omitempty"`

		// passwd username in container, defaults to passwd name of target uid or chronos
		Username string `json:"username,omitempty"`
		// absolute path to shell
		Shell *container.Absolute `json:"shell"`
		// directory to enter and use as home in the container mount namespace
		Home *container.Absolute `json:"home"`

		// extra acl ops to perform before setuid
		ExtraPerms []*ExtraPermConfig `json:"extra_perms,omitempty"`

		// numerical application id, used for init user namespace credentials
		Identity int `json:"identity"`
		// list of supplementary groups inherited by container processes
		Groups []string `json:"groups"`

		// abstract container configuration baseline
		Container *ContainerConfig `json:"container"`
	}

	// ContainerConfig describes the container configuration baseline to which the app implementation adds upon.
	ContainerConfig struct {
		// container hostname
		Hostname string `json:"hostname,omitempty"`

		// duration to wait for after interrupting a container's initial process in nanoseconds;
		// a negative value causes the container to be terminated immediately on cancellation
		WaitDelay time.Duration `json:"wait_delay,omitempty"`

		// disable project-specific filter extensions
		SeccompCompat bool `json:"seccomp_compat,omitempty"`
		// allow ptrace and friends
		Devel bool `json:"devel,omitempty"`
		// allow userns creation in container
		Userns bool `json:"userns,omitempty"`
		// share host net namespace
		HostNet bool `json:"host_net,omitempty"`
		// share abstract unix socket scope
		HostAbstract bool `json:"host_abstract,omitempty"`
		// allow dangerous terminal I/O
		Tty bool `json:"tty,omitempty"`
		// allow multiarch
		Multiarch bool `json:"multiarch,omitempty"`

		// initial process environment variables
		Env map[string]string `json:"env"`
		// map target user uid to privileged user uid in the user namespace;
		// some programs fail to connect to dbus session running as a different uid,
		// this option works around it by mapping priv-side caller uid in container
		MapRealUID bool `json:"map_real_uid"`

		// pass through all devices
		Device bool `json:"device,omitempty"`
		// container mount points;
		// if the first element targets /, it is inserted early and excluded from path hiding
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
