package dbus_test

import (
	"os"
	"testing"

	"hakurei.app/container"
	"hakurei.app/helper"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
)

func TestMain(m *testing.M) {
	container.TryArgv0(hlog.Output{}, hlog.Prepare, internal.InstallOutput)
	helper.InternalHelperStub()
	os.Exit(m.Run())
}
