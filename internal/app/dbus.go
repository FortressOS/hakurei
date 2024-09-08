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

const dbusSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"

var dbusAddress string

func (a *App) ShareDBus(c *dbus.Config) {
	a.setEnablement(state.EnableDBus)

	var binPath, address string
	target := path.Join(system.V.Share, strconv.Itoa(os.Getpid()))

	if b, ok := util.Which("xdg-dbus-proxy"); !ok {
		state.Fatal("D-Bus: Did not find 'xdg-dbus-proxy' in PATH")
	} else {
		binPath = b
	}

	if addr, ok := os.LookupEnv(dbusSessionBusAddress); !ok {
		state.Fatal("D-Bus: DBUS_SESSION_BUS_ADDRESS not set")
	} else {
		address = addr
	}

	c.Log = system.V.Verbose
	p := dbus.New(binPath, address, target)
	if system.V.Verbose {
		fmt.Println("D-Bus: sealing proxy", c.Args(address, target))
	}
	if err := p.Seal(c); err != nil {
		state.Fatal("D-Bus: invalid config when sealing proxy,", err)
	}

	ready := make(chan bool, 1)
	done := make(chan struct{})

	if system.V.Verbose {
		fmt.Printf("Starting session bus proxy '%s' for address '%s'\n", dbusAddress, address)
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
	dbusAddress = "unix:path=" + target

	if !<-ready {
		state.Fatal("D-Bus: proxy did not start correctly")
	}

	a.AppendEnv(dbusSessionBusAddress, dbusAddress)
	if err := acl.UpdatePerm(target, a.UID(), acl.Read, acl.Write); err != nil {
		state.Fatal(fmt.Sprintf("Error preparing D-Bus proxy '%s':", dbusAddress), err)
	} else {
		state.RegisterRevertPath(target)
	}
	if system.V.Verbose {
		fmt.Printf("Session bus proxy '%s' for address '%s' configured\n", dbusAddress, address)
	}
}
