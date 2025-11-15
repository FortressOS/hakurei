package wayland

import (
	"os"
	"syscall"
)

func bindRawConn(done chan struct{}, rc syscall.RawConn, p, appID, instanceID string) ([2]int, error) {
	var closeFds [2]int
	if err := syscall.Pipe2(closeFds[0:], syscall.O_CLOEXEC); err != nil {
		return closeFds, err
	}

	setupDone := make(chan error, 1) // does not block with c.done

	go func() {
		if err := rc.Control(func(fd uintptr) {
			// allow the Bind method to return after setup
			setupDone <- bind(fd, p, appID, instanceID, uintptr(closeFds[1]))
			close(setupDone)

			// keep socket alive until done is requested
			<-done
		}); err != nil {
			setupDone <- err
		}

		// notify Close that rc.Control has returned
		close(done)
	}()

	// return write end of the pipe
	return closeFds, <-setupDone
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
