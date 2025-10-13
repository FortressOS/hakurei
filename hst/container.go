package hst

import (
	"time"

	"hakurei.app/container/check"
)

// PrivateTmp is a private writable path in a hakurei container.
const PrivateTmp = "/.hakurei"

// AbsPrivateTmp is a [check.Absolute] representation of [PrivateTmp].
var AbsPrivateTmp = check.MustAbs(PrivateTmp)

const (
	// WaitDelayDefault is used when WaitDelay has its zero value.
	WaitDelayDefault = 5 * time.Second
	// WaitDelayMax is used if WaitDelay exceeds its value.
	WaitDelayMax = 30 * time.Second

	// IdentityMin is the minimum value of [Config.Identity]. This is enforced by cmd/hsu.
	IdentityMin = 0
	// IdentityMax is the maximum value of [Config.Identity]. This is enforced by cmd/hsu.
	IdentityMax = 9999

	// ShimExitRequest is returned when the priv side process requests shim exit.
	ShimExitRequest = 254
	// ShimExitOrphan is returned when the shim is orphaned before priv side delivers a signal.
	ShimExitOrphan = 3
)

// ContainerConfig describes the container configuration to be applied to an underlying [container].
type ContainerConfig struct {
	// Container UTS namespace hostname.
	Hostname string `json:"hostname,omitempty"`

	// Duration in nanoseconds to wait for after interrupting the initial process.
	// Defaults to [WaitDelayDefault] if zero, or [WaitDelayMax] if greater than [WaitDelayMax].
	// Values lesser than zero is equivalent to zero, bypassing [WaitDelayDefault].
	WaitDelay time.Duration `json:"wait_delay,omitempty"`

	// Emit Flatpak-compatible seccomp filter programs.
	SeccompCompat bool `json:"seccomp_compat,omitempty"`
	// Allow ptrace and friends.
	Devel bool `json:"devel,omitempty"`
	// Allow userns creation and container setup syscalls.
	Userns bool `json:"userns,omitempty"`
	// Share host net namespace.
	HostNet bool `json:"host_net,omitempty"`
	// Share abstract unix socket scope.
	HostAbstract bool `json:"host_abstract,omitempty"`
	// Allow dangerous terminal I/O (faking input).
	Tty bool `json:"tty,omitempty"`
	// Allow multiarch.
	Multiarch bool `json:"multiarch,omitempty"`

	// Initial process environment variables.
	Env map[string]string `json:"env"`

	/* Map target user uid to privileged user uid in the container user namespace.

	Some programs fail to connect to dbus session running as a different uid,
	this option works around it by mapping priv-side caller uid in container. */
	MapRealUID bool `json:"map_real_uid"`

	// Mount /dev/ from the init mount namespace as-is in the container mount namespace.
	Device bool `json:"device,omitempty"`

	/* Container mount points.

	If the first element targets /, it is inserted early and excluded from path hiding. */
	Filesystem []FilesystemConfigJSON `json:"filesystem"`

	// String used as the username of the emulated user, validated against the default NAME_REGEX from adduser.
	// Defaults to passwd name of target uid or chronos.
	Username string `json:"username,omitempty"`
	// Pathname of shell in the container filesystem to use for the emulated user.
	Shell *check.Absolute `json:"shell"`
	// Directory in the container filesystem to enter and use as the home directory of the emulated user.
	Home *check.Absolute `json:"home"`

	// Pathname to executable file in the container filesystem.
	Path *check.Absolute `json:"path,omitempty"`
	// Final args passed to the initial program.
	Args []string `json:"args"`
}
