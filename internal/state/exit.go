package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"git.ophivana.moe/cat/fortify/internal/acl"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/xcb"
)

func Fatal(msg ...any) {
	fmt.Println(msg...)
	BeforeExit()
	os.Exit(1)
}

func BeforeExit() {
	if u == nil {
		fmt.Println("warn: beforeExit called before app init")
		return
	}

	if statePath == "" {
		if system.V.Verbose {
			fmt.Println("State path is unset")
		}
	} else {
		if err := os.Remove(statePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			fmt.Println("Error removing state file:", err)
		}
	}

	if d, err := readLaunchers(u.Uid); err != nil {
		fmt.Println("Error reading active launchers:", err)
		os.Exit(1)
	} else if len(d) > 0 {
		// other launchers are still active
		if system.V.Verbose {
			fmt.Printf("Found %d active launchers, exiting without cleaning up\n", len(d))
		}
		return
	}

	if system.V.Verbose {
		fmt.Println("No other launchers active, will clean up")
	}

	if xcbActionComplete {
		if system.V.Verbose {
			fmt.Printf("X11: Removing XHost entry SI:localuser:%s\n", u.Username)
		}
		if err := xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+u.Username); err != nil {
			fmt.Println("Error removing XHost entry:", err)
		}
	}

	for _, candidate := range cleanupCandidate {
		if err := acl.UpdatePerm(candidate, uid); err != nil {
			fmt.Printf("Error stripping ACL entry from '%s': %s\n", candidate, err)
		}
		if system.V.Verbose {
			fmt.Printf("Stripped ACL entry for user '%s' from '%s'\n", u.Username, candidate)
		}
	}
}
