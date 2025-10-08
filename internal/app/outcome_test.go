package app

import (
	"reflect"
	"testing"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
)

func TestOutcomeStateValid(t *testing.T) {
	testCases := []struct {
		name string
		s    *outcomeState
		want bool
	}{
		{"nil", nil, false},
		{"zero", new(outcomeState), false},
		{"shim", &outcomeState{Shim: &shimParams{PrivPID: -1, Ops: []outcomeOp{}}, Container: new(hst.ContainerConfig), EnvPaths: new(EnvPaths)}, false},
		{"id", &outcomeState{Shim: &shimParams{PrivPID: 1, Ops: []outcomeOp{}}, Container: new(hst.ContainerConfig), EnvPaths: new(EnvPaths)}, false},
		{"container", &outcomeState{Shim: &shimParams{PrivPID: 1, Ops: []outcomeOp{}}, ID: new(state.ID), EnvPaths: new(EnvPaths)}, false},
		{"envpaths", &outcomeState{Shim: &shimParams{PrivPID: 1, Ops: []outcomeOp{}}, ID: new(state.ID), Container: new(hst.ContainerConfig)}, false},
		{"valid", &outcomeState{Shim: &shimParams{PrivPID: 1, Ops: []outcomeOp{}}, ID: new(state.ID), Container: new(hst.ContainerConfig), EnvPaths: new(EnvPaths)}, true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.valid(); got != tc.want {
				t.Errorf("valid: %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFromConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config *hst.Config
		want   []outcomeOp
	}{
		{"ne", new(hst.Config), []outcomeOp{
			&spParamsOp{},
			spFilesystemOp{},
			spRuntimeOp{},
			spTmpdirOp{},
			spAccountOp{},
			spFinal{},
		}},
		{"wayland pulse", &hst.Config{Enablements: hst.NewEnablements(hst.EWayland | hst.EPulse)}, []outcomeOp{
			&spParamsOp{},
			spFilesystemOp{},
			spRuntimeOp{},
			spTmpdirOp{},
			spAccountOp{},
			&spWaylandOp{},
			&spPulseOp{},
			spFinal{},
		}},
		{"all", &hst.Config{Enablements: hst.NewEnablements(0xff)}, []outcomeOp{
			&spParamsOp{},
			spFilesystemOp{},
			spRuntimeOp{},
			spTmpdirOp{},
			spAccountOp{},
			&spWaylandOp{},
			&spX11Op{},
			&spPulseOp{},
			&spDBusOp{},
			spFinal{},
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fromConfig(tc.config); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fromConfig: %#v, want %#v", got, tc.want)
			}
		})
	}
}
