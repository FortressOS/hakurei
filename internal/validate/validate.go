// Package validate provides functions for validating string values of various types.
package validate

import (
	"path/filepath"
	"strings"
)

// DeepContainsH returns whether basepath is equivalent to or is the parent of targpath.
//
// This is used for path hiding warning behaviour, the purpose of which is to improve
// user experience and is *not* a security feature and must not be treated as such.
func DeepContainsH(basepath, targpath string) (bool, error) {
	const upper = ".." + string(filepath.Separator)

	rel, err := filepath.Rel(basepath, targpath)
	return err == nil &&
		rel != ".." &&
		!strings.HasPrefix(rel, upper), err
}
