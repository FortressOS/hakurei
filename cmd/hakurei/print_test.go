package main

import (
	"strings"
	"testing"
	"time"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/system/dbus"
)

var (
	testID = state.ID{
		0x8e, 0x2c, 0x76, 0xb0,
		0x66, 0xda, 0xbe, 0x57,
		0x4c, 0xf0, 0x73, 0xbd,
		0xb4, 0x6e, 0xb5, 0xc1,
	}
	testState = &state.State{
		ID:     testID,
		PID:    0xDEADBEEF,
		Config: hst.Template(),
		Time:   testAppTime,
	}
	testTime    = time.Unix(3752, 1).UTC()
	testAppTime = time.Unix(0, 9).UTC()
)

func Test_printShowInstance(t *testing.T) {
	testCases := []struct {
		name        string
		instance    *state.State
		config      *hst.Config
		short, json bool
		want        string
	}{
		{"config", nil, hst.Template(), false, false, `App
 Identity:       9 (org.chromium.Chromium)
 Enablements:    wayland, dbus, pulseaudio
 Groups:         video, dialout, plugdev
 Data:           /var/lib/hakurei/u0/org.chromium.Chromium
 Hostname:       localhost
 Flags:          userns devel net abstract device tty mapuid
 Path:           /run/current-system/sw/bin/chromium
 Arguments:      chromium --ignore-gpu-blocklist --disable-smooth-scrolling --enable-features=UseOzonePlatform --ozone-platform=wayland

Filesystem
 autoroot:w:/var/lib/hakurei/base/org.debian
 autoetc:/etc/
 w+ephemeral(-rwxr-xr-x):/tmp/
 w*/nix/store:/mnt-root/nix/.rw-store/upper:/mnt-root/nix/.rw-store/work:/mnt-root/nix/.ro-store
 */nix/store
 &/run/current-system:*/run/current-system
 &/run/opengl-driver:*/run/opengl-driver
 w*/var/lib/hakurei/u0/org.chromium.Chromium:/data/data/org.chromium.Chromium
 d+/dev/dri

Extra ACL
 --x+:/var/lib/hakurei/u0
 rwx:/var/lib/hakurei/u0/org.chromium.Chromium

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
		{"config pd", nil, new(hst.Config), false, false, `Warning: this configuration uses permissive defaults!

App
 Identity:       0
 Enablements:    (no enablements)

`},
		{"config flag none", nil, &hst.Config{Container: new(hst.ContainerConfig)}, false, false, `App
 Identity:       0
 Enablements:    (no enablements)
 Flags:          none

`},
		{"config nil entries", nil, &hst.Config{Container: &hst.ContainerConfig{Filesystem: make([]hst.FilesystemConfigJSON, 1)}, ExtraPerms: make([]*hst.ExtraPermConfig, 1)}, false, false, `App
 Identity:       0
 Enablements:    (no enablements)
 Flags:          none

Filesystem
 <invalid>

Extra ACL

`},
		{"config pd dbus see", nil, &hst.Config{SessionBus: &dbus.Config{See: []string{"org.example.test"}}}, false, false, `Warning: this configuration uses permissive defaults!

App
 Identity:       0
 Enablements:    (no enablements)

Session bus
 Filter:    false
 See:       ["org.example.test"]

`},

		{"instance", testState, hst.Template(), false, false, `State
 Instance:    8e2c76b066dabe574cf073bdb46eb5c1 (3735928559)
 Uptime:      1h2m32s

App
 Identity:       9 (org.chromium.Chromium)
 Enablements:    wayland, dbus, pulseaudio
 Groups:         video, dialout, plugdev
 Data:           /var/lib/hakurei/u0/org.chromium.Chromium
 Hostname:       localhost
 Flags:          userns devel net abstract device tty mapuid
 Path:           /run/current-system/sw/bin/chromium
 Arguments:      chromium --ignore-gpu-blocklist --disable-smooth-scrolling --enable-features=UseOzonePlatform --ozone-platform=wayland

Filesystem
 autoroot:w:/var/lib/hakurei/base/org.debian
 autoetc:/etc/
 w+ephemeral(-rwxr-xr-x):/tmp/
 w*/nix/store:/mnt-root/nix/.rw-store/upper:/mnt-root/nix/.rw-store/work:/mnt-root/nix/.ro-store
 */nix/store
 &/run/current-system:*/run/current-system
 &/run/opengl-driver:*/run/opengl-driver
 w*/var/lib/hakurei/u0/org.chromium.Chromium:/data/data/org.chromium.Chromium
 d+/dev/dri

Extra ACL
 --x+:/var/lib/hakurei/u0
 rwx:/var/lib/hakurei/u0/org.chromium.Chromium

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
		{"instance pd", testState, new(hst.Config), false, false, `Warning: this configuration uses permissive defaults!

State
 Instance:    8e2c76b066dabe574cf073bdb46eb5c1 (3735928559)
 Uptime:      1h2m32s

App
 Identity:       0
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
    "enablements": {
      "wayland": true,
      "dbus": true,
      "pulse": true
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
    "username": "chronos",
    "shell": "/run/current-system/sw/bin/zsh",
    "data": "/var/lib/hakurei/u0/org.chromium.Chromium",
    "dir": "/data/data/org.chromium.Chromium",
    "extra_perms": [
      {
        "ensure": true,
        "path": "/var/lib/hakurei/u0",
        "x": true
      },
      {
        "path": "/var/lib/hakurei/u0/org.chromium.Chromium",
        "r": true,
        "w": true,
        "x": true
      }
    ],
    "identity": 9,
    "groups": [
      "video",
      "dialout",
      "plugdev"
    ],
    "container": {
      "hostname": "localhost",
      "wait_delay": -1,
      "seccomp_flags": 1,
      "seccomp_presets": 1,
      "seccomp_compat": true,
      "devel": true,
      "userns": true,
      "host_net": true,
      "host_abstract": true,
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
          "type": "bind",
          "dst": "/",
          "src": "/var/lib/hakurei/base/org.debian",
          "write": true,
          "special": true
        },
        {
          "type": "bind",
          "dst": "/etc/",
          "src": "/etc/",
          "special": true
        },
        {
          "type": "ephemeral",
          "dst": "/tmp/",
          "write": true,
          "perm": 493
        },
        {
          "type": "overlay",
          "dst": "/nix/store",
          "lower": [
            "/mnt-root/nix/.ro-store"
          ],
          "upper": "/mnt-root/nix/.rw-store/upper",
          "work": "/mnt-root/nix/.rw-store/work"
        },
        {
          "type": "bind",
          "src": "/nix/store"
        },
        {
          "type": "link",
          "dst": "/run/current-system",
          "linkname": "/run/current-system",
          "dereference": true
        },
        {
          "type": "link",
          "dst": "/run/opengl-driver",
          "linkname": "/run/opengl-driver",
          "dereference": true
        },
        {
          "type": "bind",
          "dst": "/data/data/org.chromium.Chromium",
          "src": "/var/lib/hakurei/u0/org.chromium.Chromium",
          "write": true
        },
        {
          "type": "bind",
          "src": "/dev/dri",
          "dev": true,
          "optional": true
        }
      ]
    }
  },
  "time": "1970-01-01T00:00:00.000000009Z"
}
`},
		{"json config", nil, hst.Template(), false, true, `{
  "id": "org.chromium.Chromium",
  "path": "/run/current-system/sw/bin/chromium",
  "args": [
    "chromium",
    "--ignore-gpu-blocklist",
    "--disable-smooth-scrolling",
    "--enable-features=UseOzonePlatform",
    "--ozone-platform=wayland"
  ],
  "enablements": {
    "wayland": true,
    "dbus": true,
    "pulse": true
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
  "username": "chronos",
  "shell": "/run/current-system/sw/bin/zsh",
  "data": "/var/lib/hakurei/u0/org.chromium.Chromium",
  "dir": "/data/data/org.chromium.Chromium",
  "extra_perms": [
    {
      "ensure": true,
      "path": "/var/lib/hakurei/u0",
      "x": true
    },
    {
      "path": "/var/lib/hakurei/u0/org.chromium.Chromium",
      "r": true,
      "w": true,
      "x": true
    }
  ],
  "identity": 9,
  "groups": [
    "video",
    "dialout",
    "plugdev"
  ],
  "container": {
    "hostname": "localhost",
    "wait_delay": -1,
    "seccomp_flags": 1,
    "seccomp_presets": 1,
    "seccomp_compat": true,
    "devel": true,
    "userns": true,
    "host_net": true,
    "host_abstract": true,
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
        "type": "bind",
        "dst": "/",
        "src": "/var/lib/hakurei/base/org.debian",
        "write": true,
        "special": true
      },
      {
        "type": "bind",
        "dst": "/etc/",
        "src": "/etc/",
        "special": true
      },
      {
        "type": "ephemeral",
        "dst": "/tmp/",
        "write": true,
        "perm": 493
      },
      {
        "type": "overlay",
        "dst": "/nix/store",
        "lower": [
          "/mnt-root/nix/.ro-store"
        ],
        "upper": "/mnt-root/nix/.rw-store/upper",
        "work": "/mnt-root/nix/.rw-store/work"
      },
      {
        "type": "bind",
        "src": "/nix/store"
      },
      {
        "type": "link",
        "dst": "/run/current-system",
        "linkname": "/run/current-system",
        "dereference": true
      },
      {
        "type": "link",
        "dst": "/run/opengl-driver",
        "linkname": "/run/opengl-driver",
        "dereference": true
      },
      {
        "type": "bind",
        "dst": "/data/data/org.chromium.Chromium",
        "src": "/var/lib/hakurei/u0/org.chromium.Chromium",
        "write": true
      },
      {
        "type": "bind",
        "src": "/dev/dri",
        "dev": true,
        "optional": true
      }
    ]
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
		{"state corruption", state.Entries{state.ID{}: testState}, false, false, "    Instance    PID    Application    Uptime\n"},

		{"valid pd", state.Entries{testID: &state.State{ID: testID, PID: 1 << 8, Config: new(hst.Config), Time: testAppTime}}, false, false, `    Instance    PID    Application                 Uptime
    8e2c76b0    256    0 (app.hakurei.8e2c76b0)    1h2m32s
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
      "enablements": {
        "wayland": true,
        "dbus": true,
        "pulse": true
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
      "username": "chronos",
      "shell": "/run/current-system/sw/bin/zsh",
      "data": "/var/lib/hakurei/u0/org.chromium.Chromium",
      "dir": "/data/data/org.chromium.Chromium",
      "extra_perms": [
        {
          "ensure": true,
          "path": "/var/lib/hakurei/u0",
          "x": true
        },
        {
          "path": "/var/lib/hakurei/u0/org.chromium.Chromium",
          "r": true,
          "w": true,
          "x": true
        }
      ],
      "identity": 9,
      "groups": [
        "video",
        "dialout",
        "plugdev"
      ],
      "container": {
        "hostname": "localhost",
        "wait_delay": -1,
        "seccomp_flags": 1,
        "seccomp_presets": 1,
        "seccomp_compat": true,
        "devel": true,
        "userns": true,
        "host_net": true,
        "host_abstract": true,
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
            "type": "bind",
            "dst": "/",
            "src": "/var/lib/hakurei/base/org.debian",
            "write": true,
            "special": true
          },
          {
            "type": "bind",
            "dst": "/etc/",
            "src": "/etc/",
            "special": true
          },
          {
            "type": "ephemeral",
            "dst": "/tmp/",
            "write": true,
            "perm": 493
          },
          {
            "type": "overlay",
            "dst": "/nix/store",
            "lower": [
              "/mnt-root/nix/.ro-store"
            ],
            "upper": "/mnt-root/nix/.rw-store/upper",
            "work": "/mnt-root/nix/.rw-store/work"
          },
          {
            "type": "bind",
            "src": "/nix/store"
          },
          {
            "type": "link",
            "dst": "/run/current-system",
            "linkname": "/run/current-system",
            "dereference": true
          },
          {
            "type": "link",
            "dst": "/run/opengl-driver",
            "linkname": "/run/opengl-driver",
            "dereference": true
          },
          {
            "type": "bind",
            "dst": "/data/data/org.chromium.Chromium",
            "src": "/var/lib/hakurei/u0/org.chromium.Chromium",
            "write": true
          },
          {
            "type": "bind",
            "src": "/dev/dri",
            "dev": true,
            "optional": true
          }
        ]
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
