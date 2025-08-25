package hst_test

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/hst"
)

func TestFSLink(t *testing.T) {
	checkFs(t, []fsTestCase{
		{"nil", (*hst.FSLink)(nil), false, nil, nil, nil, "<invalid>"},
		{"zero", new(hst.FSLink), false, nil, nil, nil, "<invalid>"},

		{"deref rel", &hst.FSLink{Target: m("/"), Linkname: ":3", Dereference: true}, false, nil, nil, nil, "<invalid>"},
		{"deref", &hst.FSLink{
			Target:      m("/run/current-system"),
			Linkname:    "/run/current-system",
			Dereference: true,
		}, true, container.Ops{
			&container.SymlinkOp{
				Target:      m("/run/current-system"),
				LinkName:    "/run/current-system",
				Dereference: true,
			},
		}, m("/run/current-system"), nil,
			"&/run/current-system:*/run/current-system"},

		{"direct", &hst.FSLink{
			Target:   m("/etc/mtab"),
			Linkname: "/proc/mounts",
		}, true, container.Ops{
			&container.SymlinkOp{
				Target:   m("/etc/mtab"),
				LinkName: "/proc/mounts",
			},
		}, m("/etc/mtab"), nil, "&/etc/mtab:/proc/mounts"},

		{"direct rel", &hst.FSLink{
			Target:   m("/etc/mtab"),
			Linkname: "../proc/mounts",
		}, true, container.Ops{
			&container.SymlinkOp{
				Target:   m("/etc/mtab"),
				LinkName: "../proc/mounts",
			},
		}, m("/etc/mtab"), nil, "&/etc/mtab:../proc/mounts"},
	})
}
