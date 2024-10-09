package ldd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Exec(p string) ([]*Entry, error) {
	t := exec.Command("ldd", p)
	t.Stdout, t.Stderr = new(strings.Builder), os.Stderr
	if err := t.Run(); err != nil {
		return nil, err
	}

	return Parse(t.Stdout.(fmt.Stringer))
}
