package main

import (
	"context"
	"os"
	"strings"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/hst"
)

func withNixDaemon(
	ctx context.Context,
	msg container.Msg,
	action string, command []string, net bool, updateConfig func(config *hst.Config) *hst.Config,
	app *appInfo, pathSet *appPathSet, dropShell bool, beforeFail func(),
) {
	mustRunAppDropShell(ctx, msg, updateConfig(&hst.Config{
		ID: app.ID,

		ExtraPerms: []*hst.ExtraPermConfig{
			{Path: dataHome, Execute: true},
			{Ensure: true, Path: pathSet.baseDir, Read: true, Write: true, Execute: true},
		},

		Identity: app.Identity,

		Container: &hst.ContainerConfig{
			Hostname:  formatHostname(app.Name) + "-" + action,
			Userns:    true, // nix sandbox requires userns
			HostNet:   net,
			Multiarch: true,
			Tty:       dropShell,
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{Target: container.AbsFHSEtc, Source: pathSet.cacheDir.Append("etc"), Special: true}},
				{FilesystemConfig: &hst.FSBind{Source: pathSet.nixPath, Target: pathNix, Write: true}},
				{FilesystemConfig: &hst.FSLink{Target: pathCurrentSystem, Linkname: app.CurrentSystem.String()}},
				{FilesystemConfig: &hst.FSLink{Target: pathBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSLink{Target: container.AbsFHSUsrBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSBind{Target: pathDataData.Append(app.ID), Source: pathSet.homeDir, Write: true, Ensure: true}},
			},

			Username: "hakurei",
			Shell:    pathShell,
			Home:     pathDataData.Append(app.ID),

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
		},
	}), dropShell, beforeFail)
}

func withCacheDir(
	ctx context.Context,
	msg container.Msg,
	action string, command []string, workDir *check.Absolute,
	app *appInfo, pathSet *appPathSet, dropShell bool, beforeFail func()) {
	mustRunAppDropShell(ctx, msg, &hst.Config{
		ID: app.ID,

		ExtraPerms: []*hst.ExtraPermConfig{
			{Path: dataHome, Execute: true},
			{Ensure: true, Path: pathSet.baseDir, Read: true, Write: true, Execute: true},
			{Path: workDir, Execute: true},
		},

		Identity: app.Identity,

		Container: &hst.ContainerConfig{
			Hostname:  formatHostname(app.Name) + "-" + action,
			Multiarch: true,
			Tty:       dropShell,
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{Target: container.AbsFHSEtc, Source: workDir.Append(container.FHSEtc), Special: true}},
				{FilesystemConfig: &hst.FSBind{Source: workDir.Append("nix"), Target: pathNix}},
				{FilesystemConfig: &hst.FSLink{Target: pathCurrentSystem, Linkname: app.CurrentSystem.String()}},
				{FilesystemConfig: &hst.FSLink{Target: pathBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSLink{Target: container.AbsFHSUsrBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSBind{Source: workDir, Target: hst.AbsTmp.Append("bundle")}},
				{FilesystemConfig: &hst.FSBind{Target: pathDataData.Append(app.ID, "cache"), Source: pathSet.cacheDir, Write: true, Ensure: true}},
			},

			Username: "nixos",
			Shell:    pathShell,
			Home:     pathDataData.Append(app.ID, "cache"),

			Path: pathShell,
			Args: []string{bash, "-lc", strings.Join(command, " && ")},
		},
	}, dropShell, beforeFail)
}

func mustRunAppDropShell(ctx context.Context, msg container.Msg, config *hst.Config, dropShell bool, beforeFail func()) {
	if dropShell {
		if config.Container != nil {
			config.Container.Args = []string{bash, "-l"}
		}
		mustRunApp(ctx, msg, config, beforeFail)
		beforeFail()
		msg.BeforeExit()
		os.Exit(0)
	}
	mustRunApp(ctx, msg, config, beforeFail)
}
