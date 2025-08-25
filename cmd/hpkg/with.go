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
			HostNet:      net,
			SeccompFlags: seccomp.AllowMultiarch,
			Tty:          dropShell,
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{Target: container.AbsFHSEtc, Source: pathSet.cacheDir.Append("etc"), Special: true}},
				{FilesystemConfig: &hst.FSBind{Source: pathSet.nixPath, Target: pathNix, Write: true}},
				{FilesystemConfig: &hst.FSLink{Target: pathCurrentSystem, Linkname: app.CurrentSystem.String()}},
				{FilesystemConfig: &hst.FSLink{Target: pathBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSLink{Target: container.AbsFHSUsrBin, Linkname: pathSwBin.String()}},
			},
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
				{FilesystemConfig: &hst.FSBind{Target: container.AbsFHSEtc, Source: workDir.Append(container.FHSEtc), Special: true}},
				{FilesystemConfig: &hst.FSBind{Source: workDir.Append("nix"), Target: pathNix}},
				{FilesystemConfig: &hst.FSLink{Target: pathCurrentSystem, Linkname: app.CurrentSystem.String()}},
				{FilesystemConfig: &hst.FSLink{Target: pathBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSLink{Target: container.AbsFHSUsrBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSBind{Source: workDir, Target: hst.AbsTmp.Append("bundle")}},
			},
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
