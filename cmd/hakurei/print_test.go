package main

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"

	"hakurei.app/container/check"
	"hakurei.app/hst"
	"hakurei.app/internal/store"
	"hakurei.app/message"
)

var (
	testID = hst.ID{
		0x8e, 0x2c, 0x76, 0xb0,
		0x66, 0xda, 0xbe, 0x57,
		0x4c, 0xf0, 0x73, 0xbd,
		0xb4, 0x6e, 0xb5, 0xc1,
	}
	testState = hst.State{
		ID:      testID,
		PID:     0xcafebabe,
		ShimPID: 0xdeadbeef,
		Config:  hst.Template(),
		Time:    testAppTime,
	}
	testStateSmall = hst.State{
		ID:      (hst.ID)(bytes.Repeat([]byte{0xaa}, len(hst.ID{}))),
		PID:     0xbeef,
		ShimPID: 0xcafe,
		Config: &hst.Config{
			Enablements: hst.NewEnablements(hst.EWayland | hst.EPulse),
			Identity:    1,
			Container: &hst.ContainerConfig{
				Shell: check.MustAbs("/bin/sh"),
				Home:  check.MustAbs("/data/data/uk.gensokyo.cat"),
				Path:  check.MustAbs("/usr/bin/cat"),
				Args:  []string{"cat"},
				Flags: hst.FUserns,
			},
		},
		Time: time.Unix(0, 0xdeadbeef).UTC(),
	}
	testTime    = time.Unix(3752, 1).UTC()
	testAppTime = time.Unix(0, 9).UTC()
)

