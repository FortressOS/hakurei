// Package ldd provides a robust parser for ldd(1) output, and a convenience function
// for running ldd(1) in a strict sandbox.
//
// Note: despite the additional hardening, great care must be taken when using ldd(1).
// As a general rule, you must never run ldd(1) against a file that you do not wish to
// execute within the same context.
package ldd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"hakurei.app/container/check"
)

var (
	// ErrUnexpectedNewline is returned when encountering an unexpected empty line.
	ErrUnexpectedNewline = errors.New("unexpected newline")
	// ErrUnexpectedSeparator is returned when encountering an unexpected separator segment.
	ErrUnexpectedSeparator = errors.New("unexpected separator")
	// ErrBadLocationFormat is returned for an incorrectly formatted [Entry.Location] segment.
	ErrBadLocationFormat = errors.New("bad location format")
)

// EntryUnexpectedSegmentsError is returned when encountering
// a line containing unexpected number of segments.
type EntryUnexpectedSegmentsError string

func (e EntryUnexpectedSegmentsError) Error() string {
	return fmt.Sprintf("unexpected segments in entry %q", string(e))
}

// An Entry represents one line of ldd(1) output.
type Entry struct {
	// File name of required object.
	Name string `json:"name"`
	// Absolute pathname of matched object. Only populated for the long variant.
	Path *check.Absolute `json:"path,omitempty"`
	// Address at which the object is loaded.
	Location uint64 `json:"location"`
}

const (
	// entrySegmentIndexName is the index of the segment holding [Entry.Name].
	entrySegmentIndexName = 0
	// entrySegmentIndexPath is the index of the segment holding [Entry.Path],
	// present only for a line describing a fully populated [Entry].
	entrySegmentIndexPath = 2
	// entrySegmentIndexSeparator is the index of the segment containing the magic bytes entrySegmentFullSeparator,
	// present only for a line describing a fully populated [Entry].
	entrySegmentIndexSeparator = 1
	// entrySegmentIndexLocation is the index of the segment holding [Entry.Location]
	// for a line describing a fully populated [Entry].
	entrySegmentIndexLocation = 3
	// entrySegmentIndexLocationShort is the index of the segment holding [Entry.Location]
	// for a line describing only [Entry.Name].
	entrySegmentIndexLocationShort = 1

	// entrySegmentSep is the byte separating segments in an [Entry] line.
	entrySegmentSep = ' '
	// entrySegmentFullSeparator is the exact contents of the segment at index entrySegmentIndexSeparator.
	entrySegmentFullSeparator = "=>"

	// entrySegmentLocationLengthMin is the minimum possible length of a segment corresponding to [Entry.Location].
	entrySegmentLocationLengthMin = 4
	// entrySegmentLocationPrefix are magic bytes prefixing a segment corresponding to [Entry.Location].
	entrySegmentLocationPrefix = "(0x"
	// entrySegmentLocationSuffix is the magic byte suffixing a segment corresponding to [Entry.Location].
	entrySegmentLocationSuffix = ')'
)

// decodeLocationSegment decodes and saves the segment corresponding to [Entry.Location].
func (e *Entry) decodeLocationSegment(segment []byte) (err error) {
	if len(segment) < entrySegmentLocationLengthMin ||
		segment[len(segment)-1] != entrySegmentLocationSuffix ||
		string(segment[:len(entrySegmentLocationPrefix)]) != entrySegmentLocationPrefix {
		return ErrBadLocationFormat
	}

	e.Location, err = strconv.ParseUint(string(segment[3:len(segment)-1]), 16, 64)
	return
}

// UnmarshalText parses a line of ldd(1) output and saves it to [Entry].
func (e *Entry) UnmarshalText(data []byte) error {
	var (
		segments = bytes.SplitN(data, []byte{entrySegmentSep}, 5)
		// segment to pass to decodeLocationSegment
		iL int
	)

	switch len(segments) {
	case 2: // /lib/ld-musl-x86_64.so.1 (0x7f04d14ef000)
		iL = entrySegmentIndexLocationShort
		e.Name = string(bytes.TrimSpace(segments[entrySegmentIndexName]))

	case 4: // libc.musl-x86_64.so.1 => /lib/ld-musl-x86_64.so.1 (0x7f04d14ef000)
		iL = entrySegmentIndexLocation
		if string(segments[entrySegmentIndexSeparator]) != entrySegmentFullSeparator {
			return ErrUnexpectedSeparator
		}
		if a, err := check.NewAbs(string(segments[entrySegmentIndexPath])); err != nil {
			return err
		} else {
			e.Path = a
		}
		e.Name = string(bytes.TrimSpace(segments[entrySegmentIndexName]))

	default:
		return EntryUnexpectedSegmentsError(data)
	}

	return e.decodeLocationSegment(segments[iL])
}

// Path returns a deduplicated slice of absolute directory paths in entries.
func Path(entries []*Entry) []*check.Absolute {
	p := make([]*check.Absolute, 0, len(entries)*2)
	for _, entry := range entries {
		if entry.Path != nil {
			p = append(p, entry.Path.Dir())
		}
		if a, err := check.NewAbs(entry.Name); err == nil {
			p = append(p, a.Dir())
		}
	}
	check.SortAbs(p)
	return check.CompactAbs(p)
}

// A Decoder reads and decodes [Entry] values from an input stream.
//
// The zero value is not safe for use.
type Decoder struct {
	s *bufio.Scanner

	// Whether the current line is not the first line.
	notFirst bool
	// Whether s has no more tokens.
	depleted bool
	// Holds onto the first error encountered while parsing.
	err error
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may read
// data from r beyond the [Entry] values requested.
func NewDecoder(r io.Reader) *Decoder { return &Decoder{s: bufio.NewScanner(r)} }

// Scan advances the [Decoder] to the next [Entry] and
// stores the result in the value pointed to by v.
func (d *Decoder) Scan(v *Entry) bool {
	if d.s == nil || d.err != nil || d.depleted {
		return false
	}
	if !d.s.Scan() {
		d.depleted = true
		return false
	}

	data := d.s.Bytes()
	if len(data) == 0 {
		if d.notFirst {
			if d.s.Scan() && d.err == nil {
				d.err = ErrUnexpectedNewline
			}
			// trailing newline is allowed (glibc)
			return false
		}

		// leading newline is allowed (musl)
		d.notFirst = true
		return d.Scan(v)
	}

	d.notFirst = true
	d.err = v.UnmarshalText(data)
	return d.err == nil
}

// Err returns the first non-EOF error that was encountered
// by the underlying [bufio.Scanner] or [Entry].
func (d *Decoder) Err() error {
	if d.err != nil || d.s == nil {
		return d.err
	}
	return d.s.Err()
}

// Decode reads from the input stream until there are no more entries
// and returns the results in a slice.
func (d *Decoder) Decode() ([]*Entry, error) {
	var entries []*Entry

	e := new(Entry)
	for d.Scan(e) {
		entries = append(entries, e)
		e = new(Entry)
	}
	return entries, d.Err()
}

// Parse returns a slice of addresses to [Entry] decoded from p.
func Parse(p []byte) ([]*Entry, error) { return NewDecoder(bytes.NewReader(p)).Decode() }
