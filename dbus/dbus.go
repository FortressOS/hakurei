// Package dbus wraps xdg-dbus-proxy and implements configuration and sandboxing of the underlying helper process.
package dbus

import (
	"fmt"
	"os"
	"sync"
)

const (
	SessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"
	SystemBusAddress  = "DBUS_SYSTEM_BUS_ADDRESS"
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
			addresses[1] = "unix:path=/run/dbus/system_bus_socket"
		} else {
			addresses[1] = addr
		}
	})

	return addresses[0], addresses[1]
}
