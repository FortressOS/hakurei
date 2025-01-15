package bwrap

import (
	"slices"
	"strconv"
)

/*
	static boolean args
*/

type BoolArg int

func (b BoolArg) Unwrap() []string {
	return boolArgs[b]
}

const (
	UnshareAll BoolArg = iota
	UnshareUser
	UnshareIPC
	UnsharePID
	UnshareNet
	UnshareUTS
	UnshareCGroup
	ShareNet

	UserNS
	Clearenv

	NewSession
	DieWithParent
	AsInit
)

var boolArgs = [...][]string{
	UnshareAll:    {"--unshare-all", "--unshare-user"},
	UnshareUser:   {"--unshare-user"},
	UnshareIPC:    {"--unshare-ipc"},
	UnsharePID:    {"--unshare-pid"},
	UnshareNet:    {"--unshare-net"},
	UnshareUTS:    {"--unshare-uts"},
	UnshareCGroup: {"--unshare-cgroup"},
	ShareNet:      {"--share-net"},

	UserNS:   {"--disable-userns", "--assert-userns-disabled"},
	Clearenv: {"--clearenv"},

	NewSession:    {"--new-session"},
	DieWithParent: {"--die-with-parent"},
	AsInit:        {"--as-pid-1"},
}

func (c *Config) boolArgs() Builder {
	b := boolArg{
		UserNS:   !c.UserNS,
		Clearenv: c.Clearenv,

		NewSession:    c.NewSession,
		DieWithParent: c.DieWithParent,
		AsInit:        c.AsInit,
	}

	if c.Unshare == nil {
		b[UnshareAll] = true
		b[ShareNet] = c.Net
	} else {
		b[UnshareUser] = c.Unshare.User
		b[UnshareIPC] = c.Unshare.IPC
		b[UnsharePID] = c.Unshare.PID
		b[UnshareNet] = c.Unshare.Net
		b[UnshareUTS] = c.Unshare.UTS
		b[UnshareCGroup] = c.Unshare.CGroup
	}

	return &b
}

type boolArg [len(boolArgs)]bool

func (b *boolArg) Len() (l int) {
	for i, v := range b {
		if v {
			l += len(boolArgs[i])
		}
	}
	return
}

func (b *boolArg) Append(args *[]string) {
	for i, v := range b {
		if v {
			*args = append(*args, BoolArg(i).Unwrap()...)
		}
	}
}

/*
	static integer args
*/

type IntArg int

func (i IntArg) Unwrap() string {
	return intArgs[i]
}

const (
	UID IntArg = iota
	GID
)

var intArgs = [...]string{
	UID: "--uid",
	GID: "--gid",
}

func (c *Config) intArgs() Builder {
	return &intArg{
		UID: c.UID,
		GID: c.GID,
	}
}

type intArg [len(intArgs)]*int

func (n *intArg) Len() (l int) {
	for _, v := range n {
		if v != nil {
			l += 2
		}
	}
	return
}

func (n *intArg) Append(args *[]string) {
	for i, v := range n {
		if v != nil {
			*args = append(*args, IntArg(i).Unwrap(), strconv.Itoa(*v))
		}
	}
}

/*
	static string args
*/

type StringArg int

func (s StringArg) Unwrap() string {
	return stringArgs[s]
}

const (
	Hostname StringArg = iota
	Chdir
	UnsetEnv
	LockFile
)

var stringArgs = [...]string{
	Hostname: "--hostname",
	Chdir:    "--chdir",
	UnsetEnv: "--unsetenv",
	LockFile: "--lock-file",
}

func (c *Config) stringArgs() Builder {
	n := stringArg{
		UnsetEnv: c.UnsetEnv,
		LockFile: c.LockFile,
	}

	if c.Hostname != "" {
		n[Hostname] = []string{c.Hostname}
	}
	if c.Chdir != "" {
		n[Chdir] = []string{c.Chdir}
	}

	return &n
}

type stringArg [len(stringArgs)][]string

func (s *stringArg) Len() (l int) {
	for _, arg := range s {
		l += len(arg) * 2
	}
	return
}

func (s *stringArg) Append(args *[]string) {
	for i, arg := range s {
		for _, v := range arg {
			*args = append(*args, StringArg(i).Unwrap(), v)
		}
	}
}

/*
	static pair args
*/

type PairArg int

func (p PairArg) Unwrap() string {
	return pairArgs[p]
}

const (
	SetEnv PairArg = iota
)

var pairArgs = [...]string{
	SetEnv: "--setenv",
}

func (c *Config) pairArgs() Builder {
	var n pairArg
	n[SetEnv] = make([][2]string, len(c.SetEnv))
	keys := make([]string, 0, len(c.SetEnv))
	for k := range c.SetEnv {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for i, k := range keys {
		n[SetEnv][i] = [2]string{k, c.SetEnv[k]}
	}

	return &n
}

type pairArg [len(pairArgs)][][2]string

func (p *pairArg) Len() (l int) {
	for _, v := range p {
		l += len(v) * 3
	}
	return
}

func (p *pairArg) Append(args *[]string) {
	for i, arg := range p {
		for _, v := range arg {
			*args = append(*args, PairArg(i).Unwrap(), v[0], v[1])
		}
	}
}
