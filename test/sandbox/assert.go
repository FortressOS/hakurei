//go:build testtool

/*
Package sandbox provides utilities for checking sandbox outcome.

This package must never be used outside integration tests, there is a much better native implementation of mountinfo
in the public sandbox/vfs package. Files in this package are excluded by the build system to prevent accidental misuse.
*/
package sandbox

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"os"
	"syscall"
)

var (
	assert     = log.New(os.Stderr, "sandbox: ", 0)
	printfFunc = assert.Printf
	fatalfFunc = assert.Fatalf
)

func printf(format string, v ...any) { printfFunc(format, v...) }
func fatalf(format string, v ...any) { fatalfFunc(format, v...) }

type TestCase struct {
	Env     []string          `json:"env"`
	FS      *FS               `json:"fs"`
	Mount   []*MountinfoEntry `json:"mount"`
	Seccomp bool              `json:"seccomp"`

	TrySocket      string `json:"try_socket,omitempty"`
	SocketAbstract bool   `json:"socket_abstract,omitempty"`
	SocketPathname bool   `json:"socket_pathname,omitempty"`
}

type T struct {
	FS fs.FS

	MountsPath string
}

func (t *T) MustCheckFile(wantFilePath string) {
	var want *TestCase
	mustDecode(wantFilePath, &want)
	t.MustCheck(want)
}

func (t *T) MustCheck(want *TestCase) {
	if want.Env != nil {
		var (
			fail bool
			i    int
			got  string
		)
		for i, got = range os.Environ() {
			if i == len(want.Env) {
				fatalf("got more than %d environment variables", len(want.Env))
			}
			if got != want.Env[i] {
				fail = true
				printf("[FAIL] %s", got)
			} else {
				printf("[ OK ] %s", got)
			}
		}

		i++
		if i != len(want.Env) {
			fatalf("got %d environment variables, want %d", i, len(want.Env))
		}

		if fail {
			fatalf("[FAIL] some environment variables did not match")
		}
	} else {
		printf("[SKIP] skipping environ check")
	}

	if want.FS != nil && t.FS != nil {
		if err := want.FS.Compare(".", t.FS); err != nil {
			fatalf("%v", err)
		}
	} else {
		printf("[SKIP] skipping fs check")
	}

	if want.Mount != nil {
		var fail bool
		m := mustParseMountinfo(t.MountsPath)
		i := 0
		for ent := range m.Entries() {
			if i == len(want.Mount) {
				fatalf("got more than %d entries", i)
			}
			if !ent.EqualWithIgnore(want.Mount[i], "//ignore") {
				fail = true
				printf("[FAIL] %s", ent)
			} else {
				printf("[ OK ] %s", ent)
			}

			i++
		}
		if err := m.Err(); err != nil {
			fatalf("%v", err)
		}

		if i != len(want.Mount) {
			fatalf("got %d entries, want %d", i, len(want.Mount))
		}

		if fail {
			fatalf("[FAIL] some mount points did not match")
		}
	} else {
		printf("[SKIP] skipping mounts check")
	}

	if want.Seccomp {
		if trySyscalls() != nil {
			os.Exit(1)
		}
	} else {
		printf("[SKIP] skipping seccomp check")
	}

	if want.TrySocket != "" {
		abstractConn, abstractErr := net.Dial("unix", "@"+want.TrySocket)
		pathnameConn, pathnameErr := net.Dial("unix", want.TrySocket)
		ok := true

		if abstractErr == nil {
			if err := abstractConn.Close(); err != nil {
				ok = false
				log.Printf("Close: %v", err)
			}
		}
		if pathnameErr == nil {
			if err := pathnameConn.Close(); err != nil {
				ok = false
				log.Printf("Close: %v", err)
			}
		}

		abstractWantErr := error(syscall.EPERM)
		pathnameWantErr := error(syscall.ENOENT)
		if want.SocketAbstract {
			abstractWantErr = nil
		}
		if want.SocketPathname {
			pathnameWantErr = nil
		}

		if !errors.Is(abstractErr, abstractWantErr) {
			ok = false
			log.Printf("abstractErr: %v, want %v", abstractErr, abstractWantErr)
		}
		if !errors.Is(pathnameErr, pathnameWantErr) {
			ok = false
			log.Printf("pathnameErr: %v, want %v", pathnameErr, pathnameWantErr)
		}

		if !ok {
			os.Exit(1)
		}
	}
}

func MustCheckFilter(pid int, want string) {
	err := CheckFilter(pid, want)
	if err == nil {
		return
	}

	var perr *ptraceError
	if !errors.As(err, &perr) {
		fatalf("%s", err)
	}
	switch perr.op {
	case "PTRACE_ATTACH":
		fatalf("cannot attach to process %d: %v", pid, err)
	case "PTRACE_SECCOMP_GET_FILTER":
		if perr.errno == syscall.ENOENT {
			fatalf("seccomp filter not installed for process %d", pid)
		}
		fatalf("cannot get filter: %v", err)
	default:
		fatalf("cannot check filter: %v", err)
	}

	*(*int)(nil) = 0 // not reached
}

func CheckFilter(pid int, want string) error {
	if err := ptraceAttach(pid); err != nil {
		return err
	}
	defer func() {
		if err := ptraceDetach(pid); err != nil {
			printf("cannot detach from process %d: %v", pid, err)
		}
	}()

	h := sha512.New()

	if buf, err := getFilter[[8]byte](pid, 0); err != nil {
		return err
	} else {
		for _, b := range buf {
			h.Write(b[:])
		}
	}

	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		printf("[FAIL] %s", got)
		return syscall.ENOTRECOVERABLE
	} else {
		printf("[ OK ] %s", got)
		return nil
	}
}

func mustDecode(wantFilePath string, v any) {
	if f, err := os.Open(wantFilePath); err != nil {
		fatalf("cannot open %q: %v", wantFilePath, err)
	} else if err = json.NewDecoder(f).Decode(v); err != nil {
		fatalf("cannot decode %q: %v", wantFilePath, err)
	} else if err = f.Close(); err != nil {
		fatalf("cannot close %q: %v", wantFilePath, err)
	}
}

func mustParseMountinfo(name string) *Mountinfo {
	m := NewMountinfo(name)
	if err := m.Parse(); err != nil {
		fatalf("%v", err)
		panic("unreachable")
	}
	return m
}
