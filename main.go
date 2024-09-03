package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"syscall"

	"git.ophivana.moe/cat/fortify/internal/acl"
	"git.ophivana.moe/cat/fortify/internal/app"
	"git.ophivana.moe/cat/fortify/internal/state"
	"git.ophivana.moe/cat/fortify/internal/system"
	"git.ophivana.moe/cat/fortify/internal/util"
	"git.ophivana.moe/cat/fortify/internal/xcb"
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

const (
	term    = "TERM"
	display = "DISPLAY"

	// https://manpages.debian.org/experimental/libwayland-doc/wl_display_connect.3.en.html
	waylandDisplay = "WAYLAND_DISPLAY"
)

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

	// ensure Wayland socket ACL (e.g. `/run/user/%d/wayland-%d`)
	if w, ok := os.LookupEnv(waylandDisplay); !ok {
		if system.V.Verbose {
			fmt.Println("Wayland: WAYLAND_DISPLAY not set, skipping")
		}
	} else {
		// add environment variable for new process
		wp := path.Join(system.V.Runtime, w)
		a.AppendEnv(waylandDisplay, wp)
		if err := acl.UpdatePerm(wp, a.UID(), acl.Read, acl.Write, acl.Execute); err != nil {
			state.Fatal(fmt.Sprintf("Error preparing Wayland '%s':", w), err)
		} else {
			state.RegisterRevertPath(wp)
		}
		if system.V.Verbose {
			fmt.Printf("Wayland socket '%s' configured\n", w)
		}
	}

	// discovery X11 and grant user permission via the `ChangeHosts` command
	if d, ok := os.LookupEnv(display); !ok {
		if system.V.Verbose {
			fmt.Println("X11: DISPLAY not set, skipping")
		}
	} else {
		// add environment variable for new process
		a.AppendEnv(display, d)

		if system.V.Verbose {
			fmt.Printf("X11: Adding XHost entry SI:localuser:%s to display '%s'\n", a.Username, d)
		}
		if err := xcb.ChangeHosts(xcb.HostModeInsert, xcb.FamilyServerInterpreted, "localuser\x00"+a.Username); err != nil {
			state.Fatal(fmt.Sprintf("Error adding XHost entry to '%s':", d), err)
		} else {
			state.XcbActionComplete()
		}
	}

	// ensure PulseAudio directory ACL (e.g. `/run/user/%d/pulse`)
	pulse := path.Join(system.V.Runtime, "pulse")
	pulseS := path.Join(pulse, "native")
	if s, err := os.Stat(pulse); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			state.Fatal("Error accessing PulseAudio directory:", err)
		}
		if mustPulse {
			state.Fatal("PulseAudio is unavailable")
		}
		if system.V.Verbose {
			fmt.Printf("PulseAudio dir '%s' not found, skipping\n", pulse)
		}
	} else {
		// add environment variable for new process
		a.AppendEnv(util.PulseServer, "unix:"+pulseS)
		if err = acl.UpdatePerm(pulse, a.UID(), acl.Execute); err != nil {
			state.Fatal("Error preparing PulseAudio:", err)
		} else {
			state.RegisterRevertPath(pulse)
		}

		// ensure PulseAudio socket permission (e.g. `/run/user/%d/pulse/native`)
		if s, err = os.Stat(pulseS); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				state.Fatal("PulseAudio directory found but socket does not exist")
			}
			state.Fatal("Error accessing PulseAudio socket:", err)
		} else {
			if m := s.Mode(); m&0o006 != 0o006 {
				state.Fatal(fmt.Sprintf("Unexpected permissions on '%s':", pulseS), m)
			}
		}

		// Publish current user's pulse-cookie for target user
		pulseCookieSource := util.DiscoverPulseCookie()
		pulseCookieFinal := path.Join(system.V.Share, "pulse-cookie")
		a.AppendEnv(util.PulseCookie, pulseCookieFinal)
		if system.V.Verbose {
			fmt.Printf("Publishing PulseAudio cookie '%s' to '%s'\n", pulseCookieSource, pulseCookieFinal)
		}
		if err = util.CopyFile(pulseCookieFinal, pulseCookieSource); err != nil {
			state.Fatal("Error copying PulseAudio cookie:", err)
		}
		if err = acl.UpdatePerm(pulseCookieFinal, a.UID(), acl.Read); err != nil {
			state.Fatal("Error publishing PulseAudio cookie:", err)
		} else {
			state.RegisterRevertPath(pulseCookieFinal)
		}

		if system.V.Verbose {
			fmt.Printf("PulseAudio dir '%s' configured\n", pulse)
		}
	}

	// pass $TERM to launcher
	if t, ok := os.LookupEnv(term); ok {
		a.AppendEnv(term, t)
	}

	a.Run()
}
