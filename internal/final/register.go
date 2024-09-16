package final

import (
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/state"
)

var (
	cleanupCandidate  []string
	enablements       *state.Enablements
	xcbActionComplete bool

	dbusProxy *dbus.Proxy
	dbusDone  *chan struct{}

	statePath string
)

func RegisterRevertPath(p string) {
	cleanupCandidate = append(cleanupCandidate, p)
}

func RegisterEnablement(e state.Enablements) {
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

func RegisterStatePath(v string) {
	if statePath != "" {
		panic("statePath set twice")
	}

	statePath = v
}
