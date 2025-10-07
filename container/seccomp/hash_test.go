package seccomp_test

import (
	"encoding/hex"

	"hakurei.app/container/bits"
	"hakurei.app/container/seccomp"
)

type (
	bpfPreset = struct {
		seccomp.ExportFlag
		bits.FilterPreset
	}
	bpfLookup map[bpfPreset][]byte
)

func toHash(s string) []byte {
	if len(s) != 128 {
		panic("bad sha512 string length")
	}
	if v, err := hex.DecodeString(s); err != nil {
		panic(err.Error())
	} else if len(v) != 64 {
		panic("unreachable")
	} else {
		return v
	}
}
