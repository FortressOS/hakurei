package app

import (
	"path"

	"git.ophivana.moe/security/fortify/acl"
	"git.ophivana.moe/security/fortify/internal/linux"
	"git.ophivana.moe/security/fortify/internal/system"
)

const (
	shell = "SHELL"
)

// shareSystem queues various system-related actions
func (seal *appSeal) shareSystem() {
	// ensure Share (e.g. `/tmp/fortify.%d`)
	// acl is unnecessary as this directory is world executable
	seal.sys.Ensure(seal.SharePath, 0711)

	// ensure process-specific share (e.g. `/tmp/fortify.%d/%s`)
	// acl is unnecessary as this directory is world executable
	seal.share = path.Join(seal.SharePath, seal.id)
	seal.sys.Ephemeral(system.Process, seal.share, 0711)

	// ensure child tmpdir parent directory (e.g. `/tmp/fortify.%d/tmpdir`)
	targetTmpdirParent := path.Join(seal.SharePath, "tmpdir")
	seal.sys.Ensure(targetTmpdirParent, 0700)
	seal.sys.UpdatePermType(system.User, targetTmpdirParent, acl.Execute)

	// ensure child tmpdir (e.g. `/tmp/fortify.%d/tmpdir/%d`)
	targetTmpdir := path.Join(targetTmpdirParent, seal.sys.user.as)
	seal.sys.Ensure(targetTmpdir, 01700)
	seal.sys.UpdatePermType(system.User, targetTmpdir, acl.Read, acl.Write, acl.Execute)
	seal.sys.bwrap.Bind(targetTmpdir, "/tmp", false, true)

	// mount tmpfs on inner shared directory (e.g. `/tmp/fortify.%d`)
	seal.sys.bwrap.Tmpfs(seal.SharePath, 1*1024*1024)
}

func (seal *appSeal) sharePasswd(os linux.System) {
	// look up shell
	sh := "/bin/sh"
	if s, ok := os.LookupEnv(shell); ok {
		seal.sys.bwrap.SetEnv[shell] = s
		sh = s
	}

	// generate /etc/passwd
	passwdPath := path.Join(seal.share, "passwd")
	username := "chronos"
	if seal.sys.user.username != "" {
		username = seal.sys.user.username
	}
	homeDir := "/var/empty"
	if seal.sys.user.home != "" {
		homeDir = seal.sys.user.home
	}

	// bind home directory
	seal.sys.bwrap.Bind(homeDir, homeDir, false, true)
	seal.sys.bwrap.Chdir = homeDir

	seal.sys.bwrap.SetEnv["USER"] = username
	seal.sys.bwrap.SetEnv["HOME"] = homeDir

	passwd := username + ":x:" + seal.sys.mappedIDString + ":" + seal.sys.mappedIDString + ":Fortify:" + homeDir + ":" + sh + "\n"
	seal.sys.Write(passwdPath, passwd)

	// write /etc/group
	groupPath := path.Join(seal.share, "group")
	seal.sys.Write(groupPath, "fortify:x:"+seal.sys.mappedIDString+":\n")

	// bind /etc/passwd and /etc/group
	seal.sys.bwrap.Bind(passwdPath, "/etc/passwd")
	seal.sys.bwrap.Bind(groupPath, "/etc/group")
}
