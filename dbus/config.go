package dbus

type Config struct {
	// See set 'see' policy for NAME (--see=NAME)
	See []string `json:"see"`
	// Talk set 'talk' policy for NAME (--talk=NAME)
	Talk []string `json:"talk"`
	// Own set 'own' policy for NAME (--own=NAME)
	Own []string `json:"own"`

	// Call set RULE for calls on NAME (--call=NAME=RULE)
	Call map[string]string `json:"call"`
	// Broadcast set RULE for broadcasts from NAME (--broadcast=NAME=RULE)
	Broadcast map[string]string `json:"broadcast"`

	Log    bool `json:"log,omitempty"`
	Filter bool `json:"filter"`
}

func (c *Config) Args(address, path string) (args []string) {
	argc := 2 + len(c.See) + len(c.Talk) + len(c.Own) + len(c.Call) + len(c.Broadcast)
	if c.Log {
		argc++
	}
	if c.Filter {
		argc++
	}

	args = make([]string, 0, argc)
	args = append(args, address, path)
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
	if c.Filter {
		args = append(args, "--filter")
	}

	return
}

// NewConfig returns a reference to a Config struct with optional defaults.
// If id is an empty string own defaults are omitted.
func NewConfig(id string, defaults, mpris bool) (c *Config) {
	c = &Config{
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

	return
}
