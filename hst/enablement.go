package hst

import (
	"encoding/json"
	"syscall"

	"hakurei.app/system"
)

// NewEnablements returns the address of [system.Enablement] as [Enablements].
func NewEnablements(e system.Enablement) *Enablements { return (*Enablements)(&e) }

// enablementsJSON is the [json] representation of the [system.Enablement] bit field.
type enablementsJSON struct {
	Wayland bool `json:"wayland,omitempty"`
	X11     bool `json:"x11,omitempty"`
	DBus    bool `json:"dbus,omitempty"`
	Pulse   bool `json:"pulse,omitempty"`
}

// Enablements is the [json] adapter for [system.Enablement].
type Enablements system.Enablement

// Unwrap returns the underlying [system.Enablement].
func (e *Enablements) Unwrap() system.Enablement {
	if e == nil {
		return 0
	}
	return system.Enablement(*e)
}

func (e *Enablements) MarshalJSON() ([]byte, error) {
	if e == nil {
		return nil, syscall.EINVAL
	}
	return json.Marshal(&enablementsJSON{
		Wayland: system.Enablement(*e)&system.EWayland != 0,
		X11:     system.Enablement(*e)&system.EX11 != 0,
		DBus:    system.Enablement(*e)&system.EDBus != 0,
		Pulse:   system.Enablement(*e)&system.EPulse != 0,
	})
}

func (e *Enablements) UnmarshalJSON(data []byte) error {
	if e == nil {
		return syscall.EINVAL
	}

	v := new(enablementsJSON)
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	var ve system.Enablement
	if v.Wayland {
		ve |= system.EWayland
	}
	if v.X11 {
		ve |= system.EX11
	}
	if v.DBus {
		ve |= system.EDBus
	}
	if v.Pulse {
		ve |= system.EPulse
	}
	*e = Enablements(ve)
	return nil
}
