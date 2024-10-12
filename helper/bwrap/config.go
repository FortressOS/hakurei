package bwrap

import (
	"os"
	"strconv"
)

func (c *Config) Args() (args []string) {
	b := c.boolArgs()
	n := c.intArgs()
	g := c.interfaceArgs()
	s := c.stringArgs()
	p := c.pairArgs()

	argc := 0
	for i, arg := range b {
		if arg {
			argc += len(boolArgs[i])
		}
	}
	for _, arg := range n {
		if arg != nil {
			argc += 2
		}
	}
	for _, arg := range g {
		argc += len(arg) * 3
	}
	for _, arg := range s {
		argc += len(arg) * 2
	}
	for _, arg := range p {
		argc += len(arg) * 3
	}

	args = make([]string, 0, argc)
	for i, arg := range b {
		if arg {
			args = append(args, boolArgs[i]...)
		}
	}
	for i, arg := range n {
		if arg != nil {
			args = append(args, intArgs[i], strconv.Itoa(*arg))
		}
	}
	for i, arg := range g {
		for _, v := range arg {
			if v.Later() {
				continue
			}
			args = append(args, v.Value(interfaceArgs[i])...)
		}
	}
	for i, arg := range s {
		for _, v := range arg {
			args = append(args, stringArgs[i], v)
		}
	}
	for i, arg := range p {
		for _, v := range arg {
			args = append(args, pairArgs[i], v[0], v[1])
		}
	}
	for i, arg := range g {
		for _, v := range arg {
			if !v.Later() {
				continue
			}
			args = append(args, v.Value(interfaceArgs[i])...)
		}
	}

	return
}

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

	// bind mount host path on sandbox
	// (--bind SRC DEST)
	Bind [][2]string `json:"bind,omitempty"`
	// equal to Bind but ignores non-existent host path
	// (--bind-try SRC DEST)
	BindTry [][2]string `json:"bind_try,omitempty"`

	// bind mount host path on sandbox, allowing device access
	// (--dev-bind SRC DEST)
	DevBind [][2]string `json:"dev_bind,omitempty"`
	// equal to DevBind but ignores non-existent host path
	// (--dev-bind-try SRC DEST)
	DevBindTry [][2]string `json:"dev_bind_try,omitempty"`

	// bind mount host path readonly on sandbox
	// (--ro-bind SRC DEST)
	ROBind [][2]string `json:"ro_bind,omitempty"`
	// equal to ROBind but ignores non-existent host path
	// (--ro-bind-try SRC DEST)
	ROBindTry [][2]string `json:"ro_bind_try,omitempty"`

	// remount path as readonly; does not recursively remount
	// (--remount-ro DEST)
	RemountRO []string `json:"remount_ro,omitempty"`

	// mount new procfs in sandbox
	// (--proc DEST)
	Procfs []string `json:"proc,omitempty"`
	// mount new dev in sandbox
	// (--dev DEST)
	DevTmpfs []string `json:"dev,omitempty"`
	// mount new tmpfs in sandbox
	// (--tmpfs DEST)
	Tmpfs []PermConfig[TmpfsConfig] `json:"tmpfs,omitempty"`
	// mount new mqueue in sandbox
	// (--mqueue DEST)
	Mqueue []string `json:"mqueue,omitempty"`
	// create dir in sandbox
	// (--dir DEST)
	Dir []PermConfig[string] `json:"dir,omitempty"`
	// create symlink within sandbox
	// (--symlink SRC DEST)
	Symlink []PermConfig[[2]string] `json:"symlink,omitempty"`

	// change permissions (must already exist)
	// (--chmod OCTAL PATH)
	Chmod map[string]os.FileMode `json:"chmod,omitempty"`

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
		--sync-fd FD                 Keep this fd open while sandbox is running
	    --exec-label LABEL           Exec label for the sandbox
	    --file-label LABEL           File label for temporary sandbox content
	    --file FD DEST               Copy from FD to destination DEST
	    --bind-data FD DEST          Copy from FD to file which is bind-mounted on DEST
	    --ro-bind-data FD DEST       Copy from FD to file which is readonly bind-mounted on DEST
	    --seccomp FD                 Load and use seccomp rules from FD (not repeatable)
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

type TmpfsConfig struct {
	// set size of tmpfs
	// (--size BYTES)
	Size int `json:"size,omitempty"`
	// mount point of new tmpfs
	// (--tmpfs DEST)
	Dir string `json:"dir"`
}

type argOf interface {
	Value(arg string) (args []string)
	Later() bool
}

func copyToArgOfSlice[T [2]string | string | TmpfsConfig](src []PermConfig[T]) (dst []argOf) {
	dst = make([]argOf, len(src))
	for i, arg := range src {
		dst[i] = arg
	}
	return
}

type PermConfig[T [2]string | string | TmpfsConfig] struct {
	// append this at the end of the argument stream
	Last bool

	// set permissions of next argument
	// (--perms OCTAL)
	Mode *os.FileMode `json:"mode,omitempty"`
	// path to get the new permission
	// (--bind-data, --file, etc.)
	Path T
}

func (p PermConfig[T]) Later() bool {
	return p.Last
}

func (p PermConfig[T]) Value(arg string) (args []string) {
	// max possible size
	if p.Mode != nil {
		args = make([]string, 0, 6)
		args = append(args, "--perms", strconv.Itoa(int(*p.Mode)))
	} else {
		args = make([]string, 0, 4)
	}

	switch v := any(p.Path).(type) {
	case string:
		args = append(args, arg, v)
		return
	case [2]string:
		args = append(args, arg, v[0], v[1])
		return
	case TmpfsConfig:
		if arg != "--tmpfs" {
			panic("unreachable")
		}

		if v.Size > 0 {
			args = append(args, "--size", strconv.Itoa(v.Size))
		}
		args = append(args, arg, v.Dir)
		return
	default:
		panic("unreachable")
	}
}
