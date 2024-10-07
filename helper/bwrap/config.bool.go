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

	boolC
)

var boolArgs = func() (b [boolC][]string) {
	b[UnshareAll] = []string{"--unshare-all"}
	b[UnshareUser] = []string{"--unshare-user"}
	b[UnshareIPC] = []string{"--unshare-ipc"}
	b[UnsharePID] = []string{"--unshare-pid"}
	b[UnshareNet] = []string{"--unshare-net"}
	b[UnshareUTS] = []string{"--unshare-uts"}
	b[UnshareCGroup] = []string{"--unshare-cgroup"}
	b[ShareNet] = []string{"--share-net"}

	b[UserNS] = []string{"--disable-userns", "--assert-userns-disabled"}
	b[Clearenv] = []string{"--clearenv"}

	b[NewSession] = []string{"--new-session"}
	b[DieWithParent] = []string{"--die-with-parent"}
	b[AsInit] = []string{"--as-pid-1"}

	return
}()

func (c *Config) boolArgs() (b [boolC]bool) {
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

	b[UserNS] = !c.UserNS
	b[Clearenv] = c.Clearenv

	b[NewSession] = c.NewSession
	b[DieWithParent] = c.DieWithParent
	b[AsInit] = c.AsInit

	return
}
