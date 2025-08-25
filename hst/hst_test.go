package hst_test

import (
	"encoding/json"
	"testing"

	"hakurei.app/hst"
)

func TestTemplate(t *testing.T) {
	const want = `{
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
	"home": "/data/data/org.chromium.Chromium",
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
				"write": true,
				"ensure": true
			},
			{
				"type": "bind",
				"src": "/dev/dri",
				"dev": true,
				"optional": true
			}
		]
	}
}`

	if p, err := json.MarshalIndent(hst.Template(), "", "\t"); err != nil {
		t.Fatalf("cannot marshal: %v", err)
	} else if s := string(p); s != want {
		t.Fatalf("Template:\n%s\nwant:\n%s",
			s, want)
	}
}
