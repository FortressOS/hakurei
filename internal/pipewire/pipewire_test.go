package pipewire_test

import (
	_ "embed"
)

var (
	//go:embed testdata/c0s0p0
	c0s0header string
	//go:embed testdata/c0s0p1
	c0s0pod string

	//go:embed testdata/c0s1p0
	c0s1header string
	//go:embed testdata/c0s1p1
	c0s1pod string

	//go:embed testdata/c0s2p0
	c0s2header string
	//go:embed testdata/c0s2p1
	c0s2pod string

	//go:embed testdata/c0s3p0
	c0s3header string
	//go:embed testdata/c0s3p1
	c0s3pod string

	//go:embed testdata/c1r0p0
	c1r0header string
	//go:embed testdata/c1r0p1
	c1r0pod string
	//go:embed testdata/c1r0p2
	c1r0footer string

	//go:embed testdata/c1r1p0
	c1r1header string
	//go:embed testdata/c1r1p1
	c1r1pod string

	//go:embed testdata/c1r2p0
	c1r2header string
	//go:embed testdata/c1r2p1
	c1r2pod string

	//go:embed testdata/c1r3p0
	c1r3header string
	//go:embed testdata/c1r3p1
	c1r3pod string

	//go:embed testdata/c1r4p0
	c1r4header string
	//go:embed testdata/c1r4p1
	c1r4pod string

	//go:embed testdata/c1r5p0
	c1r5header string
	//go:embed testdata/c1r5p1
	c1r5pod string
	//go:embed testdata/c1r5p2
	c1r5footer string
)
