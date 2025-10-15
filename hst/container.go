package hst

import (
	"encoding/json"
	"syscall"
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
)

const (
	// ExitFailure is returned if the container fails to start.
	ExitFailure = iota + 1
	// ExitCancel is returned if the container is terminated by a shim-directed signal which cancels its context.
	ExitCancel
	// ExitOrphan is returned when the shim is orphaned before priv side delivers a signal.
	ExitOrphan

	// ExitRequest is returned when the priv side process requests shim exit.
	ExitRequest = 254
)

const (
	// FMultiarch unblocks syscalls required for multiarch to work on applicable targets.
	FMultiarch uintptr = 1 << iota

	// FSeccompCompat changes emitted seccomp filter programs to be identical to that of Flatpak.
	FSeccompCompat
	// FDevel unblocks ptrace and friends.
	FDevel
	// FUserns unblocks userns creation and container setup syscalls.
	FUserns
	// FHostNet skips net namespace creation.
	FHostNet
	// FHostAbstract skips setting up abstract unix socket scope.
	FHostAbstract
	// FTty unblocks dangerous terminal I/O (faking input).
	FTty

	// FMapRealUID maps the target user uid to the privileged user uid in the container user namespace.
	//	Some programs fail to connect to dbus session running as a different uid,
	//	this option works around it by mapping priv-side caller uid in container.
	FMapRealUID

	// FDevice mount /dev/ from the init mount namespace as-is in the container mount namespace.
	FDevice

	fMax

	// FAll is [ContainerConfig.Flags] with all currently defined bits set.
	FAll = fMax - 1
)

// ContainerConfig describes the container configuration to be applied to an underlying [container].
type ContainerConfig struct {
	// Container UTS namespace hostname.
	Hostname string `json:"hostname,omitempty"`

	// Duration in nanoseconds to wait for after interrupting the initial process.
	// Defaults to [WaitDelayDefault] if zero, or [WaitDelayMax] if greater than [WaitDelayMax].
	// Values lesser than zero is equivalent to zero, bypassing [WaitDelayDefault].
	WaitDelay time.Duration `json:"wait_delay,omitempty"`

	// Initial process environment variables.
	Env map[string]string `json:"env"`

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

	// Flags holds boolean options of [ContainerConfig].
	Flags uintptr `json:"-"`
}

// ContainerConfigF is [ContainerConfig] stripped of its methods.
// The [ContainerConfig.Flags] field does not survive a [json] round trip.
type ContainerConfigF ContainerConfig

// containerConfigJSON is the [json] representation of [ContainerConfig].
type containerConfigJSON = struct {
	*ContainerConfigF

	// Corresponds to [FSeccompCompat].
	SeccompCompat bool `json:"seccomp_compat,omitempty"`
	// Corresponds to [FDevel].
	Devel bool `json:"devel,omitempty"`
	// Corresponds to [FUserns].
	Userns bool `json:"userns,omitempty"`
	// Corresponds to [FHostNet].
	HostNet bool `json:"host_net,omitempty"`
	// Corresponds to [FHostAbstract].
	HostAbstract bool `json:"host_abstract,omitempty"`
	// Corresponds to [FTty].
	Tty bool `json:"tty,omitempty"`

	// Corresponds to [FMultiarch].
	Multiarch bool `json:"multiarch,omitempty"`

	// Corresponds to [FMapRealUID].
	MapRealUID bool `json:"map_real_uid"`

	// Corresponds to [FDevice].
	Device bool `json:"device,omitempty"`
}

func (c *ContainerConfig) MarshalJSON() ([]byte, error) {
	if c == nil {
		return nil, syscall.EINVAL
	}
	return json.Marshal(&containerConfigJSON{
		ContainerConfigF: (*ContainerConfigF)(c),

		SeccompCompat: c.Flags&FSeccompCompat != 0,
		Devel:         c.Flags&FDevel != 0,
		Userns:        c.Flags&FUserns != 0,
		HostNet:       c.Flags&FHostNet != 0,
		HostAbstract:  c.Flags&FHostAbstract != 0,
		Tty:           c.Flags&FTty != 0,
		Multiarch:     c.Flags&FMultiarch != 0,
		MapRealUID:    c.Flags&FMapRealUID != 0,
		Device:        c.Flags&FDevice != 0,
	})
}

func (c *ContainerConfig) UnmarshalJSON(data []byte) error {
	if c == nil {
		return syscall.EINVAL
	}

	v := new(containerConfigJSON)
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = *(*ContainerConfig)(v.ContainerConfigF)
	if v.SeccompCompat {
		c.Flags |= FSeccompCompat
	}
	if v.Devel {
		c.Flags |= FDevel
	}
	if v.Userns {
		c.Flags |= FUserns
	}
	if v.HostNet {
		c.Flags |= FHostNet
	}
	if v.HostAbstract {
		c.Flags |= FHostAbstract
	}
	if v.Tty {
		c.Flags |= FTty
	}
	if v.Multiarch {
		c.Flags |= FMultiarch
	}
	if v.MapRealUID {
		c.Flags |= FMapRealUID
	}
	if v.Device {
		c.Flags |= FDevice
	}
	return nil
}
