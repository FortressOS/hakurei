package hst_test

import (
	"testing"

	"hakurei.app/container"
	"hakurei.app/hst"
)

func TestFSBind(t *testing.T) {
	checkFs(t, "bind", []fsTestCase{
		{"nil", (*hst.FSBind)(nil), nil, nil, nil, "<invalid>"},

		{"full", &hst.FSBind{
			Dst:      m("/dev"),
			Src:      m("/mnt/dev"),
			Optional: true,
			Device:   true,
		}, container.Ops{&container.BindMountOp{
			Source: m("/mnt/dev"),
			Target: m("/dev"),
			Flags:  container.BindWritable | container.BindDevice | container.BindOptional,
		}}, m("/dev"), ms("/mnt/dev"),
			"d+/mnt/dev:/dev"},

		{"full write dev", &hst.FSBind{
			Dst:    m("/dev"),
			Src:    m("/mnt/dev"),
			Write:  true,
			Device: true,
		}, container.Ops{&container.BindMountOp{
			Source: m("/mnt/dev"),
			Target: m("/dev"),
			Flags:  container.BindWritable | container.BindDevice,
		}}, m("/dev"), ms("/mnt/dev"),
			"d*/mnt/dev:/dev"},

		{"full write", &hst.FSBind{
			Dst:   m("/tmp"),
			Src:   m("/mnt/tmp"),
			Write: true,
		}, container.Ops{&container.BindMountOp{
			Source: m("/mnt/tmp"),
			Target: m("/tmp"),
			Flags:  container.BindWritable,
		}}, m("/tmp"), ms("/mnt/tmp"),
			"w*/mnt/tmp:/tmp"},

		{"full no flags", &hst.FSBind{
			Dst: m("/etc"),
			Src: m("/mnt/etc"),
		}, container.Ops{&container.BindMountOp{
			Source: m("/mnt/etc"),
			Target: m("/etc"),
		}}, m("/etc"), ms("/mnt/etc"),
			"*/mnt/etc:/etc"},

		{"nil dst", &hst.FSBind{
			Src: m("/"),
		}, container.Ops{&container.BindMountOp{
			Source: m("/"),
			Target: m("/"),
		}}, m("/"), ms("/"),
			"*/"},
	})
}
