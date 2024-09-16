package app

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"git.ophivana.moe/cat/fortify/internal/final"
	"os"
	"strings"
	"syscall"

	"git.ophivana.moe/cat/fortify/internal/util"
)

const launcherPayload = "FORTIFY_LAUNCHER_PAYLOAD"

func (a *App) launcherPayloadEnv() string {
	r := &bytes.Buffer{}
	enc := base64.NewEncoder(base64.StdEncoding, r)

	if err := gob.NewEncoder(enc).Encode(a.command); err != nil {
		final.Fatal("Error encoding launcher payload:", err)
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

				argv = []string{p}
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
