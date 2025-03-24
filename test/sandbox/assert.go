/*
Package sandbox provides utilities for checking sandbox outcome.

This package must never be used outside integration tests, there is a much better native implementation of mountinfo
in the public sandbox/vfs package. Files in this package are excluded by the build system to prevent accidental misuse.
*/
package sandbox

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
)

var (
	assert     = log.New(os.Stderr, "sandbox: ", 0)
	printfFunc = assert.Printf
	fatalfFunc = assert.Fatalf
)

func printf(format string, v ...any) { printfFunc(format, v...) }
func fatalf(format string, v ...any) { fatalfFunc(format, v...) }

type TestCase struct {
	FS      *FS               `json:"fs"`
	Mount   []*MountinfoEntry `json:"mount"`
	Seccomp bool              `json:"seccomp"`
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
		if TrySyscalls() != nil {
			os.Exit(1)
		}
	} else {
		printf("[SKIP] skipping seccomp check")
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
