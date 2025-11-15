package outcome

import (
	"errors"
	"os"
	"reflect"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/std"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/dbus"
	"hakurei.app/internal/system"
)

func TestSpParamsOp(t *testing.T) {
	t.Parallel()
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"invalid program path", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spParamsOp)
			}
			return &spParamsOp{Term: "xterm", TermSet: true}
		}, func() *hst.Config {
			c := hst.Template()
			c.Container.Path = nil
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"TERM"}, "xterm", nil),
		}, newI().
			Ensure(m(container.Nonexistent+"/tmp/hakurei.0"), 0711), nil, nil, nil, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  "invalid program path",
		}},

		{"success defaultargs secure", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spParamsOp)
			}
			return &spParamsOp{Term: "xterm", TermSet: true}
		}, func() *hst.Config {
			c := hst.Template()
			c.Container.Args = nil
			c.Container.Flags = hst.FHostNet | hst.FHostAbstract | hst.FMapRealUID
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"TERM"}, "xterm", nil),
		}, newI().
			Ensure(m(container.Nonexistent+"/tmp/hakurei.0"), 0711), nil, nil, nil, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Hostname:       config.Container.Hostname,
			HostNet:        true,
			HostAbstract:   true,
			Path:           config.Container.Path,
			Args:           []string{config.Container.Path.String()},
			SeccompPresets: std.PresetExt | std.PresetDenyDevel | std.PresetDenyNS | std.PresetDenyTTY,
			Uid:            1000,
			Gid:            100,
			Ops: new(container.Ops).
				Root(m("/var/lib/hakurei/base/org.debian"), std.BindWritable).
				Proc(fhs.AbsProc).Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755).
				DevWritable(fhs.AbsDev, true).
				Tmpfs(fhs.AbsDevShm, 0, 01777),
		}, paramsWantEnv(config, map[string]string{
			"TERM": "xterm",
		}, func(t *testing.T, state *outcomeStateParams) {
			if state.as.AutoEtcPrefix != wantAutoEtcPrefix {
				t.Errorf("toContainer: as.AutoEtcPrefix = %q, want %q", state.as.AutoEtcPrefix, wantAutoEtcPrefix)
			}

			wantFilesystems := config.Container.Filesystem[1:]
			if !reflect.DeepEqual(state.filesystem, wantFilesystems) {
				t.Errorf("toContainer: filesystem = %#v, want %#v", state.filesystem, wantFilesystems)
			}
		}), nil},

		{"success", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spParamsOp)
			}
			return &spParamsOp{Term: "xterm", TermSet: true}
		}, hst.Template, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{"TERM"}, "xterm", nil),
		}, newI().
			Ensure(m(container.Nonexistent+"/tmp/hakurei.0"), 0711), nil, nil, nil, []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Hostname:      config.Container.Hostname,
			RetainSession: true,
			HostNet:       true,
			HostAbstract:  true,
			Path:          config.Container.Path,
			Args:          config.Container.Args,
			SeccompFlags:  seccomp.AllowMultiarch,
			Uid:           1000,
			Gid:           100,
			Ops: new(container.Ops).
				Root(m("/var/lib/hakurei/base/org.debian"), std.BindWritable).
				Proc(fhs.AbsProc).Tmpfs(hst.AbsPrivateTmp, 1<<12, 0755).
				Bind(fhs.AbsDev, fhs.AbsDev, std.BindWritable|std.BindDevice).
				Tmpfs(fhs.AbsDevShm, 0, 01777),
		}, paramsWantEnv(config, map[string]string{
			"TERM": "xterm",
		}, func(t *testing.T, state *outcomeStateParams) {
			if state.as.AutoEtcPrefix != wantAutoEtcPrefix {
				t.Errorf("toContainer: as.AutoEtcPrefix = %q, want %q", state.as.AutoEtcPrefix, wantAutoEtcPrefix)
			}

			wantFilesystems := config.Container.Filesystem[1:]
			if !reflect.DeepEqual(state.filesystem, wantFilesystems) {
				t.Errorf("toContainer: filesystem = %#v, want %#v", state.filesystem, wantFilesystems)
			}
		}), nil},
	})
}

