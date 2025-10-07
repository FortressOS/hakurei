package hst

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"hakurei.app/container/check"
)

// FilesystemConfig is an abstract representation of a mount point.
type FilesystemConfig interface {
	// Valid returns whether the configuration is valid.
	Valid() bool
	// Path returns the target path in the container.
	Path() *check.Absolute
	// Host returns a slice of all host paths used by this operation.
	Host() []*check.Absolute
	// Apply appends the [container.Op] implementing this operation.
	Apply(z *ApplyState)

	fmt.Stringer
}

// The Ops interface enables [FilesystemConfig] to queue container ops without depending on the container package.
type Ops interface {
	// Tmpfs appends an op that mounts tmpfs on a container path.
	Tmpfs(target *check.Absolute, size int, perm os.FileMode) Ops
	// Readonly appends an op that mounts read-only tmpfs on a container path.
	Readonly(target *check.Absolute, perm os.FileMode) Ops

	// Bind appends an op that bind mounts a host path on a container path.
	Bind(source, target *check.Absolute, flags int) Ops
	// Overlay appends an op that mounts the overlay pseudo filesystem.
	Overlay(target, state, work *check.Absolute, layers ...*check.Absolute) Ops
	// OverlayReadonly appends an op that mounts the overlay pseudo filesystem readonly.
	OverlayReadonly(target *check.Absolute, layers ...*check.Absolute) Ops

	// Link appends an op that creates a symlink in the container filesystem.
	Link(target *check.Absolute, linkName string, dereference bool) Ops

	// Root appends an op that expands a directory into a toplevel bind mount mirror on container root.
	Root(host *check.Absolute, flags int) Ops
	// Etc appends an op that expands host /etc into a toplevel symlink mirror with /etc semantics.
	Etc(host *check.Absolute, prefix string) Ops
}

// ApplyState holds the address of [container.Ops] and any relevant application state.
type ApplyState struct {
	// AutoEtcPrefix is the prefix for [container.AutoEtcOp].
	AutoEtcPrefix string

	Ops
}

var (
	ErrFSNull = errors.New("unexpected null in mount point")
)

// FSTypeError is returned when [ContainerConfig.Filesystem] contains an entry with invalid type.
type FSTypeError string

func (f FSTypeError) Error() string { return fmt.Sprintf("invalid filesystem type %q", string(f)) }

// FSImplError is returned for unsupported implementations of [FilesystemConfig].
type FSImplError struct{ Value FilesystemConfig }

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
	return fmt.Sprintf("implementation %s not supported", name)
}

// FilesystemConfigJSON is the [json] adapter for [FilesystemConfig].
type FilesystemConfigJSON struct{ FilesystemConfig }

// Valid returns whether the [FilesystemConfigJSON] is valid.
func (f *FilesystemConfigJSON) Valid() bool {
	return f != nil && f.FilesystemConfig != nil && f.FilesystemConfig.Valid()
}

// fsType holds the string representation of a [FilesystemConfig]'s concrete type.
type fsType struct {
	Type string `json:"type"`
}

func (f *FilesystemConfigJSON) MarshalJSON() ([]byte, error) {
	if f == nil || f.FilesystemConfig == nil {
		return nil, ErrFSNull
	}
	var v any
	switch cv := f.FilesystemConfig.(type) {
	case *FSBind:
		v = &struct {
			fsType
			*FSBind
		}{fsType{FilesystemBind}, cv}

	case *FSEphemeral:
		v = &struct {
			fsType
			*FSEphemeral
		}{fsType{FilesystemEphemeral}, cv}

	case *FSOverlay:
		v = &struct {
			fsType
			*FSOverlay
		}{fsType{FilesystemOverlay}, cv}

	case *FSLink:
		v = &struct {
			fsType
			*FSLink
		}{fsType{FilesystemLink}, cv}

	default:
		return nil, FSImplError{f.FilesystemConfig}
	}

	return json.Marshal(v)
}

func (f *FilesystemConfigJSON) UnmarshalJSON(data []byte) error {
	t := new(fsType)
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

	case FilesystemOverlay:
		*f = FilesystemConfigJSON{new(FSOverlay)}

	case FilesystemLink:
		*f = FilesystemConfigJSON{new(FSLink)}

	default:
		return FSTypeError(t.Type)
	}

	return json.Unmarshal(data, f.FilesystemConfig)
}
