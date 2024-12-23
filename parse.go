package main

import (
	"encoding/json"
	"io"
	direct "os"
	"strings"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
)

func tryPath(name string) (config *fst.Config) {
	var r io.Reader
	config = new(fst.Config)

	if name != "-" {
		if f, err := os.Open(name); err != nil {
			fmsg.Fatalf("cannot access configuration file %q: %s", name, err)
			panic("unreachable")
		} else {
			// finalizer closes f
			r = f
		}
	} else {
		r = direct.Stdin
	}

	if err := json.NewDecoder(r).Decode(&config); err != nil {
		fmsg.Fatalf("cannot load configuration: %v", err)
		panic("unreachable")
	}

	return
}

func tryShort(name string) (config *fst.Config, instance *state.State) {
	likePrefix := false
	if len(name) <= 32 {
		likePrefix = true
		for _, c := range name {
			if c >= '0' && c <= '9' {
				continue
			}
			if c >= 'a' && c <= 'f' {
				continue
			}
			likePrefix = false
			break
		}
	}

	// try to match from state store
	if likePrefix && len(name) >= 8 {
		fmsg.VPrintln("argument looks like prefix")

		s := state.NewMulti(os.Paths().RunDirPath)
		if entries, err := state.Join(s); err != nil {
			fmsg.Printf("cannot join store: %v", err)
			// drop to fetch from file
		} else {
			for id := range entries {
				v := id.String()
				if strings.HasPrefix(v, name) {
					// match, use config from this state entry
					instance = entries[id]
					config = instance.Config
					break
				}

				fmsg.VPrintf("instance %s skipped", v)
			}
		}
	}

	return
}