func TestSpFilesystemOp(t *testing.T) {
	const nePrefix = container.Nonexistent + "/eval"
	var stubDebianRoot = stubDir("bin", "dev", "etc", "home", "lib64", "lost+found",
		"mnt", "nix", "proc", "root", "run", "srv", "sys", "tmp", "usr", "var")
	config := hst.Template()

	newConfigSmall := func() *hst.Config {
		c := hst.Template()
		c.Container.Filesystem = []hst.FilesystemConfigJSON{
			{FilesystemConfig: &hst.FSBind{Target: fhs.AbsEtc, Source: fhs.AbsEtc, Special: true}},
			{FilesystemConfig: &hst.FSOverlay{Target: m("/nix/store"), Lower: []*check.Absolute{
				fhs.AbsVarLib.Append("hakurei/base/org.nixos/.ro-store"),
				fhs.AbsVarLib.Append("hakurei/base/org.nixos/org.chromium.Chromium"),
			}}},
			{FilesystemConfig: &hst.FSEphemeral{Target: hst.AbsPrivateTmp}},
		}
		c.Container.Flags &= ^hst.FDevice
		return c
	}
	configSmall := newConfigSmall()

	needsApplyState := func(next pStateContainerFunc) pStateContainerFunc {
		return func(state *outcomeStateParams) {
			state.as = hst.ApplyState{AutoEtcPrefix: wantAutoEtcPrefix, Ops: opsAdapter{state.params.Ops}}

			if next != nil {
				next(state)
			}
		}
	}

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"readdir", func(bool, bool) outcomeOp {
			return new(spFilesystemOp)
		}, hst.Template, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/dbus"}, nePrefix+"/run/dbus", nil),
			call("readdir", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian"}, []os.DirEntry{}, stub.UniqueError(2)),
		}, nil, nil, &hst.AppError{
			Step: "access autoroot source",
			Err:  stub.UniqueError(2),
		}, nil, nil, nil, nil, nil},

		{"invalid dbus address", func(bool, bool) outcomeOp { return new(spFilesystemOp) }, func() *hst.Config {
			c := newConfigSmall()
			c.Container.Filesystem = append(c.Container.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: invalidFSHost(false)})
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid", nil),
		}, nil, nil, &hst.AppError{
			Step: "parse dbus address",
			Err: &dbus.BadAddressError{
				Type:     dbus.ErrNoColon,
				EntryVal: []byte("invalid"),
				PairPos:  -1,
			},
		}, nil, nil, nil, nil, nil},

		{"invalid fs early", func(bool, bool) outcomeOp { return new(spFilesystemOp) }, func() *hst.Config {
			c := newConfigSmall()
			c.Container.Filesystem = append(c.Container.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: invalidFSHost(false)})
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/"}, nePrefix+"/etc/dbus", nil), // to match hidePaths
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  "invalid filesystem at index 3",
		}, nil, nil, nil, nil, nil},

		{"evalSymlinks early", func(bool, bool) outcomeOp { return new(spFilesystemOp) }, newConfigSmall, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, "", stub.UniqueError(0)),
		}, nil, nil, &hst.AppError{
			Step: "evaluate path hiding target",
			Err:  stub.UniqueError(0),
		}, nil, nil, nil, nil, nil},

		{"host nil abs", func(bool, bool) outcomeOp { return new(spFilesystemOp) }, func() *hst.Config {
			c := newConfigSmall()
			c.Container.Filesystem = append(c.Container.Filesystem, hst.FilesystemConfigJSON{FilesystemConfig: invalidFSHost(true)})
			return c
		}, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/"}, nePrefix+"/etc/dbus", nil), // to match hidePaths
			call("evalSymlinks", stub.ExpectArgs{"/etc/"}, nePrefix+"/etc", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/.ro-store"}, nePrefix+"/var/lib/hakurei/base/org.nixos/.ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/org.chromium.Chromium"}, "var/lib/hakurei/base/org.nixos/org.chromium.Chromium", nil),
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  "impossible path hiding state reached",
		}, nil, nil, nil, nil, nil},

		{"evalSymlinks late", func(bool, bool) outcomeOp { return new(spFilesystemOp) }, newConfigSmall, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/"}, nePrefix+"/etc/dbus", nil), // to match hidePaths
			call("evalSymlinks", stub.ExpectArgs{"/etc/"}, nePrefix+"/etc", stub.UniqueError(1)),
		}, nil, nil, &hst.AppError{
			Step: "evaluate path hiding source",
			Err:  stub.UniqueError(1),
		}, nil, nil, nil, nil, nil},

		{"invalid contains", func(bool, bool) outcomeOp { return new(spFilesystemOp) }, newConfigSmall, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/"}, nePrefix+"/etc/dbus", nil), // to match hidePaths
			call("evalSymlinks", stub.ExpectArgs{"/etc/"}, nePrefix+"/etc", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/.ro-store"}, nePrefix+"/var/lib/hakurei/base/org.nixos/.ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/org.chromium.Chromium"}, "var/lib/hakurei/base/org.nixos/org.chromium.Chromium", nil),
			call("verbosef", stub.ExpectArgs{"hiding path %q from %q", []any{"/proc/nonexistent/eval/etc/dbus", "/etc/"}}, nil, nil),
		}, nil, nil, &hst.AppError{
			Step: "determine path hiding outcome",
			Err:  errors.New("Rel: can't make /proc/nonexistent/eval/xdg_runtime_dir relative to var/lib/hakurei/base/org.nixos/org.chromium.Chromium"),
		}, nil, nil, nil, nil, nil},

		{"invalid hide", func(bool, bool) outcomeOp { return new(spFilesystemOp) }, newConfigSmall, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, "xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, "tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "nscd", nil),
			call("evalSymlinks", stub.ExpectArgs{"/"}, "nonexistent/dbus", nil),
			call("evalSymlinks", stub.ExpectArgs{"/etc/"}, "nonexistent", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/.ro-store"}, ".ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/org.chromium.Chromium"}, "org.chromium.Chromium", nil),
			call("verbosef", stub.ExpectArgs{"hiding path %q from %q", []any{"nonexistent/dbus", "/etc/"}}, nil, nil),
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  `invalid path hiding candidate "nonexistent/dbus"`,
		}, nil, nil, nil, nil, nil},

		{"invalid fs", func(isShim, clearUnexported bool) outcomeOp {
			if !isShim {
				return new(spFilesystemOp)
			}
			return &spFilesystemOp{HidePaths: []*check.Absolute{m("/proc/nonexistent/eval/etc/dbus")}}
		}, newConfigSmall, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/"}, nePrefix+"/etc/dbus", nil), // to match hidePaths
			call("evalSymlinks", stub.ExpectArgs{"/etc/"}, nePrefix+"/etc", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/.ro-store"}, nePrefix+"/var/lib/hakurei/base/org.nixos/.ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/org.chromium.Chromium"}, nePrefix+"/var/lib/hakurei/base/org.nixos/org.chromium.Chromium", nil),
			call("verbosef", stub.ExpectArgs{"hiding path %q from %q", []any{"/proc/nonexistent/eval/etc/dbus", "/etc/"}}, nil, nil),
		}, newI().
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"),
				acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"),
				acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(needsApplyState(func(state *outcomeStateParams) {
			state.filesystem = append(configSmall.Container.Filesystem, hst.FilesystemConfigJSON{})
		})), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, nil, nil, &hst.AppError{
			Step: "finalise",
			Err:  os.ErrInvalid,
			Msg:  "invalid filesystem at index 3",
		}},

		{"success noroot nodev envdbus strangedbus dbusnotabs hide", func(isShim, clearUnexported bool) outcomeOp {
			if !isShim {
				return new(spFilesystemOp)
			}
			return &spFilesystemOp{HidePaths: []*check.Absolute{m("/proc/nonexistent/eval/etc/dbus")}}
		}, newConfigSmall, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, "invalid:meow=0;unix:path=/system_bus_socket;unix:path=system_bus_socket", nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is in an unusual location", []any{"/system_bus_socket"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"dbus socket %q is not absolute", []any{"system_bus_socket"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/"}, nePrefix+"/etc/dbus", nil), // to match hidePaths
			call("evalSymlinks", stub.ExpectArgs{"/etc/"}, nePrefix+"/etc", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/.ro-store"}, nePrefix+"/var/lib/hakurei/base/org.nixos/.ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/org.chromium.Chromium"}, nePrefix+"/var/lib/hakurei/base/org.nixos/org.chromium.Chromium", nil),
			call("verbosef", stub.ExpectArgs{"hiding path %q from %q", []any{"/proc/nonexistent/eval/etc/dbus", "/etc/"}}, nil, nil),
		}, newI().
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"),
				acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"),
				acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(needsApplyState(func(state *outcomeStateParams) {
			state.filesystem = configSmall.Container.Filesystem
		})), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Env: []string{
				"GOOGLE_API_KEY=AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
				"GOOGLE_DEFAULT_CLIENT_ID=77185425430.apps.googleusercontent.com",
				"GOOGLE_DEFAULT_CLIENT_SECRET=OTJgUOQcT7lO7GsGZq2G4IlT",
			},

			Ops: new(container.Ops).
				Etc(fhs.AbsEtc, wantAutoEtcPrefix).
				OverlayReadonly(
					check.MustAbs("/nix/store"),
					fhs.AbsVarLib.Append("hakurei/base/org.nixos/.ro-store"),
					fhs.AbsVarLib.Append("hakurei/base/org.nixos/org.chromium.Chromium")).
				Readonly(hst.AbsPrivateTmp, 0755).
				Tmpfs(m("/proc/nonexistent/eval/etc/dbus"), 1<<13, 0755).
				Remount(fhs.AbsDev, syscall.MS_RDONLY).
				Remount(fhs.AbsRoot, syscall.MS_RDONLY),
		}, nil, nil},

		{"success", func(bool, bool) outcomeOp {
			return new(spFilesystemOp)
		}, hst.Template, nil, []stub.Call{
			call("lookupEnv", stub.ExpectArgs{dbus.SystemBusAddress}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/xdg_runtime_dir"}, nePrefix+"/xdg_runtime_dir", nil),
			call("evalSymlinks", stub.ExpectArgs{container.Nonexistent + "/tmp/hakurei.0"}, nePrefix+"/tmp/hakurei.0", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/nscd"}, "", &os.PathError{Op: "lstat", Path: "/var/run/nscd", Err: os.ErrNotExist}),
			call("verbosef", stub.ExpectArgs{"path %q does not yet exist", []any{"/var/run/nscd"}}, nil, nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/run/dbus"}, nePrefix+"/run/dbus", nil),
			call("readdir", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian"}, stubDebianRoot, nil),
			call("evalSymlinks", stub.ExpectArgs{"/etc/"}, nePrefix+"/etc", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper"}, nePrefix+"/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/upper", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work"}, nePrefix+"/var/lib/hakurei/nix/u0/org.chromium.Chromium/rw-store/work", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.nixos/ro-store"}, nePrefix+"/var/lib/hakurei/base/org.nixos/ro-store", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/u0/org.chromium.Chromium"}, nePrefix+"/var/lib/hakurei/u0/org.chromium.Chromium", nil),
			call("evalSymlinks", stub.ExpectArgs{"/dev/dri"}, nePrefix+"/dev/dri", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/bin"}, nePrefix+"/var/lib/hakurei/base/org.debian/bin", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/home"}, nePrefix+"/var/lib/hakurei/base/org.debian/home", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/lib64"}, nePrefix+"/var/lib/hakurei/base/org.debian/lib64", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/lost+found"}, nePrefix+"/var/lib/hakurei/base/org.debian/lost+found", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/nix"}, nePrefix+"/var/lib/hakurei/base/org.debian/nix", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/root"}, nePrefix+"/var/lib/hakurei/base/org.debian/root", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/run"}, nePrefix+"/var/lib/hakurei/base/org.debian/run", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/srv"}, nePrefix+"/var/lib/hakurei/base/org.debian/srv", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/sys"}, nePrefix+"/var/lib/hakurei/base/org.debian/sys", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/usr"}, nePrefix+"/var/lib/hakurei/base/org.debian/usr", nil),
			call("evalSymlinks", stub.ExpectArgs{"/var/lib/hakurei/base/org.debian/var"}, nePrefix+"/var/lib/hakurei/base/org.debian/var", nil),
		}, newI().
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"),
				acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"),
				acl.Read, acl.Write, acl.Execute), nil, nil, insertsOps(needsApplyState(func(state *outcomeStateParams) {
			state.filesystem = config.Container.Filesystem[1:]
		})), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Env: []string{
				"GOOGLE_API_KEY=AIzaSyBHDrl33hwRp4rMQY0ziRbj8K9LPA6vUCY",
				"GOOGLE_DEFAULT_CLIENT_ID=77185425430.apps.googleusercontent.com",
				"GOOGLE_DEFAULT_CLIENT_SECRET=OTJgUOQcT7lO7GsGZq2G4IlT",
			},

			Ops: new(container.Ops).
				Etc(fhs.AbsEtc, wantAutoEtcPrefix).
				Tmpfs(fhs.AbsTmp, 0, 0755).
				Overlay(
					check.MustAbs("/nix/store"),
					fhs.AbsVarLib.Append("hakurei/nix/u0/org.chromium.Chromium/rw-store/upper"),
					fhs.AbsVarLib.Append("hakurei/nix/u0/org.chromium.Chromium/rw-store/work"),
					fhs.AbsVarLib.Append("hakurei/base/org.nixos/ro-store")).
				Link(fhs.AbsRun.Append("current-system"), "/run/current-system", true).
				Link(fhs.AbsRun.Append("opengl-driver"), "/run/opengl-driver", true).
				Bind(
					fhs.AbsVarLib.Append("hakurei/u0/org.chromium.Chromium"),
					check.MustAbs("/data/data/org.chromium.Chromium"),
					std.BindWritable|std.BindEnsure).
				Bind(fhs.AbsDev.Append("dri"), fhs.AbsDev.Append("dri"), std.BindDevice|std.BindWritable|std.BindOptional).
				Remount(fhs.AbsRoot, syscall.MS_RDONLY),
		}, nil, nil},
	})
}

func TestFlattenExtraPerms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		perms []hst.ExtraPermConfig
		want  *system.I
	}{
		{"path nil check", append(hst.Template().ExtraPerms, hst.ExtraPermConfig{}), newI().
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"),
				acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"),
				acl.Read, acl.Write, acl.Execute)},

		{"template", hst.Template().ExtraPerms, newI().
			Ensure(m("/var/lib/hakurei/u0"), 0700).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0"),
				acl.Execute).
			UpdatePermType(system.User, m("/var/lib/hakurei/u0/org.chromium.Chromium"),
				acl.Read, acl.Write, acl.Execute)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := newI()
			flattenExtraPerms(got, tc.perms)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("flattenExtraPerms: sys = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// invalidFSHost implements the Host method of [hst.FilesystemConfig] with an invalid response.
type invalidFSHost bool

func (f invalidFSHost) Valid() bool           { return bool(f) }
func (invalidFSHost) Path() *check.Absolute   { panic("unreachable") }
func (invalidFSHost) Host() []*check.Absolute { return []*check.Absolute{nil} }
func (invalidFSHost) Apply(*hst.ApplyState)   { panic("unreachable") }
func (invalidFSHost) String() string          { panic("unreachable") }
