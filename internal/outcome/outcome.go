// Package outcome implements the outcome of the privileged and container sides of a hakurei container.
package outcome

import (
	"errors"
	"maps"
	"strconv"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/env"
	"hakurei.app/message"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

// envAllocSize is the initial size of the env map pre-allocated when the configured env map is nil.
// It should be large enough to fit all insertions by outcomeOp.toContainer.
const envAllocSize = 1 << 6

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
	// Params only used by the shim process. Populated by populateEarly.
	Shim *shimParams

	// Generated and accounted for by the caller.
	ID *hst.ID
	// Copied from ID.
	id *stringPair[hst.ID]

	// Copied from the [hst.Config] field of the same name.
	Identity int
	// Copied from Identity.
	identity *stringPair[int]
	// Returned by [Hsu.MustID].
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
	*env.Paths

	// Copied via populateLocal.
	k syscallDispatcher
	// Copied via populateLocal.
	msg message.Msg
}

// valid checks outcomeState to be safe for use with outcomeOp.
func (s *outcomeState) valid() bool {
	return s != nil &&
		s.Shim.valid() &&
		s.ID != nil &&
		s.Container != nil &&
		s.Paths != nil
}

// newOutcomeState returns the address of a new outcomeState with its exported fields populated via syscallDispatcher.
func newOutcomeState(k syscallDispatcher, msg message.Msg, id *hst.ID, config *hst.Config, hsu *Hsu) *outcomeState {
	s := outcomeState{
		Shim:      &shimParams{PrivPID: k.getpid(), Verbose: msg.IsVerbose()},
		ID:        id,
		Identity:  config.Identity,
		UserID:    hsu.MustID(msg),
		Paths:     env.CopyPathsFunc(k.fatalf, k.tempdir, func(key string) string { v, _ := k.lookupEnv(key); return v }),
		Container: config.Container,
	}

	// enforce bounds and default early
	if s.Container.WaitDelay < 0 {
		s.Shim.WaitDelay = 0
	} else if s.Container.WaitDelay == 0 {
		s.Shim.WaitDelay = hst.WaitDelayDefault
	} else if s.Container.WaitDelay > hst.WaitDelayMax {
		s.Shim.WaitDelay = hst.WaitDelayMax
	} else {
		s.Shim.WaitDelay = s.Container.WaitDelay
	}

	if s.Container.Flags&hst.FMapRealUID != 0 {
		s.Mapuid, s.Mapgid = k.getuid(), k.getgid()
	} else {
		s.Mapuid, s.Mapgid = k.overflowUid(msg), k.overflowGid(msg)
	}

	return &s
}

