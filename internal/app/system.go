package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/helper/bwrap"
	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
	"git.ophivana.moe/cat/fortify/xcb"
)

// appSeal seals the application with child-related information
type appSeal struct {
	// application unique identifier
	id *appID
	// wayland socket path if mediated wayland is enabled
	wl string
	// wait for wayland client to exit if mediated wayland is enabled,
	// (wlDone == nil) determines whether mediated wayland setup is performed
	wlDone chan struct{}

	// freedesktop application ID
	fid string
	// argv to start process with in the final confined environment
	command []string
	// persistent process state store
	store state.Store

	// uint8 representation of launch method sealed from config
	launchOption uint8
	// process-specific share directory path
	share string
	// process-specific share directory path local to XDG_RUNTIME_DIR
	shareLocal string

	// path to launcher program
	toolPath string
	// pass-through enablement tracking from config
	et state.Enablements

	// prevents sharing from happening twice
	shared bool
	// seal system-level component
	sys *appSealTx

	// used in various sealing operations
	internal.SystemConstants

	// protected by upstream mutex
}

// appSealTx contains the system-level component of the app seal
type appSealTx struct {
	bwrap *bwrap.Config

	// reference to D-Bus proxy instance, nil if disabled
	dbus *dbus.Proxy
	// notification from goroutine waiting for dbus.Proxy
	dbusWait chan struct{}
	// upstream address/downstream path used to initialise dbus.Proxy
	dbusAddr *[2][2]string
	// whether system bus proxy is enabled
	dbusSystem bool

	// paths to append/strip ACLs (of target user) from
	acl []*appACLEntry
	// X11 ChangeHosts commands to perform
	xhost []string
	// paths of directories to ensure
	mkdir []appEnsureEntry
	// dst, data pairs of temporarily available files
	files [][2]string
	// dst, src pairs of temporarily shared files
	tmpfiles [][2]string
	// dst, src pairs of temporarily hard linked files
	hardlinks [][2]string

	// default formatted XDG_RUNTIME_DIR of User
	runtime string
	// sealed path to fortify executable, used by shim
	executable string
	// target user UID as an integer
	uid int
	// target user sealed from config
	*user.User

	// prevents commit from happening twice
	complete bool
	// prevents cleanup from happening twice
	closed bool

	// protected by upstream mutex
}

type appEnsureEntry struct {
	path   string
	perm   os.FileMode
	remove bool
}

// setEnv sets an environment variable for the child process
func (tx *appSealTx) setEnv(k, v string) {
	tx.bwrap.SetEnv[k] = v
}

// bind mounts a directory within the sandbox
func (tx *appSealTx) bind(src, dest string, ro bool) {
	if !ro {
		tx.bwrap.Bind = append(tx.bwrap.Bind, [2]string{src, dest})
	} else {
		tx.bwrap.ROBind = append(tx.bwrap.ROBind, [2]string{src, dest})
	}
}

// ensure appends a directory ensure action
func (tx *appSealTx) ensure(path string, perm os.FileMode) {
	tx.mkdir = append(tx.mkdir, appEnsureEntry{path, perm, false})
}

// ensureEphemeral appends a directory ensure action with removal in rollback
func (tx *appSealTx) ensureEphemeral(path string, perm os.FileMode) {
	tx.mkdir = append(tx.mkdir, appEnsureEntry{path, perm, true})
}

// appACLEntry contains information for applying/reverting an ACL entry
type appACLEntry struct {
	tag   state.Enablement
	path  string
	perms []acl.Perm
}

func (e *appACLEntry) ts() string {
	switch e.tag {
	case state.EnableLength:
		return "Global"
	case state.EnableLength + 1:
		return "Process"
	default:
		return e.tag.String()
	}
}

func (e *appACLEntry) String() string {
	var s = []byte("---")
	for _, p := range e.perms {
		switch p {
		case acl.Read:
			s[0] = 'r'
		case acl.Write:
			s[1] = 'w'
		case acl.Execute:
			s[2] = 'x'
		}
	}
	return string(s)
}

// updatePerm appends an untagged acl update action
func (tx *appSealTx) updatePerm(path string, perms ...acl.Perm) {
	tx.updatePermTag(state.EnableLength+1, path, perms...)
}

// updatePermTag appends an acl update action
// Tagging with state.EnableLength sets cleanup to happen at final active launcher exit,
// while tagging with state.EnableLength+1 will unconditionally clean up on exit.
func (tx *appSealTx) updatePermTag(tag state.Enablement, path string, perms ...acl.Perm) {
	tx.acl = append(tx.acl, &appACLEntry{tag, path, perms})
}

// changeHosts appends target username of an X11 ChangeHosts action
func (tx *appSealTx) changeHosts(username string) {
	tx.xhost = append(tx.xhost, username)
}

