package state

var (
	cleanupCandidate  []string
	enablements       *Enablements
	xcbActionComplete bool
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