// populateLocal populates unexported fields from transmitted exported fields.
// These fields are cheaper to recompute per-process.
func (s *outcomeState) populateLocal(k syscallDispatcher, msg message.Msg) error {
	if !s.valid() || k == nil || msg == nil {
		return newWithMessage("impossible outcome state reached")
	}

	if s.k != nil || s.msg != nil {
		panic("attempting to call populateLocal twice")
	}
	s.k = k
	s.msg = msg

	s.id = &stringPair[hst.ID]{*s.ID, s.ID.String()}

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
func (s *outcomeState) instancePath() *check.Absolute { return s.sc.SharePath.Append(s.id.String()) }

// runtimePath returns a path formatted for outcomeStateSys.runtime.
// This method must only be called from outcomeOp.toContainer if
// outcomeOp.toSystem has already called outcomeStateSys.runtime.
func (s *outcomeState) runtimePath() *check.Absolute { return s.sc.RunDirPath.Append(s.id.String()) }

// outcomeStateSys wraps outcomeState and [system.I]. Used on the priv side only.
// Implementations of outcomeOp must not access fields other than sys unless explicitly stated.
type outcomeStateSys struct {
	// Whether XDG_RUNTIME_DIR is used post hsu.
	useRuntimeDir bool
	// Process-specific directory in TMPDIR, nil if unused.
	sharePath *check.Absolute
	// Process-specific directory in XDG_RUNTIME_DIR, nil if unused.
	runtimeSharePath *check.Absolute

	// Copied from [hst.Config]. Safe for read by outcomeOp.toSystem.
	appId string
	// Copied from [hst.Config]. Safe for read by outcomeOp.toSystem.
	et hst.Enablement

	// Copied from [hst.Config]. Safe for read by spWaylandOp.toSystem only.
	directWayland bool
	// Copied header from [hst.Config]. Safe for read by spFilesystemOp.toSystem only.
	extraPerms []hst.ExtraPermConfig
	// Copied address from [hst.Config]. Safe for read by spDBusOp.toSystem only.
	sessionBus, systemBus *hst.BusConfig

	sys *system.I
	*outcomeState
}

// newSys returns the address of a new outcomeStateSys embedding the current outcomeState.
func (s *outcomeState) newSys(config *hst.Config, sys *system.I) *outcomeStateSys {
	return &outcomeStateSys{
		appId: config.ID, et: config.Enablements.Unwrap(),
		directWayland: config.DirectWayland, extraPerms: config.ExtraPerms,
		sessionBus: config.SessionBus, systemBus: config.SystemBus,
		sys: sys, outcomeState: s,
	}
}

// newParams returns the address of a new outcomeStateParams embedding the current outcomeState.
func (s *outcomeState) newParams() *outcomeStateParams {
	stateParams := outcomeStateParams{params: new(container.Params), outcomeState: s}
	if s.Container.Env == nil {
		stateParams.env = make(map[string]string, envAllocSize)
	} else {
		stateParams.env = maps.Clone(s.Container.Env)
	}
	return &stateParams
}

// ensureRuntimeDir must be called if access to paths within XDG_RUNTIME_DIR is required.
func (state *outcomeStateSys) ensureRuntimeDir() {
	if state.useRuntimeDir {
		return
	}
	state.useRuntimeDir = true
	state.sys.
		// ensure this dir in case XDG_RUNTIME_DIR is unset
		Ensure(state.sc.RuntimePath, 0700).UpdatePermType(system.User, state.sc.RuntimePath, acl.Execute).
		Ensure(state.sc.RunDirPath, 0700).UpdatePermType(system.User, state.sc.RunDirPath, acl.Execute)
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

// errNotEnabled is returned by outcomeOp.toSystem and used internally to exclude an outcomeOp from transmission.
var errNotEnabled = errors.New("op not enabled in the configuration")

// An outcomeOp inflicts an outcome on [system.I] and contains enough information to
// inflict it on [container.Params] in a separate process.
// An implementation of outcomeOp must store cross-process states in exported fields only.
type outcomeOp interface {
	// toSystem inflicts the current outcome on [system.I] in the priv side process.
	toSystem(state *outcomeStateSys) error

	// toContainer inflicts the current outcome on [container.Params] in the shim process.
	// The implementation must not write to the Env field of [container.Params] as it will be overwritten
	// by flattened env map.
	toContainer(state *outcomeStateParams) error
}

// toSystem calls the outcomeOp.toSystem method on all outcomeOp implementations and populates shimParams.Ops.
// This function assumes the caller has already called the Validate method on [hst.Config]
// and checked that it returns nil.
func (state *outcomeStateSys) toSystem() error {
	if state.Shim == nil || state.Shim.Ops != nil {
		return newWithMessage("invalid ops state reached")
	}

	ops := [...]outcomeOp{
		// must run first
		&spParamsOp{},

		&spRuntimeOp{},
		spTmpdirOp{},
		spAccountOp{},

		// optional via enablements
		&spWaylandOp{},
		&spX11Op{},
		&spPulseOp{},
		&spDBusOp{},

		// must run last
		&spFilesystemOp{},
	}

	state.Shim.Ops = make([]outcomeOp, 0, len(ops))
	for _, op := range ops {
		if err := op.toSystem(state); err != nil {
			// this error is used internally to exclude this outcomeOp from transmission
			if errors.Is(err, errNotEnabled) {
				continue
			}

			return err
		}
		state.Shim.Ops = append(state.Shim.Ops, op)
	}
	return nil
}
