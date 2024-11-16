package main

import (
	"encoding/gob"
	"errors"
	"net"
	"os"
	"path"
	"strconv"
	"syscall"

	init0 "git.ophivana.moe/security/fortify/cmd/finit/ipc"
	shim "git.ophivana.moe/security/fortify/cmd/fshim/ipc"
	"git.ophivana.moe/security/fortify/helper"
	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// everything beyond this point runs as unconstrained target user
// proceed with caution!

func main() {
	// sharing stdout with fortify
	// USE WITH CAUTION
	fmsg.SetPrefix("shim")

	// setting this prevents ptrace
	if err := internal.PR_SET_DUMPABLE__SUID_DUMP_DISABLE(); err != nil {
		fmsg.Fatalf("cannot set SUID_DUMP_DISABLE: %s", err)
		panic("unreachable")
	}

	// re-exec
	if len(os.Args) > 0 && (os.Args[0] != "fshim" || len(os.Args) != 1) && path.IsAbs(os.Args[0]) {
		if err := syscall.Exec(os.Args[0], []string{"fshim"}, os.Environ()); err != nil {
			fmsg.Println("cannot re-exec self:", err)
			// continue anyway
		}
	}

	// lookup socket path from environment
	var socketPath string
	if s, ok := os.LookupEnv(shim.Env); !ok {
		fmsg.Fatal("FORTIFY_SHIM not set")
		panic("unreachable")
	} else {
		socketPath = s
	}

	// check path to finit
	var finitPath string
	if p, ok := internal.Path(internal.Finit); !ok {
		fmsg.Fatal("invalid finit path, this copy of fshim is not compiled correctly")
	} else {
		finitPath = p
	}

	// dial setup socket
	var conn *net.UnixConn
	if c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"}); err != nil {
		fmsg.Fatal(err.Error())
		panic("unreachable")
	} else {
		conn = c
	}

	// decode payload gob stream
	var payload shim.Payload
	if err := gob.NewDecoder(conn).Decode(&payload); err != nil {
		fmsg.Fatalf("cannot decode shim payload: %v", err)
	} else {
		fmsg.SetVerbose(payload.Verbose)
	}

	if payload.Bwrap == nil {
		fmsg.Fatal("bwrap config not supplied")
	}

	// receive wayland fd over socket
	wfd := -1
	if payload.WL {
		if fd, err := receiveWLfd(conn); err != nil {
			fmsg.Fatalf("cannot receive wayland fd: %v", err)
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
		ic.Argv0 = payload.Exec[1]
	} else {
		// no argv, look up shell instead
		var ok bool
		if payload.Bwrap.SetEnv == nil {
			fmsg.Fatal("no command was specified and environment is unset")
		}
		if ic.Argv0, ok = payload.Bwrap.SetEnv["SHELL"]; !ok {
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
		fmsg.Fatalf("cannot pipe: %v", err)
	} else {
		conf.SetEnv[init0.Env] = strconv.Itoa(3 + len(extraFiles))
		extraFiles = append(extraFiles, r)

		fmsg.VPrintln("transmitting config to init")
		go func() {
			// stream config to pipe
			if err = gob.NewEncoder(w).Encode(&ic); err != nil {
				fmsg.Fatalf("cannot transmit init config: %v", err)
			}
		}()
	}

	helper.BubblewrapName = payload.Exec[0] // resolved bwrap path by parent
	if b, err := helper.NewBwrap(conf, nil, finitPath,
		func(int, int) []string { return make([]string, 0) }); err != nil {
		fmsg.Fatalf("malformed sandbox config: %v", err)
	} else {
		cmd := b.Unwrap()
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.ExtraFiles = extraFiles

		if fmsg.Verbose() {
			fmsg.VPrintln("bwrap args:", conf.Args())
		}

		// run and pass through exit code
		if err = b.Start(); err != nil {
			fmsg.Fatalf("cannot start target process: %v", err)
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
