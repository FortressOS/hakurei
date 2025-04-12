// Package fst exports shared fortify types.
package fst

import (
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/system"
)

const Tmp = "/.fortify"

// Config is used to seal an app
type Config struct {
	// reverse-DNS style arbitrary identifier string from config;
	// passed to wayland security-context-v1 as application ID
	// and used as part of defaults in dbus session proxy
	ID string `json:"id"`

	// absolute path to executable file
	Path string `json:"path,omitempty"`
	// final args passed to container init
	Args []string `json:"args"`

	Confinement ConfinementConfig `json:"confinement"`
}

// ConfinementConfig defines fortified child's confinement
type ConfinementConfig struct {
	// numerical application id, determines uid in the init namespace
	AppID int `json:"app_id"`
	// list of supplementary groups to inherit
	Groups []string `json:"groups"`
	// passwd username in container, defaults to passwd name of target uid or chronos
	Username string `json:"username,omitempty"`
	// home directory in container, empty for outer
	Inner string `json:"home_inner"`
	// home directory in init namespace
	Outer string `json:"home"`
	// absolute path to shell, empty for host shell
	Shell string `json:"shell,omitempty"`
	// abstract sandbox configuration
	Sandbox *SandboxConfig `json:"sandbox"`
	// extra acl ops, runs after everything else
	ExtraPerms []*ExtraPermConfig `json:"extra_perms,omitempty"`

	// reference to a system D-Bus proxy configuration,
	// nil value disables system bus proxy
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// reference to a session D-Bus proxy configuration,
	// nil value makes session bus proxy assume built-in defaults
	SessionBus *dbus.Config `json:"session_bus,omitempty"`

	// system resources to expose to the container
	Enablements system.Enablement `json:"enablements"`
}

type ExtraPermConfig struct {
	Ensure  bool   `json:"ensure,omitempty"`
	Path    string `json:"path"`
	Read    bool   `json:"r,omitempty"`
	Write   bool   `json:"w,omitempty"`
	Execute bool   `json:"x,omitempty"`
}

func (e *ExtraPermConfig) String() string {
	buf := make([]byte, 0, 5+len(e.Path))
	buf = append(buf, '-', '-', '-')
	if e.Ensure {
		buf = append(buf, '+')
	}
	buf = append(buf, ':')
	buf = append(buf, []byte(e.Path)...)
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
