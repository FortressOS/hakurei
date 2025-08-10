// Package hst exports shared types for invoking hakurei.
package hst

import (
	"hakurei.app/container"
	"hakurei.app/system"
	"hakurei.app/system/dbus"
)

const Tmp = "/.hakurei"

var AbsTmp = container.MustAbs(Tmp)

// Config is used to seal an app implementation.
type Config struct {
	// reverse-DNS style arbitrary identifier string from config;
	// passed to wayland security-context-v1 as application ID
	// and used as part of defaults in dbus session proxy
	ID string `json:"id"`

	// absolute path to executable file
	Path *container.Absolute `json:"path,omitempty"`
	// final args passed to container init
	Args []string `json:"args"`

	// system services to make available in the container
	Enablements system.Enablement `json:"enablements"`

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
	// absolute path to home directory in the init mount namespace
	Data *container.Absolute `json:"data"`
	// directory to enter and use as home in the container mount namespace, nil for Data
	Dir *container.Absolute `json:"dir,omitempty"`
	// extra acl ops, dispatches before container init
	ExtraPerms []*ExtraPermConfig `json:"extra_perms,omitempty"`

	// numerical application id, used for init user namespace credentials
	Identity int `json:"identity"`
	// list of supplementary groups inherited by container processes
	Groups []string `json:"groups"`

	// abstract container configuration baseline
	Container *ContainerConfig `json:"container"`
}

// ExtraPermConfig describes an acl update op.
type ExtraPermConfig struct {
	Ensure  bool                `json:"ensure,omitempty"`
	Path    *container.Absolute `json:"path"`
	Read    bool                `json:"r,omitempty"`
	Write   bool                `json:"w,omitempty"`
	Execute bool                `json:"x,omitempty"`
}

func (e *ExtraPermConfig) String() string {
	if e.Path == nil {
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
