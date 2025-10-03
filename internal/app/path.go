package app

import (
	"path/filepath"
	"strings"
)

func deepContainsH(basepath, targpath string) (bool, error) {
	const upper = ".." + string(filepath.Separator)

	rel, err := filepath.Rel(basepath, targpath)
	return err == nil &&
		rel != ".." &&
		!strings.HasPrefix(rel, upper), err
}
