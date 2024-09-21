package state

type (
	// Enablement represents an optional system resource
	Enablement uint8
	// Enablements represents optional system resources to share
	Enablements uint64
)

const (
	EnableWayland Enablement = iota
	EnableX
	EnableDBus
	EnablePulse

	EnableLength
)

var enablementString = [EnableLength]string{
	"Wayland",
	"X11",
	"D-Bus",
	"PulseAudio",
}

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
