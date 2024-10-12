package app

import (
	"os"
	"path"
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
	seal.sys.bind(passwdPath, "/etc/passwd", true)
	seal.sys.bind(groupPath, "/etc/group", true)
}
