package app

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/acl"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/util"
)

const (
	dbusSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"
	dbusSystemBusAddress  = "DBUS_SYSTEM_BUS_ADDRESS"
)

var (
	dbusAddress [2]string
	dbusSystem  bool
)

func (a *App) ShareDBus(dse, dsg *dbus.Config, verbose bool) {
	a.setEnablement(state.EnableDBus)

	dbusSystem = dsg != nil
	var binPath string
	var sessionBus, systemBus [2]string

	target := path.Join(system.V.Share, strconv.Itoa(os.Getpid()))
	sessionBus[1] = target + ".bus"
	systemBus[1] = target + ".system-bus"
	dbusAddress = [2]string{
		"unix:path=" + sessionBus[1],
		"unix:path=" + systemBus[1],
	}

	if b, ok := util.Which("xdg-dbus-proxy"); !ok {
		state.Fatal("D-Bus: Did not find 'xdg-dbus-proxy' in PATH")
	} else {
		binPath = b
	}

	if addr, ok := os.LookupEnv(dbusSessionBusAddress); !ok {
		if system.V.Verbose {
			fmt.Println("D-Bus: DBUS_SESSION_BUS_ADDRESS not set, assuming default format")
		}
		sessionBus[0] = fmt.Sprintf("unix:path=/run/user/%d/bus", os.Getuid())
	} else {
		sessionBus[0] = addr
	}

	if addr, ok := os.LookupEnv(dbusSystemBusAddress); !ok {
		if system.V.Verbose {
			fmt.Println("D-Bus: DBUS_SYSTEM_BUS_ADDRESS not set, assuming default format")
		}
		systemBus[0] = "unix:path=/run/dbus/system_bus_socket"
	} else {
		systemBus[0] = addr
	}

	p := dbus.New(binPath, sessionBus, systemBus)

	dse.Log = verbose
	if system.V.Verbose {
		fmt.Println("D-Bus: sealing session proxy", dse.Args(sessionBus))
	}
	if dsg != nil {
		dsg.Log = verbose
		if system.V.Verbose {
			fmt.Println("D-Bus: sealing system proxy", dsg.Args(systemBus))
		}
	}
	if err := p.Seal(dse, dsg); err != nil {
		state.Fatal("D-Bus: invalid config when sealing proxy,", err)
	}

	ready := make(chan bool, 1)
	done := make(chan struct{})

	if system.V.Verbose {
		fmt.Printf("Starting session bus proxy '%s' for address '%s'\n", dbusAddress[0], sessionBus[0])
		if dsg != nil {
			fmt.Printf("Starting system bus proxy '%s' for address '%s'\n", dbusAddress[1], systemBus[0])
		}
	}
	if err := p.Start(&ready); err != nil {
		state.Fatal("D-Bus: error starting proxy,", err)
	}
	if system.V.Verbose {
		fmt.Println("D-Bus proxy launch:", p)
	}

	go func() {
		if err := p.Wait(); err != nil {
			fmt.Println("warn: D-Bus proxy returned error,", err)
		} else {
			if system.V.Verbose {
				fmt.Println("D-Bus proxy uneventful wait")
			}
		}
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Println("Error removing dangling D-Bus socket:", err)
		}
		done <- struct{}{}
	}()

	// register early to enable Fatal cleanup
	state.RegisterDBus(p, &done)

	if !<-ready {
		state.Fatal("D-Bus: proxy did not start correctly")
	}

	a.AppendEnv(dbusSessionBusAddress, dbusAddress[0])
	if err := acl.UpdatePerm(sessionBus[1], a.UID(), acl.Read, acl.Write); err != nil {
		state.Fatal(fmt.Sprintf("Error preparing D-Bus session proxy '%s':", dbusAddress[0]), err)
	} else {
		state.RegisterRevertPath(sessionBus[1])
	}
	if dsg != nil {
		a.AppendEnv(dbusSystemBusAddress, dbusAddress[1])
		if err := acl.UpdatePerm(systemBus[1], a.UID(), acl.Read, acl.Write); err != nil {
			state.Fatal(fmt.Sprintf("Error preparing D-Bus system proxy '%s':", dbusAddress[1]), err)
		} else {
			state.RegisterRevertPath(systemBus[1])
		}
	}
	if system.V.Verbose {
		fmt.Printf("Session bus proxy '%s' for address '%s' configured\n", dbusAddress[0], sessionBus[0])
		if dsg != nil {
			fmt.Printf("System bus proxy '%s' for address '%s' configured\n", dbusAddress[1], systemBus[0])
		}
	}
}
