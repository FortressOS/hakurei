package main

import (
	"flag"
	"fmt"
	"git.ophivana.moe/cat/fortify/internal/state"
	"os"
	"text/tabwriter"
)

var (
	stateActionEarly [2]bool
)

func init() {
	flag.BoolVar(&stateActionEarly[0], "state", false, "print state information of active launchers")
	flag.BoolVar(&stateActionEarly[1], "state-current", false, "print state information of active launchers for the specified user")
}

// tryState is called after app initialisation
func tryState() {
	var w *tabwriter.Writer

	switch {
	case stateActionEarly[0]:
		state.MustPrintLauncherStateGlobal(&w, a.RunDir())
	case stateActionEarly[1]:
		state.MustPrintLauncherState(&w, a.RunDir(), a.Uid)
	default:
		return
	}

	if w != nil {
		if err := w.Flush(); err != nil {
			fmt.Println("warn: error formatting output:", err)
		}
	} else {
		fmt.Println("No information available")
	}

	os.Exit(0)
}
