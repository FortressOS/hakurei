package verbose

import (
	"fmt"
)

var Prefix = "fortify:"

func Println(a ...any) {
	if verbose.Load() {
		fmt.Println(append([]any{Prefix}, a...)...)
	}
}

func Printf(format string, a ...any) {
	if verbose.Load() {
		fmt.Printf(Prefix+" "+format, a...)
	}
}
