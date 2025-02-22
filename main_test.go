package main

import (
	"bytes"
	"errors"
	"flag"
	"testing"

	"git.gensokyo.uk/security/fortify/command"
)

func TestHelp(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		want string
	}{
		{
			"main", []string{}, `
Usage:	fortify [-h | --help] [-v] [--json] COMMAND [OPTIONS]

Commands:
    app         Launch app defined by the specified config file
    run         Configure and start a permissive default sandbox
    show        Show the contents of an app configuration
    ps          List active apps and their state
    version     Show fortify version
    license     Show full license text
    template    Produce a config template
    help        Show this help message

`,
		},
		{
			"run", []string{"run", "-h"}, `
Usage:	fortify run [-h | --help] [--dbus-config <value>] [--dbus-system <value>] [--mpris] [--dbus-log] [--id <value>] [-a <int>] [-g <value>] [-d <value>] [-u <value>] [--wayland] [-X] [--dbus] [--pulse] COMMAND [OPTIONS]

Flags:
  -X	Share X11 socket and allow connection
  -a int
    	Fortify application ID
  -d string
    	Application home directory (default "os")
  -dbus
    	Proxy D-Bus connection
  -dbus-config string
    	Path to D-Bus proxy config file, or "builtin" for defaults (default "builtin")
  -dbus-log
    	Force logging in the D-Bus proxy
  -dbus-system string
    	Path to system D-Bus proxy config file, or "nil" to disable (default "nil")
  -g value
    	Groups inherited by the app process
  -id string
    	App ID, leave empty to disable security context app_id
  -mpris
    	Allow owning MPRIS D-Bus path, has no effect if custom config is available
  -pulse
    	Share PulseAudio socket and cookie
  -u string
    	Passwd name within sandbox (default "chronos")
  -wayland
    	Allow Wayland connections

`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			c := buildCommand(out)
			if err := c.Parse(tc.args); !errors.Is(err, command.ErrHelp) && !errors.Is(err, flag.ErrHelp) {
				t.Errorf("Parse: error = %v; want %v",
					err, command.ErrHelp)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("Parse: %s want %s", got, tc.want)
			}
		})
	}
}
