package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"syscall"
)

// AbsoluteError is returned by [NewAbsolute] and holds the invalid pathname.
type AbsoluteError struct {
	Pathname string
}

func (e *AbsoluteError) Error() string { return fmt.Sprintf("path %q is not absolute", e.Pathname) }
func (e *AbsoluteError) Is(target error) bool {
	var ce *AbsoluteError
	if !errors.As(target, &ce) {
		return errors.Is(target, syscall.EINVAL)
	}
	return *e == *ce
}

// Absolute holds a pathname checked to be absolute.
type Absolute struct {
	pathname string
}

// isAbs wraps [path.IsAbs] in case additional checks are added in the future.
func isAbs(pathname string) bool { return path.IsAbs(pathname) }

func (a *Absolute) String() string {
	if a.pathname == zeroString {
		panic("attempted use of zero Absolute")
	}
	return a.pathname
}

// NewAbsolute checks pathname and returns a new [Absolute] if pathname is absolute.
func NewAbsolute(pathname string) (*Absolute, error) {
	if !isAbs(pathname) {
		return nil, &AbsoluteError{pathname}
	}
	return &Absolute{pathname}, nil
}

// MustAbs calls [NewAbsolute] and panics on error.
func MustAbs(pathname string) *Absolute {
	if a, err := NewAbsolute(pathname); err != nil {
		panic(err.Error())
	} else {
		return a
	}
}

func (a *Absolute) GobEncode() ([]byte, error) { return []byte(a.String()), nil }
func (a *Absolute) GobDecode(data []byte) error {
	pathname := string(data)
	if !isAbs(pathname) {
		return &AbsoluteError{pathname}
	}
	a.pathname = pathname
	return nil
}

func (a *Absolute) MarshalJSON() ([]byte, error) { return json.Marshal(a.String()) }
func (a *Absolute) UnmarshalJSON(data []byte) error {
	var pathname string
	if err := json.Unmarshal(data, &pathname); err != nil {
		return err
	}
	if !isAbs(pathname) {
		return &AbsoluteError{pathname}
	}
	a.pathname = pathname
	return nil
}
