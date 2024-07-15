package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"syscall"
)

const (
	// hidden path for main to act as a launcher
	egoLauncher = "EGO_LAUNCHER"
)

// hidden launcher path
func tryLauncher() {
	if printVersion {
		if r, ok := os.LookupEnv(egoLauncher); ok {
			// egoLauncher variable contains launcher payload
			dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(r))

			var argv []string
			if err := gob.NewDecoder(dec).Decode(&argv); err != nil {
				fmt.Println("Error decoding launcher payload:", err)
				os.Exit(1)
			}

			if err := os.Unsetenv(egoLauncher); err != nil {
				fmt.Println("Error unsetting launcher payload:", err)
				// not fatal, do not fail
			}

			var p string

			if len(argv) > 0 {
				if p, ok = which(argv[0]); !ok {
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

func launcherPayloadEnv() string {
	r := &bytes.Buffer{}
	enc := base64.NewEncoder(base64.StdEncoding, r)

	if err := gob.NewEncoder(enc).Encode(command); err != nil {
		fatal("Error encoding launcher payload:", err)
	}

	_ = enc.Close()
	return egoLauncher + "=" + r.String()
}
