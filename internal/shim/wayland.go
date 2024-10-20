package shim

import (
	"net"
	"sync"
)

// Wayland implements wayland mediation.
type Wayland struct {
	// wayland socket path
	Path string

	// wayland connection
	*net.UnixConn

	connErr error
	sync.Once
	// wait for wayland client to exit
	done chan struct{}
}

func (wl *Wayland) Close() error {
	wl.Do(func() {
		close(wl.done)
		wl.connErr = wl.UnixConn.Close()
	})

	return wl.connErr
}

func NewWayland() *Wayland {
	wl := new(Wayland)
	wl.done = make(chan struct{})
	return wl
}
