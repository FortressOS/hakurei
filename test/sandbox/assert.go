package sandbox

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"slices"
)

var (
	assert     = log.New(os.Stderr, "sandbox: ", 0)
	printfFunc = assert.Printf
	fatalfFunc = assert.Fatalf
)

func printf(format string, v ...any) { printfFunc(format, v...) }
func fatalf(format string, v ...any) { fatalfFunc(format, v...) }

type TestCase struct {
	FS      *FS       `json:"fs"`
	Mount   []*Mntent `json:"mount"`
	Seccomp bool      `json:"seccomp"`
}

type T struct {
	FS fs.FS

	MountsPath, PMountsPath string
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

	if want.Mount != nil && t.PMountsPath != "" {
		pm := mustOpenMounts(t.PMountsPath)
		passthruMounts := slices.AppendSeq(make([]*Mntent, 0, 128), pm.Entries())
		if err := pm.Err(); err != nil {
			fatalf("cannot parse host mounts: %v", err)
		}

		for _, e := range want.Mount {
			if e.Opts == "host_passthrough" {
				for _, ent := range passthruMounts {
					if e.FSName == ent.FSName && e.Type == ent.Type {
						// special case for tmpfs bind mounts
						if e.FSName == "tmpfs" && e.Dir != ent.Dir {
							continue
						}

						e.Opts = ent.Opts
						goto out
					}
				}
				fatalf("host passthrough missing %q", e.FSName)
			out:
			}
		}

		f := mustOpenMounts(t.MountsPath)
		i := 0
		for e := range f.Entries() {
			if i == len(want.Mount) {
				fatalf("got more than %d entries", i)
			}
			if !e.Is(want.Mount[i]) {
				fatalf("entry %d\n got: %s\nwant: %s", i,
					e, want.Mount[i])
			}
			printf("[ OK ] %s", e)

			i++
		}
		if err := f.Err(); err != nil {
			fatalf("cannot parse mounts: %v", err)
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

func mustOpenMounts(name string) *MountsFile {
	if f, err := OpenMounts(name); err != nil {
		fatalf("cannot open mounts %q: %v", name, err)
		panic("unreachable")
	} else {
		return f
	}
}
