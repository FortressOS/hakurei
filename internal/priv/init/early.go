package init0

import (
	"os"
	"path"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

// used by the parent process

// TryArgv0 calls [Main] if argv0 indicates the process is started from a file named "init".
func TryArgv0() {
	if len(os.Args) > 0 && path.Base(os.Args[0]) == "init" {
		Main()
		fmsg.Exit(0)
	}
}
