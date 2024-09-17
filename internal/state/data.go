package state

import (
	"encoding/gob"
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/internal"
)

// we unfortunately have to assume there are never races between processes
// this and launcher should eventually be replaced by a server process

type launcherState struct {
	PID        int
	Launcher   string
	Argv       []string
	Command    []string
	Capability internal.Enablements
}

// ReadLaunchers reads all launcher state file entries for the requested user
// and if decode is true decodes these launchers as well.
func ReadLaunchers(runDirPath, uid string, decode bool) ([]*launcherState, error) {
	var f *os.File
	var r []*launcherState
	launcherPrefix := path.Join(runDirPath, uid)

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
					if decode {
						return gob.NewDecoder(f).Decode(&s)
					} else {
						return nil
					}
				}
			}(); err != nil {
				return nil, err
			}
		}
	}

	return r, nil
}
