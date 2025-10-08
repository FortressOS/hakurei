package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"

	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/message"
)

const bash = "bash"

var (
	dataHome *check.Absolute
)

func init() {
	// dataHome
	if a, err := check.NewAbs(os.Getenv("HAKUREI_DATA_HOME")); err == nil {
		dataHome = a
	} else {
		dataHome = fhs.AbsVarLib.Append("hakurei/" + strconv.Itoa(os.Getuid()))
	}
}

var (
	pathBin = fhs.AbsRoot.Append("bin")

	pathNix           = check.MustAbs("/nix/")
	pathNixStore      = pathNix.Append("store/")
	pathCurrentSystem = fhs.AbsRun.Append("current-system")
	pathSwBin         = pathCurrentSystem.Append("sw/bin/")
	pathShell         = pathSwBin.Append(bash)

	pathData     = check.MustAbs("/data")
	pathDataData = pathData.Append("data")
)

func lookPath(file string) string {
	if p, err := exec.LookPath(file); err != nil {
		log.Fatalf("%s: command not found", file)
		return ""
	} else {
		return p
	}
}

var beforeRunFail = new(atomic.Pointer[func()])

func mustRun(msg message.Msg, name string, arg ...string) {
	msg.Verbosef("spawning process: %q %q", name, arg)
	cmd := exec.Command(name, arg...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if f := beforeRunFail.Swap(nil); f != nil {
			(*f)()
		}
		log.Fatalf("%s: %v", name, err)
	}
}

type appPathSet struct {
	// ${dataHome}/${id}
	baseDir *check.Absolute
	// ${baseDir}/app
	metaPath *check.Absolute
	// ${baseDir}/files
	homeDir *check.Absolute
	// ${baseDir}/cache
	cacheDir *check.Absolute
	// ${baseDir}/cache/nix
	nixPath *check.Absolute
}

func pathSetByApp(id string) *appPathSet {
	pathSet := new(appPathSet)
	pathSet.baseDir = dataHome.Append(id)
	pathSet.metaPath = pathSet.baseDir.Append("app")
	pathSet.homeDir = pathSet.baseDir.Append("files")
	pathSet.cacheDir = pathSet.baseDir.Append("cache")
	pathSet.nixPath = pathSet.cacheDir.Append("nix")
	return pathSet
}

func appendGPUFilesystem(config *hst.Config) {
	config.Container.Filesystem = append(config.Container.Filesystem, []hst.FilesystemConfigJSON{
		// flatpak commit 763a686d874dd668f0236f911de00b80766ffe79
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("dri"), Device: true, Optional: true}},
		// mali
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("mali"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("mali0"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("umplock"), Device: true, Optional: true}},
		// nvidia
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidiactl"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia-modeset"), Device: true, Optional: true}},
		// nvidia OpenCL/CUDA
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia-uvm"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia-uvm-tools"), Device: true, Optional: true}},

		// flatpak commit d2dff2875bb3b7e2cd92d8204088d743fd07f3ff
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia0"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia1"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia2"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia3"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia4"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia5"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia6"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia7"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia8"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia9"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia10"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia11"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia12"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia13"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia14"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia15"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia16"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia17"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia18"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: fhs.AbsDev.Append("nvidia19"), Device: true, Optional: true}},
	}...)
}
