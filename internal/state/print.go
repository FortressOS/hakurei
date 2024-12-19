package state

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/system"
)

// MustPrintLauncherStateSimpleGlobal prints active launcher states of all simple stores
// in an implementation-specific way.
func MustPrintLauncherStateSimpleGlobal(w **tabwriter.Writer, runDir string) {
	now := time.Now().UTC()
	s := NewMulti(runDir)

	// read runtime directory to get all UIDs
	if aids, err := s.List(); err != nil {
		fmsg.Fatal("cannot list store:", err)
	} else {
		for _, aid := range aids {
			// print states belonging to this store
			s.(*multiStore).mustPrintLauncherState(aid, w, now)
		}
	}

	// mustPrintLauncherState causes store activity so store needs to be closed
	if err := s.Close(); err != nil {
		fmsg.Printf("cannot close store: %v", err)
	}
}

func (s *multiStore) mustPrintLauncherState(aid int, w **tabwriter.Writer, now time.Time) {
	var innerErr error

	if ok, err := s.Do(aid, func(c Cursor) {
		innerErr = func() error {
			// read launcher states
			states, err := c.Load()
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
					_, _ = fmt.Fprintf(*w, "\t%d\t%d\t%s\t%s\t%s\n",
						state.PID, aid, now.Sub(state.Time).Round(time.Second).String(), strings.TrimPrefix(ets.String(), ", "), cs)
				} else {
					// emit argv instead when verbose
					_, _ = fmt.Fprintf(*w, "\t%d\t%d\t%s\n",
						state.PID, aid, state.ID)
				}
			}

			return nil
		}()
	}); err != nil {
		fmsg.Printf("cannot perform action on app %d: %v", aid, err)
		if !ok {
			fmsg.Fatal("store faulted before printing")
		}
	}

	if innerErr != nil {
		fmsg.Fatalf("cannot print launcher state of app %d: %s", aid, innerErr)
	}
}
