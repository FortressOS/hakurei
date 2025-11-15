package acl_test

import (
	"testing"

	"hakurei.app/internal/acl"
)

func TestPerms(t *testing.T) {
	testCases := []struct {
		name  string
		perms acl.Perms
	}{
		{"---", acl.Perms{}},
		{"r--", acl.Perms{acl.Read}},
		{"-w-", acl.Perms{acl.Write}},
		{"--x", acl.Perms{acl.Execute}},
		{"rw-", acl.Perms{acl.Read, acl.Read, acl.Write}},
		{"r-x", acl.Perms{acl.Read, acl.Execute, acl.Execute}},
		{"-wx", acl.Perms{acl.Write, acl.Write, acl.Execute, acl.Execute}},
		{"rwx", acl.Perms{acl.Read, acl.Write, acl.Execute}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.perms.String(); got != tc.name {
				t.Errorf("String: %q, want %q", got, tc.name)
			}
		})
	}
}
