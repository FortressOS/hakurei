package app_test

import (
	"io/fs"
	"reflect"
	"testing"

	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal"
	"git.ophivana.moe/security/fortify/internal/app"
	"git.ophivana.moe/security/fortify/internal/system"
)

type sealTestCase struct {
	name      string
	os        internal.System
	config    *app.Config
	id        app.ID
	wantSys   *system.I
	wantBwrap *bwrap.Config
}

func TestApp(t *testing.T) {
	testCases := append(testCasesNixos)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := app.NewWithID(tc.id, tc.os)

			if !t.Run("seal", func(t *testing.T) {
				if err := a.Seal(tc.config); err != nil {
					t.Errorf("Seal: error = %v", err)
				}
			}) {
				return
			}

			gotSys, gotBwrap := app.AppSystemBwrap(a)

			t.Run("compare sys", func(t *testing.T) {
				if !gotSys.Equal(tc.wantSys) {
					t.Errorf("Seal: sys = %#v, want %#v",
						gotSys, tc.wantSys)
				}
			})

			t.Run("compare bwrap", func(t *testing.T) {
				if !reflect.DeepEqual(gotBwrap, tc.wantBwrap) {
					t.Errorf("seal: bwrap = %#v, want %#v",
						gotBwrap, tc.wantBwrap)
				}
			})
		})
	}
}

func stubDirEntries(names ...string) (e []fs.DirEntry, err error) {
	e = make([]fs.DirEntry, len(names))
	for i, name := range names {
		e[i] = stubDirEntryPath(name)
	}
	return
}

type stubDirEntryPath string

func (p stubDirEntryPath) Name() string {
	return string(p)
}

func (p stubDirEntryPath) IsDir() bool {
	panic("attempted to call IsDir")
}

func (p stubDirEntryPath) Type() fs.FileMode {
	panic("attempted to call Type")
}

func (p stubDirEntryPath) Info() (fs.FileInfo, error) {
	panic("attempted to call Info")
}
