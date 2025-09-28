package container

import (
	"encoding/gob"
	"fmt"
)

func init() { gob.Register(new(AutoEtcOp)) }

// Etc appends an [Op] that expands host /etc into a toplevel symlink mirror with /etc semantics.
// This is not a generic setup op. It is implemented here to reduce ipc overhead.
func (f *Ops) Etc(host *Absolute, prefix string) *Ops {
	e := &AutoEtcOp{prefix}
	f.Mkdir(AbsFHSEtc, 0755)
	f.Bind(host, e.hostPath(), 0)
	*f = append(*f, e)
	return f
}

type AutoEtcOp struct{ Prefix string }

func (e *AutoEtcOp) Valid() bool                                { return e != nil }
func (e *AutoEtcOp) early(*setupState, syscallDispatcher) error { return nil }
func (e *AutoEtcOp) apply(state *setupState, k syscallDispatcher) error {
	if state.nonrepeatable&nrAutoEtc != 0 {
		return OpRepeatError("autoetc")
	}
	state.nonrepeatable |= nrAutoEtc

	const target = sysrootPath + FHSEtc
	rel := e.hostRel() + "/"

	if err := k.mkdirAll(target, 0755); err != nil {
		return err
	}
	if d, err := k.readdir(toSysroot(e.hostPath().String())); err != nil {
		return err
	} else {
		for _, ent := range d {
			n := ent.Name()
			switch n {
			case ".host", "passwd", "group":

			case "mtab":
				if err = k.symlink(FHSProc+"mounts", target+n); err != nil {
					return err
				}

			default:
				if err = k.symlink(rel+n, target+n); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (e *AutoEtcOp) hostPath() *Absolute { return AbsFHSEtc.Append(e.hostRel()) }
func (e *AutoEtcOp) hostRel() string     { return ".host/" + e.Prefix }

func (e *AutoEtcOp) Is(op Op) bool {
	ve, ok := op.(*AutoEtcOp)
	return ok && e.Valid() && ve.Valid() && *e == *ve
}
func (*AutoEtcOp) prefix() (string, bool) { return "setting up", true }
func (e *AutoEtcOp) String() string       { return fmt.Sprintf("auto etc %s", e.Prefix) }
