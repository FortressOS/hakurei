// Package dbus exposes the internal/dbus package.
//
// Deprecated: This package will be removed in 0.4.
package dbus

import (
	"context"
	"io"
	_ "unsafe" // for go:linkname

	"hakurei.app/hst"
	"hakurei.app/internal/dbus"
	"hakurei.app/message"
)

type AddrEntry = dbus.AddrEntry

// EqualAddrEntries returns whether two slices of [AddrEntry] are equal.
//
//go:linkname EqualAddrEntries hakurei.app/internal/dbus.EqualAddrEntries
func EqualAddrEntries(entries, target []AddrEntry) bool

// Parse parses D-Bus address according to
// https://dbus.freedesktop.org/doc/dbus-specification.html#addresses
//
//go:linkname Parse hakurei.app/internal/dbus.Parse
func Parse(addr []byte) ([]AddrEntry, error)

type ParseError = dbus.ParseError

const (
	ErrNoColon         = dbus.ErrNoColon
	ErrBadPairSep      = dbus.ErrBadPairSep
	ErrBadPairKey      = dbus.ErrBadPairKey
	ErrBadPairVal      = dbus.ErrBadPairVal
	ErrBadValLength    = dbus.ErrBadValLength
	ErrBadValByte      = dbus.ErrBadValByte
	ErrBadValHexLength = dbus.ErrBadValHexLength
	ErrBadValHexByte   = dbus.ErrBadValHexByte
)

type BadAddressError = dbus.BadAddressError

// ProxyPair is an upstream dbus address and a downstream socket path.
type ProxyPair = dbus.ProxyPair

// Args returns the xdg-dbus-proxy arguments equivalent of [hst.BusConfig].
//
//go:linkname Args hakurei.app/internal/dbus.Args
func Args(c *hst.BusConfig, bus ProxyPair) (args []string)

// NewConfig returns the address of a new [hst.BusConfig] with optional defaults.
//
//go:linkname NewConfig hakurei.app/internal/dbus.NewConfig
func NewConfig(id string, defaults, mpris bool) *hst.BusConfig

const (
	/*
		SessionBusAddress is the name of the environment variable where the address of the login session message bus is given in.

		If that variable is not set, applications may also try to read the address from the X Window System root window property _DBUS_SESSION_BUS_ADDRESS.
		The root window property must have type STRING. The environment variable should have precedence over the root window property.

		The address of the login session message bus is given in the DBUS_SESSION_BUS_ADDRESS environment variable.
		If DBUS_SESSION_BUS_ADDRESS is not set, or if it's set to the string "autolaunch:",
		the system should use platform-specific methods of locating a running D-Bus session server,
		or starting one if a running instance cannot be found.
		Note that this mechanism is not recommended for attempting to determine if a daemon is running.
		It is inherently racy to attempt to make this determination, since the bus daemon may be started just before or just after the determination is made.
		Therefore, it is recommended that applications do not try to make this determination for their functionality purposes, and instead they should attempt to start the server.

		This package diverges from the specification, as the caller is unlikely to be an X client, or be in a position to autolaunch a dbus server.
		So a fallback address with a socket located in the well-known default XDG_RUNTIME_DIR formatting is used.
	*/
	SessionBusAddress = dbus.SessionBusAddress

	/*
		SystemBusAddress is the name of the environment variable where the address of the system message bus is given in.

		If that variable is not set, applications should try to connect to the well-known address unix:path=/var/run/dbus/system_bus_socket.
		Implementations of the well-known system bus should listen on an address that will result in that connection being successful.
	*/
	SystemBusAddress = dbus.SystemBusAddress

	// FallbackSystemBusAddress is used when [SystemBusAddress] is not set.
	FallbackSystemBusAddress = dbus.FallbackSystemBusAddress
)

// Address returns the session and system bus addresses copied from environment,
// or appropriate fallback values if they are not set.
//
//go:linkname Address hakurei.app/internal/dbus.Address
func Address() (session, system string)

// ProxyName is the file name or path to the proxy program.
// Overriding ProxyName will only affect Proxy instance created after the change.
//
//go:linkname ProxyName hakurei.app/internal/dbus.ProxyName
var ProxyName string

// Proxy holds the state of a xdg-dbus-proxy process, and should never be copied.
type Proxy = dbus.Proxy

// Final describes the outcome of a proxy configuration.
type Final = dbus.Final

// Finalise creates a checked argument writer for [Proxy].
//
//go:linkname Finalise hakurei.app/internal/dbus.Finalise
func Finalise(sessionBus, systemBus ProxyPair, session, system *hst.BusConfig) (final *Final, err error)

// New returns a new instance of [Proxy].
//
//go:linkname New hakurei.app/internal/dbus.New
func New(ctx context.Context, msg message.Msg, final *Final, output io.Writer) *Proxy
