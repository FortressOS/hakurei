package state_test

import (
	"log"
	"math/rand"
	"reflect"
	"slices"
	"testing"
	"time"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/state"
	"hakurei.app/message"
)

func TestMulti(t *testing.T) {
	s := state.NewMulti(message.NewMsg(log.New(log.Writer(), "multi: ", 0)), check.MustAbs(t.TempDir()))

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
		if err := hst.NewInstanceID(&tc[i].ID); err != nil {
			t.Fatalf("cannot create dummy state: %v", err)
		}
		tc[i].PID = rand.Int()
		tc[i].Config = hst.Template()
		tc[i].Time = time.Now()
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

	// insert entry checked
	insert(insertEntryChecked, 0)
	check(insertEntryChecked, 0)

	// insert entry unchecked
	insert(insertEntryNoCheck, 0)

	// insert entry different identity
	insert(insertEntryOtherApp, 1)
	check(insertEntryOtherApp, 1)

	// check previous insertion
	check(insertEntryNoCheck, 0)

	// list identities
	if identities, err := s.List(); err != nil {
		t.Fatalf("List: error = %v", err)
	} else {
		slices.Sort(identities)
		want := []int{0, 1}
		if !slices.Equal(identities, want) {
			t.Fatalf("List() = %#v, want %#v", identities, want)
		}
	}

	// join store
	if entries, err := state.Join(s); err != nil {
		t.Fatalf("Join: error = %v", err)
	} else if len(entries) != 3 {
		t.Fatalf("Join(s) = %#v", entries)
	}

	// clear identity 1
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
}
