package xcb

import "errors"

var ErrChangeHosts = errors.New("xcb_change_hosts() failed")

func ChangeHosts(mode HostMode, family Family, address string) error {
	conn := new(connection)
	if err := conn.connect(); err != nil {
		conn.disconnect()
		return err
	} else {
		defer conn.disconnect()
	}

	return conn.changeHostsChecked(mode, family, address)
}
