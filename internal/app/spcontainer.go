package app

import (
	"errors"
	"io/fs"
	"path"
	"strconv"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/system/dbus"
)

const varRunNscd = container.FHSVar + "run/nscd"

// spParamsOp initialises unordered fields of [container.Params] and the optional root filesystem.
// This outcomeOp is hardcoded to always run first.
type spParamsOp struct {
	// Copied from the [hst.Config] field of the same name.
	Path *container.Absolute `json:"path,omitempty"`
	// Copied from the [hst.Config] field of the same name.
	Args []string `json:"args"`

	// Value of $TERM, stored during toSystem.
	Term string
	// Whether $TERM is set, stored during toSystem.
	TermSet bool
}

func (s *spParamsOp) toSystem(state *outcomeStateSys, _ *hst.Config) error {
	s.Term, s.TermSet = state.k.lookupEnv("TERM")
	state.sys.Ensure(state.sc.SharePath, 0711)
	return nil
}

func (s *spParamsOp) toContainer(state *outcomeStateParams) error {
	// pass $TERM for proper terminal I/O in initial process
	if s.TermSet {
		state.env["TERM"] = s.Term
	}

	// in practice there should be less than 30 system mount points
	const preallocateOpsCount = 1 << 5

	state.params.Hostname = state.Container.Hostname
	state.params.RetainSession = state.Container.Tty
	state.params.HostNet = state.Container.HostNet
	state.params.HostAbstract = state.Container.HostAbstract

	if s.Path == nil {
		return newWithMessage("invalid program path")
	}
	state.params.Path = s.Path

	if len(s.Args) == 0 {
		state.params.Args = []string{s.Path.String()}
	} else {
		state.params.Args = s.Args
	}

	// the container is canceled when shim is requested to exit or receives an interrupt or termination signal;
	// this behaviour is implemented in the shim
	state.params.ForwardCancel = state.Container.WaitDelay >= 0

	if state.Container.Multiarch {
		state.params.SeccompFlags |= seccomp.AllowMultiarch
	}

	if !state.Container.SeccompCompat {
		state.params.SeccompPresets |= seccomp.PresetExt
	}
	if !state.Container.Devel {
		state.params.SeccompPresets |= seccomp.PresetDenyDevel
	}
	if !state.Container.Userns {
		state.params.SeccompPresets |= seccomp.PresetDenyNS
	}
	if !state.Container.Tty {
		state.params.SeccompPresets |= seccomp.PresetDenyTTY
	}

	if state.Container.MapRealUID {
		state.params.Uid = state.Mapuid
		state.params.Gid = state.Mapgid
	}

	{
		state.as.AutoEtcPrefix = state.id.String()
		ops := make(container.Ops, 0, preallocateOpsCount+len(state.Container.Filesystem))
		state.params.Ops = &ops
		state.as.Ops = &ops
	}

	rootfs, filesystem, _ := resolveRoot(state.Container)
	state.filesystem = filesystem
	if rootfs != nil {
		rootfs.Apply(&state.as)
	}

	// early mount points
	state.params.
		Proc(container.AbsFHSProc).
		Tmpfs(hst.AbsTmp, 1<<12, 0755)
	if !state.Container.Device {
		state.params.DevWritable(container.AbsFHSDev, true)
	} else {
		state.params.Bind(container.AbsFHSDev, container.AbsFHSDev, container.BindWritable|container.BindDevice)
	}
	// /dev is mounted readonly later on, this prevents /dev/shm from going readonly with it
	state.params.Tmpfs(container.AbsFHSDev.Append("shm"), 0, 01777)

	return nil
}

// spFilesystemOp applies configured filesystems to [container.Params], excluding the optional root filesystem.
type spFilesystemOp struct{}

