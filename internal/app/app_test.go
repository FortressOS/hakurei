package app

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/system"
	"hakurei.app/system/acl"
	"hakurei.app/system/dbus"
)

func TestApp(t *testing.T) {
	msg := container.NewMsg(nil)
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
			&hst.Config{Username: "chronos", Home: m("/home/chronos")},
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
					Root(m("/"), container.BindWritable).
					Proc(m("/proc/")).
					Tmpfs(hst.AbsTmp, 4096, 0755).
					DevWritable(m("/dev/"), true).
					Tmpfs(m("/dev/shm"), 0, 01777).
					Bind(m("/dev/kvm"), m("/dev/kvm"), container.BindWritable|container.BindDevice|container.BindOptional).
					Readonly(m("/var/run/nscd"), 0755).
					Etc(m("/etc/"), "4a450b6596d7bc15bd01780eb9a607ac").
					Tmpfs(m("/run/user/1971"), 8192, 0755).
					Tmpfs(m("/run/dbus"), 8192, 0755).
					Remount(m("/dev/"), syscall.MS_RDONLY).
					Tmpfs(m("/run/user/"), 4096, 0755).
					Bind(m("/tmp/hakurei.0/runtime/0"), m("/run/user/65534"), container.BindWritable).
					Bind(m("/tmp/hakurei.0/tmpdir/0"), m("/tmp/"), container.BindWritable).
					Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
					Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
					Remount(m("/"), syscall.MS_RDONLY),
				SeccompPresets: seccomp.PresetExt | seccomp.PresetDenyDevel,
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
				Args:     []string{"zsh", "-c", "exec chromium "},
				Identity: 9,
				Groups:   []string{"video"},
				Username: "chronos",
				Home:     m("/home/chronos"),
				SessionBus: &dbus.Config{
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
				SystemBus: &dbus.Config{
					Talk: []string{
						"org.bluez",
						"org.freedesktop.Avahi",
						"org.freedesktop.UPower",
					},
					Filter: true,
				},
				Enablements: hst.NewEnablements(hst.EWayland | hst.EDBus | hst.EPulse),
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
				Wayland(new(*os.File), m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/1971/wayland-0"), "org.chromium.Chromium", "ebf083d1b175911782d413369b64ce7c").
				Ensure(m("/run/user/1971/hakurei"), 0700).UpdatePermType(system.User, m("/run/user/1971/hakurei"), acl.Execute).
				Ensure(m("/run/user/1971"), 0700).UpdatePermType(system.User, m("/run/user/1971"), acl.Execute). // this is ordered as is because the previous Ensure only calls mkdir if XDG_RUNTIME_DIR is unset
				Ephemeral(system.Process, m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c"), 0700).UpdatePermType(system.Process, m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c"), acl.Execute).
				Link(m("/run/user/1971/pulse/native"), m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse")).
				MustProxyDBus(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus"), &dbus.Config{
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
				}, m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), &dbus.Config{
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
					"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket",
					"HOME=/home/chronos",
					"PULSE_COOKIE=" + hst.Tmp + "/pulse-cookie",
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
					Root(m("/"), container.BindWritable).
					Proc(m("/proc/")).
					Tmpfs(hst.AbsTmp, 4096, 0755).
					DevWritable(m("/dev/"), true).
					Tmpfs(m("/dev/shm"), 0, 01777).
					Bind(m("/dev/dri"), m("/dev/dri"), container.BindWritable|container.BindDevice|container.BindOptional).
					Bind(m("/dev/kvm"), m("/dev/kvm"), container.BindWritable|container.BindDevice|container.BindOptional).
					Readonly(m("/var/run/nscd"), 0755).
					Etc(m("/etc/"), "ebf083d1b175911782d413369b64ce7c").
					Tmpfs(m("/run/user/1971"), 8192, 0755).
					Tmpfs(m("/run/dbus"), 8192, 0755).
					Remount(m("/dev/"), syscall.MS_RDONLY).
					Tmpfs(m("/run/user/"), 4096, 0755).
					Bind(m("/tmp/hakurei.0/runtime/9"), m("/run/user/65534"), container.BindWritable).
					Bind(m("/tmp/hakurei.0/tmpdir/9"), m("/tmp/"), container.BindWritable).
					Place(m("/etc/passwd"), []byte("chronos:x:65534:65534:Hakurei:/home/chronos:/run/current-system/sw/bin/zsh\n")).
					Place(m("/etc/group"), []byte("hakurei:x:65534:\n")).
					Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/wayland"), m("/run/user/65534/wayland-0"), 0).
					Bind(m("/run/user/1971/hakurei/ebf083d1b175911782d413369b64ce7c/pulse"), m("/run/user/65534/pulse/native"), 0).
					Place(m(hst.Tmp+"/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).
					Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/bus"), m("/run/user/65534/bus"), 0).
					Bind(m("/tmp/hakurei.0/ebf083d1b175911782d413369b64ce7c/system_bus_socket"), m("/run/dbus/system_bus_socket"), 0).
					Remount(m("/"), syscall.MS_RDONLY),
				SeccompPresets: seccomp.PresetExt | seccomp.PresetDenyDevel,
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
				Path:        m("/nix/store/yqivzpzzn7z5x0lq9hmbzygh45d8rhqd-chromium-start"),
				Enablements: hst.NewEnablements(hst.EWayland | hst.EDBus | hst.EPulse),
				Shell:       m("/run/current-system/sw/bin/zsh"),

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
				},
				SystemBus: &dbus.Config{
					Talk:   []string{"org.bluez", "org.freedesktop.Avahi", "org.freedesktop.UPower"},
					Filter: true,
				},
				SessionBus: &dbus.Config{
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

				Username: "u0_a1",
				Home:     m("/var/lib/persist/module/hakurei/0/1"),
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
				MustProxyDBus(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus"), &dbus.Config{
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
				}, m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), &dbus.Config{
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
					"DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket",
					"HOME=/var/lib/persist/module/hakurei/0/1",
					"PULSE_COOKIE=" + hst.Tmp + "/pulse-cookie",
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
					Tmpfs(hst.AbsTmp, 4096, 0755).
					DevWritable(m("/dev/"), true).
					Tmpfs(m("/dev/shm"), 0, 01777).
					Bind(m("/bin"), m("/bin"), 0).
					Bind(m("/usr/bin/"), m("/usr/bin/"), 0).
					Bind(m("/nix/store"), m("/nix/store"), 0).
					Bind(m("/run/current-system"), m("/run/current-system"), 0).
					Bind(m("/sys/block"), m("/sys/block"), container.BindOptional).
					Bind(m("/sys/bus"), m("/sys/bus"), container.BindOptional).
					Bind(m("/sys/class"), m("/sys/class"), container.BindOptional).
					Bind(m("/sys/dev"), m("/sys/dev"), container.BindOptional).
					Bind(m("/sys/devices"), m("/sys/devices"), container.BindOptional).
					Bind(m("/run/opengl-driver"), m("/run/opengl-driver"), 0).
					Bind(m("/dev/dri"), m("/dev/dri"), container.BindDevice|container.BindWritable|container.BindOptional).
					Etc(m("/etc/"), "8e2c76b066dabe574cf073bdb46eb5c1").
					Bind(m("/var/lib/persist/module/hakurei/0/1"), m("/var/lib/persist/module/hakurei/0/1"), container.BindWritable|container.BindEnsure).
					Remount(m("/dev/"), syscall.MS_RDONLY).
					Tmpfs(m("/run/user/"), 4096, 0755).
					Bind(m("/tmp/hakurei.0/runtime/1"), m("/run/user/1971"), container.BindWritable).
					Bind(m("/tmp/hakurei.0/tmpdir/1"), m("/tmp/"), container.BindWritable).
					Place(m("/etc/passwd"), []byte("u0_a1:x:1971:100:Hakurei:/var/lib/persist/module/hakurei/0/1:/run/current-system/sw/bin/zsh\n")).
					Place(m("/etc/group"), []byte("hakurei:x:100:\n")).
					Bind(m("/run/user/1971/wayland-0"), m("/run/user/1971/wayland-0"), 0).
					Bind(m("/run/user/1971/hakurei/8e2c76b066dabe574cf073bdb46eb5c1/pulse"), m("/run/user/1971/pulse/native"), 0).
					Place(m(hst.Tmp+"/pulse-cookie"), bytes.Repeat([]byte{0}, pulseCookieSizeMax)).
					Bind(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/bus"), m("/run/user/1971/bus"), 0).
					Bind(m("/tmp/hakurei.0/8e2c76b066dabe574cf073bdb46eb5c1/system_bus_socket"), m("/run/dbus/system_bus_socket"), 0).
					Remount(m("/"), syscall.MS_RDONLY),
				SeccompPresets: seccomp.PresetExt | seccomp.PresetDenyTTY | seccomp.PresetDenyDevel,
				HostNet:        true,
				ForwardCancel:  true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("finalise", func(t *testing.T) {
				seal := outcome{syscallDispatcher: tc.k}
				err := seal.finalise(t.Context(), msg, &tc.id, tc.config)
				if err != nil {
					if s, ok := container.GetErrorMessage(err); !ok {
						t.Fatalf("Seal: error = %v", err)
					} else {
						t.Fatalf("Seal: %s", s)
					}
				}

				t.Run("sys", func(t *testing.T) {
					if !seal.sys.Equal(tc.wantSys) {
						t.Errorf("Seal: sys = %#v, want %#v", seal.sys, tc.wantSys)
					}
				})

				t.Run("params", func(t *testing.T) {
					if !reflect.DeepEqual(&seal.container, tc.wantParams) {
						t.Errorf("seal: container =\n%s\n, want\n%s", mustMarshal(&seal.container), mustMarshal(tc.wantParams))
					}
				})
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

func m(pathname string) *container.Absolute {
	return container.MustAbs(pathname)
}

func f(c hst.FilesystemConfig) hst.FilesystemConfigJSON {
	return hst.FilesystemConfigJSON{FilesystemConfig: c}
}
