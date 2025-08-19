package main

// this works around go:embed '..' limitation
//go:generate cp ../../LICENSE .

import (
	_ "embed"
	"errors"
	"log"
	"os"

	"hakurei.app/container"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
	"hakurei.app/internal/sys"
)

var (
	errSuccess = errors.New("success")

	//go:embed LICENSE
	license string
)

func init() { hlog.Prepare("hakurei") }

var std sys.State = new(sys.Std)

func main() {
	// early init path, skips root check and duplicate PR_SET_DUMPABLE
	container.TryArgv0(hlog.Output{}, hlog.Prepare, internal.InstallOutput)

	if err := container.SetPtracer(0); err != nil {
		hlog.Verbosef("cannot enable ptrace protection via Yama LSM: %v", err)
		// not fatal: this program runs as the privileged user
	}

	if err := container.SetDumpable(container.SUID_DUMP_DISABLE); err != nil {
		log.Printf("cannot set SUID_DUMP_DISABLE: %s", err)
		// not fatal: this program runs as the privileged user
	}

	if os.Geteuid() == 0 {
		log.Fatal("this program must not run as root")
	}

	buildCommand(os.Stderr).MustParse(os.Args[1:], func(err error) {
		hlog.Verbosef("command returned %v", err)
		if errors.Is(err, errSuccess) {
			hlog.BeforeExit()
			os.Exit(0)
		}
		// this catches faulty command handlers that fail to return before this point
	})
	log.Fatal("unreachable")
}
