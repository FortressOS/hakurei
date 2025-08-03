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
		return nil, nil, syscall.EBADE
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

	if s.AutoRoot != "" {
		if !path.IsAbs(s.AutoRoot) {
			return nil, nil, fmt.Errorf("auto root target %q not absolute", s.AutoRoot)
		}
		params.Root(s.AutoRoot, prefix, s.RootFlags)
	}

	params.
		Proc(container.FHSProc).
		Tmpfs(hst.Tmp, 1<<12, 0755)

	if !s.Device {
		params.DevWritable(container.FHSDev, true)
	} else {
		params.Bind(container.FHSDev, container.FHSDev, container.BindWritable|container.BindDevice)
	}

	/* retrieve paths and hide them if they're made available in the sandbox;
	this feature tries to improve user experience of permissive defaults, and
	to warn about issues in custom configuration; it is NOT a security feature
	and should not be treated as such, ALWAYS be careful with what you bind */
	var hidePaths []string
	sc := os.Paths()
	hidePaths = append(hidePaths, sc.RuntimePath, sc.SharePath)
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
	if s.AutoRoot != "" {
		if d, err := os.ReadDir(s.AutoRoot); err != nil {
			return nil, nil, err
		} else {
			hidePathSource = slices.Grow(hidePathSource, len(d))
			for _, ent := range d {
				name := ent.Name()
				if container.IsAutoRootBindable(name) {
					name = path.Join(s.AutoRoot, name)
					srcP := [2]string{name, name}
					if err = evalSymlinks(os, &srcP[0]); err != nil {
						return nil, nil, err
					}
					hidePathSource = append(hidePathSource, srcP)
				}
			}
		}
	}

	for _, c := range s.Filesystem {
		if c == nil {
			continue
		}

		// special filesystems
		switch c.Src {
		case hst.SourceTmpfs:
			if !path.IsAbs(c.Dst) {
				return nil, nil, fmt.Errorf("tmpfs dst %q is not absolute", c.Dst)
			}
			if c.Write {
				params.Tmpfs(c.Dst, hst.TmpfsSize, hst.TmpfsPerm)
			} else {
				params.Readonly(c.Dst, hst.TmpfsPerm)
			}
			continue
		}

		if !path.IsAbs(c.Src) {
			return nil, nil, fmt.Errorf("src path %q is not absolute", c.Src)
		}

		dest := c.Dst
		if c.Dst == "" {
			dest = c.Src
		} else if !path.IsAbs(dest) {
			return nil, nil, fmt.Errorf("dst path %q is not absolute", dest)
		}

		p := [2]string{c.Src, c.Src}
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
		params.Bind(c.Src, dest, flags)
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
			params.Tmpfs(hidePaths[i], 1<<13, 0755)
		}
	}

	for _, l := range s.Link {
		params.Link(l[0], l[1])
	}

	if !s.AutoEtc {
		if s.Etc != "" {
			params.Bind(s.Etc, container.FHSEtc, 0)
		}
	} else {
		etcPath := s.Etc
		if etcPath == "" {
			etcPath = container.FHSEtc
		}
		params.Etc(etcPath, prefix)
	}

	// no more ContainerConfig paths beyond this point
	if !s.Device {
		params.Remount(container.FHSDev, syscall.MS_RDONLY)
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
