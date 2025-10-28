package main

import (
	"encoding/hex"
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
	"hakurei.app/internal/env"
	"hakurei.app/message"
)

// tryPath attempts to read [hst.Config] from multiple sources.
// tryPath reads from [os.Stdin] if name has value "-".
// Otherwise, name is passed to tryFd, and if that returns nil, name is passed to [os.Open].
func tryPath(msg message.Msg, name string) (config *hst.Config) {
	var r io.ReadCloser
	config = new(hst.Config)

	if name != "-" {
		r = tryFd(msg, name)
		if r == nil {
			msg.Verbose("load configuration from file")

			if f, err := os.Open(name); err != nil {
				log.Fatal(err.Error())
				return
			} else {
				r = f
			}
		}
	} else {
		r = os.Stdin
	}

	decodeJSON(log.Fatal, "load configuration", r, &config)
	if err := r.Close(); err != nil {
		log.Fatal(err.Error())
	}
	return
}

// tryFd returns a [io.ReadCloser] if name represents an integer corresponding to a valid file descriptor.
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

// shortLengthMin is the minimum length a short form identifier can have and still be interpreted as an identifier.
const shortLengthMin = 1 << 3

// shortIdentifier returns an eight character short representation of [hst.ID] from its random bytes.
func shortIdentifier(id *hst.ID) string {
	return shortIdentifierString(id.String())
}

// shortIdentifierString implements shortIdentifier on an arbitrary string.
func shortIdentifierString(s string) string {
	return s[len(hst.ID{}) : len(hst.ID{})+shortLengthMin]
}

// tryIdentifier attempts to match [hst.State] from a [hex] representation of [hst.ID] or a prefix of its lower half.
func tryIdentifier(msg message.Msg, name string) (config *hst.Config, entry *hst.State) {
	return tryIdentifierEntries(msg, name, func() map[hst.ID]*hst.State {
		var sc hst.Paths
		env.CopyPaths().Copy(&sc, new(app.Hsu).MustID(nil))
		s := state.NewMulti(msg, sc.RunDirPath)
		if entries, err := state.Join(s); err != nil {
			msg.GetLogger().Printf("cannot join store: %v", err) // not fatal
			return nil
		} else {
			return entries
		}
	})
}

// tryIdentifierEntries implements tryIdentifier with a custom entries pair getter.
func tryIdentifierEntries(
	msg message.Msg,
	name string,
	getEntries func() map[hst.ID]*hst.State,
) (config *hst.Config, entry *hst.State) {
	const (
		likeShort = 1 << iota
		likeFull
	)

	var likely uintptr
	if len(name) >= shortLengthMin && len(name) <= len(hst.ID{}) { // half the hex representation
		// cannot safely decode here due to unknown alignment
		for _, c := range name {
			if c >= '0' && c <= '9' {
				continue
			}
			if c >= 'a' && c <= 'f' {
				continue
			}
			return
		}
		likely |= likeShort
	} else if len(name) == hex.EncodedLen(len(hst.ID{})) {
		likely |= likeFull
	}

	if likely == 0 {
		return
	}
	entries := getEntries()
	if entries == nil {
		return
	}

	switch {
	case likely&likeShort != 0:
		msg.Verbose("argument looks like short identifier")
		for id := range entries {
			v := id.String()
			if strings.HasPrefix(v[len(hst.ID{}):], name) {
				// match, use config from this state entry
				entry = entries[id]
				config = entry.Config
				break
			}

			msg.Verbosef("instance %s skipped", v)
		}
		return

	case likely&likeFull != 0:
		var likelyID hst.ID
		if likelyID.UnmarshalText([]byte(name)) != nil {
			return
		}
		msg.Verbose("argument looks like identifier")
		if ent, ok := entries[likelyID]; ok {
			entry = ent
			config = ent.Config
		}
		return

	default:
		panic("unreachable")
	}
}