// writeFile appends a files action
func (tx *appSealTx) writeFile(dst string, data []byte) {
	tx.files = append(tx.files, [2]string{dst, string(data)})
	tx.updatePerm(dst, acl.Read)
	tx.bind(dst, dst, true)
}

// copyFile appends a tmpfiles action
func (tx *appSealTx) copyFile(dst, src string) {
	tx.tmpfiles = append(tx.tmpfiles, [2]string{dst, src})
	tx.updatePerm(dst, acl.Read)
	tx.bind(dst, dst, true)
}

// link appends a hardlink action
func (tx *appSealTx) link(oldname, newname string) {
	tx.hardlinks = append(tx.hardlinks, [2]string{oldname, newname})
}

type (
	ChangeHostsError BaseError
	EnsureDirError   BaseError
	TmpfileError     BaseError
	DBusStartError   BaseError
	ACLUpdateError   BaseError
)

// commit applies recorded actions
// order: xhost, mkdir, files, tmpfiles, hardlinks, dbus, acl
func (tx *appSealTx) commit() error {
	if tx.complete {
		panic("seal transaction committed twice")
	}
	tx.complete = true

	txp := &appSealTx{User: tx.User, bwrap: &bwrap.Config{SetEnv: make(map[string]string)}}
	defer func() {
		// rollback partial commit
		if txp != nil {
			// global changes (x11, ACLs) are always repeated and check for other launchers cannot happen here
			// attempting cleanup here will cause other fortified processes to lose access to them
			// a better (and more secure) fix is to proxy access to these resources and eliminate the ACLs altogether
			tags := new(state.Enablements)
			for e := state.Enablement(0); e < state.EnableLength+2; e++ {
				tags.Set(e)
			}
			if err := txp.revert(tags); err != nil {
				fmt.Println("fortify: errors returned reverting partial commit:", err)
			}
		}
	}()

	// insert xhost entries
	for _, username := range tx.xhost {
		verbose.Printf("inserting XHost entry SI:localuser:%s\n", username)
		if err := xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+username); err != nil {
			return (*ChangeHostsError)(wrapError(err,
				fmt.Sprintf("cannot insert XHost entry SI:localuser:%s, %s", username, err)))
		} else {
			// register partial commit
			txp.changeHosts(username)
		}
	}

	// ensure directories
	for _, dir := range tx.mkdir {
		verbose.Println("ensuring directory mode:", dir.perm.String(), "path:", dir.path)
		if err := os.Mkdir(dir.path, dir.perm); err != nil && !errors.Is(err, fs.ErrExist) {
			return (*EnsureDirError)(wrapError(err,
				fmt.Sprintf("cannot create directory '%s': %s", dir.path, err)))
		} else {
			// only ephemeral dirs require rollback
			if dir.remove {
				// register partial commit
				txp.ensureEphemeral(dir.path, dir.perm)
			}
		}
	}

	// write files
	for _, file := range tx.files {
		verbose.Println("writing", len(file[1]), "bytes of data to", file[0])
		if err := os.WriteFile(file[0], []byte(file[1]), 0600); err != nil {
			return (*TmpfileError)(wrapError(err,
				fmt.Sprintf("cannot write file '%s': %s", file[0], err)))
		} else {
			// register partial commit
			txp.writeFile(file[0], make([]byte, 0)) // data not necessary for revert
		}
	}

	// publish tmpfiles
	for _, tmpfile := range tx.tmpfiles {
		verbose.Println("publishing tmpfile", tmpfile[0], "from", tmpfile[1])
		if err := copyFile(tmpfile[0], tmpfile[1]); err != nil {
			return (*TmpfileError)(wrapError(err,
				fmt.Sprintf("cannot publish tmpfile '%s' from '%s': %s", tmpfile[0], tmpfile[1], err)))
		} else {
			// register partial commit
			txp.copyFile(tmpfile[0], tmpfile[1])
		}
	}

	// create hardlinks
	for _, link := range tx.hardlinks {
		verbose.Println("creating hardlink", link[1], "from", link[0])
		if err := os.Link(link[0], link[1]); err != nil {
			return (*TmpfileError)(wrapError(err,
				fmt.Sprintf("cannot create hardlink '%s' from '%s': %s", link[1], link[0], err)))
		} else {
			// register partial commit
			txp.link(link[0], link[1])
		}
	}

	if tx.dbus != nil {
		// start dbus proxy
		verbose.Printf("session bus proxy on '%s' for upstream '%s'\n", tx.dbusAddr[0][1], tx.dbusAddr[0][0])
		if tx.dbusSystem {
			verbose.Printf("system bus proxy on '%s' for upstream '%s'\n", tx.dbusAddr[1][1], tx.dbusAddr[1][0])
		}
		if err := tx.startDBus(); err != nil {
			return (*DBusStartError)(wrapError(err, "cannot start message bus proxy:", err))
		} else {
			txp.dbus = tx.dbus
			txp.dbusAddr = tx.dbusAddr
			txp.dbusSystem = tx.dbusSystem
			txp.dbusWait = tx.dbusWait
		}
	}

	// apply ACLs
	for _, e := range tx.acl {
		verbose.Println("applying ACL", e, "uid:", tx.Uid, "tag:", e.ts(), "path:", e.path)
		if err := acl.UpdatePerm(e.path, tx.uid, e.perms...); err != nil {
			return (*ACLUpdateError)(wrapError(err,
				fmt.Sprintf("cannot apply ACL to '%s': %s", e.path, err)))
		} else {
			// register partial commit
			txp.updatePermTag(e.tag, e.path, e.perms...)
		}
	}

	// disarm partial commit rollback
	txp = nil
	return nil
}

