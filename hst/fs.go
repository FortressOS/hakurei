package hst

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"hakurei.app/container"
)

// FilesystemConfig is an abstract representation of a mount point.
type FilesystemConfig interface {
	// Type returns the type of this mount point.
	Type() string
	// Target returns the pathname of the mount point in the container.
	Target() *container.Absolute
	// Host returns a slice of all host paths used by this mount point.
	Host() []*container.Absolute
	// Apply appends the [container.Op] implementing this mount point.
	Apply(ops *container.Ops)

	fmt.Stringer
}

var (
	ErrFSNull = errors.New("unexpected null in mount point")
)

// FSTypeError is returned when [ContainerConfig.Filesystem] contains an entry with invalid type.
type FSTypeError string

func (f FSTypeError) Error() string { return fmt.Sprintf("invalid filesystem type %q", string(f)) }

// FSImplError is returned when the underlying struct of [FilesystemConfig] does not match
// what [FilesystemConfig.Type] claims to be.
type FSImplError struct {
	Type  string
	Value FilesystemConfig
}

func (f FSImplError) Error() string {
	implType := reflect.TypeOf(f.Value)
	var name string
	for implType != nil && implType.Kind() == reflect.Ptr {
		name += "*"
		implType = implType.Elem()
	}
	if implType != nil {
		name += implType.Name()
	} else {
		name += "nil"
	}
	return fmt.Sprintf("implementation %s is not %s", name, f.Type)
}

// FilesystemConfigJSON is the [json] adapter for [FilesystemConfig].
type FilesystemConfigJSON struct {
	FilesystemConfig
}

// Valid returns whether the [FilesystemConfigJSON] is valid.
func (f *FilesystemConfigJSON) Valid() bool { return f != nil && f.FilesystemConfig != nil }

func (f *FilesystemConfigJSON) MarshalJSON() ([]byte, error) {
	if f == nil || f.FilesystemConfig == nil {
		return nil, ErrFSNull
	}
	var v any
	t := f.Type()
	switch t {
	case FilesystemBind:
		if ct, ok := f.FilesystemConfig.(*FSBind); !ok {
			return nil, FSImplError{t, f.FilesystemConfig}
		} else {
			v = &struct {
				Type string `json:"type"`
				*FSBind
			}{FilesystemBind, ct}
		}

	case FilesystemEphemeral:
		if ct, ok := f.FilesystemConfig.(*FSEphemeral); !ok {
			return nil, FSImplError{t, f.FilesystemConfig}
		} else {
			v = &struct {
				Type string `json:"type"`
				*FSEphemeral
			}{FilesystemEphemeral, ct}
		}

	default:
		return nil, FSTypeError(t)
	}

	return json.Marshal(v)
}

func (f *FilesystemConfigJSON) UnmarshalJSON(data []byte) error {
	t := new(struct {
		Type string `json:"type"`
	})
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	if t == nil {
		return ErrFSNull
	}
	switch t.Type {
	case FilesystemBind:
		*f = FilesystemConfigJSON{new(FSBind)}

	case FilesystemEphemeral:
		*f = FilesystemConfigJSON{new(FSEphemeral)}

	default:
		return FSTypeError(t.Type)
	}

	return json.Unmarshal(data, f.FilesystemConfig)
}
