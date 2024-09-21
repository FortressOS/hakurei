package verbose

import "fmt"

const prefix = "fortify:"

func Println(a ...any) {
	if verbose.Load() {
		fmt.Println(append([]any{prefix}, a...)...)
	}
}

func Printf(format string, a ...any) {
	if verbose.Load() {
		fmt.Printf(prefix+" "+format, a...)
	}
}
