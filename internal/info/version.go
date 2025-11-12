package info

// FallbackVersion is returned when a version string was not set by the linker.
const FallbackVersion = "dirty"

// buildVersion is the Hakurei tree's version string at build time.
//
// This is set by the linker.
var buildVersion string

// Version returns the Hakurei tree's version string.
// It is either the value of the constant [FallbackVersion] or,
// when possible, a release tag like "v1.0.0".
func Version() string {
	if buildVersion != "" {
		return buildVersion
	}
	return FallbackVersion
}
