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

	//go:embed testdata/08-recvmsg00-message00-header
	recvmsg00Message00Header string
	//go:embed testdata/09-recvmsg00-message00-POD
	recvmsg00Message00POD string
	//go:embed testdata/10-recvmsg00-message00-footer
	recvmsg00Message00Footer string

	//go:embed testdata/11-recvmsg00-message01-header
	recvmsg00Message01Header string
	//go:embed testdata/12-recvmsg00-message01-POD
	recvmsg00Message01POD string

	//go:embed testdata/13-recvmsg00-message02-header
	recvmsg00Message02Header string
	//go:embed testdata/14-recvmsg00-message02-POD
	recvmsg00Message02POD string

	//go:embed testdata/15-recvmsg00-message03-header
	recvmsg00Message03Header string
	//go:embed testdata/16-recvmsg00-message03-POD
	recvmsg00Message03POD string

	//go:embed testdata/17-recvmsg00-message04-header
	recvmsg00Message04Header string
	//go:embed testdata/18-recvmsg00-message04-POD
	recvmsg00Message04POD string

	//go:embed testdata/19-recvmsg00-message05-header
	recvmsg00Message05Header string
	//go:embed testdata/20-recvmsg00-message05-POD
	recvmsg00Message05POD string
	//go:embed testdata/21-recvmsg00-message05-footer
	recvmsg00Message05Footer string
)
