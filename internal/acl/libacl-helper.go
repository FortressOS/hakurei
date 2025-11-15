package acl

import "os"

func newAclPathError(name string, r int, err error) error {
	pathError := &os.PathError{Path: name, Err: err}
	switch r {
	case 0:
		return nil

	case -1:
		pathError.Op = "acl_get_file"
	case -2:
		pathError.Op = "acl_get_tag_type"
	case -3:
		pathError.Op = "acl_get_qualifier"
	case -4:
		pathError.Op = "acl_delete_entry"
	case -5:
		pathError.Op = "acl_create_entry"
	case -6:
		pathError.Op = "acl_get_permset"
	case -7:
		pathError.Op = "acl_add_perm"
	case -8:
		pathError.Op = "acl_set_tag_type"
	case -9:
		pathError.Op = "acl_set_qualifier"
	case -10:
		pathError.Op = "acl_calc_mask"
	case -11:
		pathError.Op = "acl_valid"
	case -12:
		pathError.Op = "acl_set_file"

	default: // unreachable
		pathError.Op = "setfacl"
	}
	return pathError
}
