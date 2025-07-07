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
	"os"
	"syscall"
	"time"
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
}

type T struct {
	FS fs.FS

	MountsPath string
}

func (t *T) MustCheckFile(wantFilePath, markerPath string) {
	var want *TestCase
	mustDecode(wantFilePath, &want)
	t.MustCheck(want)
	if _, err := os.Create(markerPath); err != nil {
		fatalf("cannot create success marker: %v", err)
	}
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
	{
	getFilter:
		buf, err := getFilter[[8]byte](pid, 0)
		/* this is not how ESRCH should be handled: the manpage advises the
		use of waitpid, however that is not applicable for attaching to an
		arbitrary process, and spawning target process here is not easily
		possible under the current testing framework;

		despite checking for /proc/pid/status indicating state t (tracing stop),
		it does not appear to be directly related to the internal state used to
		determine whether a process is ready to accept ptrace operations, it also
		introduces a TOCTOU that is irrelevant in the testing vm; this behaviour
		is kept anyway as it reduces the average iterations required here;

		since this code is only ever compiled into the test program, whatever
		implications this ugliness might have should not hurt anyone */
		if errors.Is(err, syscall.ESRCH) {
			time.Sleep(100 * time.Millisecond)
			goto getFilter
		}

		if err != nil {
			return err
		}

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
