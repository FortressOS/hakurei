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
			c.Filesystem = append(c.Filesystem, &pairF{pairArgs[DevBindTry], src, dest})
		} else {
			c.Filesystem = append(c.Filesystem, &pairF{pairArgs[DevBind], src, dest})
		}
		return c
	} else if write {
		if try {
			c.Filesystem = append(c.Filesystem, &pairF{pairArgs[BindTry], src, dest})
		} else {
			c.Filesystem = append(c.Filesystem, &pairF{pairArgs[Bind], src, dest})
		}
		return c
	} else {
		if try {
			c.Filesystem = append(c.Filesystem, &pairF{pairArgs[ROBindTry], src, dest})
		} else {
			c.Filesystem = append(c.Filesystem, &pairF{pairArgs[ROBind], src, dest})
		}
		return c
	}
}

// RemountRO remount path as readonly; does not recursively remount
// (--remount-ro DEST)
func (c *Config) RemountRO(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{stringArgs[RemountRO], dest})
	return c
}

// Procfs mount new procfs in sandbox
// (--proc DEST)
func (c *Config) Procfs(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{stringArgs[Procfs], dest})
	return c
}

// DevTmpfs mount new dev in sandbox
// (--dev DEST)
func (c *Config) DevTmpfs(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{stringArgs[DevTmpfs], dest})
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

// Mqueue mount new mqueue in sandbox
// (--mqueue DEST)
func (c *Config) Mqueue(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{stringArgs[Mqueue], dest})
	return c
}

// Dir create dir in sandbox
// (--dir DEST)
func (c *Config) Dir(dest string) *Config {
	c.Filesystem = append(c.Filesystem, &stringF{stringArgs[Dir], dest})
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
