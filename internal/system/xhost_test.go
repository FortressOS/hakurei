package system

import (
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/system/xcb"
)

func TestXHostOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"xcbChangeHosts revert", 0xbeef, hst.EX11, xhostOp("chronos"), []stub.Call{
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, stub.UniqueError(1)),
		}, &OpError{Op: "xhost", Err: stub.UniqueError(1)}, nil, nil},

		{"xcbChangeHosts revert", 0xbeef, hst.EX11, xhostOp("chronos"), []stub.Call{
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"deleting entry %s from X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeDelete), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, stub.UniqueError(0)),
		}, &OpError{Op: "xhost", Err: stub.UniqueError(0), Revert: true}},

		{"success skip", 0xbeef, 0, xhostOp("chronos"), []stub.Call{
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"skipping entry %s in X11", []any{xhostOp("chronos")}}, nil, nil),
		}, nil},

		{"success", 0xbeef, hst.EX11, xhostOp("chronos"), []stub.Call{
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"deleting entry %s from X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeDelete), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, nil),
		}, nil},
	})

	checkOpsBuilder(t, "ChangeHosts", []opsBuilderTestCase{
		{"xhost", 0xcafe, func(_ *testing.T, sys *I) {
			sys.ChangeHosts("chronos")
		}, []Op{
			xhostOp("chronos"),
		}, stub.Expect{}},
	})

	checkOpIs(t, []opIsTestCase{
		{"differs", xhostOp("kbd"), xhostOp("chronos"), false},
		{"equals", xhostOp("chronos"), xhostOp("chronos"), true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"xhost", xhostOp("chronos"), hst.EX11, "/tmp/.X11-unix", "SI:localuser:chronos"},
	})
}
