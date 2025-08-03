package hst

import (
	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/system"
	"hakurei.app/system/dbus"
)

// Template returns a fully populated instance of Config.
func Template() *Config {
	return &Config{
		ID: "org.chromium.Chromium",

		Path: container.FHSRun + "current-system/sw/bin/chromium",
		Args: []string{
			"chromium",
			"--ignore-gpu-blocklist",
			"--disable-smooth-scrolling",
			"--enable-features=UseOzonePlatform",
			"--ozone-platform=wayland",
		},

		Enablements: system.EWayland | system.EDBus | system.EPulse,

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
		Shell:    container.FHSRun + "current-system/sw/bin/zsh",
		Data:     container.FHSVarLib + "hakurei/u0/org.chromium.Chromium",
		Dir:      "/data/data/org.chromium.Chromium",
		ExtraPerms: []*ExtraPermConfig{
			{Path: container.FHSVarLib + "hakurei/u0", Ensure: true, Execute: true},
			{Path: container.FHSVarLib + "hakurei/u0/org.chromium.Chromium", Read: true, Write: true, Execute: true},
		},

		Identity: 9,
		Groups:   []string{"video", "dialout", "plugdev"},

		Container: &ContainerConfig{
			Hostname:       "localhost",
			Devel:          true,
			Userns:         true,
			Net:            true,
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
			Filesystem: []*FilesystemConfig{
				{Dst: container.FHSTmp, Src: SourceTmpfs, Write: true},
				{Src: "/nix/store"},
				{Src: container.FHSRun + "current-system"},
				{Src: container.FHSRun + "opengl-driver"},
				{Src: container.FHSVar + "db/nix-channels"},
				{Src: container.FHSVarLib + "hakurei/u0/org.chromium.Chromium",
					Dst: "/data/data/org.chromium.Chromium", Write: true, Must: true},
				{Src: container.FHSDev + "dri", Device: true},
			},
			Link:      [][2]string{{container.FHSRunUser + "65534", container.FHSRunUser + "150"}},
			AutoRoot:  container.FHSVarLib + "hakurei/base/org.debian",
			RootFlags: container.BindWritable,
			Etc:       container.FHSEtc,
			AutoEtc:   true,
		},
	}
}
