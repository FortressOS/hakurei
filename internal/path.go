package internal

import (
	"log"
	"path"

	"hakurei.app/internal/hlog"
)

var (
	hmain = compPoison
	hsu   = compPoison
)

func MustHakureiPath() string {
	if name, ok := checkPath(hmain); ok {
		return name
	}
	hlog.BeforeExit()
	log.Fatal("invalid hakurei path, this program is compiled incorrectly")
	return compPoison // unreachable
}

func MustHsuPath() string {
	if name, ok := checkPath(hsu); ok {
		return name
	}
	hlog.BeforeExit()
	log.Fatal("invalid hsu path, this program is compiled incorrectly")
	return compPoison // unreachable
}

func checkPath(p string) (string, bool) { return p, p != compPoison && p != "" && path.IsAbs(p) }
