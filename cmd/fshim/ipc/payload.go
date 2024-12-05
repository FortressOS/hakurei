package shim0

import (
	"encoding/gob"
	"net"

	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

const Env = "FORTIFY_SHIM"

type Payload struct {
	// child full argv
	Argv []string
	// bwrap, target full exec path
	Exec [2]string
	// bwrap config
	Bwrap *bwrap.Config
	// sync fd
	Sync *uintptr

	// verbosity pass through
	Verbose bool
}

func (p *Payload) Serve(conn *net.UnixConn) error {
	if err := gob.NewEncoder(conn).Encode(*p); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot stream shim payload:")
	}

	return fmsg.WrapErrorSuffix(conn.Close(),
		"cannot close setup connection:")
}
