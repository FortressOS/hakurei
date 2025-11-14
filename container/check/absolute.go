// Package check provides types yielding values checked to meet a condition.
package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"syscall"
)

// AbsoluteError is returned by [NewAbs] and holds the invalid pathname.
type AbsoluteError struct{ Pathname string }

func (e *AbsoluteError) Error() string { return fmt.Sprintf("path %q is not absolute", e.Pathname) }
func (e *AbsoluteError) Is(target error) bool {
	var ce *AbsoluteError
	if !errors.As(target, &ce) {
		return errors.Is(target, syscall.EINVAL)
	}
	return *e == *ce
}

// Absolute holds a pathname checked to be absolute.
type Absolute struct{ pathname string }

// unsafeAbs returns [check.Absolute] on any string value.
func unsafeAbs(pathname string) *Absolute { return &Absolute{pathname} }

func (a *Absolute) String() string {
	if a.pathname == "" {
		panic("attempted use of zero Absolute")
	}
	return a.pathname
}

func (a *Absolute) Is(v *Absolute) bool {
	if a == nil && v == nil {
		return true
	}
	return a != nil && v != nil &&
		a.pathname != "" && v.pathname != "" &&
		a.pathname == v.pathname
}

// NewAbs checks pathname and returns a new [Absolute] if pathname is absolute.
func NewAbs(pathname string) (*Absolute, error) {
	if !path.IsAbs(pathname) {
		return nil, &AbsoluteError{pathname}
	}
	return unsafeAbs(pathname), nil
}

// MustAbs calls [NewAbs] and panics on error.
func MustAbs(pathname string) *Absolute {
	if a, err := NewAbs(pathname); err != nil {
		panic(err)
	} else {
		return a
	}
}

// Append calls [path.Join] with [Absolute] as the first element.
func (a *Absolute) Append(elem ...string) *Absolute {
	return unsafeAbs(path.Join(append([]string{a.String()}, elem...)...))
}

// Dir calls [path.Dir] with [Absolute] as its argument.
func (a *Absolute) Dir() *Absolute { return unsafeAbs(path.Dir(a.String())) }

func (a *Absolute) GobEncode() ([]byte, error) { return []byte(a.String()), nil }
func (a *Absolute) GobDecode(data []byte) error {
	pathname := string(data)
	if !path.IsAbs(pathname) {
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
	if !path.IsAbs(pathname) {
		return &AbsoluteError{pathname}
	}
	a.pathname = pathname
	return nil
}

// SortAbs calls [slices.SortFunc] for a slice of [Absolute].
func SortAbs(x []*Absolute) {
	slices.SortFunc(x, func(a, b *Absolute) int { return strings.Compare(a.String(), b.String()) })
}

// CompactAbs calls [slices.CompactFunc] for a slice of [Absolute].
func CompactAbs(s []*Absolute) []*Absolute {
	return slices.CompactFunc(s, func(a *Absolute, b *Absolute) bool { return a.String() == b.String() })
}
