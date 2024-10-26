package shim

import (
	"encoding/gob"
	"errors"
	"flag"
	"net"
	"os"
	"path"
	"strconv"
	"syscall"

	"git.ophivana.moe/security/fortify/helper"
	"git.ophivana.moe/security/fortify/internal/fmsg"
	init0 "git.ophivana.moe/security/fortify/internal/init"
)

// everything beyond this point runs as target user
// proceed with caution!

func doShim(socket string) {
	fmsg.SetPrefix("shim")

	// re-exec
	if len(os.Args) > 0 && os.Args[0] != "fortify" && path.IsAbs(os.Args[0]) {
		if err := syscall.Exec(os.Args[0], []string{"fortify", "shim"}, os.Environ()); err != nil {
			fmsg.Println("cannot re-exec self:", err)
			// continue anyway
		}
	}

	// dial setup socket
	var conn *net.UnixConn
	if c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"}); err != nil {
		fmsg.Fatal("cannot dial setup socket:", err)
		panic("unreachable")
	} else {
		conn = c
	}

	// decode payload gob stream
	var payload Payload
	if err := gob.NewDecoder(conn).Decode(&payload); err != nil {
		fmsg.Fatal("cannot decode shim payload:", err)
	} else {
		// sharing stdout with parent
		// USE WITH CAUTION
		fmsg.SetVerbose(payload.Verbose)
	}

	if payload.Bwrap == nil {
		fmsg.Fatal("bwrap config not supplied")
	}

	// receive wayland fd over socket
	wfd := -1
	if payload.WL {
		if fd, err := receiveWLfd(conn); err != nil {
			fmsg.Fatal("cannot receive wayland fd:", err)
		} else {
			wfd = fd
		}
	}

	// close setup socket
	if err := conn.Close(); err != nil {
		fmsg.Println("cannot close setup socket:", err)
		// not fatal
	}

	var ic init0.Payload

	// resolve argv0
	ic.Argv = payload.Argv
	if len(ic.Argv) > 0 {
		// looked up from $PATH by parent
		ic.Argv0 = payload.Exec[2]
	} else {
		// no argv, look up shell instead
		var ok bool
		if ic.Argv0, ok = os.LookupEnv("SHELL"); !ok {
			fmsg.Fatal("no command was specified and $SHELL was unset")
		}

		ic.Argv = []string{ic.Argv0}
	}

	conf := payload.Bwrap

	var extraFiles []*os.File

	// pass wayland fd
	if wfd != -1 {
		if f := os.NewFile(uintptr(wfd), "wayland"); f != nil {
			ic.WL = 3 + len(extraFiles)
			extraFiles = append(extraFiles, f)
		}
	} else {
		ic.WL = -1
	}

	// share config pipe
	if r, w, err := os.Pipe(); err != nil {
		fmsg.Fatal("cannot pipe:", err)
	} else {
		conf.SetEnv[init0.EnvInit] = strconv.Itoa(3 + len(extraFiles))
		extraFiles = append(extraFiles, r)

		fmsg.VPrintln("transmitting config to init")
		go func() {
			// stream config to pipe
			if err = gob.NewEncoder(w).Encode(&ic); err != nil {
				fmsg.Fatal("cannot transmit init config:", err)
			}
		}()
	}

	helper.BubblewrapName = payload.Exec[1] // resolved bwrap path by parent
	if b, err := helper.NewBwrap(conf, nil, payload.Exec[0], func(int, int) []string { return []string{"init"} }); err != nil {
		fmsg.Fatal("malformed sandbox config:", err)
	} else {
		cmd := b.Unwrap()
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.ExtraFiles = extraFiles

		if fmsg.Verbose() {
			fmsg.VPrintln("bwrap args:", conf.Args())
		}

		// run and pass through exit code
		if err = b.Start(); err != nil {
			fmsg.Fatal("cannot start target process:", err)
		} else if err = b.Wait(); err != nil {
			fmsg.VPrintln("wait:", err)
		}
		if b.Unwrap().ProcessState != nil {
			fmsg.Exit(b.Unwrap().ProcessState.ExitCode())
		} else {
			fmsg.Exit(127)
		}
	}
}

func receiveWLfd(conn *net.UnixConn) (int, error) {
	oob := make([]byte, syscall.CmsgSpace(4)) // single fd

	if _, oobn, _, _, err := conn.ReadMsgUnix(nil, oob); err != nil {
		return -1, err
	} else if len(oob) != oobn {
		return -1, errors.New("invalid message length")
	}

	var msg syscall.SocketControlMessage
	if messages, err := syscall.ParseSocketControlMessage(oob); err != nil {
		return -1, err
	} else if len(messages) != 1 {
		return -1, errors.New("unexpected message count")
	} else {
		msg = messages[0]
	}

	if fds, err := syscall.ParseUnixRights(&msg); err != nil {
		return -1, err
	} else if len(fds) != 1 {
		return -1, errors.New("unexpected fd count")
	} else {
		return fds[0], nil
	}
}

// Try runs shim and stops execution if FORTIFY_SHIM is set.
func Try() {
	if args := flag.Args(); len(args) == 1 && args[0] == "shim" {
		if s, ok := os.LookupEnv(EnvShim); ok {
			doShim(s)
			panic("unreachable")
		}
	}
}
