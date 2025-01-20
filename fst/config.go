package fst

import (
	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/internal/system"
)

const Tmp = "/.fortify"

// Config is used to seal an *App
type Config struct {
	// application ID
	ID string `json:"id"`
	// value passed through to the child process as its argv
	Command []string `json:"command"`

	Confinement ConfinementConfig `json:"confinement"`
}

// ConfinementConfig defines fortified child's confinement
type ConfinementConfig struct {
	// numerical application id, determines uid in the init namespace
	AppID int `json:"app_id"`
	// list of supplementary groups to inherit
	Groups []string `json:"groups"`
	// passwd username in the sandbox, defaults to passwd name of target uid or chronos
	Username string `json:"username,omitempty"`
	// home directory in sandbox, empty for outer
	Inner string `json:"home_inner"`
	// home directory in init namespace
	Outer string `json:"home"`
	// bwrap sandbox confinement configuration
	Sandbox *SandboxConfig `json:"sandbox"`
	// seccomp syscall filter configuration
	Syscall *SyscallConfig `json:"syscall"`
	// extra acl entries to append
	ExtraPerms []*ExtraPermConfig `json:"extra_perms,omitempty"`

	// reference to a system D-Bus proxy configuration,
	// nil value disables system bus proxy
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// reference to a session D-Bus proxy configuration,
	// nil value makes session bus proxy assume built-in defaults
	SessionBus *dbus.Config `json:"session_bus,omitempty"`

	// system resources to expose to the sandbox
	Enablements system.Enablements `json:"enablements"`
}

type SyscallConfig struct {
	DenyDevel bool `json:"deny_devel"`
	Multiarch bool `json:"multiarch"`
	Linux32   bool `json:"linux32"`
	Can       bool `json:"can"`
	Bluetooth bool `json:"bluetooth"`
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

type FilesystemConfig struct {
	// mount point in sandbox, same as src if empty
	Dst string `json:"dst,omitempty"`
	// host filesystem path to make available to sandbox
	Src string `json:"src"`
	// write access
	Write bool `json:"write,omitempty"`
	// device access
	Device bool `json:"dev,omitempty"`
	// fail if mount fails
	Must bool `json:"require,omitempty"`
}

// Template returns a fully populated instance of Config.
func Template() *Config {
	return &Config{
		ID: "org.chromium.Chromium",
		Command: []string{
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
			Sandbox: &SandboxConfig{
				Hostname:      "localhost",
				UserNS:        true,
				Net:           true,
				NoNewSession:  true,
				MapRealUID:    true,
				Dev:           true,
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
					{Src: "/home/chronos", Write: true, Must: true},
					{Src: "/dev/dri", Device: true},
				},
				Link:     [][2]string{{"/run/user/65534", "/run/user/150"}},
				Etc:      "/etc",
				AutoEtc:  true,
				Override: []string{"/var/run/nscd"},
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
			Enablements: system.EWayland.Mask() | system.EDBus.Mask() | system.EPulse.Mask(),
		},
	}
}
