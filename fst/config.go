package fst

import (
	"errors"

	"git.ophivana.moe/security/fortify/dbus"
	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal/linux"
	"git.ophivana.moe/security/fortify/internal/system"
)

const fTmp = "/fortify"

// Config is used to seal an *App
type Config struct {
	// D-Bus application ID
	ID string `json:"id"`
	// value passed through to the child process as its argv
	Command []string `json:"command"`

	// child confinement configuration
	Confinement ConfinementConfig `json:"confinement"`
}

// ConfinementConfig defines fortified child's confinement
type ConfinementConfig struct {
	// numerical application id, determines uid in the init namespace
	AppID int `json:"app_id"`
	// list of supplementary groups to inherit
	Groups []string `json:"groups"`
	// passwd username in the sandbox, defaults to chronos
	Username string `json:"username,omitempty"`
	// home directory in sandbox, empty for outer
	Inner string `json:"home_inner"`
	// home directory in init namespace
	Outer string `json:"home"`
	// bwrap sandbox confinement configuration
	Sandbox *SandboxConfig `json:"sandbox"`

	// reference to a system D-Bus proxy configuration,
	// nil value disables system bus proxy
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// reference to a session D-Bus proxy configuration,
	// nil value makes session bus proxy assume built-in defaults
	SessionBus *dbus.Config `json:"session_bus,omitempty"`

	// child capability enablements
	Enablements system.Enablements `json:"enablements"`
}

// SandboxConfig describes resources made available to the sandbox.
type SandboxConfig struct {
	// unix hostname within sandbox
	Hostname string `json:"hostname,omitempty"`
	// userns availability within sandbox
	UserNS bool `json:"userns,omitempty"`
	// share net namespace
	Net bool `json:"net,omitempty"`
	// share all devices
	Dev bool `json:"dev,omitempty"`
	// do not run in new session
	NoNewSession bool `json:"no_new_session,omitempty"`
	// map target user uid to privileged user uid in the user namespace
	MapRealUID bool `json:"map_real_uid"`
	// direct access to wayland socket
	DirectWayland bool `json:"direct_wayland,omitempty"`

	// final environment variables
	Env map[string]string `json:"env"`
	// sandbox host filesystem access
	Filesystem []*FilesystemConfig `json:"filesystem"`
	// symlinks created inside the sandbox
	Link [][2]string `json:"symlink"`
	// automatically set up /etc symlinks
	AutoEtc bool `json:"auto_etc"`
	// paths to override by mounting tmpfs over them
	Override []string `json:"override"`
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
	// exit if unable to share
	Must bool `json:"require,omitempty"`
}

// Bwrap returns the address of the corresponding bwrap.Config to s.
// Note that remaining tmpfs entries must be queued by the caller prior to launch.
func (s *SandboxConfig) Bwrap(os linux.System) (*bwrap.Config, error) {
	if s == nil {
		return nil, errors.New("nil sandbox config")
	}

	var uid int
	if !s.MapRealUID {
		uid = 65534
	} else {
		uid = os.Geteuid()
	}

	conf := (&bwrap.Config{
		Net:           s.Net,
		UserNS:        s.UserNS,
		Hostname:      s.Hostname,
		Clearenv:      true,
		SetEnv:        s.Env,
		NewSession:    !s.NoNewSession,
		DieWithParent: true,
		AsInit:        true,

		// initialise map
		Chmod: make(bwrap.ChmodConfig),
	}).
		SetUID(uid).SetGID(uid).
		Procfs("/proc").
		Tmpfs(fTmp, 4*1024)

	if !s.Dev {
		conf.DevTmpfs("/dev").Mqueue("/dev/mqueue")
	} else {
		conf.Bind("/dev", "/dev", false, true, true)
	}

	if !s.AutoEtc {
		conf.Dir("/etc")
	}

	for _, c := range s.Filesystem {
		if c == nil {
			continue
		}
		src := c.Src
		dest := c.Dst
		if c.Dst == "" {
			dest = c.Src
		}
		conf.Bind(src, dest, !c.Must, c.Write, c.Device)
	}

	for _, l := range s.Link {
		conf.Symlink(l[0], l[1])
	}

	if s.AutoEtc {
		conf.Bind("/etc", fTmp+"/etc")

		// link host /etc contents to prevent passwd/group from being overwritten
		if d, err := os.ReadDir("/etc"); err != nil {
			return nil, err
		} else {
			for _, ent := range d {
				name := ent.Name()
				switch name {
				case "passwd":
				case "group":

				case "mtab":
					conf.Symlink("/proc/mounts", "/etc/"+name)
				default:
					conf.Symlink(fTmp+"/etc/"+name, "/etc/"+name)
				}
			}
		}
	}

	return conf, nil
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
