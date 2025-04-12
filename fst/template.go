package fst

import (
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/sandbox/seccomp"
	"git.gensokyo.uk/security/fortify/system"
)

// Template returns a fully populated instance of Config.
func Template() *Config {
	return &Config{
		ID: "org.chromium.Chromium",

		Path: "/run/current-system/sw/bin/chromium",
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
		Shell:    "/run/current-system/sw/bin/zsh",
		Data:     "/var/lib/fortify/u0/org.chromium.Chromium",
		Dir:      "/data/data/org.chromium.Chromium",
		ExtraPerms: []*ExtraPermConfig{
			{Path: "/var/lib/fortify/u0", Ensure: true, Execute: true},
			{Path: "/var/lib/fortify/u0/org.chromium.Chromium", Read: true, Write: true, Execute: true},
		},

		Identity: 9,
		Groups:   []string{"video", "dialout", "plugdev"},

		Container: &ContainerConfig{
			Hostname:   "localhost",
			Devel:      true,
			Userns:     true,
			Net:        true,
			Device:     true,
			Seccomp:    seccomp.FilterMultiarch,
			Tty:        true,
			Multiarch:  true,
			MapRealUID: true,
			// example API credentials pulled from Google Chrome
			// DO NOT USE THESE IN A REAL BROWSER
			Env: map[string]string{
				"GOOGLE_API_KEY":               "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
				"GOOGLE_DEFAULT_CLIENT_ID":     "77185425430.apps.googleusercontent.com",
				"GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT",
			},
			Filesystem: []*FilesystemConfig{
				{Src: "/nix/store"},
				{Src: "/run/current-system"},
				{Src: "/run/opengl-driver"},
				{Src: "/var/db/nix-channels"},
				{Src: "/var/lib/fortify/u0/org.chromium.Chromium",
					Dst: "/data/data/org.chromium.Chromium", Write: true, Must: true},
				{Src: "/dev/dri", Device: true},
			},
			Link:    [][2]string{{"/run/user/65534", "/run/user/150"}},
			Etc:     "/etc",
			AutoEtc: true,
			Cover:   []string{"/var/run/nscd"},
		},
	}
}
