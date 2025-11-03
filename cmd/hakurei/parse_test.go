package main

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/store"
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

	msg := message.New(nil)
	id := hst.ID{
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
	}
	withBase := func(extra ...hst.State) []hst.State {
		return append([]hst.State{
			{ID: (hst.ID)(bytes.Repeat([]byte{0xaa}, len(hst.ID{}))), PID: 0xbeef, ShimPID: 0xcafe, Config: hst.Template(), Time: time.Unix(0, 0xdeadbeef0)},
			{ID: (hst.ID)(bytes.Repeat([]byte{0xab}, len(hst.ID{}))), PID: 0x1beef, ShimPID: 0x1cafe, Config: hst.Template(), Time: time.Unix(0, 0xdeadbeef1)},
			{ID: (hst.ID)(bytes.Repeat([]byte{0xf0}, len(hst.ID{}))), PID: 0x2beef, ShimPID: 0x2cafe, Config: hst.Template(), Time: time.Unix(0, 0xdeadbeef2)},

			{ID: (hst.ID)(bytes.Repeat([]byte{0xfe}, len(hst.ID{}))), PID: 0xbed, ShimPID: 0xfff, Config: func() *hst.Config {
				template := hst.Template()
				template.Identity = hst.IdentityEnd
				return template
			}(), Time: time.Unix(0, 0xcafebabe0)},
			{ID: (hst.ID)(bytes.Repeat([]byte{0xfc}, len(hst.ID{}))), PID: 0x1bed, ShimPID: 0x1fff, Config: func() *hst.Config {
				template := hst.Template()
				template.Identity = 0xfc
				return template
			}(), Time: time.Unix(0, 0xcafebabe1)},
			{ID: (hst.ID)(bytes.Repeat([]byte{0xce}, len(hst.ID{}))), PID: 0x2bed, ShimPID: 0x2fff, Config: func() *hst.Config {
				template := hst.Template()
				template.Identity = 0xce
				return template
			}(), Time: time.Unix(0, 0xcafebabe2)},
		}, extra...)
	}
	sampleEntry := hst.State{
		ID:      id,
		PID:     0xcafebabe,
		ShimPID: 0xdeadbeef,
		Config:  hst.Template(),
	}

	testCases := []struct {
		name string
		s    string
		data []hst.State
		want *hst.State
	}{
		{"likely entries fault", "ffffffff", nil, nil},

		{"likely short too short", "ff", nil, nil},
		{"likely short too long", "fffffffffffffffff", nil, nil},
		{"likely short invalid lower", "fffffff\x00", nil, nil},
		{"likely short invalid higher", "0000000\xff", nil, nil},
		{"short no match", "fedcba98", withBase(), nil},
		{"short match", "fedcba98", withBase(sampleEntry), &sampleEntry},
		{"short match single", "fedcba98", []hst.State{sampleEntry}, &sampleEntry},
		{"short match longer", "fedcba98765", withBase(sampleEntry), &sampleEntry},

		{"likely long invalid", "0123456789abcdeffedcba987654321\x00", nil, nil},
		{"long no match", "0123456789abcdeffedcba9876543210", withBase(), nil},
		{"long match", "0123456789abcdeffedcba9876543210", withBase(sampleEntry), &sampleEntry},
		{"long match single", "0123456789abcdeffedcba9876543210", []hst.State{sampleEntry}, &sampleEntry},
	}
	for _, tc := range testCases {
		base := check.MustAbs(t.TempDir()).Append("store")
		s := store.New(base)
		for i := range tc.data {
			if h, err := s.Handle(tc.data[i].Identity); err != nil {
				t.Fatalf("Handle: error = %v", err)
			} else {
				var unlock func()
				if unlock, err = h.Lock(); err != nil {
					t.Fatalf("Lock: error = %v", err)
				}
				_, err = h.Save(&tc.data[i])
				unlock()
				if err != nil {
					t.Fatalf("Save: error = %v", err)
				}
			}
		}

		// store must not be written to beyond this point
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tryIdentifier(msg, tc.s, store.New(base))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tryIdentifier: %#v, want %#v", got, tc.want)
			}
		})
	}
}
