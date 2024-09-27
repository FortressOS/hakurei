package dbus

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

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

func (c *Config) Args(bus [2]string) (args []string) {
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

func (c *Config) Load(r io.Reader) error {
	return json.NewDecoder(r).Decode(&c)
}

// NewConfigFromFile opens the target config file at path and parses its contents into *Config.
func NewConfigFromFile(path string) (*Config, error) {
	if f, err := os.Open(path); err != nil {
		return nil, err
	} else {
		c := new(Config)
		err1 := c.Load(f)
		err = f.Close()

		return c, errors.Join(err1, err)
	}
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
