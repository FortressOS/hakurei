package internal

type (
	Enablement  uint8
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

func (es Enablements) Has(e Enablement) bool {
	return es&e.Mask() != 0
}
