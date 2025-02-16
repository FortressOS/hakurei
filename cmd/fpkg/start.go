package main

import (
	"flag"
	"log"
	"path"

	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/internal"
)

func actionStart(args []string) {
	set := flag.NewFlagSet("start", flag.ExitOnError)
	var (
		dropShell      bool
		dropShellNixGL bool
		autoDrivers    bool
	)
	set.BoolVar(&dropShell, "s", false, "Drop to a shell")
	set.BoolVar(&dropShellNixGL, "sg", false, "Drop to a shell on nixGL build")
	set.BoolVar(&autoDrivers, "autodrivers", false, "Attempt automatic opengl driver detection")

	// Ignore errors; set is set for ExitOnError.
	_ = set.Parse(args)

	args = set.Args()

	if len(args) < 1 {
		log.Fatal("invalid argument")
	}

	/*
		Parse app metadata.
	*/

	id := args[0]
	pathSet := pathSetByApp(id)
	app := loadBundleInfo(pathSet.metaPath, func() {})
	if app.ID != id {
		log.Fatalf("app %q claims to have identifier %q", id, app.ID)
	}

	/*
		Prepare nixGL.
	*/

	if app.GPU && autoDrivers {
		withNixDaemon("nix-gl", []string{
			"mkdir -p /nix/.nixGL/auto",
			"rm -rf /nix/.nixGL/auto",
			"export NIXPKGS_ALLOW_UNFREE=1",
			"nix build --impure " +
				"--out-link /nix/.nixGL/auto/opengl " +
				"--override-input nixpkgs path:/etc/nixpkgs " +
				"path:" + app.NixGL,
			"nix build --impure " +
				"--out-link /nix/.nixGL/auto/vulkan " +
				"--override-input nixpkgs path:/etc/nixpkgs " +
				"path:" + app.NixGL + "#nixVulkanNvidia",
		}, true, func(config *fst.Config) *fst.Config {
			config.Confinement.Sandbox.Filesystem = append(config.Confinement.Sandbox.Filesystem, []*fst.FilesystemConfig{
				{Src: "/etc/resolv.conf"},
				{Src: "/sys/block"},
				{Src: "/sys/bus"},
				{Src: "/sys/class"},
				{Src: "/sys/dev"},
				{Src: "/sys/devices"},
			}...)
			appendGPUFilesystem(config)
			return config
		}, app, pathSet, dropShellNixGL, func() {})
	}

	/*
		Create app configuration.
	*/

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
				Syscall:       &bwrap.SyscallPolicy{DenyDevel: !app.Devel, Multiarch: app.Multiarch, Bluetooth: app.Bluetooth},
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

	/*
		Expose GPU devices.
	*/

	if app.GPU {
		config.Confinement.Sandbox.Filesystem = append(config.Confinement.Sandbox.Filesystem,
			&fst.FilesystemConfig{Src: path.Join(pathSet.nixPath, ".nixGL"), Dst: path.Join(fst.Tmp, "nixGL")})
		appendGPUFilesystem(config)
	}

	/*
		Spawn app.
	*/

	fortifyApp(config, func() {})
	internal.Exit(0)
}

func appendGPUFilesystem(config *fst.Config) {
	config.Confinement.Sandbox.Filesystem = append(config.Confinement.Sandbox.Filesystem, []*fst.FilesystemConfig{
		// flatpak commit 763a686d874dd668f0236f911de00b80766ffe79
		{Src: "/dev/dri", Device: true},
		// mali
		{Src: "/dev/mali", Device: true},
		{Src: "/dev/mali0", Device: true},
		{Src: "/dev/umplock", Device: true},
		// nvidia
		{Src: "/dev/nvidiactl", Device: true},
		{Src: "/dev/nvidia-modeset", Device: true},
		// nvidia OpenCL/CUDA
		{Src: "/dev/nvidia-uvm", Device: true},
		{Src: "/dev/nvidia-uvm-tools", Device: true},

		// flatpak commit d2dff2875bb3b7e2cd92d8204088d743fd07f3ff
		{Src: "/dev/nvidia0", Device: true}, {Src: "/dev/nvidia1", Device: true},
		{Src: "/dev/nvidia2", Device: true}, {Src: "/dev/nvidia3", Device: true},
		{Src: "/dev/nvidia4", Device: true}, {Src: "/dev/nvidia5", Device: true},
		{Src: "/dev/nvidia6", Device: true}, {Src: "/dev/nvidia7", Device: true},
		{Src: "/dev/nvidia8", Device: true}, {Src: "/dev/nvidia9", Device: true},
		{Src: "/dev/nvidia10", Device: true}, {Src: "/dev/nvidia11", Device: true},
		{Src: "/dev/nvidia12", Device: true}, {Src: "/dev/nvidia13", Device: true},
		{Src: "/dev/nvidia14", Device: true}, {Src: "/dev/nvidia15", Device: true},
		{Src: "/dev/nvidia16", Device: true}, {Src: "/dev/nvidia17", Device: true},
		{Src: "/dev/nvidia18", Device: true}, {Src: "/dev/nvidia19", Device: true},
	}...)
}
