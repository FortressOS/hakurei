package check

import "strings"

const (
	// SpecialOverlayEscape is the escape string for overlay mount options.
	SpecialOverlayEscape = `\`
	// SpecialOverlayOption is the separator string between overlay mount options.
	SpecialOverlayOption = ","
	// SpecialOverlayPath is the separator string between overlay paths.
	SpecialOverlayPath = ":"
)

// EscapeOverlayDataSegment escapes a string for formatting into the data argument of an overlay mount call.
func EscapeOverlayDataSegment(s string) string {
	if s == "" {
		return ""
	}

	if f := strings.SplitN(s, "\x00", 2); len(f) > 0 {
		s = f[0]
	}

	return strings.NewReplacer(
		SpecialOverlayEscape, SpecialOverlayEscape+SpecialOverlayEscape,
		SpecialOverlayOption, SpecialOverlayEscape+SpecialOverlayOption,
		SpecialOverlayPath, SpecialOverlayEscape+SpecialOverlayPath,
	).Replace(s)
}
