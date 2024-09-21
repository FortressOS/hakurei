package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

const (
	systemdCheckPath = "/run/systemd/system"
)

var SdBootedV = func() bool {
	if v, err := SdBooted(); err != nil {
		fmt.Println("warn: read systemd marker:", err)
		return false
	} else {
		return v
	}
}()

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
