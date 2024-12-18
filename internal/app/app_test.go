package app_test

import (
	"io/fs"
	"reflect"
	"testing"
	"time"

	"git.ophivana.moe/security/fortify/fipc"
	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal/app"
	"git.ophivana.moe/security/fortify/internal/linux"
	"git.ophivana.moe/security/fortify/internal/system"
)

type sealTestCase struct {
	name      string
	os        linux.System
	config    *fipc.Config
	id        app.ID
	wantSys   *system.I
	wantBwrap *bwrap.Config
}

func TestApp(t *testing.T) {
	testCases := append(testCasesPd, testCasesNixos...)

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

type stubFileInfoMode fs.FileMode

func (s stubFileInfoMode) Name() string {
	panic("attempted to call Name")
}

func (s stubFileInfoMode) Size() int64 {
	panic("attempted to call Size")
}

func (s stubFileInfoMode) Mode() fs.FileMode {
	return fs.FileMode(s)
}

func (s stubFileInfoMode) ModTime() time.Time {
	panic("attempted to call ModTime")
}

func (s stubFileInfoMode) IsDir() bool {
	panic("attempted to call IsDir")
}

func (s stubFileInfoMode) Sys() any {
	panic("attempted to call Sys")
}

type stubFileInfoIsDir bool

func (s stubFileInfoIsDir) Name() string {
	panic("attempted to call Name")
}

func (s stubFileInfoIsDir) Size() int64 {
	panic("attempted to call Size")
}

func (s stubFileInfoIsDir) Mode() fs.FileMode {
	panic("attempted to call Mode")
}

func (s stubFileInfoIsDir) ModTime() time.Time {
	panic("attempted to call ModTime")
}

func (s stubFileInfoIsDir) IsDir() bool {
	return bool(s)
}

func (s stubFileInfoIsDir) Sys() any {
	panic("attempted to call Sys")
}
