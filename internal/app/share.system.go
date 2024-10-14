package app

import (
	"os"
	"path"

	"git.ophivana.moe/cat/fortify/acl"
	"git.ophivana.moe/cat/fortify/internal/state"
)

const (
	shell = "SHELL"
)

// shareSystem queues various system-related actions
func (seal *appSeal) shareSystem() {
	// look up shell
	sh := "/bin/sh"
	if s, ok := os.LookupEnv(shell); ok {
		seal.sys.setEnv(shell, s)
		sh = s
	}

	// generate /etc/passwd
	passwdPath := path.Join(seal.share, "passwd")
	username := "chronos"
	if seal.sys.Username != "" {
		username = seal.sys.Username
		seal.sys.setEnv("USER", seal.sys.Username)
	}
	homeDir := "/var/empty"
	if seal.sys.HomeDir != "" {
		homeDir = seal.sys.HomeDir
		seal.sys.setEnv("HOME", seal.sys.HomeDir)
	}
	passwd := username + ":x:65534:65534:Fortify:" + homeDir + ":" + sh + "\n"
	seal.sys.writeFile(passwdPath, []byte(passwd))

	// write /etc/group
	groupPath := path.Join(seal.share, "group")
	seal.sys.writeFile(groupPath, []byte("fortify:x:65534:\n"))

	// bind /etc/passwd and /etc/group
	seal.sys.bwrap.Bind(passwdPath, "/etc/passwd")
	seal.sys.bwrap.Bind(groupPath, "/etc/group")
}

func (seal *appSeal) shareTmpdirChild() string {
	// ensure child tmpdir parent directory (e.g. `/tmp/fortify.%d/tmpdir`)
	targetTmpdirParent := path.Join(seal.SharePath, "tmpdir")
	seal.sys.ensure(targetTmpdirParent, 0700)
	seal.sys.updatePermTag(state.EnableLength, targetTmpdirParent, acl.Execute)

	// ensure child tmpdir (e.g. `/tmp/fortify.%d/tmpdir/%d`)
	targetTmpdir := path.Join(targetTmpdirParent, seal.sys.Uid)
	seal.sys.ensure(targetTmpdir, 01700)
	seal.sys.updatePermTag(state.EnableLength, targetTmpdir, acl.Read, acl.Write, acl.Execute)
	seal.sys.bwrap.Bind(targetTmpdir, "/tmp", false, true)

	// mount tmpfs on inner shared directory (e.g. `/tmp/fortify.%d`)
	seal.sys.bwrap.Tmpfs(seal.SharePath, 1*1024*1024)

	return targetTmpdir
}
