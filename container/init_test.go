package container_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"hakurei.app/command"
	"hakurei.app/container"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
	"hakurei.app/ldd"
)

const (
	envDoCheck = "HAKUREI_TEST_DO_CHECK"

	helperDefaultTimeout = 5 * time.Second
	helperInnerPath      = "/usr/bin/helper"
)

var (
	absHelperInnerPath = container.MustAbs(helperInnerPath)
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

func helperNewContainerLibPaths(ctx context.Context, libPaths *[]*container.Absolute, args ...string) (c *container.Container) {
	c = container.NewCommand(ctx, absHelperInnerPath, "helper", args...)
	c.Env = append(c.Env, envDoCheck+"=1")
	c.Bind(container.MustAbs(os.Args[0]), absHelperInnerPath, 0)

	// in case test has cgo enabled
	if entries, err := ldd.Exec(ctx, os.Args[0]); err != nil {
		log.Fatalf("ldd: %v", err)
	} else {
		*libPaths = ldd.Path(entries)
	}
	for _, name := range *libPaths {
		c.Bind(name, name, 0)
	}

	return
}

func helperNewContainer(ctx context.Context, args ...string) (c *container.Container) {
	return helperNewContainerLibPaths(ctx, new([]*container.Absolute), args...)
}