// revert rolls back recorded actions
// order: acl, dbus, hardlinks, tmpfiles, files, mkdir, xhost
// errors are printed but not treated as fatal
func (tx *appSealTx) revert(tags *state.Enablements) error {
	if tx.closed {
		panic("seal transaction reverted twice")
	}
	tx.closed = true

	// will be slightly over-sized with ephemeral dirs
	errs := make([]error, 0, len(tx.acl)+1+len(tx.tmpfiles)+len(tx.mkdir)+len(tx.xhost))
	joinError := func(err error, a ...any) {
		var e error
		if err != nil {
			e = wrapError(err, a...)
		}
		errs = append(errs, e)
	}

	// revert ACLs
	for _, e := range tx.acl {
		if tags.Has(e.tag) {
			verbose.Println("stripping ACL", e, "uid:", tx.Uid, "tag:", e.ts(), "path:", e.path)
			err := acl.UpdatePerm(e.path, tx.uid)
			joinError(err, fmt.Sprintf("cannot strip ACL entry from '%s': %s", e.path, err))
		} else {
			verbose.Println("skipping ACL", e, "uid:", tx.Uid, "tag:", e.ts(), "path:", e.path)
		}
	}

	if tx.dbus != nil {
		// stop dbus proxy
		verbose.Println("terminating message bus proxy")
		err := tx.stopDBus()
		joinError(err, "cannot stop message bus proxy:", err)
	}

	// remove hardlinks
	for _, link := range tx.hardlinks {
		verbose.Println("removing hardlink", link[1])
		err := os.Remove(link[1])
		joinError(err, fmt.Sprintf("cannot remove hardlink '%s': %s", link[1], err))
	}

	// remove tmpfiles
	for _, tmpfile := range tx.tmpfiles {
		verbose.Println("removing tmpfile", tmpfile[0])
		err := os.Remove(tmpfile[0])
		joinError(err, fmt.Sprintf("cannot remove tmpfile '%s': %s", tmpfile[0], err))
	}

	// remove files
	for _, file := range tx.files {
		verbose.Println("removing file", file[0])
		err := os.Remove(file[0])
		joinError(err, fmt.Sprintf("cannot remove file '%s': %s", file[0], err))
	}

	// remove (empty) ephemeral directories
	for i := len(tx.mkdir); i > 0; i-- {
		dir := tx.mkdir[i-1]
		if !dir.remove {
			continue
		}

		verbose.Println("destroying ephemeral directory mode:", dir.perm.String(), "path:", dir.path)
		err := os.Remove(dir.path)
		joinError(err, fmt.Sprintf("cannot remove ephemeral directory '%s': %s", dir.path, err))
	}

	if tags.Has(state.EnableX) {
		// rollback xhost insertions
		for _, username := range tx.xhost {
			verbose.Printf("deleting XHost entry SI:localuser:%s\n", username)
			err := xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+username)
			joinError(err, "cannot remove XHost entry:", err)
		}
	}

	return errors.Join(errs...)
}

// shareAll calls all share methods in sequence
func (seal *appSeal) shareAll(bus [2]*dbus.Config) error {
	if seal.shared {
		panic("seal shared twice")
	}
	seal.shared = true

	seal.shareRuntime()
	seal.shareSystem()
	targetRuntime := seal.shareRuntimeChild()
	verbose.Printf("child runtime data dir '%s' configured\n", targetRuntime)
	if err := seal.shareDisplay(); err != nil {
		return err
	}
	if err := seal.sharePulse(); err != nil {
		return err
	}

	// ensure dbus session bus defaults
	if bus[0] == nil {
		bus[0] = dbus.NewConfig(seal.fid, true, true)
	}

	if err := seal.shareDBus(bus); err != nil {
		return err
	} else if seal.sys.dbusAddr != nil { // set if D-Bus enabled and share successful
		verbose.Println("sealed session proxy", bus[0].Args(seal.sys.dbusAddr[0]))
		if bus[1] != nil {
			verbose.Println("sealed system proxy", bus[1].Args(seal.sys.dbusAddr[1]))
		}
		verbose.Println("message bus proxy final args:", seal.sys.dbus)
	}

	return nil
}
