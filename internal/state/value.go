package state

import (
	"os/user"
)

var (
	u       *user.User
	uid     int
	command []string
)

func Set(val user.User, c []string, d int) {
	if u != nil {
		panic("state set twice")
	}

	u = &val
	command = c
	uid = d
}
