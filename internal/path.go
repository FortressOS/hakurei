package internal

import (
	"log"

	"hakurei.app/container"
)

var (
	hmain = compPoison
	hsu   = compPoison
)

// MustHakureiPath returns the absolute path to hakurei, configured at compile time.
func MustHakureiPath() *container.Absolute { return mustCheckPath(log.Fatal, "hakurei", hmain) }

// MustHsuPath returns the absolute path to hakurei, configured at compile time.
func MustHsuPath() *container.Absolute { return mustCheckPath(log.Fatal, "hsu", hsu) }

// mustCheckPath checks a pathname against compPoison, then [container.NewAbs], calling fatal if either step fails.
func mustCheckPath(fatal func(v ...any), name, pathname string) *container.Absolute {
	if pathname != compPoison && pathname != "" {
		if a, err := container.NewAbs(pathname); err != nil {
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
