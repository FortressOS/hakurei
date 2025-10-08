package app

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"hakurei.app/container/check"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/system/acl"
)

var absX11SocketDir = fhs.AbsTmp.Append(".X11-unix")

func init() { gob.Register(new(spX11Op)) }

// spX11Op exports the X11 display server to the container.
type spX11Op struct {
	// Value of $DISPLAY, stored during toSystem
	Display string
}

func (s *spX11Op) toSystem(state *outcomeStateSys, _ *hst.Config) error {
	if d, ok := state.k.lookupEnv("DISPLAY"); !ok {
		return newWithMessage("DISPLAY is not set")
	} else {
		s.Display = d
	}

	// the socket file at `/tmp/.X11-unix/X%d` is typically owned by the priv user
	// and not accessible by the target user
	var socketPath *check.Absolute
	if len(s.Display) > 1 && s.Display[0] == ':' { // `:%d`
		if n, err := strconv.Atoi(s.Display[1:]); err == nil && n >= 0 {
			socketPath = absX11SocketDir.Append("X" + strconv.Itoa(n))
		}
	} else if len(s.Display) > 5 && strings.HasPrefix(s.Display, "unix:") { // `unix:%s`
		if a, err := check.NewAbs(s.Display[5:]); err == nil {
			socketPath = a
		}
	}
	if socketPath != nil {
		if _, err := state.k.stat(socketPath.String()); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return &hst.AppError{Step: fmt.Sprintf("access X11 socket %q", socketPath), Err: err}
			}
		} else {
			state.sys.UpdatePermType(hst.EX11, socketPath, acl.Read, acl.Write, acl.Execute)
			if !state.Container.HostAbstract {
				s.Display = "unix:" + socketPath.String()
			}
		}
	}

	state.sys.ChangeHosts("#" + state.uid.String())
	return nil
}

func (s *spX11Op) toContainer(state *outcomeStateParams) error {
	state.env["DISPLAY"] = s.Display
	state.params.Bind(absX11SocketDir, absX11SocketDir, 0)
	return nil
}
