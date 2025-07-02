package main

import (
	"context"
	"path"
	"strings"

	"git.gensokyo.uk/security/hakurei/container/seccomp"
	"git.gensokyo.uk/security/hakurei/hst"
	"git.gensokyo.uk/security/hakurei/internal"
)

func withNixDaemon(
	ctx context.Context,
	action string, command []string, net bool, updateConfig func(config *hst.Config) *hst.Config,
	app *appInfo, pathSet *appPathSet, dropShell bool, beforeFail func(),
) {
	mustRunAppDropShell(ctx, updateConfig(&hst.Config{
		ID: app.ID,

		Path: shellPath,
		Args: []string{shellPath, "-lc", "rm -f /nix/var/nix/daemon-socket/socket && " +
			// start nix-daemon
			"nix-daemon --store / & " +
			// wait for socket to appear
			"(while [ ! -S /nix/var/nix/daemon-socket/socket ]; do sleep 0.01; done) && " +
			// create directory so nix stops complaining
			"mkdir -p /nix/var/nix/profiles/per-user/root/channels && " +
			strings.Join(command, " && ") +
			// terminate nix-daemon
			" && pkill nix-daemon",
		},

		Username: "hakurei",
		Shell:    shellPath,
		Data:     pathSet.homeDir,
		Dir:      path.Join("/data/data", app.ID),
		ExtraPerms: []*hst.ExtraPermConfig{
			{Path: dataHome, Execute: true},
			{Ensure: true, Path: pathSet.baseDir, Read: true, Write: true, Execute: true},
		},

		Identity: app.Identity,

		Container: &hst.ContainerConfig{
			Hostname:     formatHostname(app.Name) + "-" + action,
			Userns:       true, // nix sandbox requires userns
			Net:          net,
			SeccompFlags: seccomp.AllowMultiarch,
			Tty:          dropShell,
			Filesystem: []*hst.FilesystemConfig{
				{Src: pathSet.nixPath, Dst: "/nix", Write: true, Must: true},
			},
			Link: [][2]string{
				{app.CurrentSystem, "/run/current-system"},
				{"/run/current-system/sw/bin", "/bin"},
				{"/run/current-system/sw/bin", "/usr/bin"},
			},
			Etc:     path.Join(pathSet.cacheDir, "etc"),
			AutoEtc: true,
		},
	}), dropShell, beforeFail)
}

func withCacheDir(
	ctx context.Context,
	action string, command []string, workDir string,
	app *appInfo, pathSet *appPathSet, dropShell bool, beforeFail func()) {
	mustRunAppDropShell(ctx, &hst.Config{
		ID: app.ID,

		Path: shellPath,
		Args: []string{shellPath, "-lc", strings.Join(command, " && ")},

		Username: "nixos",
		Shell:    shellPath,
		Data:     pathSet.cacheDir, // this also ensures cacheDir via shim
		Dir:      path.Join("/data/data", app.ID, "cache"),
		ExtraPerms: []*hst.ExtraPermConfig{
			{Path: dataHome, Execute: true},
			{Ensure: true, Path: pathSet.baseDir, Read: true, Write: true, Execute: true},
			{Path: workDir, Execute: true},
		},

		Identity: app.Identity,

		Container: &hst.ContainerConfig{
			Hostname:     formatHostname(app.Name) + "-" + action,
			SeccompFlags: seccomp.AllowMultiarch,
			Tty:          dropShell,
			Filesystem: []*hst.FilesystemConfig{
				{Src: path.Join(workDir, "nix"), Dst: "/nix", Must: true},
				{Src: workDir, Dst: path.Join(hst.Tmp, "bundle"), Must: true},
			},
			Link: [][2]string{
				{app.CurrentSystem, "/run/current-system"},
				{"/run/current-system/sw/bin", "/bin"},
				{"/run/current-system/sw/bin", "/usr/bin"},
			},
			Etc:     path.Join(workDir, "etc"),
			AutoEtc: true,
		},
	}, dropShell, beforeFail)
}

func mustRunAppDropShell(ctx context.Context, config *hst.Config, dropShell bool, beforeFail func()) {
	if dropShell {
		config.Args = []string{shellPath, "-l"}
		mustRunApp(ctx, config, beforeFail)
		beforeFail()
		internal.Exit(0)
	}
	mustRunApp(ctx, config, beforeFail)
}
