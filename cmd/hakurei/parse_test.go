package main

import (
	"reflect"
	"testing"

	"hakurei.app/hst"
	"hakurei.app/message"
)

func TestShortIdentifier(t *testing.T) {
	t.Parallel()
	id := hst.ID{
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
	}

	const want = "fedcba98"
	if got := shortIdentifier(&id); got != want {
		t.Errorf("shortIdentifier: %q, want %q", got, want)
	}
}

func TestTryIdentifier(t *testing.T) {
	t.Parallel()
	msg := message.NewMsg(nil)
	id := hst.ID{
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
	}

	testCases := []struct {
		name    string
		s       string
		entries map[hst.ID]*hst.State
		want    *hst.State
	}{
		{"likely entries fault", "ffffffff", nil, nil},

		{"likely short too short", "ff", nil, nil},
		{"likely short too long", "fffffffffffffffff", nil, nil},
		{"likely short invalid lower", "fffffff\x00", nil, nil},
		{"likely short invalid higher", "0000000\xff", nil, nil},
		{"short no match", "fedcba98", map[hst.ID]*hst.State{hst.ID{}: nil}, nil},
		{"short match", "fedcba98", map[hst.ID]*hst.State{
			hst.ID{}: nil,
			id: {
				ID:      id,
				PID:     0xcafebabe,
				ShimPID: 0xdeadbeef,
				Config:  hst.Template(),
			},
		}, &hst.State{
			ID:      id,
			PID:     0xcafebabe,
			ShimPID: 0xdeadbeef,
			Config:  hst.Template(),
		}},
		{"short match longer", "fedcba98765", map[hst.ID]*hst.State{
			hst.ID{}: nil,
			id: {
				ID:      id,
				PID:     0xcafebabe,
				ShimPID: 0xdeadbeef,
				Config:  hst.Template(),
			},
		}, &hst.State{
			ID:      id,
			PID:     0xcafebabe,
			ShimPID: 0xdeadbeef,
			Config:  hst.Template(),
		}},

		{"likely long invalid", "0123456789abcdeffedcba987654321\x00", map[hst.ID]*hst.State{}, nil},
		{"long no match", "0123456789abcdeffedcba9876543210", map[hst.ID]*hst.State{hst.ID{}: nil}, nil},
		{"long match", "0123456789abcdeffedcba9876543210", map[hst.ID]*hst.State{
			hst.ID{}: nil,
			id: {
				ID:      id,
				PID:     0xcafebabe,
				ShimPID: 0xdeadbeef,
				Config:  hst.Template(),
			},
		}, &hst.State{
			ID:      id,
			PID:     0xcafebabe,
			ShimPID: 0xdeadbeef,
			Config:  hst.Template(),
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, got := tryIdentifierEntries(msg, tc.s, func() map[hst.ID]*hst.State { return tc.entries })
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tryIdentifier: %#v, want %#v", got, tc.want)
			}
		})
	}
}
