package state

func RegisterRevertPath(p string) {
	cleanupCandidate = append(cleanupCandidate, p)
}

func XcbActionComplete() {
	if xcbActionComplete {
		Fatal("xcb inserted twice")
	}
	xcbActionComplete = true
}
