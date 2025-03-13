package init0

import (
	"os"
	"path"

	"git.gensokyo.uk/security/fortify/internal"
)

// used by the parent process

// TryArgv0 calls [Main] if the last element of argv0 is "init0".
func TryArgv0() {
	if len(os.Args) > 0 && path.Base(os.Args[0]) == "init0" {
		Main()
		internal.Exit(0)
	}
}
