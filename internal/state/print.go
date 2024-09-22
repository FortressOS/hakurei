package state

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

// MustPrintLauncherStateSimpleGlobal prints active launcher states of all simple stores
// in an implementation-specific way.
func MustPrintLauncherStateSimpleGlobal(w **tabwriter.Writer) {
	sc := internal.GetSC()
	now := time.Now().UTC()

	// read runtime directory to get all UIDs
	if dirs, err := os.ReadDir(sc.RunDirPath); err != nil {
		fmt.Println("cannot read runtime directory:", err)
		os.Exit(1)
	} else {
		for _, e := range dirs {
			// skip non-directories
			if !e.IsDir() {
				verbose.Println("skipped non-directory entry", e.Name())
				continue
			}

			// skip non-numerical names
			if _, err = strconv.Atoi(e.Name()); err != nil {
				verbose.Println("skipped non-uid entry", e.Name())
				continue
			}

			// obtain temporary store
			s := NewSimple(sc.RunDirPath, e.Name()).(*simpleStore)

			// print states belonging to this store
			s.mustPrintLauncherState(w, now)

			// mustPrintLauncherState causes store activity so store needs to be closed
			if err = s.Close(); err != nil {
				fmt.Printf("warn: error closing store for user %s: %s\n", e.Name(), err)
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
				if !verbose.Get() {
					_, _ = fmt.Fprintln(*w, "\tUID\tPID\tUptime\tEnablements\tLauncher\tCommand")
				} else {
					// argv is emitted in body when verbose
					_, _ = fmt.Fprintln(*w, "\tUID\tPID\tArgv")
				}
			}

			// print each state
			for _, state := range states {
				// skip nil states
				if state == nil {
					_, _ = fmt.Fprintln(*w, "\tnil state entry")
					continue
				}

				// build enablements string
				ets := strings.Builder{}
				// append enablement strings in order
				for i := Enablement(0); i < EnableLength; i++ {
					if state.Capability.Has(i) {
						ets.WriteString(", " + i.String())
					}
				}
				// prevent an empty string when
				if ets.Len() == 0 {
					ets.WriteString("(No enablements)")
				}

				if !verbose.Get() {
					_, _ = fmt.Fprintf(*w, "\t%s\t%d\t%s\t%s\t%s\t%s\n",
						s.path[len(s.path)-1], state.PID, now.Sub(state.Time).Round(time.Second).String(), strings.TrimPrefix(ets.String(), ", "), state.Launcher,
						state.Command)
				} else {
					// emit argv instead when verbose
					_, _ = fmt.Fprintf(*w, "\t%s\t%d\t%s\n",
						s.path[len(s.path)-1], state.PID, state.Argv)
				}
			}

			return nil
		}()
	}); err != nil {
		fmt.Printf("cannot perform action on store '%s': %s\n", path.Join(s.path...), err)
		if !ok {
			fmt.Println("warn: store faulted before printing")
			os.Exit(1)
		}
	}

	if innerErr != nil {
		fmt.Printf("cannot print launcher state for store '%s': %s\n", path.Join(s.path...), innerErr)
		os.Exit(1)
	}
}
