package state

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"git.ophivana.moe/security/fortify/internal/fmsg"
	"git.ophivana.moe/security/fortify/internal/system"
)

// MustPrintLauncherStateSimpleGlobal prints active launcher states of all simple stores
// in an implementation-specific way.
func MustPrintLauncherStateSimpleGlobal(w **tabwriter.Writer, runDir string) {
	now := time.Now().UTC()

	// read runtime directory to get all UIDs
	if dirs, err := os.ReadDir(path.Join(runDir, "state")); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmsg.Fatal("cannot read runtime directory:", err)
	} else {
		for _, e := range dirs {
			// skip non-directories
			if !e.IsDir() {
				fmsg.VPrintf("skipped non-directory entry %q", e.Name())
				continue
			}

			// skip non-numerical names
			if _, err = strconv.Atoi(e.Name()); err != nil {
				fmsg.VPrintf("skipped non-uid entry %q", e.Name())
				continue
			}

			// obtain temporary store
			s := NewSimple(runDir, e.Name()).(*simpleStore)

			// print states belonging to this store
			s.mustPrintLauncherState(w, now)

			// mustPrintLauncherState causes store activity so store needs to be closed
			if err = s.Close(); err != nil {
				fmsg.Printf("cannot close store for user %q: %s", e.Name(), err)
			}
		}
	}
}

func (s *simpleStore) mustPrintLauncherState(w **tabwriter.Writer, now time.Time) {
	var innerErr error

	if ok, err := s.Do(func(b Backend) {
		innerErr = func() error {
			// read launcher states
			states, err := b.Load()
			if err != nil {
				return err
			}

			// initialise tabwriter if nil
			if *w == nil {
				*w = tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)

				// write header when initialising
				if !fmsg.Verbose() {
					_, _ = fmt.Fprintln(*w, "\tPID\tApp\tUptime\tEnablements\tCommand")
				} else {
					// argv is emitted in body when verbose
					_, _ = fmt.Fprintln(*w, "\tPID\tApp\tArgv")
				}
			}

			// print each state
			for _, state := range states {
				// skip nil states
				if state == nil {
					_, _ = fmt.Fprintln(*w, "\tnil state entry")
					continue
				}

				// build enablements and command string
				var (
					ets *strings.Builder
					cs  = "(No command information)"
				)

				// check if enablements are provided
				if state.Config != nil {
					ets = new(strings.Builder)
					// append enablement strings in order
					for i := system.Enablement(0); i < system.Enablement(system.ELen); i++ {
						if state.Config.Confinement.Enablements.Has(i) {
							ets.WriteString(", " + i.String())
						}
					}

					cs = fmt.Sprintf("%q", state.Config.Command)
				}
				if ets != nil {
					// prevent an empty string
					if ets.Len() == 0 {
						ets.WriteString("(No enablements)")
					}
				} else {
					ets = new(strings.Builder)
					ets.WriteString("(No confinement information)")
				}

				if !fmsg.Verbose() {
					_, _ = fmt.Fprintf(*w, "\t%d\t%s\t%s\t%s\t%s\n",
						state.PID, s.path[len(s.path)-1], now.Sub(state.Time).Round(time.Second).String(), strings.TrimPrefix(ets.String(), ", "), cs)
				} else {
					// emit argv instead when verbose
					_, _ = fmt.Fprintf(*w, "\t%d\t%s\t%s\n",
						state.PID, s.path[len(s.path)-1], state.ID)
				}
			}

			return nil
		}()
	}); err != nil {
		fmsg.Printf("cannot perform action on store %q: %s", path.Join(s.path...), err)
		if !ok {
			fmsg.Fatal("store faulted before printing")
		}
	}

	if innerErr != nil {
		fmsg.Fatalf("cannot print launcher state for store %q: %s", path.Join(s.path...), innerErr)
	}
}
