package state

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

var (
	stateActionEarly  bool
	stateActionEarlyC bool
)

func init() {
	flag.BoolVar(&stateActionEarly, "state", false, "print state information of active launchers")
	flag.BoolVar(&stateActionEarlyC, "state-current", false, "print state information of active launchers for the specified user")
}

func Early() {
	var w *tabwriter.Writer

	switch {
	case stateActionEarly:
		if runDir, err := os.ReadDir(system.V.RunDir); err != nil {
			fmt.Println("Error reading runtime directory:", err)
		} else {
			for _, e := range runDir {
				if !e.IsDir() {
					verbose.Println("Skipped non-directory entry", e.Name())
					continue
				}

				if _, err = strconv.Atoi(e.Name()); err != nil {
					verbose.Println("Skipped non-uid entry", e.Name())
					continue
				}

				printLauncherState(e.Name(), &w)
			}
		}
	case stateActionEarlyC:
		printLauncherState(u.Uid, &w)
	default:
		return
	}

	if w != nil {
		if err := w.Flush(); err != nil {
			fmt.Println("warn: error formatting output:", err)
		}
	} else {
		fmt.Println("No information available.")
	}

	os.Exit(0)
}

func printLauncherState(uid string, w **tabwriter.Writer) {
	launchers, err := readLaunchers(uid)
	if err != nil {
		fmt.Println("Error reading launchers:", err)
		os.Exit(1)
	}

	if *w == nil {
		*w = tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)

		if !verbose.Get() {
			_, _ = fmt.Fprintln(*w, "\tUID\tPID\tEnablements\tLauncher\tCommand")
		} else {
			_, _ = fmt.Fprintln(*w, "\tUID\tPID\tArgv")
		}
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

		if !verbose.Get() {
			_, _ = fmt.Fprintf(*w, "\t%s\t%d\t%s\t%s\t%s\n",
				uid, state.PID, strings.TrimPrefix(enablementsDescription.String(), ", "), state.Launcher,
				state.Command)
		} else {
			_, _ = fmt.Fprintf(*w, "\t%s\t%d\t%s\n",
				uid, state.PID, state.Argv)
		}
	}
}
