// Package wayland exposes the internal/wayland package.
//
// Deprecated: This package will be removed in 0.4.
package wayland

import (
	"errors"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	_ "unsafe" // for go:linkname

	"hakurei.app/internal/wayland"
)

// Conn represents a connection to the wayland display server.
//
// Deprecated: this interface is being replaced.
// Additionally, the package it belongs to will be removed in 0.4.
type Conn struct {
	conn *net.UnixConn

	done     chan struct{}
	doneOnce sync.Once

	mu sync.Mutex
}

// Attach connects Conn to a wayland socket.
func (c *Conn) Attach(p string) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return errors.New("socket already attached")
	}

	c.conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: p, Net: "unix"})
	return
}

// Close releases resources and closes the connection to the wayland compositor.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.done == nil {
		return errors.New("no socket bound")
	}

	c.doneOnce.Do(func() {
		c.done <- struct{}{}
		<-c.done
	})

	// closed by wayland
	runtime.SetFinalizer(c.conn, nil)
	return nil
}

//go:linkname bindRawConn hakurei.app/internal/wayland.bindRawConn
func bindRawConn(done chan struct{}, rc syscall.RawConn, p, appID, instanceID string) ([2]int, error)

// Bind binds the new socket to pathname.
func (c *Conn) Bind(pathname, appID, instanceID string) (*os.File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, errors.New("socket not attached")
	}
	if c.done != nil {
		return nil, errors.New("socket already bound")
	}

	if rc, err := c.conn.SyscallConn(); err != nil {
		// unreachable
		return nil, err
	} else {
		c.done = make(chan struct{})
		if closeFds, err := bindRawConn(c.done, rc, pathname, appID, instanceID); err != nil {
			return nil, err
		} else {
			return os.NewFile(uintptr(closeFds[1]), "close_fd"), nil
		}
	}
}

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
