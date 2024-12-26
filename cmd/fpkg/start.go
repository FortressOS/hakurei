package main

import (
	"flag"
	"path"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func actionStart(args []string) {
	set := flag.NewFlagSet("start", flag.ExitOnError)
	var dropShell bool
	set.BoolVar(&dropShell, "s", false, "Drop to a shell on activation")

	// Ignore errors; set is set for ExitOnError.
	_ = set.Parse(args)

	args = set.Args()

	if len(args) < 1 {
		fmsg.Fatal("invalid argument")
	}
	id := args[0]
	pathSet := pathSetByApp(id)
	app := loadBundleInfo(pathSet.metaPath, func() {})

	if app.ID != id {
		fmsg.Fatalf("app %q claims to have identifier %q", id, app.ID)
	}

	command := make([]string, 1, len(args))
	if !dropShell {
		command[0] = app.Launcher
	} else {
		command[0] = shell
	}
	command = append(command, args[1:]...)

	config := &fst.Config{
		ID:      app.ID,
		Command: command,
		Confinement: fst.ConfinementConfig{
			AppID:    app.AppID,
			Groups:   app.Groups,
			Username: "fortify",
			Inner:    path.Join("/data/data", app.ID),
			Outer:    pathSet.homeDir,
			Sandbox: &fst.SandboxConfig{
				Hostname:      formatHostname(app.Name),
				UserNS:        app.UserNS,
				Net:           app.Net,
				Dev:           app.Dev,
				NoNewSession:  app.NoNewSession || dropShell,
				MapRealUID:    app.MapRealUID,
				DirectWayland: app.DirectWayland,
				Filesystem: []*fst.FilesystemConfig{
					{Src: path.Join(pathSet.nixPath, "store"), Dst: "/nix/store", Must: true},
					{Src: pathSet.metaPath, Dst: path.Join(fst.Tmp, "app"), Must: true},
					{Src: "/etc/resolv.conf"},
					{Src: "/sys/block"},
					{Src: "/sys/bus"},
					{Src: "/sys/class"},
					{Src: "/sys/dev"},
					{Src: "/sys/devices"},
				},
				Link: [][2]string{
					{app.CurrentSystem, "/run/current-system"},
					{"/run/current-system/sw/bin", "/bin"},
					{"/run/current-system/sw/bin", "/usr/bin"},
				},
				Etc:     path.Join(pathSet.cacheDir, "etc"),
				AutoEtc: true,
			},
			ExtraPerms: []*fst.ExtraPermConfig{
				{Path: dataHome, Execute: true},
				{Ensure: true, Path: pathSet.baseDir, Read: true, Write: true, Execute: true},
			},
			SystemBus:   app.SystemBus,
			SessionBus:  app.SessionBus,
			Enablements: app.Enablements,
		},
	}

	if app.GPU {
		config.Confinement.Sandbox.Filesystem = append(config.Confinement.Sandbox.Filesystem,
			&fst.FilesystemConfig{Src: "/dev/dri", Device: true})
	}

	fortifyApp(config, func() {})
	fmsg.Exit(0)
}
