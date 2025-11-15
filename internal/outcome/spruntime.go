package outcome

import (
	"encoding/gob"

	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/container/std"
	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/system"
)

const (
	/*
		Path to a user-private user-writable directory that is bound
		to the user login time on the machine. It is automatically
		created the first time a user logs in and removed on the
		user's final logout. If a user logs in twice at the same time,
		both sessions will see the same $XDG_RUNTIME_DIR and the same
		contents. If a user logs in once, then logs out again, and
		logs in again, the directory contents will have been lost in
		between, but applications should not rely on this behavior and
		must be able to deal with stale files. To store
		session-private data in this directory, the user should
		include the value of $XDG_SESSION_ID in the filename. This
		directory shall be used for runtime file system objects such
		as AF_UNIX sockets, FIFOs, PID files and similar. It is
		guaranteed that this directory is local and offers the
		greatest possible file system feature set the operating system
		provides. For further details, see the XDG Base Directory
		Specification[3].  $XDG_RUNTIME_DIR is not set if the current
		user is not the original user of the session.
	*/
	envXDGRuntimeDir = "XDG_RUNTIME_DIR"

	/*
		The session class. This may be used instead of class= on the
		module parameter line, and is usually preferred.
	*/
	envXDGSessionClass = "XDG_SESSION_CLASS"

	/*
		A regular interactive user session. This is the default class
		for sessions for which a TTY or X display is known at session
		registration time.
	*/
	xdgSessionClassUser = "user"

	/*
		The session type. This may be used instead of type= on the
		module parameter line, and is usually preferred.

		One of "unspecified", "tty", "x11", "wayland", "mir", or "web".
	*/
	envXDGSessionType = "XDG_SESSION_TYPE"
)

func init() { gob.Register(new(spRuntimeOp)) }

const (
	sessionTypeUnspec = iota
	sessionTypeTTY
	sessionTypeX11
	sessionTypeWayland
)

// spRuntimeOp sets up XDG_RUNTIME_DIR inside the container.
type spRuntimeOp struct {
	// SessionType determines the value of envXDGSessionType. Populated during toSystem.
	SessionType uintptr
}

func (s *spRuntimeOp) toSystem(state *outcomeStateSys) error {
	if state.Container.Flags&hst.FShareRuntime != 0 {
		runtimeDir, runtimeDirInst := s.commonPaths(state.outcomeState)
		state.sys.Ensure(runtimeDir, 0700)
		state.sys.UpdatePermType(system.User, runtimeDir, acl.Execute)
		state.sys.Ensure(runtimeDirInst, 0700)
		state.sys.UpdatePermType(system.User, runtimeDirInst, acl.Read, acl.Write, acl.Execute)
	}

	if state.et&hst.EWayland != 0 {
		s.SessionType = sessionTypeWayland
	} else if state.et&hst.EX11 != 0 {
		s.SessionType = sessionTypeX11
	} else {
		s.SessionType = sessionTypeTTY
	}

	return nil
}

func (s *spRuntimeOp) toContainer(state *outcomeStateParams) error {
	state.runtimeDir = fhs.AbsRunUser.Append(state.mapuid.String())
	state.env[envXDGRuntimeDir] = state.runtimeDir.String()
	state.env[envXDGSessionClass] = xdgSessionClassUser

	switch s.SessionType {
	case sessionTypeUnspec:
		state.env[envXDGSessionType] = "unspecified"
	case sessionTypeTTY:
		state.env[envXDGSessionType] = "tty"
	case sessionTypeX11:
		state.env[envXDGSessionType] = "x11"
	case sessionTypeWayland:
		state.env[envXDGSessionType] = "wayland"

	}

	state.params.Tmpfs(fhs.AbsRunUser, 1<<12, 0755)
	if state.Container.Flags&hst.FShareRuntime != 0 {
		_, runtimeDirInst := s.commonPaths(state.outcomeState)
		state.params.Bind(runtimeDirInst, state.runtimeDir, std.BindWritable)
	} else {
		state.params.Mkdir(state.runtimeDir, 0700)
	}
	return nil
}

func (s *spRuntimeOp) commonPaths(state *outcomeState) (runtimeDir, runtimeDirInst *check.Absolute) {
	runtimeDir = state.sc.SharePath.Append("runtime")
	runtimeDirInst = runtimeDir.Append(state.identity.String())
	return
}
