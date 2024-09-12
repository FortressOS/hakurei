package verbose

import "fmt"

func Println(a ...any) {
	if verbose.Load() {
		fmt.Println(a...)
	}
}

func Printf(format string, a ...any) {
	if verbose.Load() {
		fmt.Printf(format, a...)
	}
}
