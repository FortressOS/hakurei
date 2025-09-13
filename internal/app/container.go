package app

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/internal/sys"
	"hakurei.app/system/dbus"
)

// in practice there should be less than 30 system mount points
const preallocateOpsCount = 1 << 5

// newContainer initialises [container.Params] via [hst.ContainerConfig].
// Note that remaining container setup must be queued by the caller.
func newContainer(s *hst.ContainerConfig, os sys.State, prefix string, uid, gid *int) (*container.Params, map[string]string, error) {
	if s == nil {
		return nil, nil, newWithMessage("invalid container configuration")
	}

	params := &container.Params{
		Hostname:       s.Hostname,
		SeccompFlags:   s.SeccompFlags,
		SeccompPresets: s.SeccompPresets,
		RetainSession:  s.Tty,
		HostNet:        s.HostNet,
		HostAbstract:   s.HostAbstract,

		// the container is canceled when shim is requested to exit or receives an interrupt or termination signal;
		// this behaviour is implemented in the shim
		ForwardCancel: s.WaitDelay >= 0,
	}

	as := &hst.ApplyState{
		AutoEtcPrefix: prefix,
	}
	{
		ops := make(container.Ops, 0, preallocateOpsCount+len(s.Filesystem))
		params.Ops = &ops
		as.Ops = &ops
	}

	if s.Multiarch {
		params.SeccompFlags |= seccomp.AllowMultiarch
	}

	if !s.SeccompCompat {
		params.SeccompPresets |= seccomp.PresetExt
	}
	if !s.Devel {
		params.SeccompPresets |= seccomp.PresetDenyDevel
	}
	if !s.Userns {
		params.SeccompPresets |= seccomp.PresetDenyNS
	}
	if !s.Tty {
		params.SeccompPresets |= seccomp.PresetDenyTTY
	}

	if s.MapRealUID {
		params.Uid = os.Getuid()
		*uid = params.Uid
		params.Gid = os.Getgid()
		*gid = params.Gid
	} else {
		*uid = container.OverflowUid()
		*gid = container.OverflowGid()
	}

	filesystem := s.Filesystem
	var autoroot *hst.FSBind
	// valid happens late, so root mount gets it here
	if len(filesystem) > 0 && filesystem[0].Valid() && filesystem[0].Path().String() == container.FHSRoot {
		// if the first element targets /, it is inserted early and excluded from path hiding
		rootfs := filesystem[0].FilesystemConfig
		filesystem = filesystem[1:]
		rootfs.Apply(as)

		// autoroot requires special handling during path hiding
		if b, ok := rootfs.(*hst.FSBind); ok && b.IsAutoRoot() {
			autoroot = b
		}
	}

	params.
		Proc(container.AbsFHSProc).
		Tmpfs(hst.AbsTmp, 1<<12, 0755)

	if !s.Device {
		params.DevWritable(container.AbsFHSDev, true)
	} else {
		params.Bind(container.AbsFHSDev, container.AbsFHSDev, container.BindWritable|container.BindDevice)
	}
	// /dev is mounted readonly later on, this prevents /dev/shm from going readonly with it
	params.Tmpfs(container.AbsFHSDev.Append("shm"), 0, 01777)

	/* retrieve paths and hide them if they're made available in the sandbox;

	this feature tries to improve user experience of permissive defaults, and
	to warn about issues in custom configuration; it is NOT a security feature
	and should not be treated as such, ALWAYS be careful with what you bind */
	var hidePaths []string
	sc := os.Paths()
	hidePaths = append(hidePaths, sc.RuntimePath.String(), sc.SharePath.String())
	_, systemBusAddr := dbus.Address()
	if entries, err := dbus.Parse([]byte(systemBusAddr)); err != nil {
		return nil, nil, err
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
							os.Printf("dbus socket %q is in an unusual location", pair[1])
						}
						hidePaths = append(hidePaths, dir)
					} else {
						os.Printf("dbus socket %q is not absolute", pair[1])
					}
				}
			}
		}
	}
	hidePathMatch := make([]bool, len(hidePaths))
	for i := range hidePaths {
		if err := evalSymlinks(os, &hidePaths[i]); err != nil {
			return nil, nil, err
		}
	}

	var hidePathSourceCount int
	for i, c := range filesystem {
		if !c.Valid() {
			return nil, nil, fmt.Errorf("invalid filesystem at index %d", i)
		}
		c.Apply(as)

		// fs counter
		hidePathSourceCount += len(c.Host())
	}

	// AutoRootOp is a collection of many BindMountOp internally
	var autoRootEntries []fs.DirEntry
	if autoroot != nil {
		if d, err := os.ReadDir(autoroot.Source.String()); err != nil {
			return nil, nil, err
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
			if container.IsAutoRootBindable(name) {
				hidePathSource = append(hidePathSource, autoroot.Source.Append(name))
			}
		}
	}

	// evaluated path, input path
	hidePathSourceEval := make([][2]string, len(hidePathSource))
	for i, a := range hidePathSource {
		if a == nil {
			// unreachable
			return nil, nil, syscall.ENOTRECOVERABLE
		}

		hidePathSourceEval[i] = [2]string{a.String(), a.String()}
		if err := evalSymlinks(os, &hidePathSourceEval[i][0]); err != nil {
			return nil, nil, err
		}
	}

	for _, p := range hidePathSourceEval {
		for i := range hidePaths {
			// skip matched entries
			if hidePathMatch[i] {
				continue
			}

			if ok, err := deepContainsH(p[0], hidePaths[i]); err != nil {
				return nil, nil, err
			} else if ok {
				hidePathMatch[i] = true
				os.Printf("hiding path %q from %q", hidePaths[i], p[1])
			}
		}
	}

	// cover matched paths
	for i, ok := range hidePathMatch {
		if ok {
			if a, err := container.NewAbs(hidePaths[i]); err != nil {
				var absoluteError *container.AbsoluteError
				if !errors.As(err, &absoluteError) {
					return nil, nil, err
				}
				if absoluteError == nil {
					return nil, nil, syscall.ENOTRECOVERABLE
				}
				return nil, nil, fmt.Errorf("invalid path hiding candidate %q", absoluteError.Pathname)
			} else {
				params.Tmpfs(a, 1<<13, 0755)
			}
		}
	}

	// no more ContainerConfig paths beyond this point
	if !s.Device {
		params.Remount(container.AbsFHSDev, syscall.MS_RDONLY)
	}

	return params, maps.Clone(s.Env), nil
}

func evalSymlinks(os sys.State, v *string) error {
	if p, err := os.EvalSymlinks(*v); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		os.Printf("path %q does not yet exist", *v)
	} else {
		*v = p
	}
	return nil
}
