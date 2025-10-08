package app

import (
	"strconv"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func newInt(v int) *stringPair[int] { return &stringPair[int]{v, strconv.Itoa(v)} }

// stringPair stores a value and its string representation.
type stringPair[T comparable] struct {
	v T
	s string
}

func (s *stringPair[T]) unwrap() T      { return s.v }
func (s *stringPair[T]) String() string { return s.s }

// outcomeState is copied to the shim process and available while applying outcomeOp.
// This is transmitted from the priv side to the shim, so exported fields should be kept to a minimum.
type outcomeState struct {
	// Generated and accounted for by the caller.
	ID *state.ID
	// Copied from ID.
	id *stringPair[state.ID]

	// Copied from the [hst.Config] field of the same name.
	Identity int
	// Copied from Identity.
	identity *stringPair[int]
	// Returned by [Hsu.MustIDMsg].
	UserID int
	// Target init namespace uid resolved from UserID and identity.
	uid *stringPair[int]

	// Included as part of [hst.Config], transmitted as-is unless permissive defaults.
	Container *hst.ContainerConfig

	// Mapped credentials within container user namespace.
	Mapuid, Mapgid int
	// Copied from their respective exported values.
	mapuid, mapgid *stringPair[int]

	// Copied from [EnvPaths] per-process.
	sc hst.Paths
	*EnvPaths

	// Matched paths to cover. Populated by spFilesystemOp.
	HidePaths []*check.Absolute

	// Copied via populateLocal.
	k syscallDispatcher
	// Copied via populateLocal.
	msg container.Msg
}

// valid checks outcomeState to be safe for use with outcomeOp.
func (s *outcomeState) valid() bool {
	return s != nil &&
		s.ID != nil &&
		s.Container != nil &&
		s.EnvPaths != nil
}

// populateEarly populates exported fields via syscallDispatcher.
// This must only be called from the priv side.
func (s *outcomeState) populateEarly(k syscallDispatcher, msg container.Msg) (waitDelay time.Duration) {
	// enforce bounds and default early
	if s.Container.WaitDelay <= 0 {
		waitDelay = hst.WaitDelayDefault
	} else if s.Container.WaitDelay > hst.WaitDelayMax {
		waitDelay = hst.WaitDelayMax
	} else {
		waitDelay = s.Container.WaitDelay
	}

	if s.Container.MapRealUID {
		s.Mapuid, s.Mapgid = k.getuid(), k.getgid()
	} else {
		s.Mapuid, s.Mapgid = k.overflowUid(msg), k.overflowGid(msg)
	}

	return
}

// populateLocal populates unexported fields from transmitted exported fields.
// These fields are cheaper to recompute per-process.
func (s *outcomeState) populateLocal(k syscallDispatcher, msg container.Msg) error {
	if !s.valid() || k == nil || msg == nil {
		return newWithMessage("impossible outcome state reached")
	}

	if s.k != nil || s.msg != nil {
		panic("attempting to call populateLocal twice")
	}
	s.k = k
	s.msg = msg

	s.id = &stringPair[state.ID]{*s.ID, s.ID.String()}

	s.Copy(&s.sc, s.UserID)
	msg.Verbosef("process share directory at %q, runtime directory at %q", s.sc.SharePath, s.sc.RunDirPath)

	s.identity = newInt(s.Identity)
	s.mapuid, s.mapgid = newInt(s.Mapuid), newInt(s.Mapgid)
	s.uid = newInt(HsuUid(s.UserID, s.identity.unwrap()))

	return nil
}

// instancePath returns a path formatted for outcomeStateSys.instance.
// This method must only be called from outcomeOp.toContainer if
// outcomeOp.toSystem has already called outcomeStateSys.instance.
func (s *outcomeState) instancePath() *check.Absolute {
	return s.sc.SharePath.Append(s.id.String())
}

// runtimePath returns a path formatted for outcomeStateSys.runtime.
// This method must only be called from outcomeOp.toContainer if
// outcomeOp.toSystem has already called outcomeStateSys.runtime.
func (s *outcomeState) runtimePath() *check.Absolute {
	return s.sc.RunDirPath.Append(s.id.String())
}

// outcomeStateSys wraps outcomeState and [system.I]. Used on the priv side only.
// Implementations of outcomeOp must not access fields other than sys unless explicitly stated.
type outcomeStateSys struct {
	// Whether XDG_RUNTIME_DIR is used post hsu.
	useRuntimeDir bool
	// Process-specific directory in TMPDIR, nil if unused.
	sharePath *check.Absolute
	// Process-specific directory in XDG_RUNTIME_DIR, nil if unused.
	runtimeSharePath *check.Absolute

	sys *system.I
	*outcomeState
}

// ensureRuntimeDir must be called if access to paths within XDG_RUNTIME_DIR is required.
func (state *outcomeStateSys) ensureRuntimeDir() {
	if state.useRuntimeDir {
		return
	}
	state.useRuntimeDir = true
	state.sys.Ensure(state.sc.RunDirPath, 0700)
	state.sys.UpdatePermType(system.User, state.sc.RunDirPath, acl.Execute)
	state.sys.Ensure(state.sc.RuntimePath, 0700) // ensure this dir in case XDG_RUNTIME_DIR is unset
	state.sys.UpdatePermType(system.User, state.sc.RuntimePath, acl.Execute)
}

// instance returns the pathname to a process-specific directory within TMPDIR.
// This directory must only hold entries bound to [system.Process].
func (state *outcomeStateSys) instance() *check.Absolute {
	if state.sharePath != nil {
		return state.sharePath
	}
	state.sharePath = state.instancePath()
	state.sys.Ephemeral(system.Process, state.sharePath, 0711)
	return state.sharePath
}

// runtime returns the pathname to a process-specific directory within XDG_RUNTIME_DIR.
// This directory must only hold entries bound to [system.Process].
func (state *outcomeStateSys) runtime() *check.Absolute {
	if state.runtimeSharePath != nil {
		return state.runtimeSharePath
	}
	state.ensureRuntimeDir()
	state.runtimeSharePath = state.runtimePath()
	state.sys.Ephemeral(system.Process, state.runtimeSharePath, 0700)
	state.sys.UpdatePerm(state.runtimeSharePath, acl.Execute)
	return state.runtimeSharePath
}

// outcomeStateParams wraps outcomeState and [container.Params]. Used on the shim side only.
type outcomeStateParams struct {
	// Overrides the embedded [container.Params] in [container.Container]. The Env field must not be used.
	params *container.Params
	// Collapsed into the Env slice in [container.Params] by the final outcomeOp.
	env map[string]string

	// Filesystems with the optional root sliced off if present. Populated by spParamsOp.
	// Safe for use by spFilesystemOp.
	filesystem []hst.FilesystemConfigJSON

	// Inner XDG_RUNTIME_DIR default formatting of `/run/user/%d` via mapped uid.
	// Populated by spRuntimeOp.
	runtimeDir *check.Absolute

	as hst.ApplyState
	*outcomeState
}

// TODO(ophestra): register outcomeOp implementations (params to shim)

// An outcomeOp inflicts an outcome on [system.I] and contains enough information to
// inflict it on [container.Params] in a separate process.
// An implementation of outcomeOp must store cross-process states in exported fields only.
type outcomeOp interface {
	// toSystem inflicts the current outcome on [system.I] in the priv side process.
	toSystem(state *outcomeStateSys, config *hst.Config) error

	// toContainer inflicts the current outcome on [container.Params] in the shim process.
	// The implementation must not write to the Env field of [container.Params] as it will be overwritten
	// by flattened env map.
	toContainer(state *outcomeStateParams) error
}
