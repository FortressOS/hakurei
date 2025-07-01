package common

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"syscall"

	"git.gensokyo.uk/security/hakurei/dbus"
	"git.gensokyo.uk/security/hakurei/hst"
	"git.gensokyo.uk/security/hakurei/internal/sys"
	"git.gensokyo.uk/security/hakurei/sandbox"
	"git.gensokyo.uk/security/hakurei/sandbox/seccomp"
)

// in practice there should be less than 30 entries added by the runtime;
// allocating slightly more as a margin for future expansion
const preallocateOpsCount = 1 << 5

// NewContainer initialises [sandbox.Params] via [hst.ContainerConfig].
// Note that remaining container setup must be queued by the caller.
func NewContainer(s *hst.ContainerConfig, os sys.State, uid, gid *int) (*sandbox.Params, map[string]string, error) {
	if s == nil {
		return nil, nil, syscall.EBADE
	}

	container := &sandbox.Params{
		Hostname:       s.Hostname,
		SeccompFlags:   s.SeccompFlags,
		SeccompPresets: s.SeccompPresets,
	}

	{
		ops := make(sandbox.Ops, 0, preallocateOpsCount+len(s.Filesystem)+len(s.Link)+len(s.Cover))
		container.Ops = &ops
	}

	if s.Multiarch {
		container.SeccompFlags |= seccomp.AllowMultiarch
	}

	if s.Devel {
		container.Flags |= sandbox.FAllowDevel
	}
	if s.Userns {
		container.Flags |= sandbox.FAllowUserns
	}
	if s.Net {
		container.Flags |= sandbox.FAllowNet
	}
	if s.Tty {
		container.Flags |= sandbox.FAllowTTY
	}

	if s.MapRealUID {
		/* some programs fail to connect to dbus session running as a different uid
		so this workaround is introduced to map priv-side caller uid in container */
		container.Uid = os.Getuid()
		*uid = container.Uid
		container.Gid = os.Getgid()
		*gid = container.Gid
	} else {
		*uid = sandbox.OverflowUid()
		*gid = sandbox.OverflowGid()
	}

	container.
		Proc("/proc").
		Tmpfs(hst.Tmp, 1<<12, 0755)

	if !s.Device {
		container.Dev("/dev").Mqueue("/dev/mqueue")
	} else {
		container.Bind("/dev", "/dev", sandbox.BindWritable|sandbox.BindDevice)
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
						if dir == "." || dir == "/" {
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

	for _, c := range s.Filesystem {
		if c == nil {
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

		srcH := c.Src
		if err := evalSymlinks(os, &srcH); err != nil {
			return nil, nil, err
		}

		for i := range hidePaths {
			// skip matched entries
			if hidePathMatch[i] {
				continue
			}

			if ok, err := deepContainsH(srcH, hidePaths[i]); err != nil {
				return nil, nil, err
			} else if ok {
				hidePathMatch[i] = true
				os.Printf("hiding paths from %q", c.Src)
			}
		}

		var flags int
		if c.Write {
			flags |= sandbox.BindWritable
		}
		if c.Device {
			flags |= sandbox.BindDevice | sandbox.BindWritable
		}
		if !c.Must {
			flags |= sandbox.BindOptional
		}
		container.Bind(c.Src, dest, flags)
	}

	// cover matched paths
	for i, ok := range hidePathMatch {
		if ok {
			container.Tmpfs(hidePaths[i], 1<<13, 0755)
		}
	}

	for _, l := range s.Link {
		container.Link(l[0], l[1])
	}

	return container, maps.Clone(s.Env), nil
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
