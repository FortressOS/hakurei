package internal

import "path"

var (
	Fmain = compPoison
	Fsu   = compPoison
	Fshim = compPoison
	Finit = compPoison
)

func Path(p string) (string, bool) {
	return p, p != compPoison && p != "" && path.IsAbs(p)
}
