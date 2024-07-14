package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
)

const (
	systemdCheckPath = "/run/systemd/system"
)

// https://www.freedesktop.org/software/systemd/man/sd_booted.html
func sdBooted() bool {
	_, err := os.Stat(systemdCheckPath)
	if err != nil {
		if verbose {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Println("System not booted through systemd")
			} else {
				fmt.Println("Error accessing", systemdCheckPath+":", err.Error())
			}
		}
		return false
	}
	return true
}

func which(file string) (string, bool) {
	path, err := exec.LookPath(file)
	return path, err == nil
}

func copyFile(dst, src string) error {
	srcD, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if srcD.Close() != nil {
			// unreachable
			panic("src file closed prematurely")
		}
	}()

	dstD, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if dstD.Close() != nil {
			// unreachable
			panic("dst file closed prematurely")
		}
	}()

	_, err = io.Copy(dstD, srcD)
	return err
}
