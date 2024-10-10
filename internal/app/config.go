package app

import (
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/helper/bwrap"
	"git.ophivana.moe/cat/fortify/internal/state"
)

// Config is used to seal an *App
type Config struct {
	// D-Bus application ID
	ID string `json:"id"`
	// username of the target user to switch to
	User string `json:"user"`
	// value passed through to the child process as its argv
	Command []string `json:"command"`
	// string representation of the child's launch method
	Method string `json:"method"`

	// child confinement configuration
	Confinement ConfinementConfig `json:"confinement"`
}

// ConfinementConfig defines fortified child's confinement
type ConfinementConfig struct {
	// bwrap sandbox confinement configuration
	Sandbox *SandboxConfig `json:"sandbox"`

	// reference to a system D-Bus proxy configuration,
	// nil value disables system bus proxy
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// reference to a session D-Bus proxy configuration,
	// nil value makes session bus proxy assume built-in defaults
	SessionBus *dbus.Config `json:"session_bus,omitempty"`

	// child capability enablements
	Enablements state.Enablements `json:"enablements"`
}

// SandboxConfig describes resources made available to the sandbox.
type SandboxConfig struct {
	// unix hostname within sandbox
	Hostname string `json:"hostname,omitempty"`
	// userns availability within sandbox
	UserNS bool `json:"userns,omitempty"`
	// share net namespace
	Net bool `json:"net,omitempty"`
	// do not run in new session
	NoNewSession bool `json:"no_new_session,omitempty"`
	// mediated access to wayland socket
	Wayland bool `json:"wayland,omitempty"`

	UID int `json:"uid,omitempty"`
	GID int `json:"gid,omitempty"`
	// final environment variables
	Env map[string]string `json:"env"`

	// paths made available within the sandbox
	Bind [][2]string `json:"bind"`
	// paths made available read-only within the sandbox
	ROBind [][2]string `json:"ro-bind"`
}

func (s *SandboxConfig) Bwrap() *bwrap.Config {
	if s == nil {
		return nil
	}

	conf := &bwrap.Config{
		Net:           s.Net,
		UserNS:        s.UserNS,
		Hostname:      s.Hostname,
		Clearenv:      true,
		SetEnv:        s.Env,
		Bind:          s.Bind,
		ROBind:        s.ROBind,
		Procfs:        []string{"/proc"},
		DevTmpfs:      []string{"/dev"},
		Mqueue:        []string{"/dev/mqueue"},
		NewSession:    !s.NoNewSession,
		DieWithParent: true,
	}
	if s.UID > 0 {
		conf.UID = &s.UID
	}
	if s.GID > 0 {
		conf.GID = &s.GID
	}

	return conf
}
