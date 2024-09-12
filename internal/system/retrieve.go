package system

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

func Retrieve() {
	if V != nil {
		panic("system info retrieved twice")
	}

	v := &Values{Share: path.Join(os.TempDir(), "fortify."+strconv.Itoa(os.Geteuid()))}

	if r, ok := os.LookupEnv(xdgRuntimeDir); !ok {
		fmt.Println("Env variable", xdgRuntimeDir, "unset")

		// too early for fatal
		os.Exit(1)
	} else {
		v.Runtime = r
		v.RunDir = path.Join(v.Runtime, "fortify")
	}

	V = v
}
