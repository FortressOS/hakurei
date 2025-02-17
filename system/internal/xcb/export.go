// Package xcb implements X11 ChangeHosts via libxcb.
package xcb

import (
	"errors"
)

var ErrChangeHosts = errors.New("xcb_change_hosts() failed")

func ChangeHosts(mode HostMode, family Family, address string) error {
	var conn *connection

	if c, err := connect(); err != nil {
		c.disconnect()
		return err
	} else {
		defer c.disconnect()
		conn = c
	}

	return conn.changeHostsChecked(mode, family, address)
}
