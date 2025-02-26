package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func fortifyApp(config *fst.Config, beforeFail func()) {
	var (
		cmd *exec.Cmd
		st  io.WriteCloser
	)
	if p, ok := internal.Path(internal.Fortify); !ok {
		beforeFail()
		log.Fatal("invalid fortify path, this copy of fpkg is not compiled correctly")
	} else if r, w, err := os.Pipe(); err != nil {
		beforeFail()
		log.Fatalf("cannot pipe: %v", err)
	} else {
		if fmsg.Load() {
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
			log.Fatalf("cannot send configuration: %v", err)
		}
	}()

	if err := cmd.Start(); err != nil {
		beforeFail()
		log.Fatalf("cannot start fortify: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			beforeFail()
			internal.Exit(exitError.ExitCode())
		} else {
			beforeFail()
			log.Fatalf("cannot wait: %v", err)
		}
	}
}
