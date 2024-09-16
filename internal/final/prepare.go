package final

import "os/user"

var (
	u   *user.User
	uid int

	runDirPath string
)

func Prepare(val user.User, d int, s string) {
	if u != nil {
		panic("final prepared twice")
	}

	u = &val
	uid = d
	runDirPath = s
}
