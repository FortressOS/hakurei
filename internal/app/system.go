package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/verbose"
	"git.ophivana.moe/cat/fortify/xcb"
)

// appSeal seals the application with child-related information
type appSeal struct {
	// application unique identifier
	id *appID

	// freedesktop application ID
	fid string
	// argv to start process with in the final confined environment
	command []string
	// environment variables of fortified process
	env []string
	// persistent process state store
	store state.Store

	// uint8 representation of launch method sealed from config
	launchOption uint8
	// process-specific share directory path
	share string

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

// appendEnv appends an environment variable for the child process
func (seal *appSeal) appendEnv(k, v string) {
	seal.env = append(seal.env, k+"="+v)
}

// appSealTx contains the system-level component of the app seal
type appSealTx struct {
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
	// dst, src pairs of temporarily shared files
	tmpfiles [][2]string

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
	path  string
	perms []acl.Perm
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

// updatePerm appends an acl update action
func (tx *appSealTx) updatePerm(path string, perms ...acl.Perm) {
	tx.acl = append(tx.acl, &appACLEntry{path, perms})
}

// changeHosts appends target username of an X11 ChangeHosts action
func (tx *appSealTx) changeHosts(username string) {
	tx.xhost = append(tx.xhost, username)
}

// copyFile appends a tmpfiles action
func (tx *appSealTx) copyFile(dst, src string) {
	tx.tmpfiles = append(tx.tmpfiles, [2]string{dst, src})
	tx.updatePerm(dst, acl.Read)
}

type (
	ChangeHostsError BaseError
	EnsureDirError   BaseError
	TmpfileError     BaseError
	DBusStartError   BaseError
	ACLUpdateError   BaseError
)

// commit applies recorded actions
// order: xhost, mkdir, tmpfiles, dbus, acl
func (tx *appSealTx) commit() error {
	if tx.complete {
		panic("seal transaction committed twice")
	}
	tx.complete = true

	txp := &appSealTx{}
	defer func() {
		// rollback partial commit
		if txp != nil {
			// global changes (x11, ACLs) are always repeated and check for other launchers cannot happen here
			// attempting cleanup here will cause other fortified processes to lose access to them
			// a better (and more secure) fix is to proxy access to these resources and eliminate the ACLs altogether
			if err := txp.revert(false); err != nil {
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
		verbose.Println("applying ACL", e, "uid:", tx.Uid, "path:", e.path)
		if err := acl.UpdatePerm(e.path, tx.uid, e.perms...); err != nil {
			return (*ACLUpdateError)(wrapError(err,
				fmt.Sprintf("cannot apply ACL to '%s': %s", e.path, err)))
		} else {
			// register partial commit
			txp.updatePerm(e.path, e.perms...)
		}
	}

	// disarm partial commit rollback
	txp = nil
	return nil
}

// revert rolls back recorded actions
// order: acl, dbus, tmpfiles, mkdir, xhost
// errors are printed but not treated as fatal
func (tx *appSealTx) revert(global bool) error {
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

	if global {
		// revert ACLs
		for _, e := range tx.acl {
			verbose.Println("stripping ACL", e, "uid:", tx.Uid, "path:", e.path)
			err := acl.UpdatePerm(e.path, tx.uid)
			joinError(err, fmt.Sprintf("cannot strip ACL entry from '%s': %s", e.path, err))
		}
	}

	if tx.dbus != nil {
		// stop dbus proxy
		verbose.Println("terminating message bus proxy")
		err := tx.stopDBus()
		joinError(err, "cannot stop message bus proxy:", err)
	}

	// remove tmpfiles
	for _, tmpfile := range tx.tmpfiles {
		verbose.Println("removing tmpfile", tmpfile[0])
		err := os.Remove(tmpfile[0])
		joinError(err, fmt.Sprintf("cannot remove tmpfile '%s': %s", tmpfile[0], err))
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

	if global {
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

	// workaround for launch method sudo
	if seal.launchOption == LaunchMethodSudo {
		targetRuntime := seal.shareRuntimeChild()
		verbose.Printf("child runtime data dir '%s' configured\n", targetRuntime)
	}

	return nil
}
