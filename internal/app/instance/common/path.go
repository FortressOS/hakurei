package common

import (
	"path/filepath"
	"strings"
)

func deepContainsH(basepath, targpath string) (bool, error) {
	rel, err := filepath.Rel(basepath, targpath)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, string([]byte{'.', '.', filepath.Separator})), err
}
