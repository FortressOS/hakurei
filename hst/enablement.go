package hst

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
)

// Enablement represents an optional host service to export to the target user.
type Enablement byte

const (
	EWayland Enablement = 1 << iota
	EX11
	EDBus
	EPulse

	EM
)

func (e Enablement) String() string {
	switch e {
	case 0:
		return "(no enablements)"
	case EWayland:
		return "wayland"
	case EX11:
		return "x11"
	case EDBus:
		return "dbus"
	case EPulse:
		return "pulseaudio"
	default:
		buf := new(strings.Builder)
		buf.Grow(32)

		for i := Enablement(1); i < EM; i <<= 1 {
			if e&i != 0 {
				buf.WriteString(", " + i.String())
			}
		}

		if buf.Len() == 0 {
			return fmt.Sprintf("e%x", byte(e))
		}
		return strings.TrimPrefix(buf.String(), ", ")
	}
}

// NewEnablements returns the address of [Enablement] as [Enablements].
func NewEnablements(e Enablement) *Enablements { return (*Enablements)(&e) }

// enablementsJSON is the [json] representation of the [Enablement] bit field.
type enablementsJSON struct {
	Wayland bool `json:"wayland,omitempty"`
	X11     bool `json:"x11,omitempty"`
	DBus    bool `json:"dbus,omitempty"`
	Pulse   bool `json:"pulse,omitempty"`
}

// Enablements is the [json] adapter for [Enablement].
type Enablements Enablement

// Unwrap returns the underlying [Enablement].
func (e *Enablements) Unwrap() Enablement {
	if e == nil {
		return 0
	}
	return Enablement(*e)
}

func (e *Enablements) MarshalJSON() ([]byte, error) {
	if e == nil {
		return nil, syscall.EINVAL
	}
	return json.Marshal(&enablementsJSON{
		Wayland: Enablement(*e)&EWayland != 0,
		X11:     Enablement(*e)&EX11 != 0,
		DBus:    Enablement(*e)&EDBus != 0,
		Pulse:   Enablement(*e)&EPulse != 0,
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

	var ve Enablement
	if v.Wayland {
		ve |= EWayland
	}
	if v.X11 {
		ve |= EX11
	}
	if v.DBus {
		ve |= EDBus
	}
	if v.Pulse {
		ve |= EPulse
	}
	*e = Enablements(ve)
	return nil
}
