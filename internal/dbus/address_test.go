package dbus_test

import (
	"errors"
	"reflect"
	"testing"

	"hakurei.app/internal/dbus"
)

func TestParse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		addr    string
		want    []dbus.AddrEntry
		wantErr error
	}{
		{
			name: "simple session unix",
			addr: "unix:path=/run/user/1971/bus",
			want: []dbus.AddrEntry{{
				Method: "unix",
				Values: [][2]string{{"path", "/run/user/1971/bus"}},
			}},
		},
		{
			name: "simple upper escape",
			addr: "debug:name=Test,cat=cute,escaped=%c3%b6",
			want: []dbus.AddrEntry{{
				Method: "debug",
				Values: [][2]string{
					{"name", "Test"},
					{"cat", "cute"},
					{"escaped", "\xc3\xb6"},
				},
			}},
		},
		{
			name: "simple bad escape",
			addr: "debug:name=%",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrBadValLength,
				EntryPos: 0, EntryVal: []byte("debug:name=%"), PairPos: 0, PairVal: []byte("name=%")},
		},

		// upstream test cases
		{
			name: "full address success",
			addr: "unix:path=/tmp/foo;debug:name=test,sliff=sloff;",
			want: []dbus.AddrEntry{
				{Method: "unix", Values: [][2]string{{"path", "/tmp/foo"}}},
				{Method: "debug", Values: [][2]string{{"name", "test"}, {"sliff", "sloff"}}},
			},
		},
		{
			name: "empty address",
			addr: "",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrNoColon,
				EntryVal: []byte{}, PairPos: -1},
		},
		{
			name: "no body",
			addr: "foo",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrNoColon,
				EntryPos: 0, EntryVal: []byte("foo"), PairPos: -1},
		},
		{
			name: "no pair separator",
			addr: "foo:bar",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrBadPairSep,
				EntryPos: 0, EntryVal: []byte("foo:bar"), PairPos: 0, PairVal: []byte("bar")},
		},
		{
			name: "no pair separator multi pair",
			addr: "foo:bar,baz",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrBadPairSep,
				EntryPos: 0, EntryVal: []byte("foo:bar,baz"), PairPos: 0, PairVal: []byte("bar")},
		},
		{
			name: "no pair separator single valid pair",
			addr: "foo:bar=foo,baz",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrBadPairSep,
				EntryPos: 0, EntryVal: []byte("foo:bar=foo,baz"), PairPos: 1, PairVal: []byte("baz")},
		},
		{
			name: "no body single valid address",
			addr: "foo:bar=foo;baz",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrNoColon,
				EntryPos: 1, EntryVal: []byte("baz"), PairPos: -1},
		},
		{
			name: "no key",
			addr: "foo:=foo",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrBadPairKey,
				EntryPos: 0, EntryVal: []byte("foo:=foo"), PairPos: 0, PairVal: []byte("=foo")},
		},
		{
			name: "no value",
			addr: "foo:foo=",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrBadPairVal,
				EntryPos: 0, EntryVal: []byte("foo:foo="), PairPos: 0, PairVal: []byte("foo=")},
		},
		{
			name: "no pair separator single valid pair trailing",
			addr: "foo:foo,bar=baz",
			wantErr: &dbus.BadAddressError{Type: dbus.ErrBadPairSep,
				EntryPos: 0, EntryVal: []byte("foo:foo,bar=baz"), PairPos: 0, PairVal: []byte("foo")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got, err := dbus.Parse([]byte(tc.addr)); !errors.Is(err, tc.wantErr) {
				t.Errorf("Parse() error = %v, wantErr %v", err, tc.wantErr)
			} else if tc.wantErr == nil && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
