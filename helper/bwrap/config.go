package bwrap

type Config struct {
	// unshare every namespace we support by default if nil
	// (--unshare-all)
	Unshare *UnshareConfig `json:"unshare,omitempty"`
	// retain the network namespace (can only combine with nil Unshare)
	// (--share-net)
	Net bool `json:"net"`

	// disable further use of user namespaces inside sandbox and fail unless
	// further use of user namespace inside sandbox is disabled if false
	// (--disable-userns) (--assert-userns-disabled)
	UserNS bool `json:"userns"`

	// custom uid in the sandbox, requires new user namespace
	// (--uid UID)
	UID *int `json:"uid,omitempty"`
	// custom gid in the sandbox, requires new user namespace
	// (--gid GID)
	GID *int `json:"gid,omitempty"`
	// custom hostname in the sandbox, requires new uts namespace
	// (--hostname NAME)
	Hostname string `json:"hostname,omitempty"`

	// change directory
	// (--chdir DIR)
	Chdir string `json:"chdir,omitempty"`
	// unset all environment variables
	// (--clearenv)
	Clearenv bool `json:"clearenv"`
	// set environment variable
	// (--setenv VAR VALUE)
	SetEnv map[string]string `json:"setenv,omitempty"`
	// unset environment variables
	// (--unsetenv VAR)
	UnsetEnv []string `json:"unsetenv,omitempty"`

	// take a lock on file while sandbox is running
	// (--lock-file DEST)
	LockFile []string `json:"lock_file,omitempty"`

	// ordered filesystem args
	Filesystem []FSBuilder `json:"filesystem,omitempty"`

	// change permissions (must already exist)
	// (--chmod OCTAL PATH)
	Chmod ChmodConfig `json:"chmod,omitempty"`

	// load and use seccomp rules from FD (not repeatable)
	// (--seccomp FD)
	Syscall *SyscallPolicy

	// create a new terminal session
	// (--new-session)
	NewSession bool `json:"new_session"`
	// kills with SIGKILL child process (COMMAND) when bwrap or bwrap's parent dies.
	// (--die-with-parent)
	DieWithParent bool `json:"die_with_parent"`
	// do not install a reaper process with PID=1
	// (--as-pid-1)
	AsInit bool `json:"as_init"`

	/* unmapped options include:
	    --unshare-user-try           Create new user namespace if possible else continue by skipping it
	    --unshare-cgroup-try         Create new cgroup namespace if possible else continue by skipping it
	    --userns FD                  Use this user namespace (cannot combine with --unshare-user)
	    --userns2 FD                 After setup switch to this user namespace, only useful with --userns
	    --pidns FD                   Use this pid namespace (as parent namespace if using --unshare-pid)
	    --bind-fd FD DEST            Bind open directory or path fd on DEST
	    --ro-bind-fd FD DEST         Bind open directory or path fd read-only on DEST
	    --exec-label LABEL           Exec label for the sandbox
	    --file-label LABEL           File label for temporary sandbox content
	    --file FD DEST               Copy from FD to destination DEST
	    --bind-data FD DEST          Copy from FD to file which is bind-mounted on DEST
	    --ro-bind-data FD DEST       Copy from FD to file which is readonly bind-mounted on DEST
	    --add-seccomp-fd FD          Load and use seccomp rules from FD (repeatable)
	    --block-fd FD                Block on FD until some data to read is available
	    --userns-block-fd FD         Block on FD until the user namespace is ready
	    --info-fd FD                 Write information about the running container to FD
	    --json-status-fd FD          Write container status to FD as multiple JSON documents
	    --cap-add CAP                Add cap CAP when running as privileged user
	    --cap-drop CAP               Drop cap CAP when running as privileged user

	among which --args is used internally for passing arguments */
}

type UnshareConfig struct {
	// (--unshare-user)
	// create new user namespace
	User bool `json:"user"`
	// (--unshare-ipc)
	// create new ipc namespace
	IPC bool `json:"ipc"`
	// (--unshare-pid)
	// create new pid namespace
	PID bool `json:"pid"`
	// (--unshare-net)
	// create new network namespace
	Net bool `json:"net"`
	// (--unshare-uts)
	// create new uts namespace
	UTS bool `json:"uts"`
	// (--unshare-cgroup)
	// create new cgroup namespace
	CGroup bool `json:"cgroup"`
}
