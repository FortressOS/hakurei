package acl

import (
	"os"
	"reflect"
	"syscall"
	"testing"

	"hakurei.app/container"
)

func TestNewAclPathError(t *testing.T) {
	testCases := []struct {
		name string
		path string
		r    int
		err  error
		want error
	}{
		{"nil", container.Nonexistent, 0, syscall.ENOTRECOVERABLE, nil},

		{"acl_get_file", container.Nonexistent, -1, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_get_file", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_get_tag_type", container.Nonexistent, -2, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_get_tag_type", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_get_qualifier", container.Nonexistent, -3, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_get_qualifier", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_delete_entry", container.Nonexistent, -4, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_delete_entry", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_create_entry", container.Nonexistent, -5, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_create_entry", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_get_permset", container.Nonexistent, -6, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_get_permset", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_add_perm", container.Nonexistent, -7, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_add_perm", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_set_tag_type", container.Nonexistent, -8, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_set_tag_type", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_set_qualifier", container.Nonexistent, -9, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_set_qualifier", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_calc_mask", container.Nonexistent, -10, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_calc_mask", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_valid", container.Nonexistent, -11, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_valid", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"acl_set_file", container.Nonexistent, -12, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "acl_set_file", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},

		{"acl", container.Nonexistent, -13, syscall.ENOTRECOVERABLE,
			&os.PathError{Op: "setfacl", Path: container.Nonexistent, Err: syscall.ENOTRECOVERABLE}},
		{"invalid", container.Nonexistent, -0xdeadbeef, nil,
			&os.PathError{Op: "setfacl", Path: container.Nonexistent}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := newAclPathError(tc.path, tc.r, tc.err)
			if !reflect.DeepEqual(err, tc.want) {
				t.Errorf("newAclPathError: %v, want %v", err, tc.want)
			}
		})
	}
}
