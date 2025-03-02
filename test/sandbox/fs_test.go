package sandbox_test

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"git.gensokyo.uk/security/fortify/test/sandbox"
)

var (
	fsPasswdSample = "u0_a20:x:65534:65534:Fortify:/var/lib/persist/module/fortify/u0/a20:/run/current-system/sw/bin/zsh"
	fsGroupSample  = "fortify:x:65534:"
)

func TestCompare(t *testing.T) {
	testCases := []struct {
		name string

		sample  fstest.MapFS
		want    *sandbox.FS
		wantOut string
		wantErr error
	}{
		{"skip", fstest.MapFS{}, &sandbox.FS{}, "[ OK ] s .\x00", nil},
		{"simple pass", fstest.MapFS{".fortify": {Mode: 0x800001ed}},
			&sandbox.FS{Dir: map[string]*sandbox.FS{".fortify": {Mode: 0x800001ed}}},
			"[ OK ] s .fortify\x00[ OK ] d .\x00", nil},
		{"bad length", fstest.MapFS{".fortify": {Mode: 0x800001ed}},
			&sandbox.FS{Dir: make(map[string]*sandbox.FS)},
			"[FAIL] d \".\": \".fortify/\"\x00", sandbox.ErrFSBadLength},
		{"top level bad mode", fstest.MapFS{".fortify": {Mode: 0x800001ed}},
			&sandbox.FS{Dir: map[string]*sandbox.FS{".fortify": {Mode: 0xdeadbeef}}},
			"[FAIL] m \".fortify\": 800001ed, want deadbeef\x00", sandbox.ErrFSBadMode},
		{"invalid entry condition", fstest.MapFS{"test": {Data: []byte{'0'}, Mode: 0644}},
			&sandbox.FS{Dir: map[string]*sandbox.FS{"test": {Dir: make(map[string]*sandbox.FS)}}},
			"[FAIL] d \".\": \"test\"\x00", sandbox.ErrFSInvalidEnt},
		{"nonexistent", fstest.MapFS{"test": {Data: []byte{'0'}, Mode: 0644}},
			&sandbox.FS{Dir: map[string]*sandbox.FS{".test": {}}},
			"[FAIL] d \".\": \"test\"\x00", fs.ErrNotExist},
		{"file", fstest.MapFS{"etc": {Mode: 0x800001c0},
			"etc/passwd": {Data: []byte(fsPasswdSample), Mode: 0644},
			"etc/group":  {Data: []byte(fsGroupSample), Mode: 0644},
		}, &sandbox.FS{Dir: map[string]*sandbox.FS{"etc": {Mode: 0x800001c0, Dir: map[string]*sandbox.FS{
			"passwd": {Mode: 0x1a4, Data: &fsPasswdSample},
			"group":  {Mode: 0x1a4, Data: &fsGroupSample},
		}}}}, "[ OK ] f etc/group\x00[ OK ] f etc/passwd\x00[ OK ] d etc\x00[ OK ] d .\x00", nil},
		{"file differ", fstest.MapFS{"etc": {Mode: 0x800001c0},
			"etc/passwd": {Data: []byte(fsPasswdSample), Mode: 0644},
			"etc/group":  {Data: []byte(fsGroupSample), Mode: 0644},
		}, &sandbox.FS{Dir: map[string]*sandbox.FS{"etc": {Mode: 0x800001c0, Dir: map[string]*sandbox.FS{
			"passwd": {Mode: 0x1a4, Data: &fsGroupSample},
			"group":  {Mode: 0x1a4, Data: &fsGroupSample},
		}}}}, "[ OK ] f etc/group\x00[FAIL] f etc/passwd\x00", sandbox.ErrFSBadData},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotOut := new(strings.Builder)
			oldPrint := sandbox.SwapPrint(func(format string, v ...any) { _, _ = fmt.Fprintf(gotOut, format+"\x00", v...) })
			t.Cleanup(func() { sandbox.SwapPrint(oldPrint) })

			err := tc.want.Compare(".", tc.sample)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Compare: error = %v; wantErr %v",
					err, tc.wantErr)
			}

			if gotOut.String() != tc.wantOut {
				t.Errorf("Compare: output %q; want %q",
					gotOut, tc.wantOut)
			}
		})
	}

	t.Run("assert", func(t *testing.T) {
		oldFatal := sandbox.SwapFatal(t.Fatalf)
		t.Cleanup(func() { sandbox.SwapFatal(oldFatal) })
		sandbox.MustAssertFS(make(fstest.MapFS), sandbox.MustWantFile(t, &sandbox.FS{Mode: 0xDEADBEEF}))
	})
}
