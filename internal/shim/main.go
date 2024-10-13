package shim

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"syscall"

	"git.ophivana.moe/cat/fortify/helper"
	init0 "git.ophivana.moe/cat/fortify/internal/init"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

// everything beyond this point runs as target user
// proceed with caution!

func doShim(socket string) {
	// re-exec
	if len(os.Args) > 0 && os.Args[0] != "fortify" && path.IsAbs(os.Args[0]) {
		if err := syscall.Exec(os.Args[0], []string{"fortify", "shim"}, os.Environ()); err != nil {
			fmt.Println("fortify-shim: cannot re-exec self:", err)
			// continue anyway
		}
	}

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
			fmt.Println("fortify-shim: no command was specified and $SHELL was unset")
			os.Exit(1)
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
		fmt.Println("fortify-shim: cannot pipe:", err)
		os.Exit(1)
	} else {
		conf.SetEnv[init0.EnvInit] = strconv.Itoa(3 + len(extraFiles))
		extraFiles = append(extraFiles, r)

		verbose.Println("transmitting config to init")
		go func() {
			// stream config to pipe
			if err = gob.NewEncoder(w).Encode(&ic); err != nil {
				fmt.Println("fortify-shim: cannot transmit init config:", err)
				os.Exit(1)
			}
		}()
	}

	helper.BubblewrapName = payload.Exec[1] // resolved bwrap path by parent
	if b, err := helper.NewBwrap(conf, nil, payload.Exec[0], func(int, int) []string { return []string{"init"} }); err != nil {
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
			doShim(s)
			panic("unreachable")
		}
	}
}
