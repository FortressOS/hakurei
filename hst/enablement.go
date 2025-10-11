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
	// EWayland exposes a wayland pathname socket via security-context-v1.
	EWayland Enablement = 1 << iota
	// EX11 adds the target user via X11 ChangeHosts and exposes the X11 pathname socket.
	EX11
	// EDBus enables the per-container xdg-dbus-proxy daemon.
	EDBus
	// EPulse copies the PulseAudio cookie to [hst.PrivateTmp] and exposes the PulseAudio socket.
	EPulse

	// EM is a noop.
	EM
)

// String returns a string representation of the flags set on [Enablement].
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

// Enablements is the [json] adapter for [Enablement].
type Enablements Enablement

// enablementsJSON is the [json] representation of [Enablements].
type enablementsJSON = struct {
	Wayland bool `json:"wayland,omitempty"`
	X11     bool `json:"x11,omitempty"`
	DBus    bool `json:"dbus,omitempty"`
	Pulse   bool `json:"pulse,omitempty"`
}

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
