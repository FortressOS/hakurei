package sandbox

/*
#include <stdlib.h>
#include <stdio.h>
#include <mntent.h>

const char *F_PROC_MOUNTS = "";
const char *F_SET_TYPE = "r";
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

type Mntent struct {
	/* name of mounted filesystem */
	FSName string `json:"fsname"`
	/* filesystem path prefix */
	Dir string `json:"dir"`
	/* mount type (see mntent.h) */
	Type string `json:"type"`
	/* mount options (see mntent.h) */
	Opts string `json:"opts"`
	/* dump frequency in days */
	Freq int `json:"freq"`
	/* pass number on parallel fsck */
	Passno int `json:"passno"`
}

func (e *Mntent) String() string {
	return fmt.Sprintf("%s %s %s %s %d %d",
		e.FSName, e.Dir, e.Type, e.Opts, e.Freq, e.Passno)
}

func IterMounts(name string, f func(e *Mntent)) error {
	m := new(mounts)
	m.p = name
	if err := m.open(); err != nil {
		return err
	}

	for m.scan() {
		e := new(Mntent)
		m.copy(e)
		f(e)
	}

	m.close()
	return m.Err()
}

type mounts struct {
	p  string
	f  *C.FILE
	mu sync.RWMutex

	ent *C.struct_mntent
	err error
}

func (m *mounts) open() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.f != nil {
		panic("open called twice")
	}

	if m.p == "" {
		m.p = "/proc/mounts"
	}

	name := C.CString(m.p)
	f, err := C.setmntent(name, C.F_SET_TYPE)
	C.free(unsafe.Pointer(name))

	if f == nil {
		return err
	}
	m.f = f
	runtime.SetFinalizer(m, (*mounts).close)
	return err
}

func (m *mounts) close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.f == nil {
		panic("close called before open")
	}

	C.endmntent(m.f)
	runtime.SetFinalizer(m, nil)
}

func (m *mounts) scan() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.f == nil {
		panic("invalid file")
	}

	m.ent, m.err = C.getmntent(m.f)
	return m.ent != nil
}

func (m *mounts) Err() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.err
}

func (m *mounts) copy(v *Mntent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ent == nil {
		panic("invalid entry")
	}
	v.FSName = C.GoString(m.ent.mnt_fsname)
	v.Dir = C.GoString(m.ent.mnt_dir)
	v.Type = C.GoString(m.ent.mnt_type)
	v.Opts = C.GoString(m.ent.mnt_opts)
	v.Freq = int(m.ent.mnt_freq)
	v.Passno = int(m.ent.mnt_passno)
}
