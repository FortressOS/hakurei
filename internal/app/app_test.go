package app_test

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"testing"
	"time"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal/app"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/hlog"
	"hakurei.app/internal/sys"
	"hakurei.app/system"
)

type sealTestCase struct {
	name          string
	os            sys.State
	config        *hst.Config
	id            state.ID
	wantSys       *system.I
	wantContainer *container.Params
}

func TestApp(t *testing.T) {
	testCases := append(testCasesPd, testCasesNixos...)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := app.NewWithID(t.Context(), tc.id, tc.os)
			var (
				gotSys       *system.I
				gotContainer *container.Params
			)
			if !t.Run("seal", func(t *testing.T) {
				if sa, err := a.Seal(tc.config); err != nil {
					hlog.PrintBaseError(err, "got generic error:")
					t.Errorf("Seal: error = %v", err)
					return
				} else {
					gotSys, gotContainer = app.AppIParams(a, sa)
				}
			}) {
				return
			}

			t.Run("compare sys", func(t *testing.T) {
				if !gotSys.Equal(tc.wantSys) {
					t.Errorf("Seal: sys = %#v, want %#v",
						gotSys, tc.wantSys)
				}
			})

			t.Run("compare params", func(t *testing.T) {
				if !reflect.DeepEqual(gotContainer, tc.wantContainer) {
					t.Errorf("seal: params =\n%s\n, want\n%s",
						mustMarshal(gotContainer), mustMarshal(tc.wantContainer))
				}
			})
		})
	}
}

func mustMarshal(v any) string {
	if b, err := json.Marshal(v); err != nil {
		panic(err.Error())
	} else {
		return string(b)
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

func (p stubDirEntryPath) Name() string               { return string(p) }
func (p stubDirEntryPath) IsDir() bool                { panic("attempted to call IsDir") }
func (p stubDirEntryPath) Type() fs.FileMode          { panic("attempted to call Type") }
func (p stubDirEntryPath) Info() (fs.FileInfo, error) { panic("attempted to call Info") }

type stubFileInfoMode fs.FileMode

func (s stubFileInfoMode) Name() string       { panic("attempted to call Name") }
func (s stubFileInfoMode) Size() int64        { panic("attempted to call Size") }
func (s stubFileInfoMode) Mode() fs.FileMode  { return fs.FileMode(s) }
func (s stubFileInfoMode) ModTime() time.Time { panic("attempted to call ModTime") }
func (s stubFileInfoMode) IsDir() bool        { panic("attempted to call IsDir") }
func (s stubFileInfoMode) Sys() any           { panic("attempted to call Sys") }

type stubFileInfoIsDir bool

func (s stubFileInfoIsDir) Name() string       { panic("attempted to call Name") }
func (s stubFileInfoIsDir) Size() int64        { panic("attempted to call Size") }
func (s stubFileInfoIsDir) Mode() fs.FileMode  { panic("attempted to call Mode") }
func (s stubFileInfoIsDir) ModTime() time.Time { panic("attempted to call ModTime") }
func (s stubFileInfoIsDir) IsDir() bool        { return bool(s) }
func (s stubFileInfoIsDir) Sys() any           { panic("attempted to call Sys") }
