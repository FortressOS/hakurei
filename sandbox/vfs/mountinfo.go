package vfs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ErrMountInfoFields = errors.New("unexpected field count")
	ErrMountInfoEmpty  = errors.New("unexpected empty field")
	ErrMountInfoDevno  = errors.New("bad maj:min field")
	ErrMountInfoSep    = errors.New("bad optional fields separator")
)

type (
	// MountInfo represents a /proc/pid/mountinfo document.
	MountInfo struct {
		Next *MountInfo
		MountInfoEntry
	}

	// MountInfoEntry represents a line in /proc/pid/mountinfo.
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

// ParseMountInfo parses a mountinfo file according to proc_pid_mountinfo(5).
func ParseMountInfo(r io.Reader) (*MountInfo, int, error) {
	var m, cur *MountInfo
	s := bufio.NewScanner(r)

	var n int
	for s.Scan() {
		n++

		if cur == nil {
			m = new(MountInfo)
			cur = m
		} else {
			cur.Next = new(MountInfo)
			cur = cur.Next
		}

		// prevent proceeding with misaligned fields due to optional fields
		f := strings.Split(s.Text(), " ")
		if len(f) < 10 {
			return nil, -1, ErrMountInfoFields
		}

		// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
		// (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)

		// (1) id
		if id, err := strconv.Atoi(f[0]); err != nil { // 0
			return nil, -1, err
		} else {
			cur.ID = id
		}

		// (2) parent
		if parent, err := strconv.Atoi(f[1]); err != nil { // 1
			return nil, -1, err
		} else {
			cur.Parent = parent
		}

		// (3) maj:min
		if n, err := fmt.Sscanf(f[2], "%d:%d", &cur.Devno[0], &cur.Devno[1]); err != nil {
			return nil, -1, err
		} else if n != 2 {
			// unreachable
			return nil, -1, ErrMountInfoDevno
		}

		// (4) mountroot
		cur.Root = Unmangle(f[3])
		if cur.Root == "" {
			return nil, -1, ErrMountInfoEmpty
		}

		// (5) target
		cur.Target = Unmangle(f[4])
		if cur.Target == "" {
			return nil, -1, ErrMountInfoEmpty
		}

		// (6) vfs options (fs-independent)
		cur.VfsOptstr = Unmangle(f[5])
		if cur.VfsOptstr == "" {
			return nil, -1, ErrMountInfoEmpty
		}

		// (7) optional fields, terminated by " - "
		i := len(f) - 4
		cur.OptFields = f[6:i]

		// (8) optional fields end marker
		if f[i] != "-" {
			return nil, -1, ErrMountInfoSep
		}
		i++

		// (9) FS type
		cur.FsType = Unmangle(f[i])
		if cur.FsType == "" {
			return nil, -1, ErrMountInfoEmpty
		}
		i++

		// (10) source -- maybe empty string
		cur.Source = Unmangle(f[i])
		i++

		// (11) fs options (fs specific)
		cur.FsOptstr = Unmangle(f[i])
	}
	return m, n, s.Err()
}
