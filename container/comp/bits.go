// Package comp contains constants from container packages without depending on cgo.
package comp

const (
	// BindOptional skips nonexistent host paths.
	BindOptional = 1 << iota
	// BindWritable mounts filesystem read-write.
	BindWritable
	// BindDevice allows access to devices (special files) on this filesystem.
	BindDevice
	// BindEnsure attempts to create the host path if it does not exist.
	BindEnsure
)

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
