package stub

import (
	"reflect"
	"sync/atomic"
	"testing"
)

// stubHolder embeds [Stub].
type stubHolder struct{ *Stub[stubHolder] }

// overrideT allows some methods of [testing.T] to be overridden.
type overrideT struct {
	*testing.T

	fatal  atomic.Pointer[func(args ...any)]
	fatalf atomic.Pointer[func(format string, args ...any)]
	errorf atomic.Pointer[func(format string, args ...any)]
}

func (t *overrideT) Fatal(args ...any) {
	fp := t.fatal.Load()
	if fp == nil || *fp == nil {
		t.T.Fatal(args...)
		return
	}
	(*fp)(args...)
}

func (t *overrideT) Fatalf(format string, args ...any) {
	fp := t.fatalf.Load()
	if fp == nil || *fp == nil {
		t.T.Fatalf(format, args...)
		return
	}
	(*fp)(format, args...)
}

func (t *overrideT) Errorf(format string, args ...any) {
	fp := t.errorf.Load()
	if fp == nil || *fp == nil {
		t.T.Errorf(format, args...)
		return
	}
	(*fp)(format, args...)
}

func TestStub(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			s := New(t, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{Calls: []Call{
				{"New", ExpectArgs{}, nil, nil},
			}, Tracks: []Expect{{Calls: []Call{
				{"done", ExpectArgs{0xbabe}, nil, nil},
			}}}})

			s.New(func(k stubHolder) {
				expect := k.Expects("done")
				if expect.Name != "done" {
					t.Errorf("New: Name = %s, want done", expect.Name)
				}
				if expect.Args != (ExpectArgs{0xbabe}) {
					t.Errorf("New: Args = %#v", expect.Args)
				}
				if expect.Ret != nil {
					t.Errorf("New: Ret = %#v", expect.Ret)
				}
				if expect.Err != nil {
					t.Errorf("New: Err = %#v", expect.Err)
				}
			})

			if pos := s.Pos(); pos != 1 {
				t.Errorf("Pos: %d, want 1", pos)
			}
			if l := s.Len(); l != 1 {
				t.Errorf("Len: %d, want 1", l)
			}

			s.VisitIncomplete(func(s *Stub[stubHolder]) { panic("unreachable") })
		})

		t.Run("overrun", func(t *testing.T) {
			ot := &overrideT{T: t}
			ot.fatal.Store(checkFatal(t, "New: track overrun"))
			s := New(ot, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{Calls: []Call{
				{"New", ExpectArgs{}, nil, nil},
				{"panic", ExpectArgs{"unreachable"}, nil, nil},
			}})
			func() { defer HandleExit(); s.New(func(k stubHolder) { panic("unreachable") }) }()

			var visit int
			s.VisitIncomplete(func(s *Stub[stubHolder]) {
				visit++
				if visit > 1 {
					panic("unexpected visit count")
				}

				want := Call{"panic", ExpectArgs{"unreachable"}, nil, nil}
				if got := s.want.Calls[s.pos]; !reflect.DeepEqual(got, want) {
					t.Errorf("VisitIncomplete: %#v, want %#v", got, want)
				}
			})
		})

		t.Run("expects", func(t *testing.T) {
			t.Run("overrun", func(t *testing.T) {
				ot := &overrideT{T: t}
				ot.fatal.Store(checkFatal(t, "Expects: advancing beyond expected calls"))
				s := New(ot, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{})
				func() { defer HandleExit(); s.Expects("unreachable") }()
			})

			t.Run("separator", func(t *testing.T) {
				t.Run("overrun", func(t *testing.T) {
					ot := &overrideT{T: t}
					ot.fatalf.Store(checkFatalf(t, "Expects: func = %s, separator overrun", "meow"))
					s := New(ot, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{Calls: []Call{
						{CallSeparator, ExpectArgs{}, nil, nil},
					}})
					func() { defer HandleExit(); s.Expects("meow") }()
				})

				t.Run("mismatch", func(t *testing.T) {
					ot := &overrideT{T: t}
					ot.fatalf.Store(checkFatalf(t, "Expects: separator, want %s", "panic"))
					s := New(ot, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{Calls: []Call{
						{"panic", ExpectArgs{}, nil, nil},
					}})
					func() { defer HandleExit(); s.Expects(CallSeparator) }()
				})
			})

			t.Run("mismatch", func(t *testing.T) {
				ot := &overrideT{T: t}
				ot.fatalf.Store(checkFatalf(t, "Expects: func = %s, want %s", "meow", "nya"))
				s := New(ot, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{Calls: []Call{
					{"nya", ExpectArgs{}, nil, nil},
				}})
				func() { defer HandleExit(); s.Expects("meow") }()
			})
		})
	})
}

