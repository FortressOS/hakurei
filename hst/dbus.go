package hst

import (
	"strconv"
	"strings"
)

// BadInterfaceError is returned when Interface fails an undocumented check in xdg-dbus-proxy,
// which would have cause a silent failure.
type BadInterfaceError struct {
	// Interface is the offending interface string.
	Interface string
	// Segment is passed through from the [BusConfig.CheckInterfaces] argument.
	Segment string
}

func (e *BadInterfaceError) Message() string { return e.Error() }
func (e *BadInterfaceError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return "bad interface string " + strconv.Quote(e.Interface) + " in " + e.Segment + " bus configuration"
}

// BusConfig configures the xdg-dbus-proxy process.
type BusConfig struct {
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

	// Log turn on logging (--log)
	Log bool `json:"log,omitempty"`
	// Filter enable filtering (--filter)
	Filter bool `json:"filter"`
}

// Interfaces iterates over all interface strings specified in [BusConfig].
func (c *BusConfig) Interfaces(yield func(string) bool) {
	if c == nil {
		return
	}

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

// CheckInterfaces checks for invalid interface strings based on an undocumented check in xdg-dbus-error,
// returning [BadInterfaceError] if one is encountered.
func (c *BusConfig) CheckInterfaces(segment string) error {
	if c == nil {
		return nil
	}

	for iface := range c.Interfaces {
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
