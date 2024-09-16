package state

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"git.ophivana.moe/cat/fortify/internal/verbose"
)

func MustPrintLauncherStateGlobal(w **tabwriter.Writer, runDirPath string) {
	if dirs, err := os.ReadDir(runDirPath); err != nil {
		fmt.Println("Error reading runtime directory:", err)
	} else {
		for _, e := range dirs {
			if !e.IsDir() {
				verbose.Println("Skipped non-directory entry", e.Name())
				continue
			}

			if _, err = strconv.Atoi(e.Name()); err != nil {
				verbose.Println("Skipped non-uid entry", e.Name())
				continue
			}

			MustPrintLauncherState(w, runDirPath, e.Name())
		}
	}
}

func MustPrintLauncherState(w **tabwriter.Writer, runDirPath, uid string) {
	launchers, err := ReadLaunchers(runDirPath, uid)
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
