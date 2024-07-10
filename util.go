package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
