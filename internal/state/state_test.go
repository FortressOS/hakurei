package state_test

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
	"time"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/state"
)

func testStore(t *testing.T, s state.Store) {
	t.Run("list empty store", func(t *testing.T) {
		if aids, err := s.List(); err != nil {
			t.Fatalf("List: error = %v", err)
		} else if len(aids) != 0 {
			t.Fatalf("List: aids = %#v", aids)
		}
	})

	const (
		insertEntryChecked = iota
		insertEntryNoCheck
		insertEntryOtherApp

		tl
	)

	var tc [tl]state.State
	for i := 0; i < tl; i++ {
		makeState(t, &tc[i])
	}

	do := func(aid int, f func(c state.Cursor)) {
		if ok, err := s.Do(aid, f); err != nil {
			t.Fatalf("Do: ok = %v, error = %v", ok, err)
		}
	}

	insert := func(i, aid int) {
		do(aid, func(c state.Cursor) {
			if err := c.Save(&tc[i]); err != nil {
				t.Fatalf("Save(&tc[%v]): error = %v", i, err)
			}
		})
	}

	check := func(i, aid int) {
		do(aid, func(c state.Cursor) {
			if entries, err := c.Load(); err != nil {
				t.Fatalf("Load: error = %v", err)
			} else if got, ok := entries[tc[i].ID]; !ok {
				t.Fatalf("Load: entry %s missing",
					&tc[i].ID)
			} else {
				got.Time = tc[i].Time
				if !reflect.DeepEqual(got, &tc[i]) {
					t.Fatalf("Load: entry %s got %#v, want %#v",
						&tc[i].ID, got, &tc[i])
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

	t.Run("insert entry different aid", func(t *testing.T) {
		insert(insertEntryOtherApp, 1)
		check(insertEntryOtherApp, 1)
	})

	t.Run("check previous insertion", func(t *testing.T) {
		check(insertEntryNoCheck, 0)
	})

	t.Run("list aids", func(t *testing.T) {
		if aids, err := s.List(); err != nil {
			t.Fatalf("List: error = %v", err)
		} else {
			slices.Sort(aids)
			want := []int{0, 1}
			if slices.Compare(aids, want) != 0 {
				t.Fatalf("List() = %#v, want %#v", aids, want)
			}
		}
	})

	t.Run("clear aid 1", func(t *testing.T) {
		do(1, func(c state.Cursor) {
			if err := c.Destroy(tc[insertEntryOtherApp].ID); err != nil {
				t.Fatalf("Destroy: error = %v", err)
			}
		})
		do(1, func(c state.Cursor) {
			if l, err := c.Len(); err != nil {
				t.Fatalf("Len: error = %v", err)
			} else if l != 0 {
				t.Fatalf("Len() = %d, want 0", l)
			}
		})
	})

	t.Run("close store", func(t *testing.T) {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: error = %v", err)
		}
	})
}

func makeState(t *testing.T, s *state.State) {
	if err := fst.NewAppID(&s.ID); err != nil {
		t.Fatalf("cannot create dummy state: %v", err)
	}
	s.Config = fst.Template()
	s.PID = rand.Int()
	s.Time = time.Now()
}
