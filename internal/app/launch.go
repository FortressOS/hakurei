package app

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"syscall"

	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/util"
)

const (
	sudoAskPass     = "SUDO_ASKPASS"
	launcherPayload = "FORTIFY_LAUNCHER_PAYLOAD"
)

func (a *App) launcherPayloadEnv() string {
	r := &bytes.Buffer{}
	enc := base64.NewEncoder(base64.StdEncoding, r)

	if err := gob.NewEncoder(enc).Encode(a.command); err != nil {
		state.Fatal("Error encoding launcher payload:", err)
	}

	_ = enc.Close()
	return launcherPayload + "=" + r.String()
}

// Early hidden launcher path
func Early(printVersion bool) {
	if printVersion {
		if r, ok := os.LookupEnv(launcherPayload); ok {
			dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(r))

			var argv []string
			if err := gob.NewDecoder(dec).Decode(&argv); err != nil {
				fmt.Println("Error decoding launcher payload:", err)
				os.Exit(1)
			}

			if err := os.Unsetenv(launcherPayload); err != nil {
				fmt.Println("Error unsetting launcher payload:", err)
				// not fatal, do not fail
			}

			var p string

			if len(argv) > 0 {
				if p, ok = util.Which(argv[0]); !ok {
					fmt.Printf("Did not find '%s' in PATH\n", argv[0])
					os.Exit(1)
				}
			} else {
				if p, ok = os.LookupEnv("SHELL"); !ok {
					fmt.Println("No command was specified and $SHELL was unset")
					os.Exit(1)
				}
			}

			if err := syscall.Exec(p, argv, os.Environ()); err != nil {
				fmt.Println("Error executing launcher payload:", err)
				os.Exit(1)
			}

			// unreachable
			os.Exit(1)
			return
		}
	}
}

func (a *App) launchBySudo() (args []string) {
	args = make([]string, 0, 4+len(a.env)+len(a.command))

	// -Hiu $USER
	args = append(args, "-Hiu", a.Username)

	// -A?
	if _, ok := os.LookupEnv(sudoAskPass); ok {
		if system.V.Verbose {
			fmt.Printf("%s set, adding askpass flag\n", sudoAskPass)
		}
		args = append(args, "-A")
	}

	// environ
	args = append(args, a.env...)

	// -- $@
	args = append(args, "--")
	args = append(args, a.command...)

	return
}

func (a *App) launchByMachineCtl(bare bool) (args []string) {
	args = make([]string, 0, 9+len(a.env))

	// shell --uid=$USER
	args = append(args, "shell", "--uid="+a.Username)

	// --quiet
	if !system.V.Verbose {
		args = append(args, "--quiet")
	}

	// environ
	envQ := make([]string, len(a.env)+1)
	for i, e := range a.env {
		envQ[i] = "-E" + e
	}
	envQ[len(a.env)] = "-E" + a.launcherPayloadEnv()
	args = append(args, envQ...)

	// -- .host
	args = append(args, "--", ".host")

	// /bin/sh -c
	if sh, ok := util.Which("sh"); !ok {
		state.Fatal("Did not find 'sh' in PATH")
	} else {
		args = append(args, sh, "-c")
	}

	if len(a.command) == 0 { // execute shell if command is not provided
		a.command = []string{"$SHELL"}
	}

	innerCommand := strings.Builder{}

	if !bare {
		innerCommand.WriteString("dbus-update-activation-environment --systemd")
		for _, e := range a.env {
			innerCommand.WriteString(" " + strings.SplitN(e, "=", 2)[0])
		}
		innerCommand.WriteString("; systemctl --user start xdg-desktop-portal-gtk; ")
	}

	if executable, err := os.Executable(); err != nil {
		state.Fatal("Error reading executable path:", err)
	} else {
		innerCommand.WriteString("exec " + executable + " -V")
	}
	args = append(args, innerCommand.String())

	return
}
