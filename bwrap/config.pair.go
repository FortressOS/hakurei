package bwrap

import "strconv"

const (
	SetEnv = iota

	Bind
	BindTry
	DevBind
	DevBindTry
	ROBind
	ROBindTry

	Chmod

	pairC
)

var pairArgs = func() (n [pairC]string) {
	n[SetEnv] = "--setenv"

	n[Bind] = "--bind"
	n[BindTry] = "--bind-try"
	n[DevBind] = "--dev-bind"
	n[DevBindTry] = "--dev-bind-try"
	n[ROBind] = "--ro-bind"
	n[ROBindTry] = "--ro-bind-try"

	n[Chmod] = "--chmod"

	return
}()

func (c *Config) pairArgs() (n [pairC][][2]string) {
	n[SetEnv] = make([][2]string, 0, len(c.SetEnv))
	for k, v := range c.SetEnv {
		n[SetEnv] = append(n[SetEnv], [2]string{k, v})
	}

	n[Bind] = c.Bind
	n[BindTry] = c.BindTry
	n[DevBind] = c.DevBind
	n[DevBindTry] = c.DevBindTry
	n[ROBind] = c.ROBind
	n[ROBindTry] = c.ROBindTry

	n[Chmod] = make([][2]string, 0, len(c.Chmod))
	for path, octal := range c.Chmod {
		n[Chmod] = append(n[Chmod], [2]string{strconv.Itoa(int(octal)), path})
	}

	return
}
