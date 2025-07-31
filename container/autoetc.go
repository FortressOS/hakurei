package container

import (
	"encoding/gob"
	"fmt"
	"os"
)

func init() { gob.Register(new(AutoEtcOp)) }

// Etc appends an [Op] that expands host /etc into a toplevel symlink mirror with /etc semantics.
// This is not a generic setup op. It is implemented here to reduce ipc overhead.
func (f *Ops) Etc(host, prefix string) *Ops {
	e := &AutoEtcOp{prefix}
	f.Mkdir("/etc", 0755)
	f.Bind(host, e.hostPath(), 0)
	*f = append(*f, e)
	return f
}

type AutoEtcOp struct{ Prefix string }

func (e *AutoEtcOp) early(*Params) error { return nil }
func (e *AutoEtcOp) apply(*Params) error {
	const target = sysrootPath + "/etc/"
	rel := e.hostRel() + "/"

	if err := os.MkdirAll(target, 0755); err != nil {
		return wrapErrSelf(err)
	}
	if d, err := os.ReadDir(toSysroot(e.hostPath())); err != nil {
		return wrapErrSelf(err)
	} else {
		for _, ent := range d {
			n := ent.Name()
			switch n {
			case ".host":

			case "passwd":
			case "group":

			case "mtab":
				if err = os.Symlink("/proc/mounts", target+n); err != nil {
					return wrapErrSelf(err)
				}

			default:
				if err = os.Symlink(rel+n, target+n); err != nil {
					return wrapErrSelf(err)
				}
			}
		}
	}

	return nil
}
func (e *AutoEtcOp) hostPath() string { return "/etc/" + e.hostRel() }
func (e *AutoEtcOp) hostRel() string  { return ".host/" + e.Prefix }

func (e *AutoEtcOp) Is(op Op) bool {
	ve, ok := op.(*AutoEtcOp)
	return ok && ((e == nil && ve == nil) || (e != nil && ve != nil && *e == *ve))
}
func (*AutoEtcOp) prefix() string   { return "setting up" }
func (e *AutoEtcOp) String() string { return fmt.Sprintf("auto etc %s", e.Prefix) }
