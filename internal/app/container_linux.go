package app

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/internal/hlog"
	"hakurei.app/internal/sys"
	"hakurei.app/system/dbus"
)

// in practice there should be less than 30 entries added by the runtime;
// allocating slightly more as a margin for future expansion
const preallocateOpsCount = 1 << 5

// newContainer initialises [container.Params] via [hst.ContainerConfig].
// Note that remaining container setup must be queued by the caller.
func newContainer(s *hst.ContainerConfig, os sys.State, prefix string, uid, gid *int) (*container.Params, map[string]string, error) {
	if s == nil {
		return nil, nil, hlog.WrapErr(syscall.EBADE, "invalid container configuration")
	}

	params := &container.Params{
		Hostname:       s.Hostname,
		SeccompFlags:   s.SeccompFlags,
		SeccompPresets: s.SeccompPresets,
		RetainSession:  s.Tty,
		HostNet:        s.Net,

		// the container is canceled when shim is requested to exit or receives an interrupt or termination signal;
		// this behaviour is implemented in the shim
		ForwardCancel: s.WaitDelay >= 0,
	}

	{
		ops := make(container.Ops, 0, preallocateOpsCount+len(s.Filesystem)+len(s.Link))
		params.Ops = &ops
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
		/* some programs fail to connect to dbus session running as a different uid
		so this workaround is introduced to map priv-side caller uid in container */
		params.Uid = os.Getuid()
		*uid = params.Uid
		params.Gid = os.Getgid()
		*gid = params.Gid
	} else {
		*uid = container.OverflowUid()
		*gid = container.OverflowGid()
	}

	if s.AutoRoot != nil {
		params.Root(s.AutoRoot, prefix, s.RootFlags)
	}

	params.
		Proc(container.AbsFHSProc).
		Tmpfs(hst.AbsTmp, 1<<12, 0755)

	if !s.Device {
		params.DevWritable(container.AbsFHSDev, true)
	} else {
		params.Bind(container.AbsFHSDev, container.AbsFHSDev, container.BindWritable|container.BindDevice)
	}

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
	// evaluated path, input path
	hidePathSource := make([][2]string, 0, len(s.Filesystem))

	// AutoRoot is a collection of many BindMountOp internally
	if s.AutoRoot != nil {
		if d, err := os.ReadDir(s.AutoRoot.String()); err != nil {
			return nil, nil, err
		} else {
			hidePathSource = slices.Grow(hidePathSource, len(d))
			for _, ent := range d {
				name := ent.Name()
				if container.IsAutoRootBindable(name) {
					name = path.Join(s.AutoRoot.String(), name)
					srcP := [2]string{name, name}
					if err = evalSymlinks(os, &srcP[0]); err != nil {
						return nil, nil, err
					}
					hidePathSource = append(hidePathSource, srcP)
				}
			}
		}
	}

	for i, c := range s.Filesystem {
		if c.Src == nil {
			return nil, nil, fmt.Errorf("invalid filesystem at index %d", i)
		}

		// special filesystems
		switch c.Src.String() {
		case container.Nonexistent:
			if c.Dst == nil {
				return nil, nil, errors.New("tmpfs dst must not be nil")
			}
			if c.Write {
				params.Tmpfs(c.Dst, hst.TmpfsSize, hst.TmpfsPerm)
			} else {
				params.Readonly(c.Dst, hst.TmpfsPerm)
			}
			continue
		}

		dst := c.Dst
		if dst == nil {
			dst = c.Src
		}

		p := [2]string{c.Src.String(), c.Src.String()}
		if err := evalSymlinks(os, &p[0]); err != nil {
			return nil, nil, err
		}
		hidePathSource = append(hidePathSource, p)

		var flags int
		if c.Write {
			flags |= container.BindWritable
		}
		if c.Device {
			flags |= container.BindDevice | container.BindWritable
		}
		if !c.Must {
			flags |= container.BindOptional
		}
		params.Bind(c.Src, dst, flags)
	}

	for _, p := range hidePathSource {
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

	for i, l := range s.Link {
		if l.Target == nil || l.Linkname == "" {
			return nil, nil, fmt.Errorf("invalid link at index %d", i)
		}
		linkname := l.Linkname
		var dereference bool
		if linkname[0] == '*' && path.IsAbs(linkname[1:]) {
			linkname = linkname[1:]
			dereference = true
		}
		params.Link(l.Target, linkname, dereference)
	}

	if !s.AutoEtc {
		if s.Etc != nil {
			params.Bind(s.Etc, container.AbsFHSEtc, 0)
		}
	} else {
		if s.Etc == nil {
			params.Etc(container.AbsFHSEtc, prefix)
		} else {
			params.Etc(s.Etc, prefix)
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
