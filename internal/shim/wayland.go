package shim

import (
	"fmt"
	"net"
	"sync"
	"syscall"

	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// Wayland implements wayland mediation.
type Wayland struct {
	// wayland socket path
	Path string

	// wayland connection
	conn *net.UnixConn

	connErr error
	sync.Once
	// wait for wayland client to exit
	done chan struct{}
}

func (wl *Wayland) WriteUnix(conn *net.UnixConn) error {
	// connect to host wayland socket
	if f, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: wl.Path, Net: "unix"}); err != nil {
		return fmsg.WrapErrorSuffix(err,
			fmt.Sprintf("cannot connect to wayland at %q:", wl.Path))
	} else {
		fmsg.VPrintf("connected to wayland at %q", wl.Path)
		wl.conn = f
	}

	// set up for passing wayland socket
	if rc, err := wl.conn.SyscallConn(); err != nil {
		return fmsg.WrapErrorSuffix(err, "cannot obtain raw wayland connection:")
	} else {
		ec := make(chan error)
		go func() {
			// pass wayland connection fd
			if err = rc.Control(func(fd uintptr) {
				if _, _, err = conn.WriteMsgUnix(nil, syscall.UnixRights(int(fd)), nil); err != nil {
					ec <- fmsg.WrapErrorSuffix(err, "cannot pass wayland connection to shim:")
					return
				}
				ec <- nil

				// block until shim exits
				<-wl.done
				fmsg.VPrintln("releasing wayland connection")
			}); err != nil {
				ec <- fmsg.WrapErrorSuffix(err, "cannot obtain wayland connection fd:")
				return
			}
		}()
		return <-ec
	}
}

func (wl *Wayland) Close() error {
	wl.Do(func() {
		close(wl.done)
		wl.connErr = wl.conn.Close()
	})

	return wl.connErr
}

func NewWayland() *Wayland {
	wl := new(Wayland)
	wl.done = make(chan struct{})
	return wl
}
