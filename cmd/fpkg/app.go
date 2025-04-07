package main

import (
	"encoding/json"
	"log"
	"os"
	"path"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/sandbox/seccomp"
	"git.gensokyo.uk/security/fortify/system"
)

type appInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`

	// passed through to [fst.Config]
	ID string `json:"id"`
	// passed through to [fst.Config]
	AppID int `json:"app_id"`
	// passed through to [fst.Config]
	Groups []string `json:"groups,omitempty"`
	// passed through to [fst.Config]
	Devel bool `json:"devel,omitempty"`
	// passed through to [fst.Config]
	Userns bool `json:"userns,omitempty"`
	// passed through to [fst.Config]
	Net bool `json:"net,omitempty"`
	// passed through to [fst.Config]
	Dev bool `json:"dev,omitempty"`
	// passed through to [fst.Config]
	Tty bool `json:"tty,omitempty"`
	// passed through to [fst.Config]
	MapRealUID bool `json:"map_real_uid,omitempty"`
	// passed through to [fst.Config]
	DirectWayland bool `json:"direct_wayland,omitempty"`
	// passed through to [fst.Config]
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// passed through to [fst.Config]
	SessionBus *dbus.Config `json:"session_bus,omitempty"`
	// passed through to [fst.Config]
	Enablements system.Enablement `json:"enablements"`

	// passed through to [fst.Config]
	Multiarch bool `json:"multiarch,omitempty"`
	// passed through to [fst.Config]
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

func (app *appInfo) toFst(pathSet *appPathSet, argv []string, flagDropShell bool) *fst.Config {
	config := &fst.Config{
		ID:   app.ID,
		Path: argv[0],
		Args: argv,
		Confinement: fst.ConfinementConfig{
			AppID:    app.AppID,
			Groups:   app.Groups,
			Username: "fortify",
			Inner:    path.Join("/data/data", app.ID),
			Outer:    pathSet.homeDir,
			Shell:    shellPath,
			Sandbox: &fst.SandboxConfig{
				Hostname:      formatHostname(app.Name),
				Devel:         app.Devel,
				Userns:        app.Userns,
				Net:           app.Net,
				Dev:           app.Dev,
				Tty:           app.Tty || flagDropShell,
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
	if app.Multiarch {
		config.Confinement.Sandbox.Seccomp |= seccomp.FilterMultiarch
	}
	if app.Bluetooth {
		config.Confinement.Sandbox.Seccomp |= seccomp.FilterBluetooth
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
		return "fortify-" + name
	} else {
		return h + "-" + name
	}
}
