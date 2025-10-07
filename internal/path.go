package internal

import (
	"log"

	"hakurei.app/container/check"
)

var (
	hmain = compPoison
	hsu   = compPoison
)

// MustHakureiPath returns the absolute path to hakurei, configured at compile time.
func MustHakureiPath() *check.Absolute { return mustCheckPath(log.Fatal, "hakurei", hmain) }

// MustHsuPath returns the absolute path to hakurei, configured at compile time.
func MustHsuPath() *check.Absolute { return mustCheckPath(log.Fatal, "hsu", hsu) }

// mustCheckPath checks a pathname against compPoison, then [container.NewAbs], calling fatal if either step fails.
func mustCheckPath(fatal func(v ...any), name, pathname string) *check.Absolute {
	if pathname != compPoison && pathname != "" {
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
