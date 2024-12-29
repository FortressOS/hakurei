package main

import (
	"encoding/json"
	"os"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/system"
)

type bundleInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`

	// passed through to [fst.Config]
	ID string `json:"id"`
	// passed through to [fst.Config]
	AppID int `json:"app_id"`
	// passed through to [fst.Config]
	Groups []string `json:"groups,omitempty"`
	// passed through to [fst.Config]
	UserNS bool `json:"userns,omitempty"`
	// passed through to [fst.Config]
	Net bool `json:"net,omitempty"`
	// passed through to [fst.Config]
	Dev bool `json:"dev,omitempty"`
	// passed through to [fst.Config]
	NoNewSession bool `json:"no_new_session,omitempty"`
	// passed through to [fst.Config]
	MapRealUID bool `json:"map_real_uid,omitempty"`
	// passed through to [fst.Config]
	DirectWayland bool `json:"direct_wayland,omitempty"`
	// passed through to [fst.Config]
	SystemBus *dbus.Config `json:"system_bus,omitempty"`
	// passed through to [fst.Config]
	SessionBus *dbus.Config `json:"session_bus,omitempty"`
	// passed through to [fst.Config]
	Enablements system.Enablements `json:"enablements"`

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

func loadBundleInfo(name string, beforeFail func()) *bundleInfo {
	bundle := new(bundleInfo)
	if f, err := os.Open(name); err != nil {
		beforeFail()
		fmsg.Fatalf("cannot open bundle: %v", err)
	} else if err = json.NewDecoder(f).Decode(&bundle); err != nil {
		beforeFail()
		fmsg.Fatalf("cannot parse bundle metadata: %v", err)
	} else if err = f.Close(); err != nil {
		fmsg.Printf("cannot close bundle metadata: %v", err)
		// not fatal
	}

	if bundle.ID == "" {
		beforeFail()
		fmsg.Fatal("application identifier must not be empty")
	}

	return bundle
}

func formatHostname(name string) string {
	if h, err := os.Hostname(); err != nil {
		fmsg.Printf("cannot get hostname: %v", err)
		return "fortify-" + name
	} else {
		return h + "-" + name
	}
}
