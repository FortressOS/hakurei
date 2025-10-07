package hst

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

	Log    bool `json:"log,omitempty"`
	Filter bool `json:"filter"`
}
