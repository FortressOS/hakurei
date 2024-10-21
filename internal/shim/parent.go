package shim

import (
	"encoding/gob"
	"errors"
	"net"
	"os"
	"syscall"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// called in the parent process

func ServeConfig(socket string, abort chan error, uid int, payload *Payload, wl *Wayland) error {
	if payload.WL {
		if f, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: wl.Path, Net: "unix"}); err != nil {
			return err
		} else {
			fmsg.VPrintf("connected to wayland at %q", wl.Path)
			wl.UnixConn = f
		}
	}

	// setup success state accessed by abort
	var success bool

	if c, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"}); err != nil {
		return err
	} else {
		c.SetUnlinkOnClose(true)

		go func() {
			err1 := <-abort
			if !success {
				fmsg.VPrintln("aborting shim setup, reason:", err1)
				if err1 = c.Close(); err1 != nil {
					fmsg.Println("cannot abort shim setup:", err1)
				}
			}
			close(abort)
		}()

		fmsg.VPrintf("configuring shim on socket %q", socket)
		if err = acl.UpdatePerm(socket, uid, acl.Read, acl.Write, acl.Execute); err != nil {
			fmsg.Println("cannot change permissions of shim setup socket:", err)
		}

		go func() {
			var conn *net.UnixConn
			if conn, err = c.AcceptUnix(); err != nil {
				if errors.Is(err, net.ErrClosed) {
					fmsg.VPrintln("accept failed due to shim setup abort")
				} else {
					fmsg.Println("cannot accept connection from shim:", err)
				}
			} else {
				if err = gob.NewEncoder(conn).Encode(*payload); err != nil {
					fmsg.Println("cannot stream shim payload:", err)
					_ = os.Remove(socket)
					return
				}

				if payload.WL {
					// get raw connection
					var rc syscall.RawConn
					if rc, err = wl.SyscallConn(); err != nil {
						fmsg.Println("cannot obtain raw wayland connection:", err)
						return
					} else {
						go func() {
							// pass wayland socket fd
							if err = rc.Control(func(fd uintptr) {
								if _, _, err = conn.WriteMsgUnix(nil, syscall.UnixRights(int(fd)), nil); err != nil {
									fmsg.Println("cannot pass wayland connection to shim:", err)
									return
								}
								_ = conn.Close()

								// block until shim exits
								<-wl.done
								fmsg.VPrintln("releasing wayland connection")
							}); err != nil {
								fmsg.Println("cannot obtain wayland connection fd:", err)
							}
						}()
					}
				} else {
					_ = conn.Close()
				}
			}

			success = true
			if err = c.Close(); err != nil {
				if errors.Is(err, net.ErrClosed) {
					fmsg.VPrintln("close failed due to shim setup abort")
				} else {
					fmsg.Println("cannot close shim socket:", err)
				}
			}
		}()
		return nil
	}
}
