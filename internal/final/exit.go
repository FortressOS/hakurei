package final

import (
	"errors"
	"fmt"
	"git.ophivana.moe/cat/fortify/internal/acl"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
	"git.ophivana.moe/cat/fortify/internal/xcb"
	"io/fs"
	"os"
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
		verbose.Println("State path is unset")
	} else {
		if err := os.Remove(statePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			fmt.Println("Error removing state file:", err)
		}
	}

	if d, err := state.ReadLaunchers(runDirPath, u.Uid); err != nil {
		fmt.Println("Error reading active launchers:", err)
		os.Exit(1)
	} else if len(d) > 0 {
		// other launchers are still active
		verbose.Printf("Found %d active launchers, exiting without cleaning up\n", len(d))
		return
	}

	verbose.Println("No other launchers active, will clean up")

	if xcbActionComplete {
		verbose.Printf("X11: Removing XHost entry SI:localuser:%s\n", u.Username)
		if err := xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+u.Username); err != nil {
			fmt.Println("Error removing XHost entry:", err)
		}
	}

	for _, candidate := range cleanupCandidate {
		if err := acl.UpdatePerm(candidate, uid); err != nil {
			fmt.Printf("Error stripping ACL entry from '%s': %s\n", candidate, err)
		}
		verbose.Printf("Stripped ACL entry for user '%s' from '%s'\n", u.Username, candidate)
	}

	if dbusProxy != nil {
		verbose.Println("D-Bus proxy registered, cleaning up")

		if err := dbusProxy.Close(); err != nil {
			if errors.Is(err, os.ErrClosed) {
				verbose.Println("D-Bus proxy already closed")
			} else {
				fmt.Println("Error closing D-Bus proxy:", err)
			}
		}

		// wait for Proxy.Wait to return
		<-*dbusDone
	}
}
