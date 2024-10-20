package shim

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/verbose"
)

// called in the parent process

func ServeConfig(socket string, uid int, payload *Payload, wl string, done chan struct{}) (*net.UnixConn, error) {
	var ws *net.UnixConn
	if payload.WL {
		if f, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: wl, Net: "unix"}); err != nil {
			return nil, err
		} else {
			verbose.Println("connected to wayland at", wl)
			ws = f
		}
	}

	if c, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"}); err != nil {
		return nil, err
	} else {
		verbose.Println("configuring shim on socket", socket)
		if err = acl.UpdatePerm(socket, uid, acl.Read, acl.Write, acl.Execute); err != nil {
			fmt.Println("fortify: cannot change permissions of shim setup socket:", err)
		}

		go func() {
			var conn *net.UnixConn
			if conn, err = c.AcceptUnix(); err != nil {
				fmt.Println("fortify: cannot accept connection from shim:", err)
			} else {
				if err = gob.NewEncoder(conn).Encode(*payload); err != nil {
					fmt.Println("fortify: cannot stream shim payload:", err)
					_ = os.Remove(socket)
					return
				}

				if payload.WL {
					// get raw connection
					var rc syscall.RawConn
					if rc, err = ws.SyscallConn(); err != nil {
						fmt.Println("fortify: cannot obtain raw wayland connection:", err)
						return
					} else {
						go func() {
							// pass wayland socket fd
							if err = rc.Control(func(fd uintptr) {
								if _, _, err = conn.WriteMsgUnix(nil, syscall.UnixRights(int(fd)), nil); err != nil {
									fmt.Println("fortify: cannot pass wayland connection to shim:", err)
									return
								}
								_ = conn.Close()

								// block until shim exits
								<-done
								verbose.Println("releasing wayland connection")
							}); err != nil {
								fmt.Println("fortify: cannot obtain wayland connection fd:", err)
							}
						}()
					}
				} else {
					_ = conn.Close()
				}
			}
			if err = c.Close(); err != nil {
				fmt.Println("fortify: cannot close shim socket:", err)
			}
			if err = os.Remove(socket); err != nil && !errors.Is(err, os.ErrNotExist) {
				fmt.Println("fortify: cannot remove dangling shim socket:", err)
			}
		}()
		return ws, nil
	}
}
