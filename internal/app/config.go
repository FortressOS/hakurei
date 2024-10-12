package app

import (
	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/helper/bwrap"
	"git.ophivana.moe/cat/fortify/internal/state"
)

// Config is used to seal an *App
type Config struct {
	// D-Bus application ID
	ID string `json:"id"`
	// username of the target user to switch to
	User string `json:"user"`
	// value passed through to the child process as its argv
	Command []string `json:"command"`
	// string representation of the child's launch method
	Method string `json:"method"`

	// child confinement configuration
	Confinement ConfinementConfig `json:"confinement"`
}

// ConfinementConfig defines fortified child's confinement
type ConfinementConfig struct {
	// bwrap sandbox confinement configuration
	Sandbox *SandboxConfig `json:"sandbox"`

	// reference to a system D-Bus proxy configuration,
	// nil value disables system bus proxy
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// reference to a session D-Bus proxy configuration,
	// nil value makes session bus proxy assume built-in defaults
	SessionBus *dbus.Config `json:"session_bus,omitempty"`

	// child capability enablements
	Enablements state.Enablements `json:"enablements"`
}

// SandboxConfig describes resources made available to the sandbox.
type SandboxConfig struct {
	// unix hostname within sandbox
	Hostname string `json:"hostname,omitempty"`
	// userns availability within sandbox
	UserNS bool `json:"userns,omitempty"`
	// share net namespace
	Net bool `json:"net,omitempty"`
	// do not run in new session
	NoNewSession bool `json:"no_new_session,omitempty"`
	// mediated access to wayland socket
	Wayland bool `json:"wayland,omitempty"`

	UID int `json:"uid,omitempty"`
	GID int `json:"gid,omitempty"`
	// final environment variables
	Env map[string]string `json:"env"`

	// sandbox host filesystem access
	Filesystem []*FilesystemConfig `json:"filesystem"`
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

func (s *SandboxConfig) Bwrap() *bwrap.Config {
	if s == nil {
		return nil
	}

	conf := &bwrap.Config{
		Net:           s.Net,
		UserNS:        s.UserNS,
		Hostname:      s.Hostname,
		Clearenv:      true,
		SetEnv:        s.Env,
		Procfs:        []string{"/proc"},
		DevTmpfs:      []string{"/dev"},
		Mqueue:        []string{"/dev/mqueue"},
		NewSession:    !s.NoNewSession,
		DieWithParent: true,
	}
	if s.UID > 0 {
		conf.UID = &s.UID
	}
	if s.GID > 0 {
		conf.GID = &s.GID
	}

	for _, c := range s.Filesystem {
		if c == nil {
			continue
		}
		p := [2]string{c.Src, c.Dst}
		if c.Dst == "" {
			p[1] = c.Src
		}

		switch {
		case c.Device:
			if c.Must {
				conf.DevBind = append(conf.DevBind, p)
			} else {
				conf.DevBindTry = append(conf.DevBindTry, p)
			}
		case c.Write:
			if c.Must {
				conf.Bind = append(conf.Bind, p)
			} else {
				conf.BindTry = append(conf.BindTry, p)
			}
		default:
			if c.Must {
				conf.ROBind = append(conf.ROBind, p)
			} else {
				conf.ROBindTry = append(conf.ROBindTry, p)
			}
		}
	}

	return conf
}

// Template returns a fully populated instance of Config.
func Template() *Config {
	return &Config{
		ID:   "org.chromium.Chromium",
		User: "chronos",
		Command: []string{
			"chromium",
			"--ignore-gpu-blocklist",
			"--disable-smooth-scrolling",
			"--enable-features=UseOzonePlatform",
			"--ozone-platform=wayland",
		},
		Method: "sudo",
		Confinement: ConfinementConfig{
			Sandbox: &SandboxConfig{
				Hostname:     "localhost",
				UserNS:       true,
				Net:          true,
				NoNewSession: true,
				Wayland:      false,
				UID:          150,
				GID:          101,
				// example API credentials pulled from Google Chrome
				// DO NOT USE THESE IN A REAL BROWSER
				Env: map[string]string{
					"GOOGLE_API_KEY":               "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
					"GOOGLE_DEFAULT_CLIENT_ID":     "77185425430.apps.googleusercontent.com",
					"GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT",
				},
				Filesystem: []*FilesystemConfig{
					{Src: "/nix"},
					{Src: "/storage/emulated/0", Write: true, Must: true},
					{Src: "/data/user/0", Dst: "/data/data", Write: true, Must: true},
					{Src: "/var/tmp", Write: true},
				},
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
			Enablements: state.EnableWayland.Mask() | state.EnableDBus.Mask() | state.EnablePulse.Mask(),
		},
	}
}
