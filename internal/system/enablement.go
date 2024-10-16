package system

type (
	// Enablement represents an optional system resource
	Enablement uint8
	// Enablements represents optional system resources to share
	Enablements uint64
)

const (
	EWayland Enablement = iota
	EX11
	EDBus
	EPulse
)

var enablementString = [...]string{
	EWayland: "Wayland",
	EX11:     "X11",
	EDBus:    "D-Bus",
	EPulse:   "PulseAudio",
}

const ELen = len(enablementString)

func (e Enablement) String() string {
	return enablementString[e]
}

func (e Enablement) Mask() Enablements {
	return 1 << e
}

// Has returns whether a feature is enabled
func (es *Enablements) Has(e Enablement) bool {
	return *es&e.Mask() != 0
}

// Set enables a feature
func (es *Enablements) Set(e Enablement) {
	if es.Has(e) {
		panic("enablement " + e.String() + " set twice")
	}

	*es |= e.Mask()
}
