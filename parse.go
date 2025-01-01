package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
)

func tryPath(name string) (config *fst.Config) {
	var r io.Reader
	config = new(fst.Config)

	if name != "-" {
		r = tryFd(name)
		if r == nil {
			fmsg.VPrintln("load configuration from file")

			if f, err := os.Open(name); err != nil {
				fmsg.Fatalf("cannot access configuration file %q: %s", name, err)
				panic("unreachable")
			} else {
				// finalizer closes f
				r = f
			}
		} else {
			defer func() {
				if err := r.(io.ReadCloser).Close(); err != nil {
					fmsg.Printf("cannot close config fd: %v", err)
				}
			}()
		}
	} else {
		r = os.Stdin
	}

	if err := json.NewDecoder(r).Decode(&config); err != nil {
		fmsg.Fatalf("cannot load configuration: %v", err)
		panic("unreachable")
	}

	return
}

func tryFd(name string) io.ReadCloser {
	if v, err := strconv.Atoi(name); err != nil {
		fmsg.VPrintf("name cannot be interpreted as int64: %v", err)
		return nil
	} else {
		fd := uintptr(v)
		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFD, 0); errno != 0 {
			if errors.Is(errno, syscall.EBADF) {
				return nil
			}
			fmsg.Fatalf("cannot get fd %d: %v", fd, errno)
		}
		return os.NewFile(fd, strconv.Itoa(v))
	}
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

		s := state.NewMulti(sys.Paths().RunDirPath)
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
