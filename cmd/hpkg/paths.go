package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"

	"hakurei.app/container"
	"hakurei.app/hst"
)

const bash = "bash"

var (
	dataHome *container.Absolute
)

func init() {
	// dataHome
	if a, err := container.NewAbs(os.Getenv("HAKUREI_DATA_HOME")); err == nil {
		dataHome = a
	} else {
		dataHome = container.AbsFHSVarLib.Append("hakurei/" + strconv.Itoa(os.Getuid()))
	}
}

var (
	pathBin = container.AbsFHSRoot.Append("bin")

	pathNix           = container.MustAbs("/nix/")
	pathNixStore      = pathNix.Append("store/")
	pathCurrentSystem = container.AbsFHSRun.Append("current-system")
	pathSwBin         = pathCurrentSystem.Append("sw/bin/")
	pathShell         = pathSwBin.Append(bash)

	pathData     = container.MustAbs("/data")
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

func mustRun(msg container.Msg, name string, arg ...string) {
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
	baseDir *container.Absolute
	// ${baseDir}/app
	metaPath *container.Absolute
	// ${baseDir}/files
	homeDir *container.Absolute
	// ${baseDir}/cache
	cacheDir *container.Absolute
	// ${baseDir}/cache/nix
	nixPath *container.Absolute
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
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("dri"), Device: true, Optional: true}},
		// mali
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("mali"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("mali0"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("umplock"), Device: true, Optional: true}},
		// nvidia
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidiactl"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia-modeset"), Device: true, Optional: true}},
		// nvidia OpenCL/CUDA
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia-uvm"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia-uvm-tools"), Device: true, Optional: true}},

		// flatpak commit d2dff2875bb3b7e2cd92d8204088d743fd07f3ff
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia0"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia1"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia2"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia3"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia4"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia5"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia6"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia7"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia8"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia9"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia10"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia11"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia12"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia13"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia14"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia15"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia16"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia17"), Device: true, Optional: true}},
		{FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia18"), Device: true, Optional: true}}, {FilesystemConfig: &hst.FSBind{Source: container.AbsFHSDev.Append("nvidia19"), Device: true, Optional: true}},
	}...)
}
