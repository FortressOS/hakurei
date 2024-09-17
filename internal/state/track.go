package state

import (
	"encoding/gob"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"

	"git.ophivana.moe/cat/fortify/internal"
)

// SaveProcess called after process start, before wait
func SaveProcess(uid string, cmd *exec.Cmd, runDirPath string, command []string, enablements internal.Enablements) (string, error) {
	statePath := path.Join(runDirPath, uid, strconv.Itoa(cmd.Process.Pid))
	state := launcherState{
		PID:        cmd.Process.Pid,
		Launcher:   cmd.Path,
		Argv:       cmd.Args,
		Command:    command,
		Capability: enablements,
	}

	if err := os.Mkdir(path.Join(runDirPath, uid), 0700); err != nil && !errors.Is(err, fs.ErrExist) {
		return statePath, err
	}

	if f, err := os.OpenFile(statePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		return statePath, err
	} else {
		defer func() {
			if f.Close() != nil {
				// unreachable
				panic("state file closed prematurely")
			}
		}()
		return statePath, gob.NewEncoder(f).Encode(state)
	}
}