func TestPrintShowInstance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		instance    *hst.State
		config      *hst.Config
		short, json bool
		want        string
		valid       bool
	}{
		{"nil", nil, nil, false, false, "Error: invalid configuration!\n\n", false},
		{"config", nil, hst.Template(), false, false, `App
 Identity:       9 (org.chromium.Chromium)
 Enablements:    wayland, dbus, pulseaudio
 Groups:         video, dialout, plugdev
 Home:           /data/data/org.chromium.Chromium
 Hostname:       localhost
 Flags:          multiarch, compat, devel, userns, net, abstract, tty, mapuid, device, runtime, tmpdir
 Path:           /run/current-system/sw/bin/chromium
 Arguments:      chromium --ignore-gpu-blocklist --disable-smooth-scrolling --enable-features=UseOzonePlatform --ozone-platform=wayland

Filesystem
 autoroot:w:/var/lib/hakurei/base/org.debian
 autoetc:/etc/
 w+ephemeral(-rwxr-xr-x):/tmp/
 w*/nix/store:/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper:/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work:/var/lib/hakurei/base/org.nixos/ro-store
 /run/current-system@
 /run/opengl-driver@
 w-/var/lib/hakurei/u0/org.chromium.Chromium:/data/data/org.chromium.Chromium
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

`, true},
		{"config pd", nil, new(hst.Config), false, false, `Error: configuration missing container state!

App
 Identity:       0
 Enablements:    (no enablements)

`, false},
		{"config flag none", nil, &hst.Config{Container: new(hst.ContainerConfig)}, false, false, `Error: container configuration missing path to home directory!

App
 Identity:       0
 Enablements:    (no enablements)
 Flags:          none

`, false},
		{"config flag none directwl", nil, &hst.Config{DirectWayland: true, Container: new(hst.ContainerConfig)}, false, false, `Error: container configuration missing path to home directory!

App
 Identity:       0
 Enablements:    (no enablements)
 Flags:          directwl

`, false},
		{"config flag directwl", nil, &hst.Config{DirectWayland: true, Container: &hst.ContainerConfig{Flags: hst.FMultiarch}}, false, false, `Error: container configuration missing path to home directory!

App
 Identity:       0
 Enablements:    (no enablements)
 Flags:          multiarch, directwl

`, false},
		{"config nil entries", nil, &hst.Config{Container: &hst.ContainerConfig{Filesystem: make([]hst.FilesystemConfigJSON, 1)}, ExtraPerms: make([]hst.ExtraPermConfig, 1)}, false, false, `Error: container configuration missing path to home directory!

App
 Identity:       0
 Enablements:    (no enablements)
 Flags:          none

Filesystem
 <invalid>

Extra ACL
 <invalid>

`, false},
		{"config pd dbus see", nil, &hst.Config{SessionBus: &hst.BusConfig{See: []string{"org.example.test"}}}, false, false, `Error: configuration missing container state!

App
 Identity:       0
 Enablements:    (no enablements)

Session bus
 Filter:    false
 See:       ["org.example.test"]

`, false},

		{"instance", &testState, hst.Template(), false, false, `State
 Instance:    8e2c76b066dabe574cf073bdb46eb5c1 (3405691582 -> 3735928559)
 Uptime:      1h2m32s

App
 Identity:       9 (org.chromium.Chromium)
 Enablements:    wayland, dbus, pulseaudio
 Groups:         video, dialout, plugdev
 Home:           /data/data/org.chromium.Chromium
 Hostname:       localhost
 Flags:          multiarch, compat, devel, userns, net, abstract, tty, mapuid, device, runtime, tmpdir
 Path:           /run/current-system/sw/bin/chromium
 Arguments:      chromium --ignore-gpu-blocklist --disable-smooth-scrolling --enable-features=UseOzonePlatform --ozone-platform=wayland

Filesystem
 autoroot:w:/var/lib/hakurei/base/org.debian
 autoetc:/etc/
 w+ephemeral(-rwxr-xr-x):/tmp/
 w*/nix/store:/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper:/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work:/var/lib/hakurei/base/org.nixos/ro-store
 /run/current-system@
 /run/opengl-driver@
 w-/var/lib/hakurei/u0/org.chromium.Chromium:/data/data/org.chromium.Chromium
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

`, true},
		{"instance pd", &testState, new(hst.Config), false, false, `Error: configuration missing container state!

State
 Instance:    8e2c76b066dabe574cf073bdb46eb5c1 (3405691582 -> 3735928559)
 Uptime:      1h2m32s

App
 Identity:       0
 Enablements:    (no enablements)

`, false},

		{"json nil", nil, nil, false, true, `null
`, true},
		{"json instance", &testState, nil, false, true, `{
  "instance": "8e2c76b066dabe574cf073bdb46eb5c1",
  "pid": 3405691582,
  "shim_pid": 3735928559,
  "id": "org.chromium.Chromium",
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
    "env": {
      "GOOGLE_API_KEY": "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
      "GOOGLE_DEFAULT_CLIENT_ID": "77185425430.apps.googleusercontent.com",
      "GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT"
    },
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
          "/var/lib/hakurei/base/org.nixos/ro-store"
        ],
        "upper": "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper",
        "work": "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work"
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
        "write": true,
        "ensure": true
      },
      {
        "type": "bind",
        "src": "/dev/dri",
        "dev": true,
        "optional": true
      }
    ],
    "username": "chronos",
    "shell": "/run/current-system/sw/bin/zsh",
    "home": "/data/data/org.chromium.Chromium",
    "path": "/run/current-system/sw/bin/chromium",
    "args": [
      "chromium",
      "--ignore-gpu-blocklist",
      "--disable-smooth-scrolling",
      "--enable-features=UseOzonePlatform",
      "--ozone-platform=wayland"
    ],
    "seccomp_compat": true,
    "devel": true,
    "userns": true,
    "host_net": true,
    "host_abstract": true,
    "tty": true,
    "multiarch": true,
    "map_real_uid": true,
    "device": true,
    "share_runtime": true,
    "share_tmpdir": true
  },
  "time": "1970-01-01T00:00:00.000000009Z"
}
`, true},
		{"json config", nil, hst.Template(), false, true, `{
  "id": "org.chromium.Chromium",
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
    "env": {
      "GOOGLE_API_KEY": "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
      "GOOGLE_DEFAULT_CLIENT_ID": "77185425430.apps.googleusercontent.com",
      "GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT"
    },
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
          "/var/lib/hakurei/base/org.nixos/ro-store"
        ],
        "upper": "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper",
        "work": "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work"
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
        "write": true,
        "ensure": true
      },
      {
        "type": "bind",
        "src": "/dev/dri",
        "dev": true,
        "optional": true
      }
    ],
    "username": "chronos",
    "shell": "/run/current-system/sw/bin/zsh",
    "home": "/data/data/org.chromium.Chromium",
    "path": "/run/current-system/sw/bin/chromium",
    "args": [
      "chromium",
      "--ignore-gpu-blocklist",
      "--disable-smooth-scrolling",
      "--enable-features=UseOzonePlatform",
      "--ozone-platform=wayland"
    ],
    "seccomp_compat": true,
    "devel": true,
    "userns": true,
    "host_net": true,
    "host_abstract": true,
    "tty": true,
    "multiarch": true,
    "map_real_uid": true,
    "device": true,
    "share_runtime": true,
    "share_tmpdir": true
  }
}
`, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			output := new(strings.Builder)
			gotValid := printShowInstance(output, testTime, tc.instance, tc.config, tc.short, tc.json)
			if got := output.String(); got != tc.want {
				t.Errorf("printShowInstance: \n%s\nwant\n%s", got, tc.want)
				return
			}
			if gotValid != tc.valid {
				t.Errorf("printShowInstance: valid = %v, want %v", gotValid, tc.valid)
			}
		})
	}
}

