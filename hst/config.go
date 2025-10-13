package hst

import (
	"errors"
	"strconv"

	"hakurei.app/container/check"
)

// Config configures an application container, implemented in internal/app.
type Config struct {
	// Reverse-DNS style configured arbitrary identifier string.
	// Passed to wayland security-context-v1 and used as part of defaults in dbus session proxy.
	ID string `json:"id"`

	// System services to make available in the container.
	Enablements *Enablements `json:"enablements,omitempty"`

	// Session D-Bus proxy configuration.
	// If set to nil, session bus proxy assume built-in defaults.
	SessionBus *BusConfig `json:"session_bus,omitempty"`
	// System D-Bus proxy configuration.
	// If set to nil, system bus proxy is disabled.
	SystemBus *BusConfig `json:"system_bus,omitempty"`
	// Direct access to wayland socket, no attempt is made to attach security-context-v1
	// and the bare socket is made available to the container.
	DirectWayland bool `json:"direct_wayland,omitempty"`

	// Extra acl updates to perform before setuid.
	ExtraPerms []ExtraPermConfig `json:"extra_perms,omitempty"`

	// Numerical application id, passed to hsu, used to derive init user namespace credentials.
	Identity int `json:"identity"`
	// Init user namespace supplementary groups inherited by all container processes.
	Groups []string `json:"groups"`

	// High level configuration applied to the underlying [container].
	Container *ContainerConfig `json:"container"`
}

var (
	// ErrConfigNull is returned by [Config.Validate] for an invalid configuration that contains a null value for any
	// field that must not be null.
	ErrConfigNull = errors.New("unexpected null in config")

	// ErrIdentityBounds is returned by [Config.Validate] for an out of bounds [Config.Identity] value.
	ErrIdentityBounds = errors.New("identity out of bounds")
)

// Validate checks [Config] and returns [AppError] if an invalid value is encountered.
func (config *Config) Validate() error {
	if config == nil {
		return &AppError{Step: "validate configuration", Err: ErrConfigNull,
			Msg: "invalid configuration"}
	}

	// this is checked again in hsu
	if config.Identity < IdentityMin || config.Identity > IdentityMax {
		return &AppError{Step: "validate configuration", Err: ErrIdentityBounds,
			Msg: "identity " + strconv.Itoa(config.Identity) + " out of range"}
	}

	if err := config.SessionBus.CheckInterfaces("session"); err != nil {
		return err
	}
	if err := config.SystemBus.CheckInterfaces("system"); err != nil {
		return err
	}

	if config.Container == nil {
		return &AppError{Step: "validate configuration", Err: ErrConfigNull,
			Msg: "configuration missing container state"}
	}
	if config.Container.Home == nil {
		return &AppError{Step: "validate configuration", Err: ErrConfigNull,
			Msg: "container configuration missing path to home directory"}
	}
	if config.Container.Shell == nil {
		return &AppError{Step: "validate configuration", Err: ErrConfigNull,
			Msg: "container configuration missing path to shell"}
	}
	if config.Container.Path == nil {
		return &AppError{Step: "validate configuration", Err: ErrConfigNull,
			Msg: "container configuration missing path to initial program"}
	}
	return nil
}

// ExtraPermConfig describes an acl update to perform before setuid.
type ExtraPermConfig struct {
	// Whether to create Path as a directory if it does not exist.
	Ensure bool `json:"ensure,omitempty"`
	// Pathname to act on.
	Path *check.Absolute `json:"path"`
	// Whether to set ACL_READ for the target user.
	Read bool `json:"r,omitempty"`
	// Whether to set ACL_WRITE for the target user.
	Write bool `json:"w,omitempty"`
	// Whether to set ACL_EXECUTE for the target user.
	Execute bool `json:"x,omitempty"`
}

// String returns a checked string representation of [ExtraPermConfig].
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
