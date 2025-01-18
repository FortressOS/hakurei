package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

const compPoison = "INVALIDINVALIDINVALIDINVALIDINVALID"

var (
	Fmain = compPoison
)

func fortifyApp(config *fst.Config, beforeFail func()) {
	var (
		cmd *exec.Cmd
		st  io.WriteCloser
	)
	if p, ok := internal.Path(Fmain); !ok {
		beforeFail()
		fmsg.Fatal("invalid fortify path, this copy of fpkg is not compiled correctly")
		panic("unreachable")
	} else if r, w, err := os.Pipe(); err != nil {
		beforeFail()
		fmsg.Fatalf("cannot pipe: %v", err)
		panic("unreachable")
	} else {
		if fmsg.Verbose() {
			cmd = exec.Command(p, "-v", "app", "3")
		} else {
			cmd = exec.Command(p, "app", "3")
		}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.ExtraFiles = []*os.File{r}
		st = w
	}

	go func() {
		if err := json.NewEncoder(st).Encode(config); err != nil {
			beforeFail()
			fmsg.Fatalf("cannot send configuration: %v", err)
			panic("unreachable")
		}
	}()

	if err := cmd.Start(); err != nil {
		beforeFail()
		fmsg.Fatalf("cannot start fortify: %v", err)
		panic("unreachable")
	}
	if err := cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			beforeFail()
			fmsg.Exit(exitError.ExitCode())
			panic("unreachable")
		} else {
			beforeFail()
			fmsg.Fatalf("cannot wait: %v", err)
			panic("unreachable")
		}
	}
}
