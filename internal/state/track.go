package state

import (
	"encoding/gob"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"

	"git.ophivana.moe/cat/fortify/internal/system"
)

// we unfortunately have to assume there are never races between processes
// this and launcher should eventually be replaced by a server process

var (
	statePath string
)

type launcherState struct {
	PID        int
	Launcher   string
	Argv       []string
	Command    []string
	Capability Enablements
}

// SaveProcess called after process start, before wait
func SaveProcess(uid string, cmd *exec.Cmd) error {
	statePath = path.Join(system.V.RunDir, uid, strconv.Itoa(cmd.Process.Pid))
	state := launcherState{
		PID:        cmd.Process.Pid,
		Launcher:   cmd.Path,
		Argv:       cmd.Args,
		Command:    command,
		Capability: *enablements,
	}

	if err := os.Mkdir(path.Join(system.V.RunDir, uid), 0700); err != nil && !errors.Is(err, fs.ErrExist) {
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

func readLaunchers(uid string) ([]*launcherState, error) {
	var f *os.File
	var r []*launcherState
	launcherPrefix := path.Join(system.V.RunDir, uid)

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
