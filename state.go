package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
)

// we unfortunately have to assume there are never races between processes
// this and launcher should eventually be replaced by a server process

var (
	stateActionEarly  bool
	statePath         string
	cleanupCandidate  []string
	xcbActionComplete bool
)

type launcherState struct {
	PID      int
	Launcher string
	Argv     []string
	Command  []string
}

func init() {
	flag.BoolVar(&stateActionEarly, "state", false, "query state value of current active launchers")
}

func tryState() {
	if !stateActionEarly {
		return
	}

	launchers, err := readLaunchers()
	if err != nil {
		fmt.Println("Error reading launchers:", err)
		os.Exit(1)
	}

	fmt.Println("\tPID\tLauncher")
	for _, state := range launchers {
		fmt.Printf("\t%d\t%s\nCommand: %s\nArgv: %s\n", state.PID, state.Launcher, state.Command, state.Argv)
	}

	os.Exit(0)
}

func registerRevertPath(p string) {
	cleanupCandidate = append(cleanupCandidate, p)
}

// called after process start, before wait
func registerProcess(uid string, cmd *exec.Cmd) error {
	statePath = path.Join(runDir, uid, strconv.Itoa(cmd.Process.Pid))
	state := launcherState{
		PID:      cmd.Process.Pid,
		Launcher: cmd.Path,
		Argv:     cmd.Args,
		Command:  command,
	}

	if err := os.Mkdir(path.Join(runDir, uid), 0700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}

	if f, err := os.OpenFile(statePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		return err
	} else {
		defer func() {
			if f.Close() != nil {
				// unreachable
				panic("state file closed prematurely")
			}
		}()
		return gob.NewEncoder(f).Encode(state)
	}
}

func readLaunchers() ([]*launcherState, error) {
	var f *os.File
	var r []*launcherState
	launcherPrefix := path.Join(runDir, ego.Uid)

	if pl, err := os.ReadDir(launcherPrefix); err != nil {
		return nil, err
	} else {
		for _, e := range pl {
			if err = func() error {
				if f, err = os.Open(path.Join(launcherPrefix, e.Name())); err != nil {
					return err
				} else {
					defer func() {
						if f.Close() != nil {
							// unreachable
							panic("foreign state file closed prematurely")
						}
					}()

					var s launcherState
					r = append(r, &s)
					return gob.NewDecoder(f).Decode(&s)
				}
			}(); err != nil {
				return nil, err
			}
		}
	}

	return r, nil
}

func beforeExit() {
	if err := os.Remove(statePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Println("Error removing state file:", err)
	}

	if a, err := readLaunchers(); err != nil {
		fmt.Println("Error reading active launchers:", err)
		os.Exit(1)
	} else if len(a) > 0 {
		// other launchers are still active
		if verbose {
			fmt.Printf("Found %d active launchers, exiting without cleaning up\n", len(a))
		}
		return
	}

	if verbose {
		fmt.Println("No other launchers active, will clean up")
	}

	if xcbActionComplete {
		if verbose {
			fmt.Printf("X11: Removing XHost entry SI:localuser:%s\n", ego.Username)
		}
		if err := changeHosts(xcbHostModeDelete, xcbFamilyServerInterpreted, "localuser\x00"+ego.Username); err != nil {
			fmt.Println("Error removing XHost entry:", err)
		}
	}

	for _, candidate := range cleanupCandidate {
		if err := aclUpdatePerm(candidate, uid); err != nil {
			fmt.Printf("Error stripping ACL entry from '%s': %s\n", candidate, err)
		}
		if verbose {
			fmt.Printf("Stripped ACL entry for user '%s' from '%s'\n", ego.Username, candidate)
		}
	}
}

func fatal(msg ...any) {
	fmt.Println(msg...)
	beforeExit()
	os.Exit(1)
}
