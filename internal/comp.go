package internal

const compPoison = "INVALIDINVALIDINVALIDINVALIDINVALID"

var (
	version = compPoison
)

// check validates string value set at compile time.
func check(s string) (string, bool) { return s, s != compPoison && s != "" }

func Version() string {
	if v, ok := check(version); ok {
		return v
	}
	return "impure"
}
