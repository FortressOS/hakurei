package store_test

import (
	"errors"
	"io"
	"iter"
	"os"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	_ "unsafe"

	"hakurei.app/container/check"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/store"
)

//go:linkname newTemplateState hakurei.app/internal/store.newTemplateState
func newTemplateState() *hst.State

//go:linkname entryDecode hakurei.app/internal/store.entryDecode
func entryDecode(r io.Reader, p *hst.State) (hst.Enablement, error)

//go:linkname newHandle hakurei.app/internal/store.newHandle
func newHandle(base *check.Absolute, identity int) *store.Handle

//go:linkname open hakurei.app/internal/store.(*EntryHandle).open
func open(eh *store.EntryHandle, flag int, perm os.FileMode) (*os.File, error)

//go:linkname save hakurei.app/internal/store.(*EntryHandle).save
func save(eh *store.EntryHandle, state *hst.State) error

func TestStateEntryHandle(t *testing.T) {
	t.Parallel()

	t.Run("lockout", func(t *testing.T) {
		t.Parallel()
		wantErr := func() error { return stub.UniqueError(0) }
		eh := store.EntryHandle{DecodeErr: wantErr(), Pathname: check.MustAbs("/proc/nonexistent")}

		if _, err := open(&eh, -1, 0); !reflect.DeepEqual(err, wantErr()) {
			t.Errorf("open: error = %v, want %v", err, wantErr())
		}
		if err := eh.Destroy(); !reflect.DeepEqual(err, wantErr()) {
			t.Errorf("destroy: error = %v, want %v", err, wantErr())
		}
		if err := save(&eh, nil); !reflect.DeepEqual(err, wantErr()) {
			t.Errorf("save: error = %v, want %v", err, wantErr())
		}
		if _, err := eh.Load(nil); !reflect.DeepEqual(err, wantErr()) {
			t.Errorf("load: error = %v, want %v", err, wantErr())
		}
	})

	t.Run("od", func(t *testing.T) {
		t.Parallel()

		{
			eh := store.EntryHandle{Pathname: check.MustAbs(t.TempDir()).Append("entry")}
			if f, err := open(&eh, os.O_CREATE|syscall.O_EXCL, 0); err != nil {
				t.Fatalf("open: error = %v", err)
			} else if err = f.Close(); err != nil {
				t.Errorf("Close: error = %v", err)
			}
			if err := eh.Destroy(); err != nil {
				t.Fatalf("destroy: error = %v", err)
			}
		}

		t.Run("nonexistent", func(t *testing.T) {
			t.Parallel()
			eh := store.EntryHandle{Pathname: check.MustAbs("/proc/nonexistent")}

			wantErrOpen := &hst.AppError{Step: "open state entry",
				Err: &os.PathError{Op: "open", Path: "/proc/nonexistent", Err: syscall.ENOENT}}
			if _, err := open(&eh, os.O_CREATE|syscall.O_EXCL, 0); !reflect.DeepEqual(err, wantErrOpen) {
				t.Errorf("open: error = %#v, want %#v", err, wantErrOpen)
			}

			wantErrDestroy := &hst.AppError{Step: "destroy state entry",
				Err: &os.PathError{Op: "remove", Path: "/proc/nonexistent", Err: syscall.ENOENT}}
			if err := eh.Destroy(); !reflect.DeepEqual(err, wantErrDestroy) {
				t.Errorf("destroy: error = %#v, want %#v", err, wantErrDestroy)
			}
		})
	})

	t.Run("saveload", func(t *testing.T) {
		t.Parallel()
		eh := store.EntryHandle{Pathname: check.MustAbs(t.TempDir()).Append("entry"),
			ID: newTemplateState().ID}

		if err := save(&eh, newTemplateState()); err != nil {
			t.Fatalf("save: error = %v", err)
		}

		t.Run("validate", func(t *testing.T) {
			t.Parallel()

			t.Run("internal", func(t *testing.T) {
				t.Parallel()

				var got hst.State
				if f, err := os.Open(eh.Pathname.String()); err != nil {
					t.Fatal(err.Error())
				} else if _, err = entryDecode(f, &got); err != nil {
					t.Fatalf("entryDecode: error = %v", err)
				} else if err = f.Close(); err != nil {
					t.Fatal(f.Close())
				}

				if want := newTemplateState(); !reflect.DeepEqual(&got, want) {
					t.Errorf("entryDecode: %#v, want %#v", &got, want)
				}
			})

			t.Run("load header only", func(t *testing.T) {
				t.Parallel()

				if et, err := eh.Load(nil); err != nil {
					t.Fatalf("load: error = %v", err)
				} else if want := newTemplateState().Enablements.Unwrap(); et != want {
					t.Errorf("load: et = %x, want %x", et, want)
				}
			})

			t.Run("load", func(t *testing.T) {
				t.Parallel()

				var got hst.State
				if _, err := eh.Load(&got); err != nil {
					t.Fatalf("load: error = %v", err)
				} else if want := newTemplateState(); !reflect.DeepEqual(&got, want) {
					t.Errorf("load: %#v, want %#v", &got, want)
				}
			})

			t.Run("load inconsistent", func(t *testing.T) {
				t.Parallel()
				wantErr := &hst.AppError{Step: "validate state identifier", Err: os.ErrInvalid,
					Msg: "state entry 00000000000000000000000000000000 has unexpected id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}

				ehi := store.EntryHandle{Pathname: eh.Pathname}
				if _, err := ehi.Load(new(hst.State)); !reflect.DeepEqual(err, wantErr) {
					t.Errorf("load: error = %#v, want %#v", err, wantErr)
				}
			})
		})
	})
}

func TestSegmentHandle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ents [2][]string
		want func(newEh func(err error, name string) *store.EntryHandle) []*store.EntryHandle
		ext  func(t *testing.T, entries iter.Seq[*store.EntryHandle], n int)
	}{
		{"errors", [2][]string{{
			"e81eb203b4190ac5c3842ef44d429945",
			"lock",
			"f0-invalid",
		}, {
			"f1-directory",
		}}, func(newEh func(err error, name string) *store.EntryHandle) []*store.EntryHandle {
			return []*store.EntryHandle{
				newEh(nil, "e81eb203b4190ac5c3842ef44d429945"),
				newEh(&hst.AppError{Step: "decode store segment entry",
					Err: hst.IdentifierDecodeError{Err: hst.ErrIdentifierLength}}, "f0-invalid"),
				newEh(&hst.AppError{Step: "read store segment entries",
					Err: errors.New(`unexpected directory "f1-directory" in store`)}, "f1-directory"),
			}
		}, nil},

		{"success", [2][]string{{
			"e81eb203b4190ac5c3842ef44d429945",
			"7958cfbb9272d9cf9cfd61c85afa13f1",
			"d0b5f7446dd5bd3424ff2f7ac9cace1e",
			"c8c8e2c4aea5c32fe47240ff8caa874e",
			"fa0d30b249d80f155a1f80ceddcc32f2",
			"lock",
		}}, func(newEh func(err error, name string) *store.EntryHandle) []*store.EntryHandle {
			return []*store.EntryHandle{
				newEh(nil, "7958cfbb9272d9cf9cfd61c85afa13f1"),
				newEh(nil, "c8c8e2c4aea5c32fe47240ff8caa874e"),
				newEh(nil, "d0b5f7446dd5bd3424ff2f7ac9cace1e"),
				newEh(nil, "e81eb203b4190ac5c3842ef44d429945"),
				newEh(nil, "fa0d30b249d80f155a1f80ceddcc32f2"),
			}
		}, func(t *testing.T, entries iter.Seq[*store.EntryHandle], n int) {
			if n != 5 {
				t.Fatalf("Entries: n = %d", n)
			}

			// check partial drain
			for range entries {
				break
			}
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := check.MustAbs(t.TempDir()).Append("store")
			segment := base.Append("9")
			if err := os.MkdirAll(segment.String(), 0700); err != nil {
				t.Fatal(err.Error())
			}
			createEntries(t, segment, tc.ents)

			var got []*store.EntryHandle
			if entries, n, err := newHandle(base, 9).Entries(); err != nil {
				t.Fatalf("Entries: error = %v", err)
			} else {
				got = slices.AppendSeq(make([]*store.EntryHandle, 0, n), entries)
				if tc.ext != nil {
					tc.ext(t, entries, n)
				}
			}

			slices.SortFunc(got, func(a, b *store.EntryHandle) int { return strings.Compare(a.Pathname.String(), b.Pathname.String()) })
			want := tc.want(func(err error, name string) *store.EntryHandle {
				eh := store.EntryHandle{DecodeErr: err, Pathname: segment.Append(name)}
				if err == nil {
					if err = eh.UnmarshalText([]byte(name)); err != nil {
						t.Fatalf("UnmarshalText: error = %v", err)
					}
				}
				return &eh
			})

			if !reflect.DeepEqual(got, want) {
				t.Errorf("Entries: %q, want %q", got, want)
			}
		})
	}

	t.Run("nonexistent", func(t *testing.T) {
		var wantErr = &hst.AppError{Step: "read store segment entries", Err: &os.PathError{
			Op:   "open",
			Path: "/proc/nonexistent",
			Err:  syscall.ENOENT,
		}}
		if _, _, err := (&store.Handle{
			Identity: -0xbad,
			Path:     check.MustAbs("/proc/nonexistent"),
		}).Entries(); !reflect.DeepEqual(err, wantErr) {
			t.Fatalf("Entries: error = %#v, want %#v", err, wantErr)
		}
	})
}

// createEntries creates file and directory entries in the specified prefix.
func createEntries(t *testing.T, prefix *check.Absolute, ents [2][]string) {
	for _, s := range ents[0] {
		if f, err := os.OpenFile(prefix.Append(s).String(), os.O_CREATE|os.O_EXCL, 0600); err != nil {
			t.Fatal(err.Error())
		} else if err = f.Close(); err != nil {
			t.Fatal(err.Error())
		}
	}
	for _, s := range ents[1] {
		if err := os.Mkdir(prefix.Append(s).String(), 0700); err != nil {
			t.Fatal(err.Error())
		}
	}
}
