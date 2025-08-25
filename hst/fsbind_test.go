package hst_test

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/hst"
)

func TestFSBind(t *testing.T) {
	checkFs(t, []fsTestCase{
		{"nil", (*hst.FSBind)(nil), false, nil, nil, nil, "<invalid>"},

		{"full", &hst.FSBind{
			Target:   m("/dev"),
			Source:   m("/mnt/dev"),
			Optional: true,
			Device:   true,
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/dev"),
			Target: m("/dev"),
			Flags:  container.BindWritable | container.BindDevice | container.BindOptional,
		}}, m("/dev"), ms("/mnt/dev"),
			"d+/mnt/dev:/dev"},

		{"full write dev", &hst.FSBind{
			Target: m("/dev"),
			Source: m("/mnt/dev"),
			Write:  true,
			Device: true,
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/dev"),
			Target: m("/dev"),
			Flags:  container.BindWritable | container.BindDevice,
		}}, m("/dev"), ms("/mnt/dev"),
			"d*/mnt/dev:/dev"},

		{"full write", &hst.FSBind{
			Target: m("/tmp"),
			Source: m("/mnt/tmp"),
			Write:  true,
		}, true, container.Ops{&container.BindMountOp{
			Source: m("/mnt/tmp"),
			Target: m("/tmp"),
			Flags:  container.BindWritable,
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

		{"autoroot nil target", &hst.FSBind{
			Source:   m("/"),
			AutoRoot: true,
		}, false, nil, nil, nil, "<invalid>"},

		{"autoroot bad target", &hst.FSBind{
			Source:   m("/"),
			Target:   m("/etc/"),
			AutoRoot: true,
		}, false, nil, nil, nil, "<invalid>"},

		{"autoroot pd", &hst.FSBind{
			Target:   m("/"),
			Source:   m("/"),
			Write:    true,
			AutoRoot: true,
		}, true, container.Ops{&container.AutoRootOp{
			Host:  m("/"),
			Flags: container.BindWritable,
		}}, m("/"), ms("/"), "autoroot:w"},

		{"autoroot silly", &hst.FSBind{
			Target:   m("/"),
			Source:   m("/etc"),
			Write:    true,
			AutoRoot: true,
		}, true, container.Ops{&container.AutoRootOp{
			Host:  m("/etc"),
			Flags: container.BindWritable,
		}}, m("/"), ms("/etc"), "autoroot:w:/etc"},
	})
}
