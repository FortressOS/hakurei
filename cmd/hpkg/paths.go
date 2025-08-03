package main

import (
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync/atomic"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal/hlog"
)

var (
	dataHome string
)

func init() {
	// dataHome
	if p, ok := os.LookupEnv("HAKUREI_DATA_HOME"); ok {
		dataHome = p
	} else {
		dataHome = container.FHSVarLib + "hakurei/" + strconv.Itoa(os.Getuid())
	}
}

func lookPath(file string) string {
	if p, err := exec.LookPath(file); err != nil {
		log.Fatalf("%s: command not found", file)
		return ""
	} else {
		return p
	}
}

var beforeRunFail = new(atomic.Pointer[func()])

func mustRun(name string, arg ...string) {
	hlog.Verbosef("spawning process: %q %q", name, arg)
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
	baseDir string
	// ${baseDir}/app
	metaPath string
	// ${baseDir}/files
	homeDir string
	// ${baseDir}/cache
	cacheDir string
	// ${baseDir}/cache/nix
	nixPath string
}

func pathSetByApp(id string) *appPathSet {
	pathSet := new(appPathSet)
	pathSet.baseDir = path.Join(dataHome, id)
	pathSet.metaPath = path.Join(pathSet.baseDir, "app")
	pathSet.homeDir = path.Join(pathSet.baseDir, "files")
	pathSet.cacheDir = path.Join(pathSet.baseDir, "cache")
	pathSet.nixPath = path.Join(pathSet.cacheDir, "nix")
	return pathSet
}

func appendGPUFilesystem(config *hst.Config) {
	config.Container.Filesystem = append(config.Container.Filesystem, []*hst.FilesystemConfig{
		// flatpak commit 763a686d874dd668f0236f911de00b80766ffe79
		{Src: "/dev/dri", Device: true},
		// mali
		{Src: "/dev/mali", Device: true},
		{Src: "/dev/mali0", Device: true},
		{Src: "/dev/umplock", Device: true},
		// nvidia
		{Src: "/dev/nvidiactl", Device: true},
		{Src: "/dev/nvidia-modeset", Device: true},
		// nvidia OpenCL/CUDA
		{Src: "/dev/nvidia-uvm", Device: true},
		{Src: "/dev/nvidia-uvm-tools", Device: true},

		// flatpak commit d2dff2875bb3b7e2cd92d8204088d743fd07f3ff
		{Src: "/dev/nvidia0", Device: true}, {Src: "/dev/nvidia1", Device: true},
		{Src: "/dev/nvidia2", Device: true}, {Src: "/dev/nvidia3", Device: true},
		{Src: "/dev/nvidia4", Device: true}, {Src: "/dev/nvidia5", Device: true},
		{Src: "/dev/nvidia6", Device: true}, {Src: "/dev/nvidia7", Device: true},
		{Src: "/dev/nvidia8", Device: true}, {Src: "/dev/nvidia9", Device: true},
		{Src: "/dev/nvidia10", Device: true}, {Src: "/dev/nvidia11", Device: true},
		{Src: "/dev/nvidia12", Device: true}, {Src: "/dev/nvidia13", Device: true},
		{Src: "/dev/nvidia14", Device: true}, {Src: "/dev/nvidia15", Device: true},
		{Src: "/dev/nvidia16", Device: true}, {Src: "/dev/nvidia17", Device: true},
		{Src: "/dev/nvidia18", Device: true}, {Src: "/dev/nvidia19", Device: true},
	}...)
}
