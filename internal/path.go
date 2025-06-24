package internal

import (
	"log"
	"path"

	"git.gensokyo.uk/security/hakurei/internal/hlog"
)

var (
	hsu = compPoison
)

func MustHsuPath() string {
	if name, ok := checkPath(hsu); ok {
		return name
	}
	hlog.BeforeExit()
	log.Fatal("invalid hsu path, this program is compiled incorrectly")
	return compPoison
}

func checkPath(p string) (string, bool) { return p, p != compPoison && p != "" && path.IsAbs(p) }
