package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
)

var Version = "impure"

var (
	ego     *user.User
	uid     int
	env     []string
	command []string
	verbose bool
	runtime string
	runDir  string
)

const (
	term          = "TERM"
	home          = "HOME"
	sudoAskPass   = "SUDO_ASKPASS"
	xdgRuntimeDir = "XDG_RUNTIME_DIR"
	xdgConfigHome = "XDG_CONFIG_HOME"
	display       = "DISPLAY"
	pulseServer   = "PULSE_SERVER"
	pulseCookie   = "PULSE_COOKIE"

	// https://manpages.debian.org/experimental/libwayland-doc/wl_display_connect.3.en.html
	waylandDisplay = "WAYLAND_DISPLAY"
)

func main() {
	flag.Parse()
	tryLauncher()
	copyArgs()

	if u, err := strconv.Atoi(ego.Uid); err != nil {
		// usually unreachable
		panic("ego uid parse")
	} else {
		uid = u
	}

	if r, ok := os.LookupEnv(xdgRuntimeDir); !ok {
		fatal("Env variable", xdgRuntimeDir, "unset")
	} else {
		runtime = r
		runDir = path.Join(runtime, "ego")
	}

	// state query command
	tryState()

	// Report warning if user home directory does not exist or has wrong ownership
	if stat, err := os.Stat(ego.HomeDir); err != nil {
		if verbose {
			switch {
			case errors.Is(err, fs.ErrPermission):
				fmt.Printf("User %s home directory %s is not accessible", ego.Username, ego.HomeDir)
			case errors.Is(err, fs.ErrNotExist):
				fmt.Printf("User %s home directory %s does not exist", ego.Username, ego.HomeDir)
			default:
				fmt.Printf("Error stat user %s home directory %s: %s", ego.Username, ego.HomeDir, err)
			}
		}
		return
	} else {
		// FreeBSD: not cross-platform
		if u := strconv.Itoa(int(stat.Sys().(*syscall.Stat_t).Uid)); u != ego.Uid {
			fmt.Printf("User %s home directory %s has incorrect ownership (expected UID %s, found %s)", ego.Username, ego.HomeDir, ego.Uid, u)
		}
	}

	// Add execute perm to runtime dir, e.g. `/run/user/%d`
	if s, err := os.Stat(runtime); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fatal("Runtime directory does not exist")
		}
		fatal("Error accessing runtime directory:", err)
	} else if !s.IsDir() {
		fatal(fmt.Sprintf("Path '%s' is not a directory", runtime))
	} else {
		if err = aclUpdatePerm(runtime, uid, aclExecute); err != nil {
			fatal("Error preparing runtime dir:", err)
		} else {
			registerRevertPath(runtime)
		}
		if verbose {
			fmt.Printf("Runtime data dir '%s' configured\n", runtime)
		}
	}

	// Create runtime dir for Ego itself (e.g. `/run/user/%d/ego`) and make it readable for target
	if err := os.Mkdir(runDir, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
		fatal("Error creating Ego runtime dir:", err)
	}
	if err := aclUpdatePerm(runDir, uid, aclExecute); err != nil {
		fatal("Error preparing Ego runtime dir:", err)
	} else {
		registerRevertPath(runDir)
	}

	// Add rwx permissions to Wayland socket (e.g. `/run/user/%d/wayland-0`)
	if w, ok := os.LookupEnv(waylandDisplay); !ok {
		if verbose {
			fmt.Println("Wayland: WAYLAND_DISPLAY not set, skipping")
		}
	} else {
		// add environment variable for new process
		env = append(env, waylandDisplay+"="+path.Join(runtime, w))
		wp := path.Join(runtime, w)
		if err := aclUpdatePerm(wp, uid, aclRead, aclWrite, aclExecute); err != nil {
			fatal(fmt.Sprintf("Error preparing Wayland '%s':", w), err)
		} else {
			registerRevertPath(wp)
		}
		if verbose {
			fmt.Printf("Wayland socket '%s' configured\n", w)
		}
	}

	// Detect `DISPLAY` and grant permissions via X11 protocol `ChangeHosts` command
	if d, ok := os.LookupEnv(display); !ok {
		if verbose {
			fmt.Println("X11: DISPLAY not set, skipping")
		}
	} else {
		// add environment variable for new process
		env = append(env, display+"="+d)

		if verbose {
			fmt.Printf("X11: Adding XHost entry SI:localuser:%s to display '%s'\n", ego.Username, d)
		}
		if err := changeHosts(xcbHostModeInsert, xcbFamilyServerInterpreted, "localuser\x00"+ego.Username); err != nil {
			fatal(fmt.Sprintf("Error adding XHost entry to '%s':", d), err)
		} else {
			xcbActionComplete = true
		}
	}

	// Add execute permissions to PulseAudio directory (e.g. `/run/user/%d/pulse`)
	pulse := path.Join(runtime, "pulse")
	pulseS := path.Join(pulse, "native")
	if s, err := os.Stat(pulse); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			fatal("Error accessing PulseAudio directory:", err)
		}
		if mustPulse {
			fatal("PulseAudio is unavailable")
		}
		if verbose {
			fmt.Printf("PulseAudio dir '%s' not found, skipping\n", pulse)
		}
	} else {
		// add environment variable for new process
		env = append(env, pulseServer+"=unix:"+pulseS)
		if err = aclUpdatePerm(pulse, uid, aclExecute); err != nil {
			fatal("Error preparing PulseAudio:", err)
		} else {
			registerRevertPath(pulse)
		}

		// Ensure permissions of PulseAudio socket `/run/user/%d/pulse/native`
		if s, err = os.Stat(pulseS); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fatal("PulseAudio directory found but socket does not exist")
			}
			fatal("Error accessing PulseAudio socket:", err)
		} else {
			if m := s.Mode(); m&0o006 != 0o006 {
				fatal(fmt.Sprintf("Unexpected permissions on '%s':", pulseS), m)
			}
		}

		// Publish current user's pulse-cookie for target user
		pulseCookieSource := discoverPulseCookie()
		env = append(env, pulseCookie+"="+pulseCookieSource)
		pulseCookieFinal := path.Join(runDir, "pulse-cookie")
		if verbose {
			fmt.Printf("Publishing PulseAudio cookie '%s' to '%s'\n", pulseCookieSource, pulseCookieFinal)
		}
		if err = copyFile(pulseCookieFinal, pulseCookieSource); err != nil {
			fatal("Error copying PulseAudio cookie:", err)
		}
		if err = aclUpdatePerm(pulseCookieFinal, uid, aclRead); err != nil {
			fatal("Error publishing PulseAudio cookie:", err)
		} else {
			registerRevertPath(pulseCookieFinal)
		}

		if verbose {
			fmt.Printf("PulseAudio dir '%s' configured\n", pulse)
		}
	}

	// pass $TERM to launcher
	if t, ok := os.LookupEnv(term); ok {
		env = append(env, term+"="+t)
	}

	f := launchBySudo
	m, b := false, false
	switch {
	case methodFlags[0]: // sudo
	case methodFlags[1]: // bare
		m, b = true, true
	default: // machinectl
		m, b = true, false
	}

	var toolPath string

	// dependency checks
	const sudoFallback = "Falling back to 'sudo', some desktop integration features may not work"
	if m {
		if !sdBooted() {
			fmt.Println("This system was not booted through systemd")
			fmt.Println(sudoFallback)
		} else if tp, ok := which("machinectl"); !ok {
			fmt.Println("Did not find 'machinectl' in PATH")
			fmt.Println(sudoFallback)
		} else {
			toolPath = tp
			f = func() []string { return launchByMachineCtl(b) }
		}
	} else if tp, ok := which("sudo"); !ok {
		fatal("Did not find 'sudo' in PATH")
	} else {
		toolPath = tp
	}

	if verbose {
		fmt.Printf("Selected launcher '%s' bare=%t\n", toolPath, b)
	}

	cmd := exec.Command(toolPath, f()...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = runDir

	if verbose {
		fmt.Println("Executing:", cmd)
	}

	if err := cmd.Start(); err != nil {
		fatal("Error starting process:", err)
	}

	if err := registerProcess(ego.Uid, cmd); err != nil {
		// process already started, shouldn't be fatal
		fmt.Println("Error registering process:", err)
	}

	var r int
	if err := cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			fatal("Error running process:", err)
		}
	}

	if verbose {
		fmt.Println("Process exited with exit code", r)
	}
	beforeExit()
	os.Exit(r)
}

