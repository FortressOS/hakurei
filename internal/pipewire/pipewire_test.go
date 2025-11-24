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

	//go:embed testdata/04-sendmsg00-message02-header
	sendmsg00Message02Header string
	//go:embed testdata/05-sendmsg00-message02-POD
	sendmsg00Message02POD string

	//go:embed testdata/06-sendmsg00-message03-header
	sendmsg00Message03Header string
	//go:embed testdata/07-sendmsg00-message03-POD
	sendmsg00Message03POD string
)
