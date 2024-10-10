package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/state"
)

var (
	stateActionEarly bool
)

func init() {
	flag.BoolVar(&stateActionEarly, "state", false, "print state information of active launchers")
}

// tryState is called after app initialisation
func tryState() {
	if stateActionEarly {
		var w *tabwriter.Writer
		state.MustPrintLauncherStateSimpleGlobal(&w, internal.GetSC().RunDirPath)
		if w != nil {
			if err := w.Flush(); err != nil {
				fmt.Println("warn: error formatting output:", err)
			}
		} else {
			fmt.Println("No information available")
		}

		os.Exit(0)
	}
}
