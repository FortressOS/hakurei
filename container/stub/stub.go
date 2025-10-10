// Package stub provides function call level stubbing and validation
// for library functions that are impossible to check otherwise.
package stub

import (
	"reflect"
	"sync"
	"testing"
)

// this should prevent stub from being inadvertently imported outside tests
var _ = func() {
	if !testing.Testing() {
		panic("stub imported while not in a test")
	}
}

const (
	// A CallSeparator denotes an injected separation between two groups of calls.
	CallSeparator = "\x00"
)

// A Stub is a collection of tracks of expected calls.
type Stub[K any] struct {
	testing.TB

	// makeK creates a new K for a descendant [Stub].
	// This function may be called concurrently.
	makeK func(s *Stub[K]) K

	// want is a hierarchy of expected calls.
	want Expect
	// pos is the current position in [Expect.Calls].
	pos int
	// goroutine counts the number of goroutines created by this [Stub].
	goroutine int
	// sub stores the addresses of descendant [Stub] created by New.
	sub []*Stub[K]
	// wg waits for all descendants to complete.
	wg *sync.WaitGroup
}

// New creates a root [Stub].
func New[K any](tb testing.TB, makeK func(s *Stub[K]) K, want Expect) *Stub[K] {
	return &Stub[K]{TB: tb, makeK: makeK, want: want, wg: new(sync.WaitGroup)}
}

func (s *Stub[K]) FailNow()          { s.Helper(); panic(panicFailNow) }
func (s *Stub[K]) Fatal(args ...any) { s.Helper(); s.Error(args...); panic(panicFatal) }
func (s *Stub[K]) Fatalf(format string, args ...any) {
	s.Helper()
	s.Errorf(format, args...)
	panic(panicFatalf)
}

func (s *Stub[K]) SkipNow()             { s.Helper(); panic("invalid call to SkipNow") }
func (s *Stub[K]) Skip(...any)          { s.Helper(); panic("invalid call to Skip") }
func (s *Stub[K]) Skipf(string, ...any) { s.Helper(); panic("invalid call to Skipf") }

// New calls f in a new goroutine
func (s *Stub[K]) New(f func(k K)) {
	s.Helper()

	s.Expects("New")
	if len(s.want.Tracks) <= s.goroutine {
		s.Fatal("New: track overrun")
	}
	ds := &Stub[K]{TB: s.TB, makeK: s.makeK, want: s.want.Tracks[s.goroutine], wg: s.wg}
	s.goroutine++
	s.sub = append(s.sub, ds)
	s.wg.Add(1)
	go func() {
		s.Helper()

		defer s.wg.Done()
		defer handleExitNew(s.TB)
		f(s.makeK(ds))
	}()
}

// Pos returns the current position of [Stub] in its [Expect.Calls]
func (s *Stub[K]) Pos() int { return s.pos }

// Len returns the length of [Expect.Calls].
func (s *Stub[K]) Len() int { return len(s.want.Calls) }

// VisitIncomplete calls f on an incomplete s and all its descendants.
func (s *Stub[K]) VisitIncomplete(f func(s *Stub[K])) {
	s.Helper()
	s.wg.Wait()

	if s.want.Calls != nil && len(s.want.Calls) != s.pos {
		f(s)
	}
	for _, ds := range s.sub {
		ds.VisitIncomplete(f)
	}
}

// Expects checks the name of and returns the current [Call] and advances pos.
func (s *Stub[K]) Expects(name string) (expect *Call) {
	s.Helper()

	if len(s.want.Calls) == s.pos {
		s.Fatal("Expects: advancing beyond expected calls")
	}
	expect = &s.want.Calls[s.pos]
	if name != expect.Name {
		if expect.Name == CallSeparator {
			s.Fatalf("Expects: func = %s, separator overrun", name)
		}
		if name == CallSeparator {
			s.Fatalf("Expects: separator, want %s", expect.Name)
		}
		s.Fatalf("Expects: func = %s, want %s", name, expect.Name)
	}
	s.pos++
	return
}

// CheckArg checks an argument comparable with the == operator. Avoid using this with pointers.
func CheckArg[T comparable, K any](s *Stub[K], arg string, got T, n int) bool {
	s.Helper()

	pos := s.pos - 1
	if pos < 0 || pos >= len(s.want.Calls) {
		panic("invalid call to CheckArg")
	}
	expect := s.want.Calls[pos]
	want, ok := expect.Args[n].(T)
	if !ok || got != want {
		s.Errorf("%s: %s = %#v, want %#v (%d)", expect.Name, arg, got, want, pos)
		return false
	}
	return true
}

// CheckArgReflect checks an argument of any type.
func CheckArgReflect[K any](s *Stub[K], arg string, got any, n int) bool {
	s.Helper()

	pos := s.pos - 1
	if pos < 0 || pos >= len(s.want.Calls) {
		panic("invalid call to CheckArgReflect")
	}
	expect := s.want.Calls[pos]
	want := expect.Args[n]
	if !reflect.DeepEqual(got, want) {
		s.Errorf("%s: %s = %#v, want %#v (%d)", expect.Name, arg, got, want, pos)
		return false
	}
	return true
}
