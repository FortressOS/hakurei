package internal

const compPoison = "INVALIDINVALIDINVALIDINVALIDINVALID"

var (
	Version = compPoison
)

// Check validates string value set at compile time.
func Check(s string) (string, bool) {
	return s, s != compPoison && s != ""
}
