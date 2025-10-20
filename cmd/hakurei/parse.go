package main

import (
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"hakurei.app/hst"
	"hakurei.app/internal/app"
	"hakurei.app/internal/app/state"
	"hakurei.app/message"
)

func tryPath(msg message.Msg, name string) (config *hst.Config) {
	var r io.Reader
	config = new(hst.Config)

	if name != "-" {
		r = tryFd(msg, name)
		if r == nil {
			msg.Verbose("load configuration from file")

			if f, err := os.Open(name); err != nil {
				log.Fatalf("cannot access configuration file %q: %s", name, err)
			} else {
				// finalizer closes f
				r = f
			}
		} else {
			defer func() {
				if err := r.(io.ReadCloser).Close(); err != nil {
					log.Printf("cannot close config fd: %v", err)
				}
			}()
		}
	} else {
		r = os.Stdin
	}

	decodeJSON(log.Fatal, "load configuration", r, &config)
	return
}

func tryFd(msg message.Msg, name string) io.ReadCloser {
	if v, err := strconv.Atoi(name); err != nil {
		if !errors.Is(err, strconv.ErrSyntax) {
			msg.Verbosef("name cannot be interpreted as int64: %v", err)
		}
		return nil
	} else {
		msg.Verbosef("trying config stream from %d", v)
		fd := uintptr(v)
		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFD, 0); errno != 0 {
			if errors.Is(errno, syscall.EBADF) {
				return nil
			}
			log.Fatalf("cannot get fd %d: %v", fd, errno)
		}
		return os.NewFile(fd, strconv.Itoa(v))
	}
}

func tryShort(msg message.Msg, name string) (config *hst.Config, entry *state.State) {
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
		msg.Verbose("argument looks like prefix")

		var sc hst.Paths
		app.CopyPaths().Copy(&sc, new(app.Hsu).MustID(nil))
		s := state.NewMulti(msg, sc.RunDirPath.String())
		if entries, err := state.Join(s); err != nil {
			log.Printf("cannot join store: %v", err)
			// drop to fetch from file
		} else {
			for id := range entries {
				v := id.String()
				if strings.HasPrefix(v, name) {
					// match, use config from this state entry
					entry = entries[id]
					config = entry.Config
					break
				}

				msg.Verbosef("instance %s skipped", v)
			}
		}
	}

	return
}
