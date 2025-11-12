//go:build testtool

package sandbox

import (
	"encoding/json"
	"os"
	"path"
	"testing"
)

type F func(format string, v ...any)

func SwapPrint(f F) (old F) { old = printfFunc; printfFunc = f; return }
func SwapFatal(f F) (old F) { old = fatalfFunc; fatalfFunc = f; return }

func MustWantFile(t *testing.T, v any) (wantFile string) {
	wantFile = path.Join(t.TempDir(), "want.json")
	if f, err := os.OpenFile(wantFile, os.O_CREATE|os.O_WRONLY, 0400); err != nil {
		t.Fatalf("cannot create %q: %v", wantFile, err)
	} else if err = json.NewEncoder(f).Encode(v); err != nil {
		t.Fatalf("cannot encode to %q: %v", wantFile, err)
	} else if err = f.Close(); err != nil {
		t.Fatalf("cannot close %q: %v", wantFile, err)
	}

	t.Cleanup(func() {
		if err := os.Remove(wantFile); err != nil {
			t.Fatalf("cannot remove %q: %v", wantFile, err)
		}
	})

	return
}
