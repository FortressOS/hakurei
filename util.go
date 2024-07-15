package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
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

// Try various ways to discover the current user's PulseAudio authentication cookie.
func discoverPulseCookie() string {
	if p, ok := os.LookupEnv(pulseCookie); ok {
		return p
	}

	if p, ok := os.LookupEnv(home); ok {
		p = path.Join(p, ".pulse-cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				fatal("Error accessing PulseAudio cookie:", err)
				// unreachable
				return p
			}
		} else if !s.IsDir() {
			return p
		}
	}

	if p, ok := os.LookupEnv(xdgConfigHome); ok {
		p = path.Join(p, "pulse", "cookie")
		if s, err := os.Stat(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				fatal("Error accessing PulseAudio cookie:", err)
				// unreachable
				return p
			}
		} else if !s.IsDir() {
			return p
		}
	}

	fatal(fmt.Sprintf("Cannot locate PulseAudio cookie (tried $%s, $%s/pulse/cookie, $%s/.pulse-cookie)",
		pulseCookie, xdgConfigHome, home))
	return ""
}

func which(file string) (string, bool) {
	p, err := exec.LookPath(file)
	return p, err == nil
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
