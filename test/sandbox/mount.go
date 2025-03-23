package sandbox

/*
#cgo linux pkg-config: --static mount

#include <stdlib.h>
#include <stdio.h>
#include <libmount.h>

const char *F_MOUNTINFO_PATH = "/proc/self/mountinfo";
*/
import "C"

import (
	"errors"
	"fmt"
	"iter"
	"runtime"
	"sync"
	"unsafe"
)

var (
	ErrMountinfoParse = errors.New("invalid mountinfo records")
	ErrMountinfoIter  = errors.New("cannot allocate iterator")
	ErrMountinfoFault = errors.New("cannot iterate on filesystems")
)

type (
	Mountinfo struct {
		mu  sync.RWMutex
		p   string
		err error

		tb  *C.struct_libmnt_table
		itr *C.struct_libmnt_iter

		fs *C.struct_libmnt_fs
	}

	// MountinfoEntry represents deterministic mountinfo parts of a libmnt_fs entry.
	MountinfoEntry struct {
		// mount ID: a unique ID for the mount (may be reused after umount(2)).
		ID int `json:"id"`
		// parent ID: the ID of the parent mount (or of self for the root of this mount namespace's mount tree).
		Parent int `json:"parent"`
		// root: the pathname of the directory in the filesystem which forms the root of this mount.
		Root string `json:"root"`
		// mount point: the pathname of the mount point relative to the process's root directory.
		Target string `json:"target"`
		// mount options: per-mount options (see mount(2)).
		VfsOptstr string `json:"vfs_optstr"`
		// filesystem type: the filesystem type in the form "type[.subtype]".
		FsType string `json:"fstype"`
		// mount source: filesystem-specific information or "none".
		Source string `json:"source"`
		// super options: per-superblock options (see mount(2)).
		FsOptstr string `json:"fs_optstr"`
	}
)

func (m *Mountinfo) copy(v *MountinfoEntry) {
	if m.fs == nil {
		panic("invalid entry")
	}
	v.ID = int(C.mnt_fs_get_id(m.fs))
	v.Parent = int(C.mnt_fs_get_parent_id(m.fs))
	v.Root = C.GoString(C.mnt_fs_get_root(m.fs))
	v.Target = C.GoString(C.mnt_fs_get_target(m.fs))
	v.VfsOptstr = C.GoString(C.mnt_fs_get_vfs_options(m.fs))
	v.FsType = C.GoString(C.mnt_fs_get_fstype(m.fs))
	v.Source = C.GoString(C.mnt_fs_get_source(m.fs))
	v.FsOptstr = C.GoString(C.mnt_fs_get_fs_options(m.fs))
}

func NewMountinfo(p string) *Mountinfo { m := new(Mountinfo); m.p = p; return m }

func (m *Mountinfo) Err() error { m.mu.RLock(); defer m.mu.RUnlock(); return m.err }

func (m *Mountinfo) Parse() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tb != nil {
		panic("open called twice")
	}

	if m.p == "" {
		m.tb = C.mnt_new_table_from_file(C.F_MOUNTINFO_PATH)
	} else {
		name := C.CString(m.p)
		m.tb = C.mnt_new_table_from_file(name)
		C.free(unsafe.Pointer(name))
	}
	if m.tb == nil {
		return ErrMountinfoParse
	}
	m.itr = C.mnt_new_iter(C.MNT_ITER_FORWARD)
	if m.itr == nil {
		C.mnt_unref_table(m.tb)
		return ErrMountinfoIter
	}

	runtime.SetFinalizer(m, (*Mountinfo).Unref)
	return nil
}

func (m *Mountinfo) Unref() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tb == nil {
		panic("unref called before parse")
	}

	C.mnt_unref_table(m.tb)
	C.mnt_free_iter(m.itr)
	runtime.SetFinalizer(m, nil)
}

func (m *Mountinfo) Entries() iter.Seq[*MountinfoEntry] {
	return func(yield func(*MountinfoEntry) bool) {
		m.mu.Lock()
		defer m.mu.Unlock()

		C.mnt_reset_iter(m.itr, -1)

		var rc C.int
		ent := new(MountinfoEntry)
		for rc = C.mnt_table_next_fs(m.tb, m.itr, &m.fs); rc == 0; rc = C.mnt_table_next_fs(m.tb, m.itr, &m.fs) {
			m.copy(ent)
			if !yield(ent) {
				return
			}
		}
		if rc < 0 {
			m.err = ErrMountinfoFault
			return
		}
	}
}

func (e *MountinfoEntry) EqualWithIgnore(want *MountinfoEntry, ignore string) bool {
	return (e.ID == want.ID || want.ID == -1) &&
		(e.Parent == want.Parent || want.Parent == -1) &&
		(e.Root == want.Root || want.Root == ignore) &&
		(e.Target == want.Target || want.Target == ignore) &&
		(e.VfsOptstr == want.VfsOptstr || want.VfsOptstr == ignore) &&
		(e.FsType == want.FsType || want.FsType == ignore) &&
		(e.Source == want.Source || want.Source == ignore) &&
		(e.FsOptstr == want.FsOptstr || want.FsOptstr == ignore)
}

func (e *MountinfoEntry) String() string {
	return fmt.Sprintf("%d %d %s %s %s %s %s %s",
		e.ID, e.Parent, e.Root, e.Target, e.VfsOptstr, e.FsType, e.Source, e.FsOptstr)
}
