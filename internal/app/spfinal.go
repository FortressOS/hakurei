package app

import (
	"encoding/gob"
	"fmt"
	"slices"
	"strings"
	"syscall"

	"hakurei.app/container/fhs"
	"hakurei.app/hst"
	"hakurei.app/system"
	"hakurei.app/system/acl"
)

func init() { gob.Register(spFinal{}) }

// spFinal is a transitional op destined for removal after #3, #8, #9 has been resolved.
// It exists to avoid reordering the expected entries in test cases.
type spFinal struct{}

func (s spFinal) toSystem(state *outcomeStateSys) error {
	// append ExtraPerms last
	for _, p := range state.config.ExtraPerms {
		if p == nil || p.Path == nil {
			continue
		}

		if p.Ensure {
			state.sys.Ensure(p.Path, 0700)
		}

		perms := make(acl.Perms, 0, 3)
		if p.Read {
			perms = append(perms, acl.Read)
		}
		if p.Write {
			perms = append(perms, acl.Write)
		}
		if p.Execute {
			perms = append(perms, acl.Execute)
		}
		state.sys.UpdatePermType(system.User, p.Path, perms...)
	}
	return nil
}

func (s spFinal) toContainer(state *outcomeStateParams) error {
	// TODO(ophestra): move this to spFilesystemOp after #8 and #9

	// mount root read-only as the final setup Op
	state.params.Remount(fhs.AbsRoot, syscall.MS_RDONLY)

	state.params.Env = make([]string, 0, len(state.env))
	for key, value := range state.env {
		if strings.IndexByte(key, '=') != -1 {
			return &hst.AppError{Step: "flatten environment", Err: syscall.EINVAL,
				Msg: fmt.Sprintf("invalid environment variable %s", key)}
		}
		state.params.Env = append(state.params.Env, key+"="+value)
	}
	// range over map has randomised order
	slices.Sort(state.params.Env)

	return nil
}
