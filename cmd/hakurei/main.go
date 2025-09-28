package main

// this works around go:embed '..' limitation
//go:generate cp ../../LICENSE .

import (
	"context"
	_ "embed"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"hakurei.app/container"
)

var (
	errSuccess = errors.New("success")

	//go:embed LICENSE
	license string
)

// earlyHardeningErrs are errors collected while setting up early hardening feature.
type earlyHardeningErrs struct{ yamaLSM, dumpable error }

func main() {
	// early init path, skips root check and duplicate PR_SET_DUMPABLE
	container.TryArgv0(nil)

	log.SetPrefix("hakurei: ")
	log.SetFlags(0)
	msg := container.NewMsg(log.Default())

	early := earlyHardeningErrs{
		yamaLSM:  container.SetPtracer(0),
		dumpable: container.SetDumpable(container.SUID_DUMP_DISABLE),
	}

	if os.Geteuid() == 0 {
		log.Fatal("this program must not run as root")
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop() // unreachable

	buildCommand(ctx, msg, &early, os.Stderr).MustParse(os.Args[1:], func(err error) {
		msg.Verbosef("command returned %v", err)
		if errors.Is(err, errSuccess) {
			msg.BeforeExit()
			os.Exit(0)
		}
		// this catches faulty command handlers that fail to return before this point
	})
	log.Fatal("unreachable")
}