func (s spFilesystemOp) toSystem(state *outcomeStateSys, _ *hst.Config) error {
	/* retrieve paths and hide them if they're made available in the sandbox;

	this feature tries to improve user experience of permissive defaults, and
	to warn about issues in custom configuration; it is NOT a security feature
	and should not be treated as such, ALWAYS be careful with what you bind */
	hidePaths := []string{
		state.sc.RuntimePath.String(),
		state.sc.SharePath.String(),

		// this causes emulated passwd database to be bypassed on some /etc/ setups
		varRunNscd,
	}

	_, systemBusAddr := dbus.Address()
	if entries, err := dbus.Parse([]byte(systemBusAddr)); err != nil {
		return &hst.AppError{Step: "parse dbus address", Err: err}
	} else {
		// there is usually only one, do not preallocate
		for _, entry := range entries {
			if entry.Method != "unix" {
				continue
			}
			for _, pair := range entry.Values {
				if pair[0] == "path" {
					if path.IsAbs(pair[1]) {
						// get parent dir of socket
						dir := path.Dir(pair[1])
						if dir == "." || dir == container.FHSRoot {
							state.msg.Verbosef("dbus socket %q is in an unusual location", pair[1])
						}
						hidePaths = append(hidePaths, dir)
					} else {
						state.msg.Verbosef("dbus socket %q is not absolute", pair[1])
					}
				}
			}
		}
	}
	hidePathMatch := make([]bool, len(hidePaths))
	for i := range hidePaths {
		if err := evalSymlinks(state.msg, state.k, &hidePaths[i]); err != nil {
			return &hst.AppError{Step: "evaluate path hiding target", Err: err}
		}
	}

	_, filesystem, autoroot := resolveRoot(state.Container)

	var hidePathSourceCount int
	for i, c := range filesystem {
		if !c.Valid() {
			return newWithMessage("invalid filesystem at index " + strconv.Itoa(i))
		}

		// fs counter
		hidePathSourceCount += len(c.Host())
	}

	// AutoRootOp is a collection of many BindMountOp internally
	var autoRootEntries []fs.DirEntry
	if autoroot != nil {
		if d, err := state.k.readdir(autoroot.Source.String()); err != nil {
			return &hst.AppError{Step: "access autoroot source", Err: err}
		} else {
			// autoroot counter
			hidePathSourceCount += len(d)
			autoRootEntries = d
		}
	}

	hidePathSource := make([]*container.Absolute, 0, hidePathSourceCount)

	// fs append
	for _, c := range filesystem {
		// all entries already checked above
		hidePathSource = append(hidePathSource, c.Host()...)
	}

	// autoroot append
	if autoroot != nil {
		for _, ent := range autoRootEntries {
			name := ent.Name()
			if container.IsAutoRootBindable(state.msg, name) {
				hidePathSource = append(hidePathSource, autoroot.Source.Append(name))
			}
		}
	}

	// evaluated path, input path
	hidePathSourceEval := make([][2]string, len(hidePathSource))
	for i, a := range hidePathSource {
		if a == nil {
			// unreachable
			return newWithMessage("impossible path hiding state reached")
		}

		hidePathSourceEval[i] = [2]string{a.String(), a.String()}
		if err := evalSymlinks(state.msg, state.k, &hidePathSourceEval[i][0]); err != nil {
			return &hst.AppError{Step: "evaluate path hiding source", Err: err}
		}
	}

	for _, p := range hidePathSourceEval {
		for i := range hidePaths {
			// skip matched entries
			if hidePathMatch[i] {
				continue
			}

			if ok, err := deepContainsH(p[0], hidePaths[i]); err != nil {
				return &hst.AppError{Step: "determine path hiding outcome", Err: err}
			} else if ok {
				hidePathMatch[i] = true
				state.msg.Verbosef("hiding path %q from %q", hidePaths[i], p[1])
			}
		}
	}

	// copy matched paths for shim
	for i, ok := range hidePathMatch {
		if ok {
			if a, err := container.NewAbs(hidePaths[i]); err != nil {
				var absoluteError *container.AbsoluteError
				if !errors.As(err, &absoluteError) {
					return newWithMessageError(absoluteError.Error(), absoluteError)
				}
				if absoluteError == nil {
					return newWithMessage("impossible path checking state reached")
				}
				return newWithMessage("invalid path hiding candidate " + strconv.Quote(absoluteError.Pathname))
			} else {
				state.HidePaths = append(state.HidePaths, a)
			}
		}
	}

	return nil
}

func (s spFilesystemOp) toContainer(state *outcomeStateParams) error {
	for i, c := range state.filesystem {
		if !c.Valid() {
			return newWithMessage("invalid filesystem at index " + strconv.Itoa(i))
		}
		c.Apply(&state.as)
	}

	for _, a := range state.HidePaths {
		state.params.Tmpfs(a, 1<<13, 0755)
	}

	// no more configured paths beyond this point
	if !state.Container.Device {
		state.params.Remount(container.AbsFHSDev, syscall.MS_RDONLY)
	}
	return nil
}

// resolveRoot handles the root filesystem special case for [hst.FilesystemConfig] and additionally resolves autoroot
// as it requires special handling during path hiding.
func resolveRoot(c *hst.ContainerConfig) (rootfs hst.FilesystemConfig, filesystem []hst.FilesystemConfigJSON, autoroot *hst.FSBind) {
	// root filesystem special case
	filesystem = c.Filesystem
	// valid happens late, so root gets it here
	if len(filesystem) > 0 && filesystem[0].Valid() && filesystem[0].Path().String() == container.FHSRoot {
		// if the first element targets /, it is inserted early and excluded from path hiding
		rootfs = filesystem[0].FilesystemConfig
		filesystem = filesystem[1:]

		// autoroot requires special handling during path hiding
		if b, ok := rootfs.(*hst.FSBind); ok && b.IsAutoRoot() {
			autoroot = b
		}
	}
	return
}

// evalSymlinks calls syscallDispatcher.evalSymlinks but discards errors unwrapping to [fs.ErrNotExist].
func evalSymlinks(msg container.Msg, k syscallDispatcher, v *string) error {
	if p, err := k.evalSymlinks(*v); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		msg.Verbosef("path %q does not yet exist", *v)
	} else {
		*v = p
	}
	return nil
}
