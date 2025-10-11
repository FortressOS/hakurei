// Package dbus wraps xdg-dbus-proxy and implements configuration and sandboxing of the underlying helper process.
package dbus

import (
	"fmt"
	"os"
	"sync"
)

const (
	/*SessionBusAddress is the name of the environment variable where the address of the login session message bus is given in.

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
	So a fallback address with a socket located in the well-known default XDG_RUNTIME_DIR formatting is used.*/
	SessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"

	/*SystemBusAddress is the name of the environment variable where the address of the system message bus is given in.

	If that variable is not set, applications should try to connect to the well-known address unix:path=/var/run/dbus/system_bus_socket.
	Implementations of the well-known system bus should listen on an address that will result in that connection being successful.*/
	SystemBusAddress = "DBUS_SYSTEM_BUS_ADDRESS"

	// FallbackSystemBusAddress is used when [SystemBusAddress] is not set.
	FallbackSystemBusAddress = "unix:path=/var/run/dbus/system_bus_socket"
)

var (
	addresses   [2]string
	addressOnce sync.Once
)

func Address() (session, system string) {
	addressOnce.Do(func() {
		// resolve upstream session bus address
		if addr, ok := os.LookupEnv(SessionBusAddress); !ok {
			// fall back to default format
			addresses[0] = fmt.Sprintf("unix:path=/run/user/%d/bus", os.Getuid())
		} else {
			addresses[0] = addr
		}

		// resolve upstream system bus address
		if addr, ok := os.LookupEnv(SystemBusAddress); !ok {
			// fall back to default hardcoded value
			addresses[1] = FallbackSystemBusAddress
		} else {
			addresses[1] = addr
		}
	})

	return addresses[0], addresses[1]
}
