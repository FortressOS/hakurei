package bwrap

const (
	UID = iota
	GID

	intC
)

var intArgs = func() (n [intC]string) {
	n[UID] = "--uid"
	n[GID] = "--gid"

	return
}()

func (c *Config) intArgs() (n [intC]*int) {
	n[UID] = c.UID
	n[GID] = c.GID

	return
}
