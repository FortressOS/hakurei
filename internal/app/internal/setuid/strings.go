package setuid

import (
	"strconv"

	. "git.gensokyo.uk/security/fortify/internal/app"
)

func newInt(v int) *stringPair[int] { return &stringPair[int]{v, strconv.Itoa(v)} }
func newID(id *ID) *stringPair[ID]  { return &stringPair[ID]{*id, id.String()} }

// stringPair stores a value and its string representation.
type stringPair[T comparable] struct {
	v T
	s string
}

func (s *stringPair[T]) unwrap() T      { return s.v }
func (s *stringPair[T]) String() string { return s.s }
