package shim

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"

	"git.ophivana.moe/cat/fortify/helper"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

// everything beyond this point runs as target user
// proceed with caution!

func shim(socket string) {
	verbose.Prefix = "fortify-shim:"

	// dial setup socket
	var conn *net.UnixConn
	if c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"}); err != nil {
		fmt.Println("fortify-shim: cannot dial setup socket:", err)
		os.Exit(1)
	} else {
		conn = c
	}

	// decode payload gob stream
	var payload Payload
	if err := gob.NewDecoder(conn).Decode(&payload); err != nil {
		fmt.Println("fortify-shim: cannot decode shim payload:", err)
		os.Exit(1)
	} else {
		// sharing stdout with parent
		// USE WITH CAUTION
		verbose.Set(payload.Verbose)
	}

	if payload.Bwrap == nil {
		fmt.Println("fortify-shim: bwrap config not supplied")
		os.Exit(1)
	}

	// receive wayland fd over socket
	wfd := -1
	if payload.WL {
		if fd, err := receiveWLfd(conn); err != nil {
			fmt.Println("fortify-shim: cannot receive wayland fd:", err)
			os.Exit(1)
		} else {
			wfd = fd
		}
	}

	// close setup socket
	if err := conn.Close(); err != nil {
		fmt.Println("fortify-shim: cannot close setup socket:", err)
		// not fatal
	}

	// resolve argv0
	var (
		argv0 string
		argv  = payload.Argv
	)
	if len(argv) > 0 {
		// looked up from $PATH by parent
		argv0 = payload.Exec[1]
	} else {
		// no argv, look up shell instead
		var ok bool
		if argv0, ok = os.LookupEnv("SHELL"); !ok {
			fmt.Println("fortify-shim: no command was specified and $SHELL was unset")
			os.Exit(1)
		}

		argv = []string{argv0}
	}

	_ = conn.Close()

	conf := payload.Bwrap

	var extraFiles []*os.File

	// pass wayland fd
	if wfd != -1 {
		if f := os.NewFile(uintptr(wfd), "wayland"); f != nil {
			conf.SetEnv["WAYLAND_SOCKET"] = strconv.Itoa(3 + len(extraFiles))
			extraFiles = append(extraFiles, f)
		}
	}

	helper.BubblewrapName = payload.Exec[0] // resolved bwrap path by parent
	if b, err := helper.NewBwrap(conf, nil, argv0, func(_, _ int) []string { return argv[1:] }); err != nil {
		fmt.Println("fortify-shim: malformed sandbox config:", err)
		os.Exit(1)
	} else {
		cmd := b.Unwrap()
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.ExtraFiles = extraFiles

		if verbose.Get() {
			verbose.Println("bwrap args:", conf.Args())
		}

		// run and pass through exit code
		if err = b.Start(); err != nil {
			fmt.Println("fortify-shim: cannot start target process:", err)
			os.Exit(1)
		} else if err = b.Wait(); err != nil {
			verbose.Println("wait:", err)
		}
		if b.Unwrap().ProcessState != nil {
			os.Exit(b.Unwrap().ProcessState.ExitCode())
		} else {
			os.Exit(127)
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
			shim(s)
			panic("unreachable")
		}
	}
}
