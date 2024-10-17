package system

import (
	"testing"
)

func TestChangeHosts(t *testing.T) {
	testCases := []string{"chronos", "keyring", "cat", "kbd", "yonah"}
	for _, tc := range testCases {
		t.Run("append ChangeHosts operation for "+tc, func(t *testing.T) {
			sys := New(150)
			sys.ChangeHosts(tc)
			(&tcOp{EX11, tc}).test(t, sys.ops, []Op{
				XHost(tc),
			}, "ChangeHosts")
		})
	}
}

func TestXHost_String(t *testing.T) {
	testCases := []struct {
		username string
		want     string
	}{
		{"chronos", "SI:localuser:chronos"},
	}
	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			if got := XHost(tc.username).String(); got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}
