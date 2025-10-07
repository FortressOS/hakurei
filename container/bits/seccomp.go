package bits

// FilterPreset specifies parts of the syscall filter preset to enable.
type FilterPreset int

const (
	// PresetExt are project-specific extensions.
	PresetExt FilterPreset = 1 << iota
	// PresetDenyNS denies namespace setup syscalls.
	PresetDenyNS
	// PresetDenyTTY denies faking input.
	PresetDenyTTY
	// PresetDenyDevel denies development-related syscalls.
	PresetDenyDevel
	// PresetLinux32 sets PER_LINUX32.
	PresetLinux32

	// PresetStrict is a strict preset useful as a default value.
	PresetStrict = PresetExt | PresetDenyNS | PresetDenyTTY | PresetDenyDevel
)
