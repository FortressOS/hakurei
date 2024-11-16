package internal

import "path"

var (
	Fsu   = compPoison
	Finit = compPoison
)

func Path(p string) (string, bool) {
	return p, p != compPoison && p != "" && path.IsAbs(p)
}
