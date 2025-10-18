package app

import (
	"encoding/gob"
	"errors"
	"io/fs"
	"os"
	"path"
	"slices"
	"strconv"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/container/bits"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/message"
	"hakurei.app/system"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
)

const varRunNscd = fhs.Var + "run/nscd"

func init() { gob.Register(new(spParamsOp)) }

// spParamsOp initialises unordered fields of [container.Params] and the optional root filesystem.
// This outcomeOp is hardcoded to always run first.
type spParamsOp struct {
	// Value of $TERM, stored during toSystem.
	Term string
	// Whether $TERM is set, stored during toSystem.
	TermSet bool
}

func (s *spParamsOp) toSystem(state *outcomeStateSys) error {
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
	state.params.RetainSession = state.Container.Flags&hst.FTty != 0
	state.params.HostNet = state.Container.Flags&hst.FHostNet != 0
	state.params.HostAbstract = state.Container.Flags&hst.FHostAbstract != 0

	if state.Container.Path == nil {
		return newWithMessage("invalid program path")
	}
	state.params.Path = state.Container.Path

	if len(state.Container.Args) == 0 {
		state.params.Args = []string{state.Container.Path.String()}
	} else {
		state.params.Args = state.Container.Args
	}

	// the container is canceled when shim is requested to exit or receives an interrupt or termination signal;
	// this behaviour is implemented in the shim
	state.params.ForwardCancel = state.Shim.WaitDelay > 0

	if state.Container.Flags&hst.FMultiarch != 0 {
		state.params.SeccompFlags |= seccomp.AllowMultiarch
	}

	if state.Container.Flags&hst.FSeccompCompat == 0 {
		state.params.SeccompPresets |= bits.PresetExt
	}
	if state.Container.Flags&hst.FDevel == 0 {
		state.params.SeccompPresets |= bits.PresetDenyDevel
	}
	if state.Container.Flags&hst.FUserns == 0 {
		state.params.SeccompPresets |= bits.PresetDenyNS
	}
	if state.Container.Flags&hst.FTty == 0 {
		state.params.SeccompPresets |= bits.PresetDenyTTY
	}

	if state.Container.Flags&hst.FMapRealUID != 0 {
		state.params.Uid = state.Mapuid
		state.params.Gid = state.Mapgid
	}

	{
		state.as.AutoEtcPrefix = state.id.String()
		ops := make(container.Ops, 0, preallocateOpsCount+len(state.Container.Filesystem))
		state.params.Ops = &ops
		state.as.Ops = opsAdapter{&ops}
	}

	rootfs, filesystem, _ := resolveRoot(state.Container)
	state.filesystem = filesystem
	if rootfs != nil {
		rootfs.Apply(&state.as)
	}

	// early mount points
	state.params.
		Proc(fhs.AbsProc).
		Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755)
	if state.Container.Flags&hst.FDevice == 0 {
		state.params.DevWritable(fhs.AbsDev, true)
	} else {
		state.params.Bind(fhs.AbsDev, fhs.AbsDev, bits.BindWritable|bits.BindDevice)
	}
	// /dev is mounted readonly later on, this prevents /dev/shm from going readonly with it
	state.params.Tmpfs(fhs.AbsDev.Append("shm"), 0, 01777)

	return nil
}

func init() { gob.Register(new(spFilesystemOp)) }

// spFilesystemOp applies configured filesystems to [container.Params], excluding the optional root filesystem.
// This outcomeOp is hardcoded to always run last.
type spFilesystemOp struct {
	// Matched paths to cover. Stored during toSystem.
	HidePaths []*check.Absolute
}

func (s *spFilesystemOp) toSystem(state *outcomeStateSys) error {
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

	// dbus.Address does not go through syscallDispatcher
	systemBusAddr := dbus.FallbackSystemBusAddress
	if addr, ok := state.k.lookupEnv(dbus.SystemBusAddress); ok {
		systemBusAddr = addr
	}

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
						if dir == "." || dir == fhs.Root {
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

	hidePathSource := make([]*check.Absolute, 0, hidePathSourceCount)

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
			if a, err := check.NewAbs(hidePaths[i]); err != nil {
				return newWithMessage("invalid path hiding candidate " + strconv.Quote(hidePaths[i]))
			} else {
				s.HidePaths = append(s.HidePaths, a)
			}
		}
	}

	// append ExtraPerms last
	flattenExtraPerms(state.sys, state.extraPerms)
	return nil
}

