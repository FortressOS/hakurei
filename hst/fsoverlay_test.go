package hst_test

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/hst"
)

func TestFSOverlay(t *testing.T) {
	t.Parallel()

	checkFs(t, []fsTestCase{
		{"nil", (*hst.FSOverlay)(nil), false, nil, nil, nil, "<invalid>"},
		{"nil lower", &hst.FSOverlay{Target: m("/etc"), Lower: []*check.Absolute{nil}}, false, nil, nil, nil, "<invalid>"},
		{"zero lower", &hst.FSOverlay{Target: m("/etc"), Upper: m("/"), Work: m("/")}, false, nil, nil, nil, "<invalid>"},
		{"zero lower ro", &hst.FSOverlay{Target: m("/etc")}, false, nil, nil, nil, "<invalid>"},
		{"short lower", &hst.FSOverlay{Target: m("/etc"), Lower: ms("/etc")}, false, nil, nil, nil, "<invalid>"},

		{"full", &hst.FSOverlay{
			Target: m("/nix/store"),
			Lower:  ms("/mnt-root/nix/.ro-store"),
			Upper:  m("/mnt-root/nix/.rw-store/upper"),
			Work:   m("/mnt-root/nix/.rw-store/work"),
		}, true, container.Ops{&container.MountOverlayOp{
			Target: m("/nix/store"),
			Lower:  ms("/mnt-root/nix/.ro-store"),
			Upper:  m("/mnt-root/nix/.rw-store/upper"),
			Work:   m("/mnt-root/nix/.rw-store/work"),
		}}, m("/nix/store"), ms("/mnt-root/nix/.rw-store/upper", "/mnt-root/nix/.rw-store/work", "/mnt-root/nix/.ro-store"),
			"w*/nix/store:/mnt-root/nix/.rw-store/upper:/mnt-root/nix/.rw-store/work:/mnt-root/nix/.ro-store"},

		{"ro", &hst.FSOverlay{
			Target: m("/mnt/src"),
			Lower:  ms("/tmp/.src0", "/tmp/.src1"),
		}, true, container.Ops{&container.MountOverlayOp{
			Target: m("/mnt/src"),
			Lower:  ms("/tmp/.src0", "/tmp/.src1"),
		}}, m("/mnt/src"), ms("/tmp/.src0", "/tmp/.src1"),
			"*/mnt/src:/tmp/.src0:/tmp/.src1"},

		{"ro work", &hst.FSOverlay{
			Target: m("/mnt/src"),
			Lower:  ms("/tmp/.src0", "/tmp/.src1"),
			Work:   m("/tmp"),
		}, true, container.Ops{&container.MountOverlayOp{
			Target: m("/mnt/src"),
			Lower:  ms("/tmp/.src0", "/tmp/.src1"),
		}}, m("/mnt/src"), ms("/tmp/.src0", "/tmp/.src1"),
			"*/mnt/src:/tmp/.src0:/tmp/.src1"},
	})
}
