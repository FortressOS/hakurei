package fst_test

import (
	"encoding/json"
	"testing"

	"git.gensokyo.uk/security/fortify/fst"
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
}`

	if p, err := json.MarshalIndent(fst.Template(), "", "\t"); err != nil {
		t.Fatalf("cannot marshal: %v", err)
	} else if s := string(p); s != want {
		t.Fatalf("Template:\n%s\nwant:\n%s",
			s, want)
	}
}