func TestPrintPs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		data        []hst.State
		short, json bool
		want, log   string
	}{
		{"no entries", []hst.State{}, false, false, "    Instance    PID    Application    Uptime\n", ""},
		{"no entries short", []hst.State{}, true, false, "", ""},

		{"invalid config", []hst.State{{ID: testID, PID: 1 << 8, Config: new(hst.Config), Time: testAppTime}}, false, false, "    Instance    PID    Application    Uptime\n", "check: configuration missing container state\n"},

		{"valid", []hst.State{testStateSmall, testState}, false, false, `    Instance    PID           Application                  Uptime
    4cf073bd    3405691582    9 (org.chromium.Chromium)    1h2m32s
    aaaaaaaa    48879         1 (app.hakurei.aaaaaaaa)     1h2m28s
`, ""},
		{"valid single", []hst.State{testState}, false, false, `    Instance    PID           Application                  Uptime
    4cf073bd    3405691582    9 (org.chromium.Chromium)    1h2m32s
`, ""},

		{"valid short", []hst.State{testStateSmall, testState}, true, false, "4cf073bd\naaaaaaaa\n", ""},
		{"valid short single", []hst.State{testState}, true, false, "4cf073bd\n", ""},

		{"valid json", []hst.State{testState, testStateSmall}, false, true, `[
  {
    "instance": "8e2c76b066dabe574cf073bdb46eb5c1",
    "pid": 3405691582,
    "shim_pid": 3735928559,
    "id": "org.chromium.Chromium",
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
      "env": {
        "GOOGLE_API_KEY": "AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
        "GOOGLE_DEFAULT_CLIENT_ID": "77185425430.apps.googleusercontent.com",
        "GOOGLE_DEFAULT_CLIENT_SECRET": "OTJgUOQcT7lO7GsGZq2G4IlT"
      },
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
            "/var/lib/hakurei/base/org.nixos/ro-store"
          ],
          "upper": "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper",
          "work": "/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work"
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
          "write": true,
          "ensure": true
        },
        {
          "type": "bind",
          "src": "/dev/dri",
          "dev": true,
          "optional": true
        }
      ],
      "username": "chronos",
      "shell": "/run/current-system/sw/bin/zsh",
      "home": "/data/data/org.chromium.Chromium",
      "path": "/run/current-system/sw/bin/chromium",
      "args": [
        "chromium",
        "--ignore-gpu-blocklist",
        "--disable-smooth-scrolling",
        "--enable-features=UseOzonePlatform",
        "--ozone-platform=wayland"
      ],
      "seccomp_compat": true,
      "devel": true,
      "userns": true,
      "host_net": true,
      "host_abstract": true,
      "tty": true,
      "multiarch": true,
      "map_real_uid": true,
      "device": true,
      "share_runtime": true,
      "share_tmpdir": true
    },
    "time": "1970-01-01T00:00:00.000000009Z"
  },
  {
    "instance": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "pid": 48879,
    "shim_pid": 51966,
    "enablements": {
      "wayland": true,
      "pulse": true
    },
    "identity": 1,
    "groups": null,
    "container": {
      "env": null,
      "filesystem": null,
      "shell": "/bin/sh",
      "home": "/data/data/uk.gensokyo.cat",
      "path": "/usr/bin/cat",
      "args": [
        "cat"
      ],
      "userns": true,
      "map_real_uid": false
    },
    "time": "1970-01-01T00:00:03.735928559Z"
  }
]
`, ""},
		{"valid short json", []hst.State{testStateSmall, testState}, true, true, `["8e2c76b066dabe574cf073bdb46eb5c1","aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]
`, ""},
	}

	for _, tc := range testCases {
		s := store.New(check.MustAbs(t.TempDir()).Append("store"))
		for i := range tc.data {
			if h, err := s.Handle(tc.data[i].Identity); err != nil {
				t.Fatalf("Handle: error = %v", err)
			} else {
				var unlock func()
				if unlock, err = h.Lock(); err != nil {
					t.Fatalf("Lock: error = %v", err)
				}
				_, err = h.Save(&tc.data[i])
				unlock()
				if err != nil {
					t.Fatalf("Save: error = %v", err)
				}
			}
		}

		// store must not be written to beyond this point
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var printBuf, logBuf bytes.Buffer
			msg := message.New(log.New(&logBuf, "check: ", 0))
			msg.SwapVerbose(true)
			printPs(msg, &printBuf, testTime, s, tc.short, tc.json)
			if got := printBuf.String(); got != tc.want {
				t.Errorf("printPs:\n%s\nwant\n%s", got, tc.want)
				return
			}
			if got := logBuf.String(); got != tc.log {
				t.Errorf("msg:\n%s\nwant\n%s", got, tc.log)
			}
		})
	}
}