func (s *spFilesystemOp) toContainer(state *outcomeStateParams) error {
	for i, c := range state.filesystem {
		if !c.Valid() {
			return newWithMessage("invalid filesystem at index " + strconv.Itoa(i))
		}
		c.Apply(&state.as)
	}

	for _, a := range s.HidePaths {
		state.params.Tmpfs(a, 1<<13, 0755)
	}

	// no more configured paths beyond this point
	if state.Container.Flags&hst.FDevice == 0 {
		state.params.Remount(fhs.AbsDev, syscall.MS_RDONLY)
	}
	state.params.Remount(fhs.AbsRoot, syscall.MS_RDONLY)

	state.params.Env = make([]string, 0, len(state.env))
	for key, value := range state.env {
		// key validated early via hst
		state.params.Env = append(state.params.Env, key+"="+value)
	}
	slices.Sort(state.params.Env)

	return nil
}

// resolveRoot handles the root filesystem special case for [hst.FilesystemConfig] and additionally resolves autoroot
// as it requires special handling during path hiding.
func resolveRoot(c *hst.ContainerConfig) (rootfs hst.FilesystemConfig, filesystem []hst.FilesystemConfigJSON, autoroot *hst.FSBind) {
	// root filesystem special case
	filesystem = c.Filesystem
	// valid happens late, so root gets it here
	if len(filesystem) > 0 && filesystem[0].Valid() && filesystem[0].Path().String() == fhs.Root {
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
func evalSymlinks(msg message.Msg, k syscallDispatcher, v *string) error {
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

// flattenExtraPerms expands a slice of [hst.ExtraPermConfig] into [system.I].
func flattenExtraPerms(sys *system.I, extraPerms []hst.ExtraPermConfig) {
	for i := range extraPerms {
		p := &extraPerms[i]
		if p.Path == nil {
			continue
		}

		if p.Ensure {
			sys.Ensure(p.Path, 0700)
		}

		perms := make(acl.Perms, 0, 3)
		if p.Read {
			perms = append(perms, acl.Read)
		}
		if p.Write {
			perms = append(perms, acl.Write)
		}
		if p.Execute {
			perms = append(perms, acl.Execute)
		}
		sys.UpdatePermType(system.User, p.Path, perms...)
	}
}

// opsAdapter implements [hst.Ops] on [container.Ops].
type opsAdapter struct{ *container.Ops }

func (p opsAdapter) Tmpfs(target *check.Absolute, size int, perm os.FileMode) hst.Ops {
	return opsAdapter{p.Ops.Tmpfs(target, size, perm)}
}

func (p opsAdapter) Readonly(target *check.Absolute, perm os.FileMode) hst.Ops {
	return opsAdapter{p.Ops.Readonly(target, perm)}
}

func (p opsAdapter) Bind(source, target *check.Absolute, flags int) hst.Ops {
	return opsAdapter{p.Ops.Bind(source, target, flags)}
}

func (p opsAdapter) Overlay(target, state, work *check.Absolute, layers ...*check.Absolute) hst.Ops {
	return opsAdapter{p.Ops.Overlay(target, state, work, layers...)}
}

func (p opsAdapter) OverlayReadonly(target *check.Absolute, layers ...*check.Absolute) hst.Ops {
	return opsAdapter{p.Ops.OverlayReadonly(target, layers...)}
}

func (p opsAdapter) Link(target *check.Absolute, linkName string, dereference bool) hst.Ops {
	return opsAdapter{p.Ops.Link(target, linkName, dereference)}
}

func (p opsAdapter) Root(host *check.Absolute, flags int) hst.Ops {
	return opsAdapter{p.Ops.Root(host, flags)}
}

func (p opsAdapter) Etc(host *check.Absolute, prefix string) hst.Ops {
	return opsAdapter{p.Ops.Etc(host, prefix)}
}
