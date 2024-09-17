package util

import (
	"errors"
	"io/fs"
	"os"
)

const (
	systemdCheckPath = "/run/systemd/system"
)

// SdBooted implements https://www.freedesktop.org/software/systemd/man/sd_booted.html
func SdBooted() (bool, error) {
	_, err := os.Stat(systemdCheckPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return false, err
	}

	return true, nil
}
