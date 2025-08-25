// Package hst exports stable shared types for interacting with hakurei.
package hst

import (
	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/system"
	"hakurei.app/system/dbus"
)

// Paths contains environment-dependent paths used by hakurei.
type Paths struct {
	// temporary directory returned by [os.TempDir] (usually `/tmp`)
	TempDir *container.Absolute `json:"temp_dir"`
	// path to shared directory (usually `/tmp/hakurei.%d`)
	SharePath *container.Absolute `json:"share_path"`
	// XDG_RUNTIME_DIR value (usually `/run/user/%d`)
	RuntimePath *container.Absolute `json:"runtime_path"`
	// application runtime directory (usually `/run/user/%d/hakurei`)
	RunDirPath *container.Absolute `json:"run_dir_path"`
}

type Info struct {
	User int `json:"user"`
	Paths
}

// Template returns a fully populated instance of Config.
func Template() *Config {
	return &Config{
		ID: "org.chromium.Chromium",

		Path: container.AbsFHSRun.Append("current-system/sw/bin/chromium"),
		Args: []string{
			"chromium",
			"--ignore-gpu-blocklist",
			"--disable-smooth-scrolling",
			"--enable-features=UseOzonePlatform",
			"--ozone-platform=wayland",
		},

		Enablements: NewEnablements(system.EWayland | system.EDBus | system.EPulse),

		SessionBus: &dbus.Config{
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
		SystemBus: &dbus.Config{
			See:       nil,
			Talk:      []string{"org.bluez", "org.freedesktop.Avahi", "org.freedesktop.UPower"},
			Own:       nil,
			Call:      nil,
			Broadcast: nil,
			Log:       false,
			Filter:    true,
		},
		DirectWayland: false,

		Username: "chronos",
		Shell:    container.AbsFHSRun.Append("current-system/sw/bin/zsh"),
		Data:     container.AbsFHSVarLib.Append("hakurei/u0/org.chromium.Chromium"),
		Dir:      container.MustAbs("/data/data/org.chromium.Chromium"),
		ExtraPerms: []*ExtraPermConfig{
			{Path: container.AbsFHSVarLib.Append("hakurei/u0"), Ensure: true, Execute: true},
			{Path: container.AbsFHSVarLib.Append("hakurei/u0/org.chromium.Chromium"), Read: true, Write: true, Execute: true},
		},

		Identity: 9,
		Groups:   []string{"video", "dialout", "plugdev"},

		Container: &ContainerConfig{
			Hostname:       "localhost",
			Devel:          true,
			Userns:         true,
			HostNet:        true,
			HostAbstract:   true,
			Device:         true,
			WaitDelay:      -1,
			SeccompFlags:   seccomp.AllowMultiarch,
			SeccompPresets: seccomp.PresetExt,
			SeccompCompat:  true,
			Tty:            true,
			Multiarch:      true,
			MapRealUID:     true,
			// example API credentials pulled from Google Chrome
			// DO NOT USE THESE IN A REAL BROWSER
			Env: map[string]string{
				"GOOGLE_API_KEY":               "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
				"GOOGLE_DEFAULT_CLIENT_ID":     "77185425430.apps.googleusercontent.com",
				"GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT",
			},
			Filesystem: []FilesystemConfigJSON{
				{&FSBind{container.AbsFHSRoot, container.AbsFHSVarLib.Append("hakurei/base/org.debian"), true, false, false, true}},
				{&FSEphemeral{Target: container.AbsFHSTmp, Write: true, Perm: 0755}},
				{&FSOverlay{
					Target: container.MustAbs("/nix/store"),
					Lower:  []*container.Absolute{container.MustAbs("/mnt-root/nix/.ro-store")},
					Upper:  container.MustAbs("/mnt-root/nix/.rw-store/upper"),
					Work:   container.MustAbs("/mnt-root/nix/.rw-store/work"),
				}},
				{&FSBind{Source: container.MustAbs("/nix/store")}},
				{&FSBind{Source: container.AbsFHSRun.Append("current-system")}},
				{&FSBind{Source: container.AbsFHSRun.Append("opengl-driver")}},
				{&FSBind{Source: container.AbsFHSVarLib.Append("hakurei/u0/org.chromium.Chromium"),
					Target: container.MustAbs("/data/data/org.chromium.Chromium"), Write: true}},
				{&FSBind{Source: container.AbsFHSDev.Append("dri"), Device: true, Optional: true}},
			},
			Link:    []LinkConfig{{container.AbsFHSRunUser.Append("65534"), container.FHSRunUser + "150"}},
			Etc:     container.AbsFHSEtc,
			AutoEtc: true,
		},
	}
}
