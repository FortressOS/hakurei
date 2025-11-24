package pipewire_test

import (
	_ "embed"
)

var (
	//go:embed testdata/00-sendmsg00-message00-header
	sendmsg00Message00Header string
	//go:embed testdata/01-sendmsg00-message00-POD
	sendmsg00Message00POD string

	//go:embed testdata/02-sendmsg00-message01-header
	sendmsg00Message01Header string
	//go:embed testdata/03-sendmsg00-message01-POD
	sendmsg00Message01POD string
)
