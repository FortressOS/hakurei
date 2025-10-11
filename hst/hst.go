// Package hst exports stable shared types for interacting with hakurei.
package hst

import (
	"errors"
	"net"
	"os"

	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
)

// An AppError is returned while starting an app according to [hst.Config].
type AppError struct {
	// A user-facing description of where the error occurred.
	Step string `json:"step"`
	// The underlying error value.
	Err error
	// An arbitrary error message, overriding the return value of Message if not empty.
	Msg string `json:"message,omitempty"`
}

func (e *AppError) Error() string { return e.Err.Error() }
func (e *AppError) Unwrap() error { return e.Err }
func (e *AppError) Message() string {
	if e.Msg != "" {
		return e.Msg
	}

	switch {
	case errors.As(e.Err, new(*os.PathError)),
		errors.As(e.Err, new(*os.LinkError)),
		errors.As(e.Err, new(*os.SyscallError)),
		errors.As(e.Err, new(*net.OpError)):
		return "cannot " + e.Error()

	default:
		return "cannot " + e.Step + ": " + e.Error()
	}
}

// Paths contains environment-dependent paths used by hakurei.
type Paths struct {
	// Temporary directory returned by [os.TempDir], usually equivalent to [fhs.AbsTmp].
	TempDir *check.Absolute `json:"temp_dir"`
	// Shared directory specific to the hsu userid, usually (`/tmp/hakurei.%d`, [Info.User]).
	SharePath *check.Absolute `json:"share_path"`
	// Checked XDG_RUNTIME_DIR value, usually (`/run/user/%d`, uid).
	RuntimePath *check.Absolute `json:"runtime_path"`
	// Shared directory specific to the hsu userid located in RuntimePath, usually (`/run/user/%d/hakurei`, uid).
	RunDirPath *check.Absolute `json:"run_dir_path"`
}

type Info struct {
	// User is the userid according to hsu.
	User int `json:"user"`

	Paths
}

// Template returns a fully populated instance of Config.
func Template() *Config {
	return &Config{
		ID: "org.chromium.Chromium",

		Enablements: NewEnablements(EWayland | EDBus | EPulse),

		SessionBus: &BusConfig{
			See: nil,
			Talk: []string{"org.freedesktop.Notifications", "org.freedesktop.FileManager1", "org.freedesktop.ScreenSaver",
				"org.freedesktop.secrets", "org.kde.kwalletd5", "org.kde.kwalletd6", "org.gnome.SessionManager"},
			Own: []string{"org.chromium.Chromium.*", "org.mpris.MediaPlayer2.org.chromium.Chromium.*",
				"org.mpris.MediaPlayer2.chromium.*"},
			Call:      map[string]string{"org.freedesktop.portal.*": "*"},
			Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
			Log:       false,
			Filter:    true,
		},
		SystemBus: &BusConfig{
			See:       nil,
			Talk:      []string{"org.bluez", "org.freedesktop.Avahi", "org.freedesktop.UPower"},
			Own:       nil,
			Call:      nil,
			Broadcast: nil,
			Log:       false,
			Filter:    true,
		},
		DirectWayland: false,

		ExtraPerms: []*ExtraPermConfig{
			{Path: fhs.AbsVarLib.Append("hakurei/u0"), Ensure: true, Execute: true},
			{Path: fhs.AbsVarLib.Append("hakurei/u0/org.chromium.Chromium"), Read: true, Write: true, Execute: true},
		},

		Identity: 9,
		Groups:   []string{"video", "dialout", "plugdev"},

		Container: &ContainerConfig{
			Hostname:      "localhost",
			Devel:         true,
			Userns:        true,
			HostNet:       true,
			HostAbstract:  true,
			Device:        true,
			WaitDelay:     -1,
			SeccompCompat: true,
			Tty:           true,
			Multiarch:     true,
			MapRealUID:    true,
			// example API credentials pulled from Google Chrome
			// DO NOT USE THESE IN A REAL BROWSER
			Env: map[string]string{
				"GOOGLE_API_KEY":               "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
				"GOOGLE_DEFAULT_CLIENT_ID":     "77185425430.apps.googleusercontent.com",
				"GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT",
			},
			Filesystem: []FilesystemConfigJSON{
				{&FSBind{Target: fhs.AbsRoot, Source: fhs.AbsVarLib.Append("hakurei/base/org.debian"), Write: true, Special: true}},
				{&FSBind{Target: fhs.AbsEtc, Source: fhs.AbsEtc, Special: true}},
				{&FSEphemeral{Target: fhs.AbsTmp, Write: true, Perm: 0755}},
				{&FSOverlay{
					Target: check.MustAbs("/nix/store"),
					Lower:  []*check.Absolute{fhs.AbsVarLib.Append("hakurei/base/org.nixos/ro-store")},
					Upper:  fhs.AbsVarLib.Append("hakurei/nix/u0/org.chromium.Chromium/rw-store/upper"),
					Work:   fhs.AbsVarLib.Append("hakurei/nix/u0/org.chromium.Chromium/rw-store/work"),
				}},
				{&FSLink{Target: fhs.AbsRun.Append("current-system"), Linkname: "/run/current-system", Dereference: true}},
				{&FSLink{Target: fhs.AbsRun.Append("opengl-driver"), Linkname: "/run/opengl-driver", Dereference: true}},
				{&FSBind{Source: fhs.AbsVarLib.Append("hakurei/u0/org.chromium.Chromium"),
					Target: check.MustAbs("/data/data/org.chromium.Chromium"), Write: true, Ensure: true}},
				{&FSBind{Source: fhs.AbsDev.Append("dri"), Device: true, Optional: true}},
			},

			Username: "chronos",
			Shell:    fhs.AbsRun.Append("current-system/sw/bin/zsh"),
			Home:     check.MustAbs("/data/data/org.chromium.Chromium"),

			Path: fhs.AbsRun.Append("current-system/sw/bin/chromium"),
			Args: []string{
				"chromium",
				"--ignore-gpu-blocklist",
				"--disable-smooth-scrolling",
				"--enable-features=UseOzonePlatform",
				"--ozone-platform=wayland",
			},
		},
	}
}
