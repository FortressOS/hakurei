// Package bits contains constants for configuring the container.
package bits

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
