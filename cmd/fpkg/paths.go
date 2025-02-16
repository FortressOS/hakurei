package main

import (
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync/atomic"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

var (
	dataHome string
)

func init() {
	// dataHome
	if p, ok := os.LookupEnv("FORTIFY_DATA_HOME"); ok {
		dataHome = p
	} else {
		dataHome = "/var/lib/fortify/" + strconv.Itoa(os.Getuid())
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
	fmsg.Verbosef("spawning process: %q %q", name, arg)
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
