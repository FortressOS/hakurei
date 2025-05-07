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
  -X	Enable direct connection to X11
  -a int
    	Application identity
  -d string
    	Container home directory (default "os")
  -dbus
    	Enable proxied connection to D-Bus
  -dbus-config string
    	Path to session bus proxy config file, or "builtin" for defaults (default "builtin")
  -dbus-log
    	Force buffered logging in the D-Bus proxy
  -dbus-system string
    	Path to system bus proxy config file, or "nil" to disable (default "nil")
  -g value
    	Groups inherited by all container processes
  -id string
    	Reverse-DNS style Application identifier, leave empty to inherit instance identifier
  -mpris
    	Allow owning MPRIS D-Bus path, has no effect if custom config is available
  -pulse
    	Enable direct connection to PulseAudio
  -u string
    	Passwd user name within sandbox (default "chronos")
  -wayland
    	Enable connection to Wayland via security-context-v1

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
