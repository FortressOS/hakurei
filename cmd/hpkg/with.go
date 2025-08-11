package main

import (
	"context"
	"strings"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/internal"
)

func withNixDaemon(
	ctx context.Context,
	action string, command []string, net bool, updateConfig func(config *hst.Config) *hst.Config,
	app *appInfo, pathSet *appPathSet, dropShell bool, beforeFail func(),
) {
	mustRunAppDropShell(ctx, updateConfig(&hst.Config{
		ID: app.ID,

		Path: pathShell,
		Args: []string{bash, "-lc", "rm -f /nix/var/nix/daemon-socket/socket && " +
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
		Shell:    pathShell,
		Data:     pathSet.homeDir,
		Dir:      pathDataData.Append(app.ID),
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
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{Src: pathSet.nixPath, Dst: pathNix, Write: true}},
			},
			Link: []hst.LinkConfig{
				{pathCurrentSystem, app.CurrentSystem.String()},
				{pathBin, pathSwBin.String()},
				{container.AbsFHSUsrBin, pathSwBin.String()},
			},
			Etc:     pathSet.cacheDir.Append("etc"),
			AutoEtc: true,
		},
	}), dropShell, beforeFail)
}

func withCacheDir(
	ctx context.Context,
	action string, command []string, workDir *container.Absolute,
	app *appInfo, pathSet *appPathSet, dropShell bool, beforeFail func()) {
	mustRunAppDropShell(ctx, &hst.Config{
		ID: app.ID,

		Path: pathShell,
		Args: []string{bash, "-lc", strings.Join(command, " && ")},

		Username: "nixos",
		Shell:    pathShell,
		Data:     pathSet.cacheDir, // this also ensures cacheDir via shim
		Dir:      pathDataData.Append(app.ID, "cache"),
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
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{Src: workDir.Append("nix"), Dst: pathNix}},
				{FilesystemConfig: &hst.FSBind{Src: workDir, Dst: hst.AbsTmp.Append("bundle")}},
			},
			Link: []hst.LinkConfig{
				{pathCurrentSystem, app.CurrentSystem.String()},
				{pathBin, pathSwBin.String()},
				{container.AbsFHSUsrBin, pathSwBin.String()},
			},
			Etc:     workDir.Append(container.FHSEtc),
			AutoEtc: true,
		},
	}, dropShell, beforeFail)
}

func mustRunAppDropShell(ctx context.Context, config *hst.Config, dropShell bool, beforeFail func()) {
	if dropShell {
		config.Args = []string{bash, "-l"}
		mustRunApp(ctx, config, beforeFail)
		beforeFail()
		internal.Exit(0)
	}
	mustRunApp(ctx, config, beforeFail)
}