func TestCheckArg(t *testing.T) {
	t.Run("oob negative", func(t *testing.T) {
		defer func() {
			want := "invalid call to CheckArg"
			if r := recover(); r != want {
				t.Errorf("recover: %v, want %v", r, want)
			}
		}()
		s := New(t, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{})
		CheckArg(s, "unreachable", struct{}{}, 0)
	})

	ot := &overrideT{T: t}
	s := New(ot, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{Calls: []Call{
		{"panic", ExpectArgs{PanicExit}, nil, nil},
		{"meow", ExpectArgs{-1}, nil, nil},
	}})
	t.Run("match", func(t *testing.T) {
		s.Expects("panic")
		if !CheckArg(s, "v", PanicExit, 0) {
			t.Errorf("CheckArg: unexpected false")
		}
	})
	t.Run("mismatch", func(t *testing.T) {
		defer HandleExit()
		s.Expects("meow")
		ot.errorf.Store(checkFatalf(t, "%s: %s = %#v, want %#v (%d)", "meow", "time", 0, -1, 1))
		if CheckArg(s, "time", 0, 0) {
			t.Errorf("CheckArg: unexpected true")
		}
	})
	t.Run("oob", func(t *testing.T) {
		s.pos++
		defer func() {
			want := "invalid call to CheckArg"
			if r := recover(); r != want {
				t.Errorf("recover: %v, want %v", r, want)
			}
		}()
		CheckArg(s, "unreachable", struct{}{}, 0)
	})
}

func TestCheckArgReflect(t *testing.T) {
	t.Run("oob lower", func(t *testing.T) {
		defer func() {
			want := "invalid call to CheckArgReflect"
			if r := recover(); r != want {
				t.Errorf("recover: %v, want %v", r, want)
			}
		}()
		s := New(t, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{})
		CheckArgReflect(s, "unreachable", struct{}{}, 0)
	})

	ot := &overrideT{T: t}
	s := New(ot, func(s *Stub[stubHolder]) stubHolder { return stubHolder{s} }, Expect{Calls: []Call{
		{"panic", ExpectArgs{PanicExit}, nil, nil},
		{"meow", ExpectArgs{-1}, nil, nil},
	}})
	t.Run("match", func(t *testing.T) {
		s.Expects("panic")
		if !CheckArgReflect(s, "v", PanicExit, 0) {
			t.Errorf("CheckArgReflect: unexpected false")
		}
	})
	t.Run("mismatch", func(t *testing.T) {
		defer HandleExit()
		s.Expects("meow")
		ot.errorf.Store(checkFatalf(t, "%s: %s = %#v, want %#v (%d)", "meow", "time", 0, -1, 1))
		if CheckArgReflect(s, "time", 0, 0) {
			t.Errorf("CheckArgReflect: unexpected true")
		}
	})
	t.Run("oob", func(t *testing.T) {
		s.pos++
		defer func() {
			want := "invalid call to CheckArgReflect"
			if r := recover(); r != want {
				t.Errorf("recover: %v, want %v", r, want)
			}
		}()
		CheckArgReflect(s, "unreachable", struct{}{}, 0)
	})
}

func checkFatal(t *testing.T, wantArgs ...any) *func(args ...any) {
	var called bool
	f := func(args ...any) {
		if called {
			panic("invalid call to fatal")
		}
		called = true

		if !reflect.DeepEqual(args, wantArgs) {
			t.Errorf("Fatal: %#v, want %#v", args, wantArgs)
		}
		panic(PanicExit)
	}
	return &f
}

func checkFatalf(t *testing.T, wantFormat string, wantArgs ...any) *func(format string, args ...any) {
	var called bool
	f := func(format string, args ...any) {
		if called {
			panic("invalid call to fatalf")
		}
		called = true

		if format != wantFormat {
			t.Errorf("Fatalf: format = %q, want %q", format, wantFormat)
		}
		if !reflect.DeepEqual(args, wantArgs) {
			t.Errorf("Fatalf: args = %#v, want %#v", args, wantArgs)
		}
		panic(PanicExit)
	}
	return &f
}
