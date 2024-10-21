package internal

import (
	"os"
	"path"
	"strconv"
	"sync"

	"git.ophivana.moe/security/fortify/internal/fmsg"
)

// state that remain constant for the lifetime of the process
// fetched and cached here

const (
	xdgRuntimeDir = "XDG_RUNTIME_DIR"
)

// SystemConstants contains state from the operating system
type SystemConstants struct {
	// path to shared directory e.g. /tmp/fortify.%d
	SharePath string `json:"share_path"`
	// XDG_RUNTIME_DIR value e.g. /run/user/%d
	RuntimePath string `json:"runtime_path"`
	// application runtime directory e.g. /run/user/%d/fortify
	RunDirPath string `json:"run_dir_path"`
}

var (
	scVal  SystemConstants
	scOnce sync.Once
)

func copySC() {
	sc := SystemConstants{
		SharePath: path.Join(os.TempDir(), "fortify."+strconv.Itoa(os.Geteuid())),
	}

	fmsg.VPrintf("process share directory at %q", sc.SharePath)

	// runtimePath, runDirPath
	if r, ok := os.LookupEnv(xdgRuntimeDir); !ok {
		fmsg.Println("variable", xdgRuntimeDir, "unset")
		os.Exit(1)
	} else {
		sc.RuntimePath = r
		sc.RunDirPath = path.Join(sc.RuntimePath, "fortify")
		fmsg.VPrintf("XDG runtime directory at %q", sc.RunDirPath)
	}

	scVal = sc
}

// GetSC returns a populated SystemConstants value
func GetSC() SystemConstants {
	scOnce.Do(copySC)
	return scVal
}
