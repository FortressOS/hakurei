package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"

	"git.gensokyo.uk/security/hakurei/hst"
	"git.gensokyo.uk/security/hakurei/internal"
	"git.gensokyo.uk/security/hakurei/internal/hlog"
)

var hakureiPath = internal.MustHakureiPath()

func mustRunApp(ctx context.Context, config *hst.Config, beforeFail func()) {
	var (
		cmd *exec.Cmd
		st  io.WriteCloser
	)

	if r, w, err := os.Pipe(); err != nil {
		beforeFail()
		log.Fatalf("cannot pipe: %v", err)
	} else {
		if hlog.Load() {
			cmd = exec.CommandContext(ctx, hakureiPath, "-v", "app", "3")
		} else {
			cmd = exec.CommandContext(ctx, hakureiPath, "app", "3")
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
		log.Fatalf("cannot start hakurei: %v", err)
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
