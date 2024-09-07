package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"syscall"

	"git.ophivana.moe/cat/fortify/internal/acl"
	"git.ophivana.moe/cat/fortify/internal/app"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
)

var (
	Version = "impure"
	a       *app.App
)

func tryVersion() {
	if printVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
}

func main() {
	flag.Parse()

	// launcher payload early exit
	app.Early(printVersion)

	// version/license command early exit
	tryVersion()
	tryLicense()

	system.Retrieve(flagVerbose)
	a = app.New(userName, flag.Args())
	state.Set(*a.User, a.Command(), a.UID())

	// ensure RunDir (e.g. `/run/user/%d/fortify`)
	if err := os.Mkdir(system.V.RunDir, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
		state.Fatal("Error creating runtime directory:", err)
	}

	// state query command early exit
	state.Early()

	// ensure Share (e.g. `/tmp/fortify.%d`)
	// acl is unnecessary as this directory is world executable
	if err := os.Mkdir(system.V.Share, 0701); err != nil && !errors.Is(err, fs.ErrExist) {
		state.Fatal("Error creating shared directory:", err)
	}

	// warn about target user home directory ownership
	if stat, err := os.Stat(a.HomeDir); err != nil {
		if system.V.Verbose {
			switch {
			case errors.Is(err, fs.ErrPermission):
				fmt.Printf("User %s home directory %s is not accessible", a.Username, a.HomeDir)
			case errors.Is(err, fs.ErrNotExist):
				fmt.Printf("User %s home directory %s does not exist", a.Username, a.HomeDir)
			default:
				fmt.Printf("Error stat user %s home directory %s: %s", a.Username, a.HomeDir, err)
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
	if s, err := os.Stat(system.V.Runtime); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			state.Fatal("Runtime directory does not exist")
		}
		state.Fatal("Error accessing runtime directory:", err)
	} else if !s.IsDir() {
		state.Fatal(fmt.Sprintf("Path '%s' is not a directory", system.V.Runtime))
	} else {
		if err = acl.UpdatePerm(system.V.Runtime, a.UID(), acl.Execute); err != nil {
			state.Fatal("Error preparing runtime dir:", err)
		} else {
			state.RegisterRevertPath(system.V.Runtime)
		}
		if system.V.Verbose {
			fmt.Printf("Runtime data dir '%s' configured\n", system.V.Runtime)
		}
	}

	if mustWayland {
		a.ShareWayland()
	}

	if mustX {
		a.ShareX()
	}

	if mustDBus {
		a.ShareDBus()
	}

	if mustPulse {
		a.SharePulse()
	}

	a.Run()
}
