package app

import (
	"testing"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
)

func TestOutcomeStateValid(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			if got := tc.s.valid(); got != tc.want {
				t.Errorf("valid: %v, want %v", got, tc.want)
			}
		})
	}
}
