package bwrap

import "os"

/*
Bind binds mount src on host to dest in sandbox.

Bind(src, dest) bind mount host path readonly on sandbox
(--ro-bind SRC DEST).
Bind(src, dest, true) equal to ROBind but ignores non-existent host path
(--ro-bind-try SRC DEST).

Bind(src, dest, false, true) bind mount host path on sandbox.
(--bind SRC DEST).
Bind(src, dest, true, true) equal to Bind but ignores non-existent host path
(--bind-try SRC DEST).

Bind(src, dest, false, true, true) bind mount host path on sandbox, allowing device access
(--dev-bind SRC DEST).
Bind(src, dest, true, true, true) equal to DevBind but ignores non-existent host path
(--dev-bind-try SRC DEST).
*/
func (c *Config) Bind(src, dest string, opts ...bool) *Config {
	var (
		try   bool
		write bool
		dev   bool
	)

	if len(opts) > 0 {
		try = opts[0]
	}
	if len(opts) > 1 {
		write = opts[1]
	}
	if len(opts) > 2 {
		dev = opts[2]
	}

	if dev {
		if try {
			c.Filesystem = append(c.Filesystem, &pairF{DevBindTry.Unwrap(), src, dest})
		} else {
			c.Filesystem = append(c.Filesystem, &pairF{DevBind.Unwrap(), src, dest})
		}
		return c
	} else if write {
		if try {
			c.Filesystem = append(c.Filesystem, &pairF{BindTry.Unwrap(), src, dest})
		} else {
			c.Filesystem = append(c.Filesystem, &pairF{Bind.Unwrap(), src, dest})
		}
		return c
	} else {
		if try {
			c.Filesystem = append(c.Filesystem, &pairF{ROBindTry.Unwrap(), src, dest})
		} else {
			c.Filesystem = append(c.Filesystem, &pairF{ROBind.Unwrap(), src, dest})
		}
		return c
	}
}

// Dir create dir in sandbox
// (--dir DEST)
func (c *Config) Dir(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{Dir.Unwrap(), dest})
	return c
}

// RemountRO remount path as readonly; does not recursively remount
// (--remount-ro DEST)
func (c *Config) RemountRO(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{RemountRO.Unwrap(), dest})
	return c
}

// Procfs mount new procfs in sandbox
// (--proc DEST)
func (c *Config) Procfs(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{Procfs.Unwrap(), dest})
	return c
}

// DevTmpfs mount new dev in sandbox
// (--dev DEST)
func (c *Config) DevTmpfs(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{DevTmpfs.Unwrap(), dest})
	return c
}

// Mqueue mount new mqueue in sandbox
// (--mqueue DEST)
func (c *Config) Mqueue(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{Mqueue.Unwrap(), dest})
	return c
}

// Tmpfs mount new tmpfs in sandbox
// (--tmpfs DEST)
func (c *Config) Tmpfs(dest string, size int, perm ...os.FileMode) *Config {
	tmpfs := &PermConfig[*TmpfsConfig]{Inner: &TmpfsConfig{Dir: dest}}
	if size >= 0 {
		tmpfs.Inner.Size = size
	}
	if len(perm) == 1 {
		tmpfs.Mode = &perm[0]
	}
	c.Filesystem = append(c.Filesystem, tmpfs)
	return c
}

// Overlay mount overlayfs on DEST, with writes going to an invisible tmpfs
// (--tmp-overlay DEST)
func (c *Config) Overlay(dest string, src ...string) *Config {
	c.Filesystem = append(c.Filesystem, &OverlayConfig{Src: src, Dest: dest})
	return c
}

// Join mount overlayfs read-only on DEST
// (--ro-overlay DEST)
func (c *Config) Join(dest string, src ...string) *Config {
	c.Filesystem = append(c.Filesystem, &OverlayConfig{Src: src, Dest: dest, Persist: new([2]string)})
	return c
}

// Persist mount overlayfs on DEST, with RWSRC as the host path for writes and
// WORKDIR an empty directory on the same filesystem as RWSRC
// (--overlay RWSRC WORKDIR DEST)
func (c *Config) Persist(dest, rwsrc, workdir string, src ...string) *Config {
	if rwsrc == "" || workdir == "" {
		panic("persist called without required paths")
	}
	c.Filesystem = append(c.Filesystem, &OverlayConfig{Src: src, Dest: dest, Persist: &[2]string{rwsrc, workdir}})
	return c
}

// Symlink create symlink within sandbox
// (--symlink SRC DEST)
func (c *Config) Symlink(src, dest string, perm ...os.FileMode) *Config {
	symlink := &PermConfig[SymlinkConfig]{Inner: SymlinkConfig{src, dest}}
	if len(perm) == 1 {
		symlink.Mode = &perm[0]
	}
	c.Filesystem = append(c.Filesystem, symlink)
	return c
}

// SetUID sets custom uid in the sandbox, requires new user namespace (--uid UID).
func (c *Config) SetUID(uid int) *Config {
	if uid >= 0 {
		c.UID = &uid
	}
	return c
}

// SetGID sets custom gid in the sandbox, requires new user namespace (--gid GID).
func (c *Config) SetGID(gid int) *Config {
	if gid >= 0 {
		c.GID = &gid
	}
	return c
}

// SetSync sets the sync pipe kept open while sandbox is running
// (--sync-fd FD)
func (c *Config) SetSync(s *os.File) *Config {
	c.sync = s
	return c
}
