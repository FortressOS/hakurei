package state

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
	"strings"
	"text/tabwriter"

	"git.ophivana.moe/cat/fortify/internal/system"
)

// we unfortunately have to assume there are never races between processes
// this and launcher should eventually be replaced by a server process

var (
	stateActionEarly  bool
	statePath         string
	cleanupCandidate  []string
	xcbActionComplete bool
	enablements       *Enablements
)

type launcherState struct {
	PID        int
	Launcher   string
	Argv       []string
	Command    []string
	Capability Enablements
}

func init() {
	flag.BoolVar(&stateActionEarly, "state", false, "query state value of current active launchers")
}

func Early() {
	if !stateActionEarly {
		return
	}

	launchers, err := readLaunchers(u.Uid)
	if err != nil {
		fmt.Println("Error reading launchers:", err)
		os.Exit(1)
	}

	stdout := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
	if !system.V.Verbose {
		_, _ = fmt.Fprintln(stdout, "\tPID\tEnablements\tLauncher\tCommand")
	} else {
		_, _ = fmt.Fprintln(stdout, "\tPID\tArgv")
	}

	for _, state := range launchers {
		enablementsDescription := strings.Builder{}
		for i := Enablement(0); i < enableLength; i++ {
			if state.Capability.Has(i) {
				enablementsDescription.WriteString(", " + i.String())
			}
		}
		if enablementsDescription.Len() == 0 {
			enablementsDescription.WriteString("none")
		}

		if !system.V.Verbose {
			_, _ = fmt.Fprintf(stdout, "\t%d\t%s\t%s\t%s\n",
				state.PID, strings.TrimPrefix(enablementsDescription.String(), ", "), state.Launcher,
				state.Command)
		} else {
			_, _ = fmt.Fprintf(stdout, "\t%d\t%s\n",
				state.PID, state.Argv)
		}
	}
	if err = stdout.Flush(); err != nil {
		fmt.Println("warn: error formatting output:", err)
	}

	os.Exit(0)
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
