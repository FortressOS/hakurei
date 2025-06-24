package main

import (
	"log"
	"path"
)

const compPoison = "INVALIDINVALIDINVALIDINVALIDINVALID"

var (
	hmain = compPoison
	fpkg  = compPoison
)

func mustCheckPath(p string) string {
	if p != compPoison && p != "" && path.IsAbs(p) {
		return p
	}
	log.Fatal("this program is compiled incorrectly")
	return compPoison
}
