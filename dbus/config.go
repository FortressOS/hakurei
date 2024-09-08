package dbus

type Config struct {
	See  []string `json:"see"`
	Talk []string `json:"talk"`
	Own  []string `json:"own"`

	Log    bool `json:"log,omitempty"`
	Filter bool `json:"filter"`
}

func (c *Config) Args(address, path string) (args []string) {
	argc := 2 + len(c.See) + len(c.Talk) + len(c.Own)
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
	c = &Config{Filter: true}

	if defaults {
		c.Talk = []string{"org.freedesktop.DBus", "org.freedesktop.portal.*", "org.freedesktop.Notifications"}

		if id != "" {
			c.Own = []string{id}
			if mpris {
				c.Own = append(c.Own, "org.mpris.MediaPlayer2."+id)
			}
		}
	}

	return
}
