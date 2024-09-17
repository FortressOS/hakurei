package internal

import (
	"fmt"

	"git.ophivana.moe/cat/fortify/internal/util"
)

var SdBootedV = func() bool {
	if v, err := util.SdBooted(); err != nil {
		fmt.Println("warn: read systemd marker:", err)
		return false
	} else {
		return v
	}
}()
