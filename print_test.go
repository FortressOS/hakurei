package main

import (
	"strings"
	"testing"
	"time"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/app"
	"git.gensokyo.uk/security/fortify/internal/state"
)

var (
	testID = app.ID{
		0x8e, 0x2c, 0x76, 0xb0,
		0x66, 0xda, 0xbe, 0x57,
		0x4c, 0xf0, 0x73, 0xbd,
		0xb4, 0x6e, 0xb5, 0xc1,
	}
	testState = &state.State{
		ID:     testID,
		PID:    0xDEADBEEF,
		Config: fst.Template(),
		Time:   testAppTime,
	}
	testTime    = time.Unix(3752, 1).UTC()
	testAppTime = time.Unix(0, 9).UTC()
)

func Test_printShowInstance(t *testing.T) {
	testCases := []struct {
		name        string
		instance    *state.State
		config      *fst.Config
		short, json bool
		want        string
	}{
		{"config", nil, fst.Template(), false, false, `App
 ID:             9 (org.chromium.Chromium)
 Enablements:    wayland, dbus, pulseaudio
 Groups:         ["video"]
 Directory:      /var/lib/persist/home/org.chromium.Chromium
 Hostname:       "localhost"
 Flags:          userns devel net device tty mapuid autoetc
 Etc:            /etc
 Cover:          /var/run/nscd
 Path:           /run/current-system/sw/bin/chromium
 Arguments:      chromium --ignore-gpu-blocklist --disable-smooth-scrolling --enable-features=UseOzonePlatform --ozone-platform=wayland

Filesystem
 +/nix/store
 +/run/current-system
 +/run/opengl-driver
 +/var/db/nix-channels
 w*/var/lib/fortify/u0/org.chromium.Chromium:/data/data/org.chromium.Chromium
 d+/dev/dri

Extra ACL
 --x+:/var/lib/fortify/u0
 rwx:/var/lib/fortify/u0/org.chromium.Chromium

Session bus
 Filter:       true
 Talk:         ["org.freedesktop.Notifications" "org.freedesktop.FileManager1" "org.freedesktop.ScreenSaver" "org.freedesktop.secrets" "org.kde.kwalletd5" "org.kde.kwalletd6" "org.gnome.SessionManager"]
 Own:          ["org.chromium.Chromium.*" "org.mpris.MediaPlayer2.org.chromium.Chromium.*" "org.mpris.MediaPlayer2.chromium.*"]
 Call:         map["org.freedesktop.portal.*":"*"]
 Broadcast:    map["org.freedesktop.portal.*":"@/org/freedesktop/portal/*"]

System bus
 Filter:    true
 Talk:      ["org.bluez" "org.freedesktop.Avahi" "org.freedesktop.UPower"]

`},
		{"config pd", nil, new(fst.Config), false, false, `Warning: this configuration uses permissive defaults!

App
 ID:             0
 Enablements:    (no enablements)

`},
		{"config flag none", nil, &fst.Config{Confinement: fst.ConfinementConfig{Sandbox: new(fst.SandboxConfig)}}, false, false, `App
 ID:             0
 Enablements:    (no enablements)
 Flags:          none
 Etc:            /etc
 Path:           

`},
		{"config nil entries", nil, &fst.Config{Confinement: fst.ConfinementConfig{Sandbox: &fst.SandboxConfig{Filesystem: make([]*fst.FilesystemConfig, 1)}, ExtraPerms: make([]*fst.ExtraPermConfig, 1)}}, false, false, `App
 ID:             0
 Enablements:    (no enablements)
 Flags:          none
 Etc:            /etc
 Path:           

Filesystem

Extra ACL

`},
		{"config pd dbus see", nil, &fst.Config{Confinement: fst.ConfinementConfig{SessionBus: &dbus.Config{See: []string{"org.example.test"}}}}, false, false, `Warning: this configuration uses permissive defaults!

App
 ID:             0
 Enablements:    (no enablements)

Session bus
 Filter:    false
 See:       ["org.example.test"]

`},

		{"instance", testState, fst.Template(), false, false, `State
 Instance:    8e2c76b066dabe574cf073bdb46eb5c1 (3735928559)
 Uptime:      1h2m32s

App
 ID:             9 (org.chromium.Chromium)
 Enablements:    wayland, dbus, pulseaudio
 Groups:         ["video"]
 Directory:      /var/lib/persist/home/org.chromium.Chromium
 Hostname:       "localhost"
 Flags:          userns devel net device tty mapuid autoetc
 Etc:            /etc
 Cover:          /var/run/nscd
 Path:           /run/current-system/sw/bin/chromium
 Arguments:      chromium --ignore-gpu-blocklist --disable-smooth-scrolling --enable-features=UseOzonePlatform --ozone-platform=wayland

Filesystem
 +/nix/store
 +/run/current-system
 +/run/opengl-driver
 +/var/db/nix-channels
 w*/var/lib/fortify/u0/org.chromium.Chromium:/data/data/org.chromium.Chromium
 d+/dev/dri

Extra ACL
 --x+:/var/lib/fortify/u0
 rwx:/var/lib/fortify/u0/org.chromium.Chromium

Session bus
 Filter:       true
 Talk:         ["org.freedesktop.Notifications" "org.freedesktop.FileManager1" "org.freedesktop.ScreenSaver" "org.freedesktop.secrets" "org.kde.kwalletd5" "org.kde.kwalletd6" "org.gnome.SessionManager"]
 Own:          ["org.chromium.Chromium.*" "org.mpris.MediaPlayer2.org.chromium.Chromium.*" "org.mpris.MediaPlayer2.chromium.*"]
 Call:         map["org.freedesktop.portal.*":"*"]
 Broadcast:    map["org.freedesktop.portal.*":"@/org/freedesktop/portal/*"]

System bus
 Filter:    true
 Talk:      ["org.bluez" "org.freedesktop.Avahi" "org.freedesktop.UPower"]

`},
		{"instance pd", testState, new(fst.Config), false, false, `Warning: this configuration uses permissive defaults!

State
 Instance:    8e2c76b066dabe574cf073bdb46eb5c1 (3735928559)
 Uptime:      1h2m32s

App
 ID:             0
 Enablements:    (no enablements)

`},

		{"json nil", nil, nil, false, true, `null
`},
		{"json instance", testState, nil, false, true, `{
  "instance": [
    142,
    44,
    118,
    176,
    102,
    218,
    190,
    87,
    76,
    240,
    115,
    189,
    180,
    110,
    181,
    193
  ],
  "pid": 3735928559,
  "config": {
    "id": "org.chromium.Chromium",
    "path": "/run/current-system/sw/bin/chromium",
    "args": [
      "chromium",
      "--ignore-gpu-blocklist",
      "--disable-smooth-scrolling",
      "--enable-features=UseOzonePlatform",
      "--ozone-platform=wayland"
    ],
    "confinement": {
      "app_id": 9,
      "groups": [
        "video"
      ],
      "username": "chronos",
      "home_inner": "/var/lib/fortify",
      "home": "/var/lib/persist/home/org.chromium.Chromium",
      "shell": "/run/current-system/sw/bin/zsh",
      "sandbox": {
        "hostname": "localhost",
        "seccomp": 32,
        "devel": true,
        "userns": true,
        "net": true,
        "tty": true,
        "multiarch": true,
        "env": {
          "GOOGLE_API_KEY": "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
          "GOOGLE_DEFAULT_CLIENT_ID": "77185425430.apps.googleusercontent.com",
          "GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT"
        },
        "map_real_uid": true,
        "device": true,
        "filesystem": [
          {
            "src": "/nix/store"
          },
          {
            "src": "/run/current-system"
          },
          {
            "src": "/run/opengl-driver"
          },
          {
            "src": "/var/db/nix-channels"
          },
          {
            "dst": "/data/data/org.chromium.Chromium",
            "src": "/var/lib/fortify/u0/org.chromium.Chromium",
            "write": true,
            "require": true
          },
          {
            "src": "/dev/dri",
            "dev": true
          }
        ],
        "symlink": [
          [
            "/run/user/65534",
            "/run/user/150"
          ]
        ],
        "etc": "/etc",
        "auto_etc": true,
        "cover": [
          "/var/run/nscd"
        ]
      },
      "extra_perms": [
        {
          "ensure": true,
          "path": "/var/lib/fortify/u0",
          "x": true
        },
        {
          "path": "/var/lib/fortify/u0/org.chromium.Chromium",
          "r": true,
          "w": true,
          "x": true
        }
      ],
      "system_bus": {
        "see": null,
        "talk": [
          "org.bluez",
          "org.freedesktop.Avahi",
          "org.freedesktop.UPower"
        ],
        "own": null,
        "call": null,
        "broadcast": null,
        "filter": true
      },
      "session_bus": {
        "see": null,
        "talk": [
          "org.freedesktop.Notifications",
          "org.freedesktop.FileManager1",
          "org.freedesktop.ScreenSaver",
          "org.freedesktop.secrets",
          "org.kde.kwalletd5",
          "org.kde.kwalletd6",
          "org.gnome.SessionManager"
        ],
        "own": [
          "org.chromium.Chromium.*",
          "org.mpris.MediaPlayer2.org.chromium.Chromium.*",
          "org.mpris.MediaPlayer2.chromium.*"
        ],
        "call": {
          "org.freedesktop.portal.*": "*"
        },
        "broadcast": {
          "org.freedesktop.portal.*": "@/org/freedesktop/portal/*"
        },
        "filter": true
      },
      "enablements": 13
    }
  },
  "time": "1970-01-01T00:00:00.000000009Z"
}
`},
		{"json config", nil, fst.Template(), false, true, `{
  "id": "org.chromium.Chromium",
  "path": "/run/current-system/sw/bin/chromium",
  "args": [
    "chromium",
    "--ignore-gpu-blocklist",
    "--disable-smooth-scrolling",
    "--enable-features=UseOzonePlatform",
    "--ozone-platform=wayland"
  ],
  "confinement": {
    "app_id": 9,
    "groups": [
      "video"
    ],
    "username": "chronos",
    "home_inner": "/var/lib/fortify",
    "home": "/var/lib/persist/home/org.chromium.Chromium",
    "shell": "/run/current-system/sw/bin/zsh",
    "sandbox": {
      "hostname": "localhost",
      "seccomp": 32,
      "devel": true,
      "userns": true,
      "net": true,
      "tty": true,
      "multiarch": true,
      "env": {
        "GOOGLE_API_KEY": "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
        "GOOGLE_DEFAULT_CLIENT_ID": "77185425430.apps.googleusercontent.com",
        "GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT"
      },
      "map_real_uid": true,
      "device": true,
      "filesystem": [
        {
          "src": "/nix/store"
        },
        {
          "src": "/run/current-system"
        },
        {
          "src": "/run/opengl-driver"
        },
        {
          "src": "/var/db/nix-channels"
        },
        {
          "dst": "/data/data/org.chromium.Chromium",
          "src": "/var/lib/fortify/u0/org.chromium.Chromium",
          "write": true,
          "require": true
        },
        {
          "src": "/dev/dri",
          "dev": true
        }
      ],
      "symlink": [
        [
          "/run/user/65534",
          "/run/user/150"
        ]
      ],
      "etc": "/etc",
      "auto_etc": true,
      "cover": [
        "/var/run/nscd"
      ]
    },
    "extra_perms": [
      {
        "ensure": true,
        "path": "/var/lib/fortify/u0",
        "x": true
      },
      {
        "path": "/var/lib/fortify/u0/org.chromium.Chromium",
        "r": true,
        "w": true,
        "x": true
      }
    ],
    "system_bus": {
      "see": null,
      "talk": [
        "org.bluez",
        "org.freedesktop.Avahi",
        "org.freedesktop.UPower"
      ],
      "own": null,
      "call": null,
      "broadcast": null,
      "filter": true
    },
    "session_bus": {
      "see": null,
      "talk": [
        "org.freedesktop.Notifications",
        "org.freedesktop.FileManager1",
        "org.freedesktop.ScreenSaver",
        "org.freedesktop.secrets",
        "org.kde.kwalletd5",
        "org.kde.kwalletd6",
        "org.gnome.SessionManager"
      ],
      "own": [
        "org.chromium.Chromium.*",
        "org.mpris.MediaPlayer2.org.chromium.Chromium.*",
        "org.mpris.MediaPlayer2.chromium.*"
      ],
      "call": {
        "org.freedesktop.portal.*": "*"
      },
      "broadcast": {
        "org.freedesktop.portal.*": "@/org/freedesktop/portal/*"
      },
      "filter": true
    },
    "enablements": 13
  }
}
`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := new(strings.Builder)
			printShowInstance(output, testTime, tc.instance, tc.config, tc.short, tc.json)
			if got := output.String(); got != tc.want {
				t.Errorf("printShowInstance: got\n%s\nwant\n%s",
					got, tc.want)
				return
			}
		})
	}
}

