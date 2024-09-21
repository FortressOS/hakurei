package app

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const shimPayload = "FORTIFY_SHIM_PAYLOAD"

func (a *app) shimPayloadEnv() string {
	r := &bytes.Buffer{}
	enc := base64.NewEncoder(base64.StdEncoding, r)

	if err := gob.NewEncoder(enc).Encode(a.seal.command); err != nil {
		// should be unreachable
		panic(err)
	}

	_ = enc.Close()
	return shimPayload + "=" + r.String()
}

// TryShim attempts the early hidden launcher shim path
func TryShim() {
	// environment variable contains encoded argv
	if r, ok := os.LookupEnv(shimPayload); ok {
		// everything beyond this point runs as target user
		// proceed with caution!

		// parse base64 revealing underlying gob stream
		dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(r))

		// decode argv gob stream
		var argv []string
		if err := gob.NewDecoder(dec).Decode(&argv); err != nil {
			fmt.Println("fortify-shim: cannot decode shim payload:", err)
			os.Exit(1)
		}

		// remove payload variable since the child does not need to see it
		if err := os.Unsetenv(shimPayload); err != nil {
			fmt.Println("fortify-shim: cannot unset shim payload:", err)
			// not fatal, do not fail
		}

		// look up argv0
		var argv0 string

		if len(argv) > 0 {
			// look up program from $PATH
			if p, err := exec.LookPath(argv[0]); err != nil {
				fmt.Printf("%s not found: %s\n", argv[0], err)
				os.Exit(1)
			} else {
				argv0 = p
			}
		} else {
			// no argv, look up shell instead
			if argv0, ok = os.LookupEnv("SHELL"); !ok {
				fmt.Println("fortify-shim: no command was specified and $SHELL was unset")
				os.Exit(1)
			}

			argv = []string{argv0}
		}

		// exec target process
		if err := syscall.Exec(argv0, argv, os.Environ()); err != nil {
			fmt.Println("fortify-shim: cannot execute shim payload:", err)
			os.Exit(1)
		}

		// unreachable
		os.Exit(1)
		return
	}
}
