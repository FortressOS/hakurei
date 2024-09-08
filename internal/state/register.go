package state

import "git.ophivana.moe/cat/fortify/dbus"

var (
	cleanupCandidate  []string
	enablements       *Enablements
	xcbActionComplete bool

	dbusProxy *dbus.Proxy
	dbusDone  *chan struct{}
)

func RegisterRevertPath(p string) {
	cleanupCandidate = append(cleanupCandidate, p)
}

func RegisterEnablement(e Enablements) {
	if enablements != nil {
		panic("enablement state set twice")
	}
	enablements = &e
}

func XcbActionComplete() {
	if xcbActionComplete {
		Fatal("xcb inserted twice")
	}
	xcbActionComplete = true
}

func RegisterDBus(p *dbus.Proxy, done *chan struct{}) {
	dbusProxy = p
	dbusDone = done
}
