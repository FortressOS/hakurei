package internal

const compPoison = "INVALIDINVALIDINVALIDINVALIDINVALID"

var (
	version = compPoison
)

// checkComp validates string value set at compile time.
func checkComp(s string) (string, bool) { return s, s != compPoison && s != "" }

func Version() string {
	if v, ok := checkComp(version); ok {
		return v
	}
	return "impure"
}
