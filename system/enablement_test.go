package system_test

import (
	"testing"

	"git.gensokyo.uk/security/fortify/system"
)

func TestEnablementString(t *testing.T) {
	testCases := []struct {
		flags system.Enablement
		want  string
	}{
		{0, "(no enablements)"},
		{system.EWayland, "wayland"},
		{system.EX11, "x11"},
		{system.EDBus, "dbus"},
		{system.EPulse, "pulseaudio"},
		{system.EWayland | system.EX11, "wayland, x11"},
		{system.EWayland | system.EDBus, "wayland, dbus"},
		{system.EWayland | system.EPulse, "wayland, pulseaudio"},
		{system.EX11 | system.EDBus, "x11, dbus"},
		{system.EX11 | system.EPulse, "x11, pulseaudio"},
		{system.EDBus | system.EPulse, "dbus, pulseaudio"},
		{system.EWayland | system.EX11 | system.EDBus, "wayland, x11, dbus"},
		{system.EWayland | system.EX11 | system.EPulse, "wayland, x11, pulseaudio"},
		{system.EWayland | system.EDBus | system.EPulse, "wayland, dbus, pulseaudio"},
		{system.EX11 | system.EDBus | system.EPulse, "x11, dbus, pulseaudio"},
		{system.EWayland | system.EX11 | system.EDBus | system.EPulse, "wayland, x11, dbus, pulseaudio"},

		{1 << 5, "e20"},
		{1 << 6, "e40"},
		{1 << 7, "e80"},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.flags.String(); got != tc.want {
				t.Errorf("String: %q, want %q", got, tc.want)
			}
		})
	}
}
