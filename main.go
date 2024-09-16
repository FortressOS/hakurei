package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"syscall"

	"git.ophivana.moe/cat/fortify/internal/final"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/app"
	"git.ophivana.moe/cat/fortify/internal/verbose"
)

var (
	Version = "impure"

	a *app.App

	dbusSession *dbus.Config
	dbusSystem  *dbus.Config

	launchOptionText string
)

func tryVersion() {
	if printVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
}

func main() {
	flag.Parse()
	verbose.Set(flagVerbose)

	// launcher payload early exit
	app.Early(printVersion)

	// version/license command early exit
	tryVersion()
	tryLicense()

	a = app.New(userName, flag.Args(), launchOptionText)
	final.Prepare(*a.User, a.UID(), a.RunDir())

	// parse D-Bus config file if applicable
	if mustDBus {
		if dbusConfigSession == "builtin" {
			dbusSession = dbus.NewConfig(dbusID, true, mpris)
		} else {
			if f, err := os.Open(dbusConfigSession); err != nil {
				final.Fatal("Error opening D-Bus proxy config file:", err)
			} else {
				if err = json.NewDecoder(f).Decode(&dbusSession); err != nil {
					final.Fatal("Error parsing D-Bus proxy config file:", err)
				}
			}
		}

		// system bus proxy is optional
		if dbusConfigSystem != "nil" {
			if f, err := os.Open(dbusConfigSystem); err != nil {
				final.Fatal("Error opening D-Bus proxy config file:", err)
			} else {
				if err = json.NewDecoder(f).Decode(&dbusSystem); err != nil {
					final.Fatal("Error parsing D-Bus proxy config file:", err)
				}
			}
		}
	}

	// ensure RunDir (e.g. `/run/user/%d/fortify`)
	a.EnsureRunDir()

	// state query command early exit
	tryState()

	// ensure Share (e.g. `/tmp/fortify.%d`)
	a.EnsureShare()

	// warn about target user home directory ownership
	if stat, err := os.Stat(a.HomeDir); err != nil {
		if verbose.Get() {
			switch {
			case errors.Is(err, fs.ErrPermission):
				fmt.Printf("User %s home directory %s is not accessible\n", a.Username, a.HomeDir)
			case errors.Is(err, fs.ErrNotExist):
				fmt.Printf("User %s home directory %s does not exis\n", a.Username, a.HomeDir)
			default:
				fmt.Printf("Error stat user %s home directory %s: %s\n", a.Username, a.HomeDir, err)
			}
		}
		return
	} else {
		// FreeBSD: not cross-platform
		if u := strconv.Itoa(int(stat.Sys().(*syscall.Stat_t).Uid)); u != a.Uid {
			fmt.Printf("User %s home directory %s has incorrect ownership (expected UID %s, found %s)", a.Username, a.HomeDir, a.Uid, u)
		}
	}

	// ensure runtime directory ACL (e.g. `/run/user/%d`)
	a.EnsureRuntime()

	if mustWayland {
		a.ShareWayland()
	}

	if mustX {
		a.ShareX()
	}

	if mustDBus {
		a.ShareDBus(dbusSession, dbusSystem, dbusVerbose)
	}

	if mustPulse {
		a.SharePulse()
	}

	a.Run()
}
