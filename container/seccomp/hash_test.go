package seccomp_test

import (
	"crypto/sha512"
	"encoding/hex"

	"hakurei.app/container/comp"
	"hakurei.app/container/seccomp"
)

type (
	bpfPreset = struct {
		seccomp.ExportFlag
		comp.FilterPreset
	}
	bpfLookup map[bpfPreset][sha512.Size]byte
)

func toHash(s string) [sha512.Size]byte {
	if len(s) != sha512.Size*2 {
		panic("bad sha512 string length")
	}
	if v, err := hex.DecodeString(s); err != nil {
		panic(err.Error())
	} else if len(v) != sha512.Size {
		panic("unreachable")
	} else {
		return ([sha512.Size]byte)(v)
	}
}
