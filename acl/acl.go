// Package acl implements simple ACL manipulation via libacl.
package acl

type Perms []Perm

func (ps Perms) String() string {
	var s = []byte("---")
	for _, p := range ps {
		switch p {
		case Read:
			s[0] = 'r'
		case Write:
			s[1] = 'w'
		case Execute:
			s[2] = 'x'
		}
	}
	return string(s)
}
