package container

import "testing"

func TestAutoRootOp(t *testing.T) {
	checkOpsValid(t, []opValidTestCase{
		{"nil", (*AutoRootOp)(nil), false},
		{"zero", new(AutoRootOp), false},
		{"valid", &AutoRootOp{Host: MustAbs("/")}, true},
	})

	checkOpsBuilder(t, []opsBuilderTestCase{
		{"pd", new(Ops).Root(MustAbs("/"), "048090b6ed8f9ebb10e275ff5d8c0659", BindWritable), Ops{
			&AutoRootOp{
				Host:   MustAbs("/"),
				Prefix: "048090b6ed8f9ebb10e275ff5d8c0659",
				Flags:  BindWritable,
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(AutoRootOp), new(AutoRootOp), false},

		{"internal ne", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:     MustAbs("/"),
			Prefix:   ":3",
			Flags:    BindWritable,
			resolved: []Op{new(BindMountOp)},
		}, true},

		{"prefix differs", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: "\x00",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, false},

		{"flags differs", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable | BindDevice,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, false},

		{"host differs", &AutoRootOp{
			Host:   MustAbs("/tmp/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, false},

		{"equals", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"root", &AutoRootOp{
			Host:   MustAbs("/"),
			Prefix: ":3",
			Flags:  BindWritable,
		}, "setting up", `auto root "/" prefix :3 flags 0x2`},
	})
}

func TestIsAutoRootBindable(t *testing.T) {
	testCases := []struct {
		name string
		want bool
	}{
		{"proc", false},
		{"dev", false},
		{"tmp", false},
		{"mnt", false},
		{"etc", false},
		{"", false},

		{"var", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAutoRootBindable(tc.name); got != tc.want {
				t.Errorf("IsAutoRootBindable: %v, want %v", got, tc.want)
			}
		})
	}
}
