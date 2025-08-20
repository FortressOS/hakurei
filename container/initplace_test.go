package container

import "testing"

func TestTmpfileOp(t *testing.T) {
	checkOpsBuilder(t, []opsBuilderTestCase{
		{"noref", new(Ops).Place(MustAbs("/etc/passwd"), []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`)), Ops{
			&TmpfileOp{
				Path: MustAbs("/etc/passwd"),
				Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`),
			},
		}},

		{"ref", new(Ops).PlaceP(MustAbs("/etc/passwd"), new(*[]byte)), Ops{
			&TmpfileOp{
				Path: MustAbs("/etc/passwd"),
				Data: []byte{},
			},
		}},
	})

	checkOpIs(t, []opIsTestCase{
		{"zero", new(TmpfileOp), new(TmpfileOp), false},

		{"differs path", &TmpfileOp{
			Path: MustAbs("/etc/group"),
			Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`),
		}, &TmpfileOp{
			Path: MustAbs("/etc/passwd"),
			Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`),
		}, false},

		{"differs data", &TmpfileOp{
			Path: MustAbs("/etc/passwd"),
			Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh` + "\x00"),
		}, &TmpfileOp{
			Path: MustAbs("/etc/passwd"),
			Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`),
		}, false},

		{"equals", &TmpfileOp{
			Path: MustAbs("/etc/passwd"),
			Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`),
		}, &TmpfileOp{
			Path: MustAbs("/etc/passwd"),
			Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`),
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"passwd", &TmpfileOp{
			Path: MustAbs("/etc/passwd"),
			Data: []byte(`chronos:x:65534:65534:Hakurei:/var/empty:/bin/zsh`),
		}, "placing", `tmpfile "/etc/passwd" (49 bytes)`},
	})
}
