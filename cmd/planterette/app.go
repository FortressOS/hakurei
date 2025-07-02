package main

import (
	"encoding/json"
	"log"
	"os"
	"path"

	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/dbus"
)

type appInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`

	// passed through to [hst.Config]
	ID string `json:"id"`
	// passed through to [hst.Config]
	Identity int `json:"identity"`
	// passed through to [hst.Config]
	Groups []string `json:"groups,omitempty"`
	// passed through to [hst.Config]
	Devel bool `json:"devel,omitempty"`
	// passed through to [hst.Config]
	Userns bool `json:"userns,omitempty"`
	// passed through to [hst.Config]
	Net bool `json:"net,omitempty"`
	// passed through to [hst.Config]
	Device bool `json:"dev,omitempty"`
	// passed through to [hst.Config]
	Tty bool `json:"tty,omitempty"`
	// passed through to [hst.Config]
	MapRealUID bool `json:"map_real_uid,omitempty"`
	// passed through to [hst.Config]
	DirectWayland bool `json:"direct_wayland,omitempty"`
	// passed through to [hst.Config]
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// passed through to [hst.Config]
	SessionBus *dbus.Config `json:"session_bus,omitempty"`
	// passed through to [hst.Config]
	Enablements system.Enablement `json:"enablements"`

	// passed through to [hst.Config]
	Multiarch bool `json:"multiarch,omitempty"`
	// passed through to [hst.Config]
	Bluetooth bool `json:"bluetooth,omitempty"`

	// allow gpu access within sandbox
	GPU bool `json:"gpu"`
	// store path to nixGL mesa wrappers
	Mesa string `json:"mesa,omitempty"`
	// store path to nixGL source
	NixGL string `json:"nix_gl,omitempty"`
	// store path to activate-and-exec script
	Launcher string `json:"launcher"`
	// store path to /run/current-system
	CurrentSystem string `json:"current_system"`
	// store path to home-manager activation package
	ActivationPackage string `json:"activation_package"`
}

func (app *appInfo) toFst(pathSet *appPathSet, argv []string, flagDropShell bool) *hst.Config {
	config := &hst.Config{
		ID: app.ID,

		Path: argv[0],
		Args: argv,

		Enablements: app.Enablements,

		SystemBus:     app.SystemBus,
		SessionBus:    app.SessionBus,
		DirectWayland: app.DirectWayland,

		Username: "hakurei",
		Shell:    shellPath,
		Data:     pathSet.homeDir,
		Dir:      path.Join("/data/data", app.ID),

		Identity: app.Identity,
		Groups:   app.Groups,

		Container: &hst.ContainerConfig{
			Hostname:   formatHostname(app.Name),
			Devel:      app.Devel,
			Userns:     app.Userns,
			Net:        app.Net,
			Device:     app.Device,
			Tty:        app.Tty || flagDropShell,
			MapRealUID: app.MapRealUID,
			Filesystem: []*hst.FilesystemConfig{
				{Src: path.Join(pathSet.nixPath, "store"), Dst: "/nix/store", Must: true},
				{Src: pathSet.metaPath, Dst: path.Join(hst.Tmp, "app"), Must: true},
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
		ExtraPerms: []*hst.ExtraPermConfig{
			{Path: dataHome, Execute: true},
			{Ensure: true, Path: pathSet.baseDir, Read: true, Write: true, Execute: true},
		},
	}
	if app.Multiarch {
		config.Container.SeccompFlags |= seccomp.AllowMultiarch
	}
	if app.Bluetooth {
		config.Container.SeccompFlags |= seccomp.AllowBluetooth
	}
	return config
}

func loadAppInfo(name string, beforeFail func()) *appInfo {
	bundle := new(appInfo)
	if f, err := os.Open(name); err != nil {
		beforeFail()
		log.Fatalf("cannot open bundle: %v", err)
	} else if err = json.NewDecoder(f).Decode(&bundle); err != nil {
		beforeFail()
		log.Fatalf("cannot parse bundle metadata: %v", err)
	} else if err = f.Close(); err != nil {
		log.Printf("cannot close bundle metadata: %v", err)
		// not fatal
	}

	if bundle.ID == "" {
		beforeFail()
		log.Fatal("application identifier must not be empty")
	}

	return bundle
}

func formatHostname(name string) string {
	if h, err := os.Hostname(); err != nil {
		log.Printf("cannot get hostname: %v", err)
		return "hakurei-" + name
	} else {
		return h + "-" + name
	}
}
