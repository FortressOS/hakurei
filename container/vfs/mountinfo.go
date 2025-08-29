// Package vfs provides bindings and iterators over proc_pid_mountinfo(5).
package vfs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

const (
	MS_NOSYMFOLLOW = 0x100
)

var (
	ErrMountInfoFields = errors.New("unexpected field count")
	ErrMountInfoEmpty  = errors.New("unexpected empty field")
	ErrMountInfoDevno  = errors.New("bad maj:min field")
	ErrMountInfoSep    = errors.New("bad optional fields separator")
)

type DecoderError struct {
	Op   string
	Line int
	Err  error
}

func (e *DecoderError) Unwrap() error { return e.Err }
func (e *DecoderError) Error() string {
	var s string

	var numError *strconv.NumError
	switch {
	case errors.As(e.Err, &numError) && numError != nil:
		s = "numeric field " + strconv.Quote(numError.Num) + " " + numError.Err.Error()

	default:
		s = e.Err.Error()
	}

	var atLine string
	if e.Line >= 0 {
		atLine = " at line " + strconv.Itoa(e.Line)
	}
	return e.Op + " mountinfo" + atLine + ": " + s
}

type (
	// A MountInfoDecoder reads and decodes proc_pid_mountinfo(5) entries from an input stream.
	MountInfoDecoder struct {
		s *bufio.Scanner
		m *MountInfo

		current  *MountInfo
		parseErr error
		curLine  int
		complete bool
	}

	// MountInfo represents the contents of a proc_pid_mountinfo(5) document.
	MountInfo struct {
		Next *MountInfo
		MountInfoEntry
	}

	// MountInfoEntry represents a proc_pid_mountinfo(5) entry.
	MountInfoEntry struct {
		// mount ID: a unique ID for the mount (may be reused after umount(2)).
		ID int `json:"id"`
		// parent ID: the ID of the parent mount (or of self for the root of this mount namespace's mount tree).
		Parent int `json:"parent"`
		// major:minor: the value of st_dev for files on this filesystem (see stat(2)).
		Devno DevT `json:"devno"`
		// root: the pathname of the directory in the filesystem which forms the root of this mount.
		Root string `json:"root"`
		// mount point: the pathname of the mount point relative to the process's root directory.
		Target string `json:"target"`
		// mount options: per-mount options (see mount(2)).
		VfsOptstr string `json:"vfs_optstr"`
		// optional fields: zero or more fields of the form "tag[:value]"; see below.
		// separator: the end of the optional fields is marked by a single hyphen.
		OptFields []string `json:"opt_fields"`
		// filesystem type: the filesystem type in the form "type[.subtype]".
		FsType string `json:"fstype"`
		// mount source: filesystem-specific information or "none".
		Source string `json:"source"`
		// super options: per-superblock options (see mount(2)).
		FsOptstr string `json:"fs_optstr"`
	}

	DevT [2]int
)

// Flags interprets VfsOptstr and returns the resulting flags and unmatched options.
func (e *MountInfoEntry) Flags() (flags uintptr, unmatched []string) {
	for _, s := range strings.Split(e.VfsOptstr, ",") {
		switch s {
		case "rw":
		case "ro":
			flags |= syscall.MS_RDONLY
		case "nosuid":
			flags |= syscall.MS_NOSUID
		case "nodev":
			flags |= syscall.MS_NODEV
		case "noexec":
			flags |= syscall.MS_NOEXEC
		case "nosymfollow":
			flags |= MS_NOSYMFOLLOW
		case "noatime":
			flags |= syscall.MS_NOATIME
		case "nodiratime":
			flags |= syscall.MS_NODIRATIME
		case "relatime":
			flags |= syscall.MS_RELATIME
		default:
			unmatched = append(unmatched, s)
		}
	}
	return
}

// NewMountInfoDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may read data from r beyond the mountinfo entries requested.
func NewMountInfoDecoder(r io.Reader) *MountInfoDecoder {
	return &MountInfoDecoder{s: bufio.NewScanner(r)}
}

func (d *MountInfoDecoder) Decode(v **MountInfo) (err error) {
	for d.scan() {
	}
	err = d.Err()
	if err == nil {
		*v = d.m
	}
	return
}

