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
	"hakurei.app/internal/outcome"
	"hakurei.app/internal/store"
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
		if v < 3 { // reject standard streams
			return nil
		}

		msg.Verbosef("trying config stream from %d", v)
		fd := uintptr(v)
		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFD, 0); errno != 0 {
			if errors.Is(errno, syscall.EBADF) { // reject bad fd
				return nil
			}
			log.Fatalf("cannot get fd %d: %v", fd, errno)
		}

		if outcome.IsPollDescriptor(fd) { // reject runtime internals
			log.Fatalf("invalid config stream %d", fd)
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
func tryIdentifier(msg message.Msg, name string, s *store.Store) *hst.State {
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
			return nil
		}
		likely |= likeShort
	} else if len(name) == hex.EncodedLen(len(hst.ID{})) {
		likely |= likeFull
	}

	if likely == 0 {
		return nil
	}

	entries, copyError := s.All()
	defer func() {
		if err := copyError(); err != nil {
			msg.GetLogger().Println(getMessage("cannot iterate over store:", err))
		}
	}()

	switch {
	case likely&likeShort != 0:
		msg.Verbose("argument looks like short identifier")
		for eh := range entries {
			if eh.DecodeErr != nil {
				msg.Verbose(getMessage("skipping instance:", eh.DecodeErr))
				continue
			}

			if strings.HasPrefix(eh.ID.String()[len(hst.ID{}):], name) {
				var entry hst.State
				if _, err := eh.Load(&entry); err != nil {
					msg.GetLogger().Println(getMessage("cannot load state entry:", err))
					continue
				}
				return &entry
			}
		}
		return nil

	case likely&likeFull != 0:
		var likelyID hst.ID
		if likelyID.UnmarshalText([]byte(name)) != nil {
			return nil
		}
		msg.Verbose("argument looks like identifier")
		for eh := range entries {
			if eh.DecodeErr != nil {
				msg.Verbose(getMessage("skipping instance:", eh.DecodeErr))
				continue
			}

			if eh.ID == likelyID {
				var entry hst.State
				if _, err := eh.Load(&entry); err != nil {
					msg.GetLogger().Println(getMessage("cannot load state entry:", err))
					continue
				}
				return &entry
			}
		}
		return nil

	default:
		panic("unreachable")
	}
}
