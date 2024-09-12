package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"syscall"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/internal/acl"
	"git.ophivana.moe/cat/fortify/internal/app"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/util"
)

var (
	Version = "impure"

	a *app.App

	dbusSession *dbus.Config
	dbusSystem  *dbus.Config

	launchOptionText string
)

func init() {
	methodHelpString := "Method of launching the child process, can be one of \"sudo\", \"bubblewrap\""
	if util.SdBootedV {
		methodHelpString += ", \"systemd\""
	}

	flag.StringVar(&launchOptionText, "method", "sudo", methodHelpString)
}

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
	a = app.New(userName, flag.Args(), launchOptionText)
	state.Set(*a.User, a.Command(), a.UID())

	// parse D-Bus config file if applicable
	if mustDBus {
		if dbusConfigSession == "builtin" {
			dbusSession = dbus.NewConfig(dbusID, true, mpris)
		} else {
			if f, err := os.Open(dbusConfigSession); err != nil {
				state.Fatal("Error opening D-Bus proxy config file:", err)
			} else {
				if err = json.NewDecoder(f).Decode(&dbusSession); err != nil {
					state.Fatal("Error parsing D-Bus proxy config file:", err)
				}
			}
		}

		// system bus proxy is optional
		if dbusConfigSystem != "nil" {
			if f, err := os.Open(dbusConfigSystem); err != nil {
				state.Fatal("Error opening D-Bus proxy config file:", err)
			} else {
				if err = json.NewDecoder(f).Decode(&dbusSystem); err != nil {
					state.Fatal("Error parsing D-Bus proxy config file:", err)
				}
			}
		}
	}

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

	if a.LaunchOption() == app.LaunchMethodSudo {
		// ensure child runtime directory (e.g. `/tmp/fortify.%d/%d.share`)
		cr := path.Join(system.V.Share, a.Uid+".share")
		if err := os.Mkdir(cr, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			state.Fatal("Error creating child runtime directory:", err)
		} else {
			if err = acl.UpdatePerm(cr, a.UID(), acl.Read, acl.Write, acl.Execute); err != nil {
				state.Fatal("Error preparing child runtime directory:", err)
			} else {
				state.RegisterRevertPath(cr)
			}
			a.AppendEnv("XDG_RUNTIME_DIR", cr)
			a.AppendEnv("XDG_SESSION_CLASS", "user")
			a.AppendEnv("XDG_SESSION_TYPE", "tty")
			if system.V.Verbose {
				fmt.Printf("Child runtime data dir '%s' configured\n", cr)
			}
		}
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
			state.Fatal("Error preparing runtime directory:", err)
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
		a.ShareDBus(dbusSession, dbusSystem, dbusVerbose)
	}

	if mustPulse {
		a.SharePulse()
	}

	a.Run()
}
