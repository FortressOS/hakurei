package hst_test

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/bits"
	"hakurei.app/hst"
)

func TestFSBind(t *testing.T) {
	checkFs(t, []fsTestCase{
		{"nil", (*hst.FSBind)(nil), false, nil, nil, nil, "<invalid>"},
		{"ensure optional", &hst.FSBind{Source: m("/"), Ensure: true, Optional: true},
			false, nil, nil, nil, "<invalid>"},

		{"full", &hst.FSBind{
			Target:   m("/dev"),
			Source:   m("/mnt/dev"),
			Optional: true,
			Device:   true,
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/dev"),
			Target: m("/dev"),
			Flags:  bits.BindWritable | bits.BindDevice | bits.BindOptional,
		}}, m("/dev"), ms("/mnt/dev"),
			"d+/mnt/dev:/dev"},

		{"full ensure", &hst.FSBind{
			Target: m("/dev"),
			Source: m("/mnt/dev"),
			Ensure: true,
			Device: true,
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/dev"),
			Target: m("/dev"),
			Flags:  bits.BindWritable | bits.BindDevice | bits.BindEnsure,
		}}, m("/dev"), ms("/mnt/dev"),
			"d-/mnt/dev:/dev"},

		{"full write dev", &hst.FSBind{
			Target: m("/dev"),
			Source: m("/mnt/dev"),
			Write:  true,
			Device: true,
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/dev"),
			Target: m("/dev"),
			Flags:  bits.BindWritable | bits.BindDevice,
		}}, m("/dev"), ms("/mnt/dev"),
			"d*/mnt/dev:/dev"},

		{"full write", &hst.FSBind{
			Target: m("/tmp"),
			Source: m("/mnt/tmp"),
			Write:  true,
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/tmp"),
			Target: m("/tmp"),
			Flags:  bits.BindWritable,
		}}, m("/tmp"), ms("/mnt/tmp"),
			"w*/mnt/tmp:/tmp"},

		{"full no flags", &hst.FSBind{
			Target: m("/etc"),
			Source: m("/mnt/etc"),
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/etc"),
			Target: m("/etc"),
		}}, m("/etc"), ms("/mnt/etc"),
			"*/mnt/etc:/etc"},

		{"nil dst", &hst.FSBind{
			Source: m("/"),
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/"),
			Target: m("/"),
		}}, m("/"), ms("/"),
			"*/"},

		{"special nil target", &hst.FSBind{
			Source:  m("/"),
			Special: true,
		}, false, nil, nil, nil, "<invalid>"},

		{"special bad target", &hst.FSBind{
			Source:  m("/"),
			Target:  m("/var/"),
			Special: true,
		}, false, nil, nil, nil, "<invalid>"},

		{"autoroot pd", &hst.FSBind{
			Target:  m("/"),
			Source:  m("/"),
			Write:   true,
			Special: true,
		}, true, container.Ops{&container.AutoRootOp{
			Host:  m("/"),
			Flags: bits.BindWritable,
		}}, m("/"), ms("/"), "autoroot:w"},

		{"autoroot silly", &hst.FSBind{
			Target:  m("/"),
			Source:  m("/etc"),
			Write:   true,
			Special: true,
		}, true, container.Ops{&container.AutoRootOp{
			Host:  m("/etc"),
			Flags: bits.BindWritable,
		}}, m("/"), ms("/etc"), "autoroot:w:/etc"},

		{"autoetc", &hst.FSBind{
			Target:  m("/etc/"),
			Source:  m("/etc/"),
			Special: true,
		}, true, container.Ops{
			&container.MkdirOp{Path: m("/etc/"), Perm: 0755},
			&container.BindMountOp{Source: m("/etc/"), Target: m("/etc/.host/:3")},
			&container.AutoEtcOp{Prefix: ":3"},
		}, m("/etc/"), ms("/etc/"), "autoetc:/etc/"},
	})
}
