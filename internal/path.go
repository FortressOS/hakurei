package internal

import (
	"log"

	"hakurei.app/container/check"
)

// Absolute paths to the Hakurei installation.
//
// These are set by the linker.
var hakureiPath, hsuPath string

// MustHakureiPath returns the [check.Absolute] path to hakurei.
func MustHakureiPath() *check.Absolute { return mustCheckPath(log.Fatal, "hakurei", hakureiPath) }

// MustHsuPath returns the [check.Absolute] to hsu.
func MustHsuPath() *check.Absolute { return mustCheckPath(log.Fatal, "hsu", hsuPath) }

// mustCheckPath checks a pathname to not be zero, then [check.NewAbs], calling fatal if either step fails.
func mustCheckPath(fatal func(v ...any), name, pathname string) *check.Absolute {
	if pathname != "" {
		if a, err := check.NewAbs(pathname); err != nil {
			fatal(err.Error())
			return nil // unreachable
		} else {
			return a
		}
	} else {
		fatal("invalid " + name + " path, this program is compiled incorrectly")
		return nil // unreachable
	}
}
