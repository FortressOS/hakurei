// Package wayland exposes the internal/system/wayland package.
//
// Deprecated: This package will be removed in 0.4.
package wayland

import (
	_ "unsafe" // for go:linkname

	"hakurei.app/internal/system/wayland"
)

// Conn represents a connection to the wayland display server.
type Conn = wayland.Conn

const (
	// WaylandDisplay contains the name of the server socket
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1147)
	// which is concatenated with XDG_RUNTIME_DIR
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1171)
	// or used as-is if absolute
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1176).
	WaylandDisplay = wayland.Display

	// FallbackName is used as the wayland socket name if WAYLAND_DISPLAY is unset
	// (https://gitlab.freedesktop.org/wayland/wayland/-/blob/1.23.1/src/wayland-client.c#L1149).
	FallbackName = wayland.FallbackName
)
