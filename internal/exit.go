package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/verbose"
	"git.ophivana.moe/cat/fortify/xcb"
)

// ExitState keeps track of various changes fortify made to the system
// as well as other resources that need to be manually released.
// NOT thread safe.
type ExitState struct {
	// target fortified user inherited from app.App
	user *user.User
	// integer UID of targeted user
	uid int
	// returns amount of launcher states read
	launcherStateCount func() (int, error)

	// paths to strip ACLs (of target user) from
	aclCleanupCandidate []string
	// target process capability enablements
	enablements *Enablements
	// whether the xcb.ChangeHosts action was complete
	xcbActionComplete bool

	// reference to D-Bus proxy instance, nil if disabled
	dbusProxy *dbus.Proxy
	// D-Bus wait complete notification
	dbusDone *chan struct{}

	// path to fortify process state information
	statePath string

	// prevents cleanup from happening twice
	complete bool
}

// RegisterRevertPath registers a path with ACLs added by fortify
func (s *ExitState) RegisterRevertPath(p string) {
	s.aclCleanupCandidate = append(s.aclCleanupCandidate, p)
}

// SealEnablements submits the child process enablements
func (s *ExitState) SealEnablements(e Enablements) {
	if s.enablements != nil {
		panic("enablement exit state set twice")
	}
	s.enablements = &e
}

// XcbActionComplete submits xcb.ChangeHosts action completion
func (s *ExitState) XcbActionComplete() {
	if s.xcbActionComplete {
		Fatal("xcb inserted twice")
	}
	s.xcbActionComplete = true
}

// SealDBus submits the child's D-Bus proxy instance
func (s *ExitState) SealDBus(p *dbus.Proxy, done *chan struct{}) {
	if p == nil {
		Fatal("unexpected nil dbus proxy exit state submitted")
	}
	if s.dbusProxy != nil {
		Fatal("dbus proxy exit state set twice")
	}
	s.dbusProxy = p
	s.dbusDone = done
}

// SealStatePath submits filesystem path to the fortify process's state file
func (s *ExitState) SealStatePath(v string) {
	if s.statePath != "" {
		panic("statePath set twice")
	}

	s.statePath = v
}

// NewExit initialises a new ExitState containing basic, unchanging information
// about the fortify process required during cleanup
func NewExit(u *user.User, uid int, f func() (int, error)) *ExitState {
	return &ExitState{
		uid:  uid,
		user: u,

		launcherStateCount: f,
	}
}

func Fatal(msg ...any) {
	fmt.Println(msg...)
	BeforeExit()
	os.Exit(1)
}

var exitState *ExitState

func SealExit(s *ExitState) {
	if exitState != nil {
		panic("exit state submitted twice")
	}

	exitState = s
}

func BeforeExit() {
	if exitState == nil {
		fmt.Println("warn: cleanup attempted before exit state submission")
		return
	}

	exitState.beforeExit()
}

func (s *ExitState) beforeExit() {
	if s.complete {
		panic("beforeExit called twice")
	}

	if s.statePath == "" {
		verbose.Println("State path is unset")
	} else {
		if err := os.Remove(s.statePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			fmt.Println("Error removing state file:", err)
		}
	}

	if count, err := s.launcherStateCount(); err != nil {
		fmt.Println("Error reading active launchers:", err)
		os.Exit(1)
	} else if count > 0 {
		// other launchers are still active
		verbose.Printf("Found %d active launchers, exiting without cleaning up\n", count)
		return
	}

	verbose.Println("No other launchers active, will clean up")

	if s.xcbActionComplete {
		verbose.Printf("X11: Removing XHost entry SI:localuser:%s\n", s.user.Username)
		if err := xcb.ChangeHosts(xcb.HostModeDelete, xcb.FamilyServerInterpreted, "localuser\x00"+s.user.Username); err != nil {
			fmt.Println("Error removing XHost entry:", err)
		}
	}

	for _, candidate := range s.aclCleanupCandidate {
		if err := acl.UpdatePerm(candidate, s.uid); err != nil {
			fmt.Printf("Error stripping ACL entry from '%s': %s\n", candidate, err)
		}
		verbose.Printf("Stripped ACL entry for user '%s' from '%s'\n", s.user.Username, candidate)
	}

	if s.dbusProxy != nil {
		verbose.Println("D-Bus proxy registered, cleaning up")

		if err := s.dbusProxy.Close(); err != nil {
			if errors.Is(err, os.ErrClosed) {
				verbose.Println("D-Bus proxy already closed")
			} else {
				fmt.Println("Error closing D-Bus proxy:", err)
			}
		}

		// wait for Proxy.Wait to return
		<-*s.dbusDone
	}
}
