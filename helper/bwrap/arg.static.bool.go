package bwrap

const (
	UnshareAll = iota
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
			*args = append(*args, boolArgs[i]...)
		}
	}
}
