package container

import "testing"

func TestBindMountOp(t *testing.T) {
	checkOpsBuilder(t, []opsBuilderTestCase{
		{"autoetc", new(Ops).Bind(
			MustAbs("/etc/"),
			MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			0,
		), Ops{
			&BindMountOp{
				Source: MustAbs("/etc/"),
				Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(BindMountOp), new(BindMountOp), false},

		{"internal ne", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source:      MustAbs("/etc/"),
			Target:      MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			sourceFinal: MustAbs("/etc/"),
		}, true},

		{"flags differs", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
			Flags:  BindOptional,
		}, false},

		{"source differs", &BindMountOp{
			Source: MustAbs("/.hakurei/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, false},

		{"target differs", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/"),
		}, false},

		{"equals", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"invalid", new(BindMountOp), "mounting", "<invalid>"},

		{"autoetc", &BindMountOp{
			Source: MustAbs("/etc/"),
			Target: MustAbs("/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659"),
		}, "mounting", `"/etc/" on "/etc/.host/048090b6ed8f9ebb10e275ff5d8c0659" flags 0x0`},

		{"hostdev", &BindMountOp{
			Source: MustAbs("/dev/"),
			Target: MustAbs("/dev/"),
			Flags:  BindWritable | BindDevice,
		}, "mounting", `"/dev/" flags 0x6`},
	})
}
