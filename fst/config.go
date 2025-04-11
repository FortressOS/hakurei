package fst

import (
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/sandbox/seccomp"
	"git.gensokyo.uk/security/fortify/system"
)

const Tmp = "/.fortify"

// Config is used to seal an app
type Config struct {
	// reverse-DNS style arbitrary identifier string from config;
	// passed to wayland security-context-v1 as application ID
	// and used as part of defaults in dbus session proxy
	ID string `json:"id"`

	// absolute path to executable file
	Path string `json:"path,omitempty"`
	// final args passed to container init
	Args []string `json:"args"`

	Confinement ConfinementConfig `json:"confinement"`
}

// ConfinementConfig defines fortified child's confinement
type ConfinementConfig struct {
	// numerical application id, determines uid in the init namespace
	AppID int `json:"app_id"`
	// list of supplementary groups to inherit
	Groups []string `json:"groups"`
	// passwd username in container, defaults to passwd name of target uid or chronos
	Username string `json:"username,omitempty"`
	// home directory in container, empty for outer
	Inner string `json:"home_inner"`
	// home directory in init namespace
	Outer string `json:"home"`
	// absolute path to shell, empty for host shell
	Shell string `json:"shell,omitempty"`
	// abstract sandbox configuration
	Sandbox *SandboxConfig `json:"sandbox"`
	// extra acl ops, runs after everything else
	ExtraPerms []*ExtraPermConfig `json:"extra_perms,omitempty"`

	// reference to a system D-Bus proxy configuration,
	// nil value disables system bus proxy
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// reference to a session D-Bus proxy configuration,
	// nil value makes session bus proxy assume built-in defaults
	SessionBus *dbus.Config `json:"session_bus,omitempty"`

	// system resources to expose to the container
	Enablements system.Enablement `json:"enablements"`
}

type ExtraPermConfig struct {
	Ensure  bool   `json:"ensure,omitempty"`
	Path    string `json:"path"`
	Read    bool   `json:"r,omitempty"`
	Write   bool   `json:"w,omitempty"`
	Execute bool   `json:"x,omitempty"`
}

func (e *ExtraPermConfig) String() string {
	buf := make([]byte, 0, 5+len(e.Path))
	buf = append(buf, '-', '-', '-')
	if e.Ensure {
		buf = append(buf, '+')
	}
	buf = append(buf, ':')
	buf = append(buf, []byte(e.Path)...)
	if e.Read {
		buf[0] = 'r'
	}
	if e.Write {
		buf[1] = 'w'
	}
	if e.Execute {
		buf[2] = 'x'
	}
	return string(buf)
}

// Template returns a fully populated instance of Config.
func Template() *Config {
	return &Config{
		ID:   "org.chromium.Chromium",
		Path: "/run/current-system/sw/bin/chromium",
		Args: []string{
			"chromium",
			"--ignore-gpu-blocklist",
			"--disable-smooth-scrolling",
			"--enable-features=UseOzonePlatform",
			"--ozone-platform=wayland",
		},
		Confinement: ConfinementConfig{
			AppID:    9,
			Groups:   []string{"video"},
			Username: "chronos",
			Outer:    "/var/lib/persist/home/org.chromium.Chromium",
			Inner:    "/var/lib/fortify",
			Shell:    "/run/current-system/sw/bin/zsh",
			Sandbox: &SandboxConfig{
				Hostname:      "localhost",
				Devel:         true,
				Userns:        true,
				Net:           true,
				Device:        true,
				Seccomp:       seccomp.FilterMultiarch,
				Tty:           true,
				Multiarch:     true,
				MapRealUID:    true,
				DirectWayland: false,
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
			ExtraPerms: []*ExtraPermConfig{
				{Path: "/var/lib/fortify/u0", Ensure: true, Execute: true},
				{Path: "/var/lib/fortify/u0/org.chromium.Chromium", Read: true, Write: true, Execute: true},
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
			Enablements: system.EWayland | system.EDBus | system.EPulse,
		},
	}
}
