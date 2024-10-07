package helper_test

import (
	"strconv"
	"strings"

	"git.ophivana.moe/cat/fortify/helper"
)

var (
	want = []string{
		"unix:path=/run/dbus/system_bus_socket",
		"/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47/system_bus_socket",
		"--filter",
		"--talk=org.bluez",
		"--talk=org.freedesktop.Avahi",
		"--talk=org.freedesktop.UPower",
	}

	wantPayload = strings.Join(want, "\x00") + "\x00"
	argsWt      = helper.MustNewCheckedArgs(want)
)

func argF(argsFD int, _ int) []string {
	return []string{"--args", strconv.Itoa(argsFD)}
}

func argFStatus(argsFD int, statFD int) []string {
	return []string{"--args", strconv.Itoa(argsFD), "--fd", strconv.Itoa(statFD)}
}
