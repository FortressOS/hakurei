package app

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os/exec"
	"os/user"
	"reflect"
	"syscall"
	"testing"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/bits"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/message"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func TestApp(t *testing.T) {
	msg := message.NewMsg(nil)
	msg.SwapVerbose(testing.Verbose())

	testCases := []struct {
		name       string
		k          syscallDispatcher
		config     *hst.Config
		id         state.ID
		wantSys    *system.I
		wantParams *container.Params
	}{
		{
			"nixos permissive defaults no enablements", new(stubNixOS),
			&hst.Config{Container: &hst.ContainerConfig{
				Userns: true, HostNet: true, HostAbstract: true, Tty: true,

				Filesystem: []hst.FilesystemConfigJSON{
					{FilesystemConfig: &hst.FSBind{
						Target:  fhs.AbsRoot,
						Source:  fhs.AbsRoot,
						Write:   true,
						Special: true,
					}},
					{FilesystemConfig: &hst.FSBind{
						Source:   fhs.AbsDev.Append("kvm"),
						Device:   true,
						Optional: true,
					}},
					{FilesystemConfig: &hst.FSBind{
						Target:  fhs.AbsEtc,
						Source:  fhs.AbsEtc,
						Special: true,
					}},
				},

				Username: "chronos",
				Shell:    m("/run/current-system/sw/bin/zsh"),
				Home:     m("/home/chronos"),

				Path: m("/run/current-system/sw/bin/zsh"),
				Args: []string{"/run/current-system/sw/bin/zsh"},
			}},
			state.ID{
				0x4a, 0x45, 0x0b, 0x65,
				0x96, 0xd7, 0xbc, 0x15,
				0xbd, 0x01, 0x78, 0x0e,
				0xb9, 0xa6, 0x07, 0xac,
			},
			system.New(t.Context(), msg, 1000000).
				Ensure(m("/tmp/hakurei.0"), 0711).
				Ensure(m("/tmp/hakurei.0/runtime"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime"), acl.Execute).
				Ensure(m("/tmp/hakurei.0/runtime/0"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime/0"), acl.Read, acl.Write, acl.Execute).
				Ensure(m("/tmp/hakurei.0/tmpdir"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir"), acl.Execute).
				Ensure(m("/tmp/hakurei.0/tmpdir/0"), 01700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir/0"), acl.Read, acl.Write, acl.Execute),
			&container.Params{
				Dir:  m("/home/chronos"),
				Path: m("/run/current-system/sw/bin/zsh"),
				Args: []string{"/run/current-system/sw/bin/zsh"},
				Env: []string{
					"HOME=/home/chronos",
					"SHELL=/run/current-system/sw/bin/zsh",
					"TERM=xterm-256color",
					"USER=chronos",
					"XDG_RUNTIME_DIR=/run/user/65534",
					"XDG_SESSION_CLASS=user",
					"XDG_SESSION_TYPE=tty",
				},
				Ops: new(container.Ops).
					Root(m("/"), bits.BindWritable).
					Proc(m("/proc/")).
					Tmpfs(hst.AbsPrivateTmp, 4096, 0755).
					DevWritable(m("/dev/"), true).
					Tmpfs(m("/dev/shm"), 0, 01777).
					Bind(m("/dev/kvm"), m("/dev/kvm"), bits.BindWritable|bits.BindDevice|bits.BindOptional).
					Etc(m("/etc/"), "4a450b6596d7bc15bd01780eb9a607ac").
					Tmpfs(m("/run/user/1971"), 8192, 0755).
					Tmpfs(m("/run/nscd"), 8192, 0755).
					Tmpfs(m("/run/dbus"), 8192, 0755).
					Remount(m("/dev/"), syscall.MS_RDONLY).
					Tmpfs(m("/run/user/"), 4096, 0755).
					Bind(m("/tmp/hakurei.0/runtime/0"), m("/run/user/65534"), bits.BindWritable).
					Bind(m("/tmp/hakurei.0/tmpdir/0"), m("/tmp/"), bits.BindWritable).
					Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
					Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
					Remount(m("/"), syscall.MS_RDONLY),
				SeccompPresets: bits.PresetExt | bits.PresetDenyDevel,
				HostNet:        true,
				HostAbstract:   true,
				RetainSession:  true,
				ForwardCancel:  true,
			},
		},
		{
			"nixos permissive defaults chromium", new(stubNixOS),
			&hst.Config{
				ID:       "org.chromium.Chromium",
				Identity: 9,
				Groups:   []string{"video"},
				SessionBus: &hst.BusConfig{
					Talk: []string{
						"org.freedesktop.Notifications",
						"org.freedesktop.FileManager1",
						"org.freedesktop.ScreenSaver",
						"org.freedesktop.secrets",
						"org.kde.kwalletd5",
						"org.kde.kwalletd6",
						"org.gnome.SessionManager",
					},
					Own: []string{
						"org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.chromium.*",
					},
					Call: map[string]string{
						"org.freedesktop.portal.*": "*",
					},
					Broadcast: map[string]string{
						"org.freedesktop.portal.*": "@/org/freedesktop/portal/*",
					},
					Filter: true,
				},
				SystemBus: &hst.BusConfig{
					Talk: []string{
						"org.bluez",
						"org.freedesktop.Avahi",
						"org.freedesktop.UPower",
					},
					Filter: true,
				},
				Enablements: hst.NewEnablements(hst.EWayland | hst.EDBus | hst.EPulse),

				Container: &hst.ContainerConfig{
					Userns: true, HostNet: true, HostAbstract: true, Tty: true,

					Filesystem: []hst.FilesystemConfigJSON{
						{FilesystemConfig: &hst.FSBind{
							Target:  fhs.AbsRoot,
							Source:  fhs.AbsRoot,
							Write:   true,
							Special: true,
						}},
						{FilesystemConfig: &hst.FSBind{
							Source:   fhs.AbsDev.Append("dri"),
							Device:   true,
							Optional: true,
						}},
						{FilesystemConfig: &hst.FSBind{
							Source:   fhs.AbsDev.Append("kvm"),
							Device:   true,
							Optional: true,
						}},
						{FilesystemConfig: &hst.FSBind{
							Target:  fhs.AbsEtc,
							Source:  fhs.AbsEtc,
							Special: true,
						}},
					},

					Username: "chronos",
					Shell:    m("/run/current-system/sw/bin/zsh"),
					Home:     m("/home/chronos"),

					Path: m("/run/current-system/sw/bin/zsh"),
					Args: []string{"zsh", "-c", "exec chromium "},
				},
			},
			state.ID{
				0xeb, 0xf0, 0x83, 0xd1,
				0xb1, 0x75, 0x91, 0x17,
				0x82, 0xd4, 0x13, 0x36,
				0x9b, 0x64, 0xce, 0x7c,
			},
			system.New(t.Context(), msg, 1000009).
				Ensure(m("/tmp/hakurei.0"), 0711).
				Ensure(m("/tmp/hakurei.0/runtime"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime"), acl.Execute).
				Ensure(m("/tmp/hakurei.0/runtime/9"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime/9"), acl.Read, acl.Write, acl.Execute).
				Ensure(m("/tmp/hakurei.0/tmpdir"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir"), acl.Execute).
				Ensure(m("/tmp/hakurei.0/tmpdir/9"), 01700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir/9"), acl.Read, acl.Write, acl.Execute).
				Ephemeral(system.Process, m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c"), 0711).
				Wayland(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/1971/wayland-0"), "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
				Ensure(m("/run/user/1971/hakurei"), 0700).UpdatePermType(system.User, m("/run/user/1971/hakurei"), acl.Execute).
				Ensure(m("/run/user/1971"), 0700).UpdatePermType(system.User, m("/run/user/1971"), acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
				Ephemeral(system.Process, m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c"), 0700).UpdatePermType(system.Process, m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c"), acl.Execute).
				Link(m("/run/user/1971/pulse/native"), m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse")).
				MustProxyDBus(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus"), &hst.BusConfig{
					Talk: []string{
						"org.freedesktop.Notifications",
						"org.freedesktop.FileManager1",
						"org.freedesktop.ScreenSaver",
						"org.freedesktop.secrets",
						"org.kde.kwalletd5",
						"org.kde.kwalletd6",
						"org.gnome.SessionManager",
					},
					Own: []string{
						"org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.chromium.*",
					},
					Call: map[string]string{
						"org.freedesktop.portal.*": "*",
					},
					Broadcast: map[string]string{
						"org.freedesktop.portal.*": "@/org/freedesktop/portal/*",
					},
					Filter: true,
				}, m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), &hst.BusConfig{
					Talk: []string{
						"org.bluez",
						"org.freedesktop.Avahi",
						"org.freedesktop.UPower",
					},
					Filter: true,
				}).
				UpdatePerm(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus"), acl.Read, acl.Write).
				UpdatePerm(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), acl.Read, acl.Write),
			&container.Params{
				Dir:  m("/home/chronos"),
				Path: m("/run/current-system/sw/bin/zsh"),
				Args: []string{"zsh", "-c", "exec chromium "},
				Env: []string{
					"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/65534/bus",
					"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/var/run/dbus/system_bus_socket",
					"HOME=/home/chronos",
					"PULSE_COOKIE=" + hst.PrivateTmp + "/pulse-cookie",
					"PULSE_SERVER=unix:/run/user/65534/pulse/native",
					"SHELL=/run/current-system/sw/bin/zsh",
					"TERM=xterm-256color",
					"USER=chronos",
					"WAYLAND_DISPLAY=wayland-0",
					"XDG_RUNTIME_DIR=/run/user/65534",
					"XDG_SESSION_CLASS=user",
					"XDG_SESSION_TYPE=tty",
				},
				Ops: new(container.Ops).
					Root(m("/"), bits.BindWritable).
					Proc(m("/proc/")).
					Tmpfs(hst.AbsPrivateTmp, 4096, 0755).
					DevWritable(m("/dev/"), true).
					Tmpfs(m("/dev/shm"), 0, 01777).
					Bind(m("/dev/dri"), m("/dev/dri"), bits.BindWritable|bits.BindDevice|bits.BindOptional).
					Bind(m("/dev/kvm"), m("/dev/kvm"), bits.BindWritable|bits.BindDevice|bits.BindOptional).
					Etc(m("/etc/"), "ebf083d1b175911782d413369b64ce7c").
					Tmpfs(m("/run/user/1971"), 8192, 0755).
					Tmpfs(m("/run/nscd"), 8192, 0755).
					Tmpfs(m("/run/dbus"), 8192, 0755).
					Remount(m("/dev/"), syscall.MS_RDONLY).
					Tmpfs(m("/run/user/"), 4096, 0755).
					Bind(m("/tmp/hakurei.0/runtime/9"), m("/run/user/65534"), bits.BindWritable).
					Bind(m("/tmp/hakurei.0/tmpdir/9"), m("/tmp/"), bits.BindWritable).
					Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
					Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
					Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/65534/wayland-0"), 0).
					Bind(m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse"), m("/run/user/65534/pulse/native"), 0).
					Place(m(hst.PrivateTmp+"/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).
					Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus"), m("/run/user/65534/bus"), 0).
					Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), m("/var/run/dbus/system_bus_socket"), 0).
					Remount(m("/"), syscall.MS_RDONLY),
				SeccompPresets: bits.PresetExt | bits.PresetDenyDevel,
				HostNet:        true,
				HostAbstract:   true,
				RetainSession:  true,
				ForwardCancel:  true,
			},
		},

		{
			"nixos chromium direct wayland", new(stubNixOS),
			&hst.Config{
				ID:          "org.chromium.Chromium",
				Enablements: hst.NewEnablements(hst.EWayland | hst.EDBus | hst.EPulse),
				Container: &hst.ContainerConfig{
					Userns: true, HostNet: true, MapRealUID: true, Env: nil,
					Filesystem: []hst.FilesystemConfigJSON{
						f(&hst.FSBind{Source: m("/bin")}),
						f(&hst.FSBind{Source: m("/usr/bin/")}),
						f(&hst.FSBind{Source: m("/nix/store")}),
						f(&hst.FSBind{Source: m("/run/current-system")}),
						f(&hst.FSBind{Source: m("/sys/block"), Optional: true}),
						f(&hst.FSBind{Source: m("/sys/bus"), Optional: true}),
						f(&hst.FSBind{Source: m("/sys/class"), Optional: true}),
						f(&hst.FSBind{Source: m("/sys/dev"), Optional: true}),
						f(&hst.FSBind{Source: m("/sys/devices"), Optional: true}),
						f(&hst.FSBind{Source: m("/run/opengl-driver")}),
						f(&hst.FSBind{Source: m("/dev/dri"), Device: true, Optional: true}),
						f(&hst.FSBind{Source: m("/etc/"), Target: m("/etc/"), Special: true}),
						f(&hst.FSBind{Source: m("/var/lib/persist/module/hakurei/0/1"), Write: true, Ensure: true}),
					},

					Username: "u0_a1",
					Shell:    m("/run/current-system/sw/bin/zsh"),
					Home:     m("/var/lib/persist/module/hakurei/0/1"),

					Path: m("/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"),
				},
				SystemBus: &hst.BusConfig{
					Talk:   []string{"org.bluez", "org.freedesktop.Avahi", "org.freedesktop.UPower"},
					Filter: true,
				},
				SessionBus: &hst.BusConfig{
					Talk: []string{
						"org.freedesktop.FileManager1", "org.freedesktop.Notifications",
						"org.freedesktop.ScreenSaver", "org.freedesktop.secrets",
						"org.kde.kwalletd5", "org.kde.kwalletd6",
					},
					Own: []string{
						"org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.chromium.*",
					},
					Call: map[string]string{}, Broadcast: map[string]string{},
					Filter: true,
				},
				DirectWayland: true,

				Identity: 1, Groups: []string{},
			},
			state.ID{
				0x8e, 0x2c, 0x76, 0xb0,
				0x66, 0xda, 0xbe, 0x57,
				0x4c, 0xf0, 0x73, 0xbd,
				0xb4, 0x6e, 0xb5, 0xc1,
			},
			system.New(t.Context(), msg, 1000001).
				Ensure(m("/tmp/hakurei.0"), 0711).
				Ensure(m("/tmp/hakurei.0/runtime"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime"), acl.Execute).
				Ensure(m("/tmp/hakurei.0/runtime/1"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/runtime/1"), acl.Read, acl.Write, acl.Execute).
				Ensure(m("/tmp/hakurei.0/tmpdir"), 0700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir"), acl.Execute).
				Ensure(m("/tmp/hakurei.0/tmpdir/1"), 01700).UpdatePermType(system.User, m("/tmp/hakurei.0/tmpdir/1"), acl.Read, acl.Write, acl.Execute).
				Ensure(m("/run/user/1971/hakurei"), 0700).UpdatePermType(system.User, m("/run/user/1971/hakurei"), acl.Execute).
				Ensure(m("/run/user/1971"), 0700).UpdatePermType(system.User, m("/run/user/1971"), acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
				UpdatePermType(hst.EWayland, m("/run/user/1971/wayland-0"), acl.Read, acl.Write, acl.Execute).
				Ephemeral(system.Process, m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1"), 0700).UpdatePermType(system.Process, m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1"), acl.Execute).
				Link(m("/run/user/1971/pulse/native"), m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1/pulse")).
				Ephemeral(system.Process, m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1"), 0711).
				MustProxyDBus(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus"), &hst.BusConfig{
					Talk: []string{
						"org.freedesktop.FileManager1", "org.freedesktop.Notifications",
						"org.freedesktop.ScreenSaver", "org.freedesktop.secrets",
						"org.kde.kwalletd5", "org.kde.kwalletd6",
					},
					Own: []string{
						"org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.org.chromium.Chromium.*",
						"org.mpris.MediaPlayer2.chromium.*",
					},
					Call: map[string]string{}, Broadcast: map[string]string{},
					Filter: true,
				}, m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), &hst.BusConfig{
					Talk: []string{
						"org.bluez",
						"org.freedesktop.Avahi",
						"org.freedesktop.UPower",
					},
					Filter: true,
				}).
				UpdatePerm(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus"), acl.Read, acl.Write).
				UpdatePerm(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), acl.Read, acl.Write),
			&container.Params{
				Uid:  1971,
				Gid:  100,
				Dir:  m("/var/lib/persist/module/hakurei/0/1"),
				Path: m("/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"),
				Args: []string{"/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"},
				Env: []string{
					"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1971/bus",
					"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/var/run/dbus/system_bus_socket",
					"HOME=/var/lib/persist/module/hakurei/0/1",
					"PULSE_COOKIE=" + hst.PrivateTmp + "/pulse-cookie",
					"PULSE_SERVER=unix:/run/user/1971/pulse/native",
					"SHELL=/run/current-system/sw/bin/zsh",
					"TERM=xterm-256color",
					"USER=u0_a1",
					"WAYLAND_DISPLAY=wayland-0",
					"XDG_RUNTIME_DIR=/run/user/1971",
					"XDG_SESSION_CLASS=user",
					"XDG_SESSION_TYPE=tty",
				},
				Ops: new(container.Ops).
					Proc(m("/proc/")).
					Tmpfs(hst.AbsPrivateTmp, 4096, 0755).
					DevWritable(m("/dev/"), true).
					Tmpfs(m("/dev/shm"), 0, 01777).
					Bind(m("/bin"), m("/bin"), 0).
					Bind(m("/usr/bin/"), m("/usr/bin/"), 0).
					Bind(m("/nix/store"), m("/nix/store"), 0).
					Bind(m("/run/current-system"), m("/run/current-system"), 0).
					Bind(m("/sys/block"), m("/sys/block"), bits.BindOptional).
					Bind(m("/sys/bus"), m("/sys/bus"), bits.BindOptional).
					Bind(m("/sys/class"), m("/sys/class"), bits.BindOptional).
					Bind(m("/sys/dev"), m("/sys/dev"), bits.BindOptional).
					Bind(m("/sys/devices"), m("/sys/devices"), bits.BindOptional).
					Bind(m("/run/opengl-driver"), m("/run/opengl-driver"), 0).
					Bind(m("/dev/dri"), m("/dev/dri"), bits.BindDevice|bits.BindWritable|bits.BindOptional).
					Etc(m("/etc/"), "8e2c76b066dabe574cf073bdb46eb5c1").
					Bind(m("/var/lib/persist/module/hakurei/0/1"), m("/var/lib/persist/module/hakurei/0/1"), bits.BindWritable|bits.BindEnsure).
					Remount(m("/dev/"), syscall.MS_RDONLY).
					Tmpfs(m("/run/user/"), 4096, 0755).
					Bind(m("/tmp/hakurei.0/runtime/1"), m("/run/user/1971"), bits.BindWritable).
					Bind(m("/tmp/hakurei.0/tmpdir/1"), m("/tmp/"), bits.BindWritable).
					Place(m("/etc/passwd"), []byte("u0_a1:x:1971:100:Hakurei:/var/lib/persist/module/hakurei/0/1:/run/current-system/sw/bin/zsh\n")).
					Place(m("/etc/group"), []byte("hakurei:x:100:\n")).
					Bind(m("/run/user/1971/wayland-0"), m("/run/user/1971/wayland-0"), 0).
					Bind(m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1/pulse"), m("/run/user/1971/pulse/native"), 0).
					Place(m(hst.PrivateTmp+"/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).
					Bind(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus"), m("/run/user/1971/bus"), 0).
					Bind(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), m("/var/run/dbus/system_bus_socket"), 0).
					Remount(m("/"), syscall.MS_RDONLY),
				SeccompPresets: bits.PresetExt | bits.PresetDenyTTY | bits.PresetDenyDevel,
				HostNet:        true,
				ForwardCancel:  true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gr, gw := io.Pipe()

			var gotSys *system.I
			{
				sPriv := newOutcomeState(tc.k, msg, &tc.id, tc.config, &Hsu{k: tc.k})
				if err := sPriv.populateLocal(tc.k, msg); err != nil {
					t.Fatalf("populateLocal: error = %#v", err)
				}

				gotSys = system.New(t.Context(), msg, sPriv.uid.unwrap())
				if err := sPriv.newSys(tc.config, gotSys).toSystem(); err != nil {
					t.Fatalf("toSystem: error = %#v", err)
				}

				go func() {
					e := gob.NewEncoder(gw)
					if err := errors.Join(e.Encode(&sPriv)); err != nil {
						t.Errorf("Encode: error = %v", err)
						panic("unexpected encode fault")
					}
				}()
			}

			var gotParams *container.Params
			{
				var sShim outcomeState

				d := gob.NewDecoder(gr)
				if err := errors.Join(d.Decode(&sShim)); err != nil {
					t.Fatalf("Decode: error = %v", err)
				}
				if err := sShim.populateLocal(tc.k, msg); err != nil {
					t.Fatalf("populateLocal: error = %#v", err)
				}

				stateParams := sShim.newParams()
				for _, op := range sShim.Shim.Ops {
					if err := op.toContainer(stateParams); err != nil {
						t.Fatalf("toContainer: error = %#v", err)
					}
				}
				gotParams = stateParams.params
			}

			t.Run("sys", func(t *testing.T) {
				if !gotSys.Equal(tc.wantSys) {
					t.Errorf("toSystem: sys = %#v, want %#v", gotSys, tc.wantSys)
				}
			})

			t.Run("params", func(t *testing.T) {
				if !reflect.DeepEqual(gotParams, tc.wantParams) {
					t.Errorf("toContainer: params =\n%s\n, want\n%s", mustMarshal(gotParams), mustMarshal(tc.wantParams))
				}
			})
		})
	}
}

func mustMarshal(v any) string {
	if b, err := json.Marshal(v); err != nil {
		panic(err.Error())
	} else {
		return string(b)
	}
}

func stubDirEntries(names ...string) (e []fs.DirEntry, err error) {
	e = make([]fs.DirEntry, len(names))
	for i, name := range names {
		e[i] = stubDirEntryPath(name)
	}
	return
}

type stubDirEntryPath string

func (p stubDirEntryPath) Name() string               { return string(p) }
func (p stubDirEntryPath) IsDir() bool                { panic("attempted to call IsDir") }
func (p stubDirEntryPath) Type() fs.FileMode          { panic("attempted to call Type") }
func (p stubDirEntryPath) Info() (fs.FileInfo, error) { panic("attempted to call Info") }

type stubFileInfoMode fs.FileMode

func (s stubFileInfoMode) Name() string       { panic("attempted to call Name") }
func (s stubFileInfoMode) Size() int64        { panic("attempted to call Size") }
func (s stubFileInfoMode) Mode() fs.FileMode  { return fs.FileMode(s) }
func (s stubFileInfoMode) ModTime() time.Time { panic("attempted to call ModTime") }
func (s stubFileInfoMode) IsDir() bool        { panic("attempted to call IsDir") }
func (s stubFileInfoMode) Sys() any           { panic("attempted to call Sys") }

type stubFileInfoIsDir bool

func (s stubFileInfoIsDir) Name() string       { panic("attempted to call Name") }
func (s stubFileInfoIsDir) Size() int64        { panic("attempted to call Size") }
func (s stubFileInfoIsDir) Mode() fs.FileMode  { panic("attempted to call Mode") }
func (s stubFileInfoIsDir) ModTime() time.Time { panic("attempted to call ModTime") }
func (s stubFileInfoIsDir) IsDir() bool        { return bool(s) }
func (s stubFileInfoIsDir) Sys() any           { panic("attempted to call Sys") }

type stubFileInfoPulseCookie struct{ stubFileInfoIsDir }

func (s stubFileInfoPulseCookie) Size() int64 { return pulseCookieSizeMax }

type stubOsFileReadCloser struct{ io.ReadCloser }

func (s stubOsFileReadCloser) Name() string               { panic("attempting to call Name") }
func (s stubOsFileReadCloser) Write([]byte) (int, error)  { panic("attempting to call Write") }
func (s stubOsFileReadCloser) Stat() (fs.FileInfo, error) { panic("attempting to call Stat") }

type stubNixOS struct {
	usernameErr map[string]error
}

func (k *stubNixOS) new(func(k syscallDispatcher)) { panic("not implemented") }

func (k *stubNixOS) getpid() int { return 0xdeadbeef }
func (k *stubNixOS) getuid() int { return 1971 }
func (k *stubNixOS) getgid() int { return 100 }

func (k *stubNixOS) lookupEnv(key string) (string, bool) {
	switch key {
	case "SHELL":
		return "/run/current-system/sw/bin/zsh", true
	case "TERM":
		return "xterm-256color", true
	case "WAYLAND_DISPLAY":
		return "wayland-0", true
	case "PULSE_COOKIE":
		return "", false
	case "HOME":
		return "/home/ophestra", true
	case "XDG_RUNTIME_DIR":
		return "/run/user/1971", true
	case "XDG_CONFIG_HOME":
		return "/home/ophestra/xdg/config", true
	case "DBUS_SYSTEM_BUS_ADDRESS":
		return "", false
	default:
		panic(fmt.Sprintf("attempted to access unexpected environment variable %q", key))
	}
}

func (k *stubNixOS) stat(name string) (fs.FileInfo, error) {
	switch name {
	case "/var/run/nscd":
		return nil, nil
	case "/run/user/1971/pulse":
		return nil, nil
	case "/run/user/1971/pulse/native":
		return stubFileInfoMode(0666), nil
	case "/home/ophestra/.pulse-cookie":
		return stubFileInfoIsDir(true), nil
	case "/home/ophestra/xdg/config/pulse/cookie":
		return stubFileInfoPulseCookie{false}, nil
	default:
		panic(fmt.Sprintf("attempted to stat unexpected path %q", name))
	}
}

func (k *stubNixOS) open(name string) (osFile, error) {
	switch name {
	case "/home/ophestra/xdg/config/pulse/cookie":
		return stubOsFileReadCloser{io.NopCloser(bytes.NewReader(bytes.Repeat([]byte{0}, pulseCookieSizeMax)))}, nil
	default:
		panic(fmt.Sprintf("attempted to open unexpected path %q", name))
	}
}

func (k *stubNixOS) readdir(name string) ([]fs.DirEntry, error) {
	switch name {
	case "/":
		return stubDirEntries("bin", "boot", "dev", "etc", "home", "lib",
			"lib64", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var")

	case "/run":
		return stubDirEntries("agetty.reload", "binfmt", "booted-system",
			"credentials", "cryptsetup", "current-system", "dbus", "host", "keys",
			"libvirt", "libvirtd.pid", "lock", "log", "lvm", "mount", "NetworkManager",
			"nginx", "nixos", "nscd", "opengl-driver", "pppd", "resolvconf", "sddm",
			"store", "syncoid", "system", "systemd", "tmpfiles.d", "udev", "udisks2",
			"user", "utmp", "virtlogd.pid", "wrappers", "zed.pid", "zed.state")

	case "/etc":
		return stubDirEntries("alsa", "bashrc", "binfmt.d", "dbus-1", "default",
			"ethertypes", "fonts", "fstab", "fuse.conf", "group", "host.conf", "hostid",
			"hostname", "hostname.CHECKSUM", "hosts", "inputrc", "ipsec.d", "issue", "kbd",
			"libblockdev", "locale.conf", "localtime", "login.defs", "lsb-release", "lvm",
			"machine-id", "man_db.conf", "modprobe.d", "modules-load.d", "mtab", "nanorc",
			"netgroup", "NetworkManager", "nix", "nixos", "NIXOS", "nscd.conf", "nsswitch.conf",
			"opensnitchd", "os-release", "pam", "pam.d", "passwd", "pipewire", "pki", "polkit-1",
			"profile", "protocols", "qemu", "resolv.conf", "resolvconf.conf", "rpc", "samba",
			"sddm.conf", "secureboot", "services", "set-environment", "shadow", "shells", "ssh",
			"ssl", "static", "subgid", "subuid", "sudoers", "sysctl.d", "systemd", "terminfo",
			"tmpfiles.d", "udev", "udisks2", "UPower", "vconsole.conf", "X11", "zfs", "zinputrc",
			"zoneinfo", "zprofile", "zshenv", "zshrc")

	default:
		panic(fmt.Sprintf("attempted to read unexpected directory %q", name))
	}
}

func (k *stubNixOS) tempdir() string { return "/tmp/" }

func (k *stubNixOS) evalSymlinks(path string) (string, error) {
	switch path {
	case "/var/run/nscd":
		return "/run/nscd", nil
	case "/run/user/1971":
		return "/run/user/1971", nil
	case "/tmp/hakurei.0":
		return "/tmp/hakurei.0", nil
	case "/var/run/dbus":
		return "/run/dbus", nil
	case "/dev/kvm":
		return "/dev/kvm", nil
	case "/etc/":
		return "/etc/", nil
	case "/bin":
		return "/bin", nil
	case "/boot":
		return "/boot", nil
	case "/home":
		return "/home", nil
	case "/lib":
		return "/lib", nil
	case "/lib64":
		return "/lib64", nil
	case "/nix":
		return "/nix", nil
	case "/root":
		return "/root", nil
	case "/run":
		return "/run", nil
	case "/srv":
		return "/srv", nil
	case "/sys":
		return "/sys", nil
	case "/usr":
		return "/usr", nil
	case "/var":
		return "/var", nil
	case "/dev/dri":
		return "/dev/dri", nil
	case "/usr/bin/":
		return "/usr/bin/", nil
	case "/nix/store":
		return "/nix/store", nil
	case "/run/current-system":
		return "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-nixos-system-satori-25.05.99999999.aaaaaaa", nil
	case "/sys/block":
		return "/sys/block", nil
	case "/sys/bus":
		return "/sys/bus", nil
	case "/sys/class":
		return "/sys/class", nil
	case "/sys/dev":
		return "/sys/dev", nil
	case "/sys/devices":
		return "/sys/devices", nil
	case "/run/opengl-driver":
		return "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-graphics-drivers", nil
	case "/var/lib/persist/module/hakurei/0/1":
		return "/var/lib/persist/module/hakurei/0/1", nil
	default:
		panic(fmt.Sprintf("attempted to evaluate unexpected path %q", path))
	}
}

func (k *stubNixOS) lookupGroupId(name string) (string, error) {
	switch name {
	case "video":
		return "26", nil
	default:
		return "", user.UnknownGroupError(name)
	}
}

func (k *stubNixOS) cmdOutput(cmd *exec.Cmd) ([]byte, error) {
	switch cmd.Path {
	case "/proc/nonexistent/hsu":
		return []byte{'0'}, nil
	default:
		panic(fmt.Sprintf("unexpected cmd %#v", cmd))
	}
}

func (k *stubNixOS) overflowUid(message.Msg) int { return 65534 }
func (k *stubNixOS) overflowGid(message.Msg) int { return 65534 }

func (k *stubNixOS) mustHsuPath() *check.Absolute { return m("/proc/nonexistent/hsu") }

func (k *stubNixOS) fatalf(format string, v ...any) { panic(fmt.Sprintf(format, v...)) }

func (k *stubNixOS) isVerbose() bool                  { return true }
func (k *stubNixOS) verbose(v ...any)                 { log.Print(v...) }
func (k *stubNixOS) verbosef(format string, v ...any) { log.Printf(format, v...) }

func m(pathname string) *check.Absolute {
	return check.MustAbs(pathname)
}

func f(c hst.FilesystemConfig) hst.FilesystemConfigJSON {
	return hst.FilesystemConfigJSON{FilesystemConfig: c}
}
