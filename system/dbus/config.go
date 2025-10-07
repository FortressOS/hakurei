package dbus

import (
	"hakurei.app/hst"
)

// ProxyPair is an upstream dbus address and a downstream socket path.
type ProxyPair [2]string

// Args returns the xdg-dbus-proxy arguments equivalent of [hst.BusConfig].
func Args(c *hst.BusConfig, bus ProxyPair) (args []string) {
	argc := 2 + len(c.See) + len(c.Talk) + len(c.Own) + len(c.Call) + len(c.Broadcast)
	if c.Log {
		argc++
	}
	if c.Filter {
		argc++
	}

	args = make([]string, 0, argc)
	args = append(args, bus[0], bus[1])
	if c.Filter {
		args = append(args, "--filter")
	}
	for _, name := range c.See {
		args = append(args, "--see="+name)
	}
	for _, name := range c.Talk {
		args = append(args, "--talk="+name)
	}
	for _, name := range c.Own {
		args = append(args, "--own="+name)
	}
	for name, rule := range c.Call {
		args = append(args, "--call="+name+"="+rule)
	}
	for name, rule := range c.Broadcast {
		args = append(args, "--broadcast="+name+"="+rule)
	}
	if c.Log {
		args = append(args, "--log")
	}

	return
}

// NewConfig returns the address of a new [hst.BusConfig] with optional defaults.
func NewConfig(id string, defaults, mpris bool) *hst.BusConfig {
	c := hst.BusConfig{
		Call:      make(map[string]string),
		Broadcast: make(map[string]string),

		Filter: true,
	}

	if defaults {
		c.Talk = []string{"org.freedesktop.DBus", "org.freedesktop.Notifications"}

		c.Call["org.freedesktop.portal.*"] = "*"
		c.Broadcast["org.freedesktop.portal.*"] = "@/org/freedesktop/portal/*"

		if id != "" {
			c.Own = []string{id + ".*"}
			if mpris {
				c.Own = append(c.Own, "org.mpris.MediaPlayer2."+id+".*")
			}
		}
	}

	return &c
}
