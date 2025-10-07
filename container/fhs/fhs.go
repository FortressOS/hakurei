// Package fhs provides constant and checked pathname values for common FHS paths.
package fhs

const (
	// Root points to the file system root.
	Root = "/"
	// Etc points to the directory for system-specific configuration.
	Etc = "/etc/"
	// Tmp points to the place for small temporary files.
	Tmp = "/tmp/"

	// Run points to a "tmpfs" file system for system packages to place runtime data, socket files, and similar.
	Run = "/run/"
	// RunUser points to a directory containing per-user runtime directories,
	// each usually individually mounted "tmpfs" instances.
	RunUser = Run + "user/"

	// Usr points to vendor-supplied operating system resources.
	Usr = "/usr/"
	// UsrBin points to binaries and executables for user commands that shall appear in the $PATH search path.
	UsrBin = Usr + "bin/"

	// Var points to persistent, variable system data. Writable during normal system operation.
	Var = "/var/"
	// VarLib points to persistent system data.
	VarLib = Var + "lib/"
	// VarEmpty points to a nonstandard directory that is usually empty.
	VarEmpty = Var + "empty/"

	// Dev points to the root directory for device nodes.
	Dev = "/dev/"
	// Proc points to a virtual kernel file system exposing the process list and other functionality.
	Proc = "/proc/"
	// ProcSys points to a hierarchy below /proc/ that exposes a number of kernel tunables.
	ProcSys = Proc + "sys/"
	// Sys points to a virtual kernel file system exposing discovered devices and other functionality.
	Sys = "/sys/"
)