func Test_printPs(t *testing.T) {
	testCases := []struct {
		name        string
		entries     state.Entries
		short, json bool
		want        string
	}{
		{"no entries", make(state.Entries), false, false, "    Instance    PID    Application    Uptime\n"},
		{"no entries short", make(state.Entries), true, false, ""},
		{"nil instance", state.Entries{testID: nil}, false, false, "    Instance    PID    Application    Uptime\n"},
		{"state corruption", state.Entries{app.ID{}: testState}, false, false, "    Instance    PID    Application    Uptime\n"},

		{"valid pd", state.Entries{testID: &state.State{ID: testID, PID: 1 << 8, Config: new(fst.Config), Time: testAppTime}}, false, false, `    Instance    PID    Application                         Uptime
    8e2c76b0    256    0 (uk.gensokyo.fortify.8e2c76b0)    1h2m32s
`},

		{"valid", state.Entries{testID: testState}, false, false, `    Instance    PID           Application                  Uptime
    8e2c76b0    3735928559    9 (org.chromium.Chromium)    1h2m32s
`},
		{"valid short", state.Entries{testID: testState}, true, false, "8e2c76b0\n"},
		{"valid json", state.Entries{testID: testState}, false, true, `{
  "8e2c76b066dabe574cf073bdb46eb5c1": {
    "instance": [
      142,
      44,
      118,
      176,
      102,
      218,
      190,
      87,
      76,
      240,
      115,
      189,
      180,
      110,
      181,
      193
    ],
    "pid": 3735928559,
    "config": {
      "id": "org.chromium.Chromium",
      "path": "/run/current-system/sw/bin/chromium",
      "args": [
        "chromium",
        "--ignore-gpu-blocklist",
        "--disable-smooth-scrolling",
        "--enable-features=UseOzonePlatform",
        "--ozone-platform=wayland"
      ],
      "confinement": {
        "app_id": 9,
        "groups": [
          "video"
        ],
        "username": "chronos",
        "home_inner": "/var/lib/fortify",
        "home": "/var/lib/persist/home/org.chromium.Chromium",
        "shell": "/run/current-system/sw/bin/zsh",
        "sandbox": {
          "hostname": "localhost",
          "seccomp": 32,
          "devel": true,
          "userns": true,
          "net": true,
          "tty": true,
          "multiarch": true,
          "env": {
            "GOOGLE_API_KEY": "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
            "GOOGLE_DEFAULT_CLIENT_ID": "77185425430.apps.googleusercontent.com",
            "GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT"
          },
          "map_real_uid": true,
          "device": true,
          "filesystem": [
            {
              "src": "/nix/store"
            },
            {
              "src": "/run/current-system"
            },
            {
              "src": "/run/opengl-driver"
            },
            {
              "src": "/var/db/nix-channels"
            },
            {
              "dst": "/data/data/org.chromium.Chromium",
              "src": "/var/lib/fortify/u0/org.chromium.Chromium",
              "write": true,
              "require": true
            },
            {
              "src": "/dev/dri",
              "dev": true
            }
          ],
          "symlink": [
            [
              "/run/user/65534",
              "/run/user/150"
            ]
          ],
          "etc": "/etc",
          "auto_etc": true,
          "cover": [
            "/var/run/nscd"
          ]
        },
        "extra_perms": [
          {
            "ensure": true,
            "path": "/var/lib/fortify/u0",
            "x": true
          },
          {
            "path": "/var/lib/fortify/u0/org.chromium.Chromium",
            "r": true,
            "w": true,
            "x": true
          }
        ],
        "system_bus": {
          "see": null,
          "talk": [
            "org.bluez",
            "org.freedesktop.Avahi",
            "org.freedesktop.UPower"
          ],
          "own": null,
          "call": null,
          "broadcast": null,
          "filter": true
        },
        "session_bus": {
          "see": null,
          "talk": [
            "org.freedesktop.Notifications",
            "org.freedesktop.FileManager1",
            "org.freedesktop.ScreenSaver",
            "org.freedesktop.secrets",
            "org.kde.kwalletd5",
            "org.kde.kwalletd6",
            "org.gnome.SessionManager"
          ],
          "own": [
            "org.chromium.Chromium.*",
            "org.mpris.MediaPlayer2.org.chromium.Chromium.*",
            "org.mpris.MediaPlayer2.chromium.*"
          ],
          "call": {
            "org.freedesktop.portal.*": "*"
          },
          "broadcast": {
            "org.freedesktop.portal.*": "@/org/freedesktop/portal/*"
          },
          "filter": true
        },
        "enablements": 13
      }
    },
    "time": "1970-01-01T00:00:00.000000009Z"
  }
}
`},
		{"valid short json", state.Entries{testID: testState}, true, true, `["8e2c76b066dabe574cf073bdb46eb5c1"]
`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := new(strings.Builder)
			printPs(output, testTime, stubStore(tc.entries), tc.short, tc.json)
			if got := output.String(); got != tc.want {
				t.Errorf("printPs: got\n%s\nwant\n%s",
					got, tc.want)
				return
			}
		})
	}
}

// stubStore implements [state.Store] and returns test samples via [state.Joiner].
type stubStore state.Entries

func (s stubStore) Join() (state.Entries, error)               { return state.Entries(s), nil }
func (s stubStore) Do(int, func(c state.Cursor)) (bool, error) { panic("unreachable") }
func (s stubStore) List() ([]int, error)                       { panic("unreachable") }
func (s stubStore) Close() error                               { return nil }
