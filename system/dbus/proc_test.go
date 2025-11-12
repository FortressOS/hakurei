package dbus_test

import (
	"os"
	"testing"

	"hakurei.app/container"
	"hakurei.app/internal/helper"
)

func TestMain(m *testing.M) { container.TryArgv0(nil); helper.InternalHelperStub(); os.Exit(m.Run()) }