// Entries returns an iterator over mountinfo entries.
func (d *MountInfoDecoder) Entries() iter.Seq[*MountInfoEntry] {
	return func(yield func(*MountInfoEntry) bool) {
		for cur := d.m; cur != nil; cur = cur.Next {
			if !yield(&cur.MountInfoEntry) {
				return
			}
		}
		for d.scan() {
			if !yield(&d.current.MountInfoEntry) {
				return
			}
		}
	}
}

func (d *MountInfoDecoder) Err() error {
	if err := d.s.Err(); err != nil {
		return &DecoderError{"scan", d.curLine, err}
	}
	if d.parseErr != nil {
		return &DecoderError{"parse", d.curLine, d.parseErr}
	}
	return nil
}

func (d *MountInfoDecoder) scan() bool {
	if d.complete {
		return false
	}
	if !d.s.Scan() {
		d.complete = true
		return false
	}

	m := new(MountInfo)
	if err := parseMountInfoLine(d.s.Text(), &m.MountInfoEntry); err != nil {
		d.parseErr = err
		d.complete = true
		return false
	}

	if d.current == nil {
		d.m = m
		d.current = d.m
	} else {
		d.current.Next = m
		d.current = d.current.Next
	}
	d.curLine++
	return true
}

func parseMountInfoLine(s string, ent *MountInfoEntry) error {
	// prevent proceeding with misaligned fields due to optional fields
	f := strings.Split(s, " ")
	if len(f) < 10 {
		return ErrMountInfoFields
	}

	// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
	// (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)

	// (1) id
	if id, err := strconv.Atoi(f[0]); err != nil { // 0
		return err
	} else {
		ent.ID = id
	}

	// (2) parent
	if parent, err := strconv.Atoi(f[1]); err != nil { // 1
		return err
	} else {
		ent.Parent = parent
	}

	// (3) maj:min
	if n, err := fmt.Sscanf(f[2], "%d:%d", &ent.Devno[0], &ent.Devno[1]); err != nil {
		return err
	} else if n != 2 {
		// unreachable
		return ErrMountInfoDevno
	}

	// (4) mountroot
	ent.Root = Unmangle(f[3])
	if ent.Root == "" {
		return ErrMountInfoEmpty
	}

	// (5) target
	ent.Target = Unmangle(f[4])
	if ent.Target == "" {
		return ErrMountInfoEmpty
	}

	// (6) vfs options (fs-independent)
	ent.VfsOptstr = Unmangle(f[5])
	if ent.VfsOptstr == "" {
		return ErrMountInfoEmpty
	}

	// (7) optional fields, terminated by " - "
	i := len(f) - 4
	ent.OptFields = f[6:i]

	// (8) optional fields end marker
	if f[i] != "-" {
		return ErrMountInfoSep
	}
	i++

	// (9) FS type
	ent.FsType = Unmangle(f[i])
	if ent.FsType == "" {
		return ErrMountInfoEmpty
	}
	i++

	// (10) source -- maybe empty string
	ent.Source = Unmangle(f[i])
	i++

	// (11) fs options (fs specific)
	ent.FsOptstr = Unmangle(f[i])

	return nil
}

func (e *MountInfoEntry) EqualWithIgnore(want *MountInfoEntry, ignore string) bool {
	return (e.ID == want.ID || want.ID == -1) &&
		(e.Parent == want.Parent || want.Parent == -1) &&
		(e.Devno == want.Devno || (want.Devno[0] == -1 && want.Devno[1] == -1)) &&
		(e.Root == want.Root || want.Root == ignore) &&
		(e.Target == want.Target || want.Target == ignore) &&
		(e.VfsOptstr == want.VfsOptstr || want.VfsOptstr == ignore) &&
		(slices.Equal(e.OptFields, want.OptFields) || (len(want.OptFields) == 1 && want.OptFields[0] == ignore)) &&
		(e.FsType == want.FsType || want.FsType == ignore) &&
		(e.Source == want.Source || want.Source == ignore) &&
		(e.FsOptstr == want.FsOptstr || want.FsOptstr == ignore)
}

func (e *MountInfoEntry) String() string {
	return fmt.Sprintf("%d %d %d:%d %s %s %s %s %s %s %s",
		e.ID, e.Parent, e.Devno[0], e.Devno[1], e.Root, e.Target, e.VfsOptstr,
		strings.Join(append(e.OptFields, "-"), " "), e.FsType, e.Source, e.FsOptstr)
}
