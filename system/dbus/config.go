package dbus

import (
	"strings"

	"hakurei.app/hst"
)

// ProxyPair is an upstream dbus address and a downstream socket path.
type ProxyPair [2]string

// interfacesAll returns an iterator over all interfaces specified in c.
func interfacesAll(c *hst.BusConfig) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for _, iface := range c.See {
			if !yield(iface) {
				return
			}
		}
		for _, iface := range c.Talk {
			if !yield(iface) {
				return
			}
		}
		for _, iface := range c.Own {
			if !yield(iface) {
				return
			}
		}

		for iface := range c.Call {
			if !yield(iface) {
				return
			}
		}
		for iface := range c.Broadcast {
			if !yield(iface) {
				return
			}
		}
	}
}

// checkInterfaces checks [hst.BusConfig] for invalid interfaces based on an undocumented check in xdg-dbus-error,
// returning [BadInterfaceError] if one is encountered.
func checkInterfaces(c *hst.BusConfig, segment string) error {
	for iface := range interfacesAll(c) {
		/*
			xdg-dbus-proxy fails without output when this condition is not met:
				char *dot = strrchr (filter->interface, '.');
				if (dot != NULL)
				  {
				    *dot = 0;
				    if (strcmp (dot + 1, "*") != 0)
				      filter->member = g_strdup (dot + 1);
				  }

			trim ".*" since they are removed before searching for '.':
				if (g_str_has_suffix (name, ".*"))
				  {
				    name[strlen (name) - 2] = 0;
				    wildcard = TRUE;
				  }
		*/
		if strings.IndexByte(strings.TrimSuffix(iface, ".*"), '.') == -1 {
			return &BadInterfaceError{iface, segment}
		}
	}
	return nil
}

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
