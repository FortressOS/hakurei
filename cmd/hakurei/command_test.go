package main

import (
	"bytes"
	"errors"
	"flag"
	"testing"

	"hakurei.app/command"
	"hakurei.app/message"
)

func TestHelp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want string
	}{
		{
			"main", []string{}, `
Usage:	hakurei [-h | --help] [-v] [--json] COMMAND [OPTIONS]

Commands:
    app         Load and start container from configuration file
    run         Configure and start a permissive container
    show        Show live or local app configuration
    ps          List active instances
    version     Display version information
    license     Show full license text
    template    Produce a config template
    help        Show this help message

`,
		},
		{
			"run", []string{"run", "-h"}, `
Usage:	hakurei run [-h | --help] [--dbus-config <value>] [--dbus-system <value>] [--mpris] [--dbus-log] [--id <value>] [-a <int>] [-g <value>] [-d <value>] [-u <value>] [--wayland] [-X] [--dbus] [--pulse] COMMAND [OPTIONS]

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
			t.Parallel()

			out := new(bytes.Buffer)
			c := buildCommand(t.Context(), message.NewMsg(nil), new(earlyHardeningErrs), out)
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
