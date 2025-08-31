package stub

import (
	"errors"
	"strconv"
)

var (
	ErrCheck = errors.New("one or more arguments did not match")
)

// UniqueError is an error that only equivalates to other [UniqueError] with the same magic value.
type UniqueError uintptr

func (e UniqueError) Error() string {
	return "unique error " + strconv.Itoa(int(e)) + " injected by the test suite"
}

func (e UniqueError) Is(target error) bool {
	var u UniqueError
	if !errors.As(target, &u) {
		return false
	}
	return e == u
}
