package container

import (
	"encoding/gob"
	"fmt"
	"syscall"
)

const (
	// intermediate root file name pattern for [TmpfileOp]
	intermediatePatternTmpfile = "tmp.*"
)

func init() { gob.Register(new(TmpfileOp)) }

// Place appends an [Op] that places a file in container path [TmpfileOp.Path] containing [TmpfileOp.Data].
func (f *Ops) Place(name *Absolute, data []byte) *Ops {
	*f = append(*f, &TmpfileOp{name, data})
	return f
}

// PlaceP is like Place but writes the address of [TmpfileOp.Data] to the pointer dataP points to.
func (f *Ops) PlaceP(name *Absolute, dataP **[]byte) *Ops {
	t := &TmpfileOp{Path: name}
	*dataP = &t.Data

	*f = append(*f, t)
	return f
}

// TmpfileOp places a file on container Path containing Data.
type TmpfileOp struct {
	Path *Absolute
	Data []byte
}

func (t *TmpfileOp) Valid() bool                                { return t != nil && t.Path != nil }
func (t *TmpfileOp) early(*setupState, syscallDispatcher) error { return nil }
func (t *TmpfileOp) apply(state *setupState, k syscallDispatcher) error {
	var tmpPath string
	if f, err := k.createTemp(FHSRoot, intermediatePatternTmpfile); err != nil {
		return err
	} else if _, err = f.Write(t.Data); err != nil {
		return err
	} else if err = f.Close(); err != nil {
		return err
	} else {
		tmpPath = f.Name()
	}

	target := toSysroot(t.Path.String())
	if err := k.ensureFile(target, 0444, state.ParentPerm); err != nil {
		return err
	} else if err = k.bindMount(
		tmpPath,
		target,
		syscall.MS_RDONLY|syscall.MS_NODEV,
	); err != nil {
		return err
	} else if err = k.remove(tmpPath); err != nil {
		return err
	}
	return nil
}

func (t *TmpfileOp) Is(op Op) bool {
	vt, ok := op.(*TmpfileOp)
	return ok && t.Valid() && vt.Valid() &&
		t.Path.Is(vt.Path) &&
		string(t.Data) == string(vt.Data)
}
func (*TmpfileOp) prefix() (string, bool) { return "placing", true }
func (t *TmpfileOp) String() string {
	return fmt.Sprintf("tmpfile %q (%d bytes)", t.Path, len(t.Data))
}
