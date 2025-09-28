package hst_test

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
)

func TestAppError(t *testing.T) {
	testCases := []struct {
		name    string
		err     error
		s       string
		message string
		is, isF error
	}{
		{"message", &hst.AppError{Step: "obtain uid from hsu", Err: stub.UniqueError(0),
			Msg: "the setuid helper is missing: /run/wrappers/bin/hsu"},
			"unique error 0 injected by the test suite",
			"the setuid helper is missing: /run/wrappers/bin/hsu",
			stub.UniqueError(0), os.ErrNotExist},

		{"os.PathError", &hst.AppError{Step: "passthrough os.PathError",
			Err: &os.PathError{Op: "stat", Path: "/proc/nonexistent", Err: os.ErrNotExist}},
			"stat /proc/nonexistent: file does not exist",
			"cannot stat /proc/nonexistent: file does not exist",
			os.ErrNotExist, stub.UniqueError(0xdeadbeef)},

		{"os.LinkError", &hst.AppError{Step: "passthrough os.LinkError",
			Err: &os.LinkError{Op: "link", Old: "/proc/self", New: "/proc/nonexistent", Err: os.ErrNotExist}},
			"link /proc/self /proc/nonexistent: file does not exist",
			"cannot link /proc/self /proc/nonexistent: file does not exist",
			os.ErrNotExist, stub.UniqueError(0xdeadbeef)},

		{"os.SyscallError", &hst.AppError{Step: "passthrough os.SyscallError",
			Err: &os.SyscallError{Syscall: "meow", Err: syscall.ENOSYS}},
			"meow: function not implemented",
			"cannot meow: function not implemented",
			syscall.ENOSYS, syscall.ENOTRECOVERABLE},

		{"net.OpError", &hst.AppError{Step: "passthrough net.OpError",
			Err: &net.OpError{Op: "dial", Net: "cat", Err: net.UnknownNetworkError("cat")}},
			"dial cat: unknown network cat",
			"cannot dial cat: unknown network cat",
			net.UnknownNetworkError("cat"), syscall.ENOTRECOVERABLE},

		{"default", &hst.AppError{Step: "initialise container configuration", Err: stub.UniqueError(1)},
			"unique error 1 injected by the test suite",
			"cannot initialise container configuration: unique error 1 injected by the test suite",
			stub.UniqueError(1), os.ErrInvalid},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("error", func(t *testing.T) {
				if got := tc.err.Error(); got != tc.s {
					t.Errorf("Error: %s, want %s", got, tc.s)
				}
			})

			t.Run("message", func(t *testing.T) {
				gotMessage, gotMessageOk := container.GetErrorMessage(tc.err)
				if want := tc.message != "\x00"; gotMessageOk != want {
					t.Errorf("GetErrorMessage: ok = %v, want %v", gotMessage, want)
				}

				if gotMessageOk {
					if gotMessage != tc.message {
						t.Errorf("GetErrorMessage: %s, want %s", gotMessage, tc.message)
					}
				}
			})

			t.Run("is", func(t *testing.T) {
				if !errors.Is(tc.err, tc.is) {
					t.Errorf("Is: unexpected false for %v", tc.is)
				}
				if errors.Is(tc.err, tc.isF) {
					t.Errorf("Is: unexpected true for %v", tc.isF)
				}
			})
		})
	}
}

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
