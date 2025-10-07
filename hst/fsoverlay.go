package hst

import (
	"encoding/gob"
	"strings"

	"hakurei.app/container/check"
)

func init() { gob.Register(new(FSOverlay)) }

// FilesystemOverlay is the type string of an overlay mount point.
const FilesystemOverlay = "overlay"

// FSOverlay represents an overlay mount point.
type FSOverlay struct {
	// mount point in container
	Target *check.Absolute `json:"dst"`

	// any filesystem, does not need to be on a writable filesystem, must not be nil
	Lower []*check.Absolute `json:"lower"`
	// the upperdir is normally on a writable filesystem, leave as nil to mount Lower readonly
	Upper *check.Absolute `json:"upper,omitempty"`
	// the workdir needs to be an empty directory on the same filesystem as Upper, must not be nil if Upper is populated
	Work *check.Absolute `json:"work,omitempty"`
}

func (o *FSOverlay) Valid() bool {
	if o == nil || o.Target == nil {
		return false
	}

	for _, a := range o.Lower {
		if a == nil {
			return false
		}
	}

	if o.Upper != nil { // rw
		return o.Work != nil && len(o.Lower) > 0
	} else { // ro
		return len(o.Lower) >= 2
	}
}

func (o *FSOverlay) Path() *check.Absolute {
	if !o.Valid() {
		return nil
	}
	return o.Target
}

func (o *FSOverlay) Host() []*check.Absolute {
	if !o.Valid() {
		return nil
	}
	p := make([]*check.Absolute, 0, 2+len(o.Lower))
	if o.Upper != nil && o.Work != nil {
		p = append(p, o.Upper, o.Work)
	}
	p = append(p, o.Lower...)
	return p
}

func (o *FSOverlay) Apply(z *ApplyState) {
	if !o.Valid() {
		return
	}

	if o.Upper != nil && o.Work != nil { // rw
		z.Overlay(o.Target, o.Upper, o.Work, o.Lower...)
	} else { // ro
		z.OverlayReadonly(o.Target, o.Lower...)
	}
}

func (o *FSOverlay) String() string {
	if !o.Valid() {
		return "<invalid>"
	}

	lower := make([]string, len(o.Lower))
	for i, a := range o.Lower {
		lower[i] = check.EscapeOverlayDataSegment(a.String())
	}

	if o.Upper != nil && o.Work != nil {
		return "w*" + strings.Join(append([]string{
			check.EscapeOverlayDataSegment(o.Target.String()),
			check.EscapeOverlayDataSegment(o.Upper.String()),
			check.EscapeOverlayDataSegment(o.Work.String())},
			lower...), check.SpecialOverlayPath)
	} else {
		return "*" + strings.Join(append([]string{
			check.EscapeOverlayDataSegment(o.Target.String())},
			lower...), check.SpecialOverlayPath)
	}
}
