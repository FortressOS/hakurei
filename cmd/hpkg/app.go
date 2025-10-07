package main

import (
	"encoding/json"
	"log"
	"os"

	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
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
	HostNet bool `json:"net,omitempty"`
	// passed through to [hst.Config]
	HostAbstract bool `json:"abstract,omitempty"`
	// passed through to [hst.Config]
	Device bool `json:"dev,omitempty"`
	// passed through to [hst.Config]
	Tty bool `json:"tty,omitempty"`
	// passed through to [hst.Config]
	MapRealUID bool `json:"map_real_uid,omitempty"`
	// passed through to [hst.Config]
	DirectWayland bool `json:"direct_wayland,omitempty"`
	// passed through to [hst.Config]
	SystemBus *hst.BusConfig `json:"system_bus,omitempty"`
	// passed through to [hst.Config]
	SessionBus *hst.BusConfig `json:"session_bus,omitempty"`
	// passed through to [hst.Config]
	Enablements *hst.Enablements `json:"enablements,omitempty"`

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
	Launcher *check.Absolute `json:"launcher"`
	// store path to /run/current-system
	CurrentSystem *check.Absolute `json:"current_system"`
	// store path to home-manager activation package
	ActivationPackage string `json:"activation_package"`
}

func (app *appInfo) toHst(pathSet *appPathSet, pathname *check.Absolute, argv []string, flagDropShell bool) *hst.Config {
	config := &hst.Config{
		ID: app.ID,

		Enablements: app.Enablements,

		SystemBus:     app.SystemBus,
		SessionBus:    app.SessionBus,
		DirectWayland: app.DirectWayland,

		Identity: app.Identity,
		Groups:   app.Groups,

		Container: &hst.ContainerConfig{
			Hostname:     formatHostname(app.Name),
			Devel:        app.Devel,
			Userns:       app.Userns,
			HostNet:      app.HostNet,
			HostAbstract: app.HostAbstract,
			Device:       app.Device,
			Tty:          app.Tty || flagDropShell,
			MapRealUID:   app.MapRealUID,
			Multiarch:    app.Multiarch,
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{Target: fhs.AbsEtc, Source: pathSet.cacheDir.Append("etc"), Special: true}},
				{FilesystemConfig: &hst.FSBind{Source: pathSet.nixPath.Append("store"), Target: pathNixStore}},
				{FilesystemConfig: &hst.FSLink{Target: pathCurrentSystem, Linkname: app.CurrentSystem.String()}},
				{FilesystemConfig: &hst.FSLink{Target: pathBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSLink{Target: fhs.AbsUsrBin, Linkname: pathSwBin.String()}},
				{FilesystemConfig: &hst.FSBind{Source: pathSet.metaPath, Target: hst.AbsTmp.Append("app")}},
				{FilesystemConfig: &hst.FSBind{Source: fhs.AbsEtc.Append("resolv.conf"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Source: fhs.AbsSys.Append("block"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Source: fhs.AbsSys.Append("bus"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Source: fhs.AbsSys.Append("class"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Source: fhs.AbsSys.Append("dev"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Source: fhs.AbsSys.Append("devices"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Target: pathDataData.Append(app.ID), Source: pathSet.homeDir, Write: true, Ensure: true}},
			},

			Username: "hakurei",
			Shell:    pathShell,
			Home:     pathDataData.Append(app.ID),

			Path: pathname,
			Args: argv,
		},
		ExtraPerms: []*hst.ExtraPermConfig{
			{Path: dataHome, Execute: true},
			{Ensure: true, Path: pathSet.baseDir, Read: true, Write: true, Execute: true},
		},
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
	if bundle.Launcher == nil {
		beforeFail()
		log.Fatal("launcher must not be empty")
	}
	if bundle.CurrentSystem == nil {
		beforeFail()
		log.Fatal("current-system must not be empty")
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
