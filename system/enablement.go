package system

import (
	"fmt"
	"strings"
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
