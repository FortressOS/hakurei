package main

import (
	"encoding/json"
	"log"
	"os"

	"hakurei.app/container"
	"hakurei.app/container/seccomp"
	"hakurei.app/hst"
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
	Launcher *container.Absolute `json:"launcher"`
	// store path to /run/current-system
	CurrentSystem *container.Absolute `json:"current_system"`
	// store path to home-manager activation package
	ActivationPackage string `json:"activation_package"`
}

func (app *appInfo) toHst(pathSet *appPathSet, pathname *container.Absolute, argv []string, flagDropShell bool) *hst.Config {
	config := &hst.Config{
		ID: app.ID,

		Path: pathname,
		Args: argv,

		Enablements: app.Enablements,

		SystemBus:     app.SystemBus,
		SessionBus:    app.SessionBus,
		DirectWayland: app.DirectWayland,

		Username: "hakurei",
		Shell:    pathShell,
		Data:     pathSet.homeDir,
		Dir:      pathDataData.Append(app.ID),

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
			Filesystem: []hst.FilesystemConfigJSON{
				{FilesystemConfig: &hst.FSBind{Src: pathSet.nixPath.Append("store"), Dst: pathNixStore}},
				{FilesystemConfig: &hst.FSBind{Src: pathSet.metaPath, Dst: hst.AbsTmp.Append("app")}},
				{FilesystemConfig: &hst.FSBind{Src: container.AbsFHSEtc.Append("resolv.conf"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Src: container.AbsFHSSys.Append("block"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Src: container.AbsFHSSys.Append("bus"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Src: container.AbsFHSSys.Append("class"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Src: container.AbsFHSSys.Append("dev"), Optional: true}},
				{FilesystemConfig: &hst.FSBind{Src: container.AbsFHSSys.Append("devices"), Optional: true}},
			},
			Link: []hst.LinkConfig{
				{pathCurrentSystem, app.CurrentSystem.String()},
				{pathBin, pathSwBin.String()},
				{container.AbsFHSUsrBin, pathSwBin.String()},
			},
			Etc:     pathSet.cacheDir.Append("etc"),
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
