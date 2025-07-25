package container_test

import (
	"log"
	"os"
	"testing"

	"hakurei.app/command"
	"hakurei.app/container"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
)

const (
	envDoCheck = "HAKUREI_TEST_DO_CHECK"
)

var helperCommands []func(c command.Command)

func TestMain(m *testing.M) {
	container.TryArgv0(hlog.Output{}, hlog.Prepare, internal.InstallOutput)

	if os.Getenv(envDoCheck) == "1" {
		c := command.New(os.Stderr, log.Printf, "helper", func(args []string) error {
			log.SetFlags(0)
			log.SetPrefix("helper: ")
			return nil
		})
		for _, f := range helperCommands {
			f(c)
		}
		c.MustParse(os.Args[1:], func(err error) {
			if err != nil {
				log.Fatal(err.Error())
			}
		})
		return
	}

	os.Exit(m.Run())
}

func prepareHelper(c *container.Container) { c.Env = append(c.Env, envDoCheck+"=1") }
