package util

import "fmt"

var SdBootedV = func() bool {
	if v, err := SdBooted(); err != nil {
		fmt.Println("warn: read systemd marker:", err)
		return false
	} else {
		return v
	}
}()
