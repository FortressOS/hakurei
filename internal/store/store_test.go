package store_test

import (
	"cmp"
	"iter"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	_ "unsafe"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/store"
)

//go:linkname bigLock hakurei.app/internal/store.(*Store).bigLock
func bigLock(s *store.Store) (unlock func(), err error)

func TestStateStoreBigLock(t *testing.T) {
	t.Parallel()

	{
		s := store.New(check.MustAbs(t.TempDir()).Append("state"))
		for i := 0; i < 2; i++ { // check once behaviour
			if unlock, err := bigLock(s); err != nil {
				t.Fatalf("bigLock: error = %v", err)
			} else {
				unlock()
			}
		}
	}

	t.Run("mkdir", func(t *testing.T) {
		t.Parallel()

		wantErr := &hst.AppError{Step: "create state store directory",
			Err: &os.PathError{Op: "mkdir", Path: "/proc/nonexistent", Err: syscall.ENOENT}}
		for i := 0; i < 2; i++ { // check once behaviour
			if _, err := bigLock(store.New(check.MustAbs("/proc/nonexistent"))); !reflect.DeepEqual(err, wantErr) {
				t.Errorf("bigLock: error = %#v, want %#v", err, wantErr)
			}
		}
	})

	t.Run("access", func(t *testing.T) {
		t.Parallel()

		base := check.MustAbs(t.TempDir()).Append("inaccessible")
		if err := os.MkdirAll(base.String(), 0); err != nil {
			t.Fatal(err.Error())
		}

		wantErr := &hst.AppError{Step: "acquire lock on the state store",
			Err: &os.PathError{Op: "open", Path: base.Append(store.MutexName).String(), Err: syscall.EACCES}}
		if _, err := bigLock(store.New(base)); !reflect.DeepEqual(err, wantErr) {
			t.Errorf("bigLock: error = %#v, want %#v", err, wantErr)
		}
	})
}

func TestStateStoreHandle(t *testing.T) {
	t.Parallel()

	t.Run("loadstore", func(t *testing.T) {
		t.Parallel()

		s := store.New(check.MustAbs(t.TempDir()).Append("store"))

		var handleAddr [8]*store.Handle
		checkHandle := func(identity int, load bool) {
			if h, err := s.Handle(identity); err != nil {
				t.Fatalf("Handle: error = %v", err)
			} else if load != (handleAddr[identity] != nil) {
				t.Fatalf("Handle: load = %v, want %v", load, handleAddr[identity] != nil)
			} else if !load {
				handleAddr[identity] = h

				if h.Identity != identity {
					t.Errorf("Handle: identity = %d, want %d", h.Identity, identity)
				}
			} else if h != handleAddr[identity] {
				t.Fatalf("Handle: %p, want %p", h, handleAddr[identity])
			}
		}

		checkHandle(0, false)
		checkHandle(1, false)
		checkHandle(2, false)
		checkHandle(3, false)
		checkHandle(7, false)
		checkHandle(7, true)
		checkHandle(2, true)
		checkHandle(1, true)
		checkHandle(2, true)
		checkHandle(0, true)
	})

	t.Run("access", func(t *testing.T) {
		t.Parallel()

		base := check.MustAbs(t.TempDir()).Append("inaccessible")
		if err := os.MkdirAll(base.String(), 0); err != nil {
			t.Fatal(err.Error())
		}

		wantErr := &hst.AppError{Step: "acquire lock on the state store",
			Err: &os.PathError{Op: "open", Path: base.Append(store.MutexName).String(), Err: syscall.EACCES}}
		if _, err := store.New(base).Handle(0); !reflect.DeepEqual(err, wantErr) {
			t.Errorf("Handle: error = %#v, want %#v", err, wantErr)
		}
	})

	t.Run("access segment", func(t *testing.T) {
		t.Parallel()

		base := check.MustAbs(t.TempDir()).Append("inaccessible")
		if err := os.MkdirAll(base.String(), 0700); err != nil {
			t.Fatal(err.Error())
		}
		if f, err := os.Create(base.Append(store.MutexName).String()); err != nil {
			t.Fatal(err.Error())
		} else if err = f.Close(); err != nil {
			t.Fatal(err.Error())
		}
		if err := os.Chmod(base.String(), 0100); err != nil {
			t.Fatal(err.Error())
		}
		t.Cleanup(func() {
			if err := os.Chmod(base.String(), 0700); err != nil {
				t.Fatal(err.Error())
			}
		})

		wantErr := &hst.AppError{Step: "create store segment directory",
			Err: &os.PathError{Op: "mkdir", Path: base.Append("0").String(), Err: syscall.EACCES}}
		if _, err := store.New(base).Handle(0); !reflect.DeepEqual(err, wantErr) {
			t.Errorf("Handle: error = %#v, want %#v", err, wantErr)
		}
	})
}

func TestStateStoreSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ents [2][]string
		want []store.SegmentIdentity
		ext  func(t *testing.T, segments iter.Seq[store.SegmentIdentity], n int)
	}{
		{"errors", [2][]string{{
			"f0-invalid-file",
		}, {
			"f1-invalid-syntax",
			"9999",
			"16384",
		}}, []store.SegmentIdentity{
			{-1, &hst.AppError{Step: "process store segment", Err: syscall.EISDIR,
				Msg: `skipped non-directory entry "f0-invalid-file"`}},
			{-1, &hst.AppError{Step: "process store segment", Err: syscall.ERANGE,
				Msg: `skipped out of bounds entry 16384`}},
			{-1, &hst.AppError{Step: "process store segment",
				Err: &strconv.NumError{Func: "Atoi", Num: "f1-invalid-syntax", Err: strconv.ErrSyntax},
				Msg: `skipped non-identity entry "f1-invalid-syntax"`}},
			{9999, nil},
		}, nil},

		{"success", [2][]string{{
			"lock",
		}, {
			"0",
			"1",
			"2",
			"3",
			"4",
			"5",
			"6",
			"7",
			"9",
			"13",
			"20",
			"31",
			"197",
		}}, []store.SegmentIdentity{
			{0, nil},
			{1, nil},
			{2, nil},
			{3, nil},
			{4, nil},
			{5, nil},
			{6, nil},
			{7, nil},
			{9, nil},
			{13, nil},
			{20, nil},
			{31, nil},
			{197, nil},
		}, func(t *testing.T, segments iter.Seq[store.SegmentIdentity], n int) {
			if n != 13 {
				t.Fatalf("Segments: n = %d", n)
			}

			// check partial drain
			for range segments {
				break
			}
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := check.MustAbs(t.TempDir()).Append("store")
			if err := os.Mkdir(base.String(), 0700); err != nil {
				t.Fatal(err.Error())
			}
			createEntries(t, base, tc.ents)

			var got []store.SegmentIdentity
			if segments, n, err := store.New(base).Segments(); err != nil {
				t.Fatalf("Segments: error = %v", err)
			} else {
				got = slices.AppendSeq(make([]store.SegmentIdentity, 0, n), segments)
				if tc.ext != nil {
					tc.ext(t, segments, n)
				}
			}

			slices.SortFunc(got, func(a, b store.SegmentIdentity) int {
				if a.Identity == b.Identity {
					return strings.Compare(a.Err.Error(), b.Err.Error())
				}
				return cmp.Compare(a.Identity, b.Identity)
			})
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Segments: %#v, want %#v", got, tc.want)
			}
		})
	}

	t.Run("access", func(t *testing.T) {
		t.Parallel()

		base := check.MustAbs(t.TempDir()).Append("inaccessible")
		if err := os.MkdirAll(base.String(), 0); err != nil {
			t.Fatal(err.Error())
		}

		wantErr := &hst.AppError{Step: "acquire lock on the state store",
			Err: &os.PathError{Op: "open", Path: base.Append(store.MutexName).String(), Err: syscall.EACCES}}
		if _, _, err := store.New(base).Segments(); !reflect.DeepEqual(err, wantErr) {
			t.Errorf("Segments: error = %#v, want %#v", err, wantErr)
		}
	})
}