func launchBySudo() (args []string) {
	args = make([]string, 0, 4+len(env)+len(command))

	// -Hiu $USER
	args = append(args, "-Hiu", ego.Username)

	// -A?
	if _, ok := os.LookupEnv(sudoAskPass); ok {
		if verbose {
			fmt.Printf("%s set, adding askpass flag\n", sudoAskPass)
		}
		args = append(args, "-A")
	}

	// environ
	args = append(args, env...)

	// -- $@
	args = append(args, "--")
	args = append(args, command...)

	return
}

func launchByMachineCtl(bare bool) (args []string) {
	args = make([]string, 0, 9+len(env))

	// shell --uid=$USER
	args = append(args, "shell", "--uid="+ego.Username)

	// --quiet
	if !verbose {
		args = append(args, "--quiet")
	}

	// environ
	envQ := make([]string, len(env)+1)
	for i, e := range env {
		envQ[i] = "-E" + e
	}
	envQ[len(env)] = "-E" + launcherPayloadEnv()
	args = append(args, envQ...)

	// -- .host
	args = append(args, "--", ".host")

	// /bin/sh -c
	if sh, ok := which("sh"); !ok {
		fatal("Did not find 'sh' in PATH")
	} else {
		args = append(args, sh, "-c")
	}

	if len(command) == 0 { // execute shell if command is not provided
		command = []string{"$SHELL"}
	}

	innerCommand := strings.Builder{}

	if !bare {
		innerCommand.WriteString("dbus-update-activation-environment --systemd")
		for _, e := range env {
			innerCommand.WriteString(" " + strings.SplitN(e, "=", 2)[0])
		}
		innerCommand.WriteString("; systemctl --user start xdg-desktop-portal-gtk; ")
	}

	if executable, err := os.Executable(); err != nil {
		fatal("Error reading executable path:", err)
	} else {
		innerCommand.WriteString("exec " + executable + " -V")
	}
	args = append(args, innerCommand.String())

	return
}
