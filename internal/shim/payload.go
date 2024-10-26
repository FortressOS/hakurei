package shim

import (
	"encoding/gob"
	"errors"
	"net"

	"git.ophivana.moe/security/fortify/helper/bwrap"
	"git.ophivana.moe/security/fortify/internal/fmsg"
)

const EnvShim = "FORTIFY_SHIM"

type Payload struct {
	// child full argv
	Argv []string
	// fortify, bwrap, target full exec path
	Exec [3]string
	// bwrap config
	Bwrap *bwrap.Config
	// whether to pass wayland fd
	WL bool

	// verbosity pass through
	Verbose bool
}

func (p *Payload) serve(conn *net.UnixConn, wl *Wayland) error {
	if err := gob.NewEncoder(conn).Encode(*p); err != nil {
		return fmsg.WrapErrorSuffix(err,
			"cannot stream shim payload:")
	}

	if wl != nil {
		if err := wl.WriteUnix(conn); err != nil {
			return errors.Join(err, conn.Close())
		}
	}

	return fmsg.WrapErrorSuffix(conn.Close(),
		"cannot close setup connection:")
}
