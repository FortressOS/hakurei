package wayland

import (
	"errors"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
)

// Conn represents a connection to the wayland display server.
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
		return bindRawConn(c.done, rc, pathname, appID, instanceID)
	}
}

func bindRawConn(done chan struct{}, rc syscall.RawConn, p, appID, instanceID string) (*os.File, error) {
	var syncPipe [2]*os.File

	if r, w, err := os.Pipe(); err != nil {
		return nil, err
	} else {
		syncPipe[0] = r
		syncPipe[1] = w
	}

	setupDone := make(chan error, 1) // does not block with c.done

	go func() {
		if err := rc.Control(func(fd uintptr) {
			// prevent runtime from closing the read end of sync fd
			runtime.SetFinalizer(syncPipe[0], nil)

			// allow the Bind method to return after setup
			setupDone <- bind(fd, p, appID, instanceID, syncPipe[0].Fd())
			close(setupDone)

			// keep socket alive until done is requested
			<-done
			runtime.KeepAlive(syncPipe[1])
		}); err != nil {
			setupDone <- err
		}

		// notify Close that rc.Control has returned
		close(done)
	}()

	// return write end of the pipe
	return syncPipe[1], <-setupDone
}

func bind(fd uintptr, p, appID, instanceID string, syncFd uintptr) error {
	// ensure p is available
	if f, err := os.Create(p); err != nil {
		return err
	} else if err = f.Close(); err != nil {
		return err
	} else if err = os.Remove(p); err != nil {
		return err
	}

	return bindWaylandFd(p, fd, appID, instanceID, syncFd)
}
