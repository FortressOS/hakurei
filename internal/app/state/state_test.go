package state_test

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
	"time"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
)

func testStore(t *testing.T, s state.Store) {
	t.Run("list empty store", func(t *testing.T) {
		if identities, err := s.List(); err != nil {
			t.Fatalf("List: error = %v", err)
		} else if len(identities) != 0 {
			t.Fatalf("List: identities = %#v", identities)
		}
	})

	const (
		insertEntryChecked = iota
		insertEntryNoCheck
		insertEntryOtherApp

		tl
	)

	var tc [tl]hst.State
	for i := 0; i < tl; i++ {
		makeState(t, &tc[i])
	}

	do := func(identity int, f func(c state.Cursor)) {
		if ok, err := s.Do(identity, f); err != nil {
			t.Fatalf("Do: ok = %v, error = %v", ok, err)
		}
	}

	insert := func(i, identity int) {
		do(identity, func(c state.Cursor) {
			if err := c.Save(&tc[i]); err != nil {
				t.Fatalf("Save: error = %v", err)
			}
		})
	}

	check := func(i, identity int) {
		do(identity, func(c state.Cursor) {
			if entries, err := c.Load(); err != nil {
				t.Fatalf("Load: error = %v", err)
			} else if got, ok := entries[tc[i].ID]; !ok {
				t.Fatalf("Load: entry %s missing", &tc[i].ID)
			} else {
				got.Time = tc[i].Time
				if !reflect.DeepEqual(got, &tc[i]) {
					t.Fatalf("Load: entry %s got %#v, want %#v", &tc[i].ID, got, &tc[i])
				}
			}
		})
	}

	t.Run("insert entry checked", func(t *testing.T) {
		insert(insertEntryChecked, 0)
		check(insertEntryChecked, 0)
	})

	t.Run("insert entry unchecked", func(t *testing.T) {
		insert(insertEntryNoCheck, 0)
	})

	t.Run("insert entry different identity", func(t *testing.T) {
		insert(insertEntryOtherApp, 1)
		check(insertEntryOtherApp, 1)
	})

	t.Run("check previous insertion", func(t *testing.T) {
		check(insertEntryNoCheck, 0)
	})

	t.Run("list identities", func(t *testing.T) {
		if identities, err := s.List(); err != nil {
			t.Fatalf("List: error = %v", err)
		} else {
			slices.Sort(identities)
			want := []int{0, 1}
			if !slices.Equal(identities, want) {
				t.Fatalf("List() = %#v, want %#v", identities, want)
			}
		}
	})

	t.Run("join store", func(t *testing.T) {
		if entries, err := state.Join(s); err != nil {
			t.Fatalf("Join: error = %v", err)
		} else if len(entries) != 3 {
			t.Fatalf("Join(s) = %#v", entries)
		}
	})

	t.Run("clear identity 1", func(t *testing.T) {
		do(1, func(c state.Cursor) {
			if err := c.Destroy(tc[insertEntryOtherApp].ID); err != nil {
				t.Fatalf("Destroy: error = %v", err)
			}
		})
		do(1, func(c state.Cursor) {
			if l, err := c.Len(); err != nil {
				t.Fatalf("Len: error = %v", err)
			} else if l != 0 {
				t.Fatalf("Len: %d, want 0", l)
			}
		})
	})

	t.Run("close store", func(t *testing.T) {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: error = %v", err)
		}
	})
}

func makeState(t *testing.T, s *hst.State) {
	if err := hst.NewInstanceID(&s.ID); err != nil {
		t.Fatalf("cannot create dummy state: %v", err)
	}
	s.PID = rand.Int()
	s.Config = hst.Template()
	s.Time = time.Now()
}
