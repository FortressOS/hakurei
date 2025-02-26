package internal

import (
	"log"
	"path"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

var (
	fsu = compPoison
)

func MustFsuPath() string {
	if name, ok := checkPath(fsu); ok {
		return name
	}
	fmsg.BeforeExit()
	log.Fatal("invalid fsu path, this program is compiled incorrectly")
	return compPoison
}

func checkPath(p string) (string, bool) { return p, p != compPoison && p != "" && path.IsAbs(p) }
