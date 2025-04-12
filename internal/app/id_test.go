package app_test

import (
	"errors"
	"testing"

	. "git.gensokyo.uk/security/fortify/internal/app"
)

func TestParseAppID(t *testing.T) {
	t.Run("bad length", func(t *testing.T) {
		if err := ParseAppID(new(ID), "meow"); !errors.Is(err, ErrInvalidLength) {
			t.Errorf("ParseAppID: error = %v, wantErr =  %v", err, ErrInvalidLength)
		}
	})

	t.Run("bad byte", func(t *testing.T) {
		wantErr := "invalid char '\\n' at byte 15"
		if err := ParseAppID(new(ID), "02bc7f8936b2af6\n\ne2535cd71ef0bb7"); err == nil || err.Error() != wantErr {
			t.Errorf("ParseAppID: error = %v, wantErr =  %v", err, wantErr)
		}
	})

	t.Run("fuzz 16 iterations", func(t *testing.T) {
		for i := 0; i < 16; i++ {
			testParseAppIDWithRandom(t)
		}
	})
}

func FuzzParseAppID(f *testing.F) {
	for i := 0; i < 16; i++ {
		id := new(ID)
		if err := NewAppID(id); err != nil {
			panic(err.Error())
		}
		f.Add(id[0], id[1], id[2], id[3], id[4], id[5], id[6], id[7], id[8], id[9], id[10], id[11], id[12], id[13], id[14], id[15])
	}

	f.Fuzz(func(t *testing.T, b0, b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14, b15 byte) {
		testParseAppID(t, &ID{b0, b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14, b15})
	})
}

func testParseAppIDWithRandom(t *testing.T) {
	id := new(ID)
	if err := NewAppID(id); err != nil {
		t.Fatalf("cannot generate app ID: %v", err)
	}
	testParseAppID(t, id)
}

func testParseAppID(t *testing.T, id *ID) {
	s := id.String()
	got := new(ID)
	if err := ParseAppID(got, s); err != nil {
		t.Fatalf("cannot parse app ID: %v", err)
	}

	if *got != *id {
		t.Fatalf("ParseAppID(%#v) = \n%#v, want \n%#v", s, got, id)
	}
}
