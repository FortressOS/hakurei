package app

import (
	"git.ophivana.moe/cat/fortify/dbus"
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
	// reference to a system D-Bus proxy configuration,
	// nil value disables system bus proxy
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// reference to a session D-Bus proxy configuration,
	// nil value makes session bus proxy assume built-in defaults
	SessionBus *dbus.Config `json:"session_bus,omitempty"`

	// child capability enablements
	Enablements state.Enablements `json:"enablements"`
}
