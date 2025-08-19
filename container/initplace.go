package container

import (
	"encoding/gob"
	"fmt"
	"os"
	"slices"
	. "syscall"
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

func (t *TmpfileOp) early(*setupState) error { return nil }
func (t *TmpfileOp) apply(state *setupState) error {
	if t.Path == nil {
		return EBADE
	}

	var tmpPath string
	if f, err := os.CreateTemp(FHSRoot, intermediatePatternTmpfile); err != nil {
		return wrapErrSelf(err)
	} else if _, err = f.Write(t.Data); err != nil {
		return wrapErrSuffix(err,
			"cannot write to intermediate file:")
	} else if err = f.Close(); err != nil {
		return wrapErrSuffix(err,
			"cannot close intermediate file:")
	} else {
		tmpPath = f.Name()
	}

	target := toSysroot(t.Path.String())
	if err := ensureFile(target, 0444, state.ParentPerm); err != nil {
		return err
	} else if err = hostProc.bindMount(
		tmpPath,
		target,
		MS_RDONLY|MS_NODEV,
		false,
	); err != nil {
		return err
	} else if err = os.Remove(tmpPath); err != nil {
		return wrapErrSelf(err)
	}
	return nil
}

func (t *TmpfileOp) Is(op Op) bool {
	vt, ok := op.(*TmpfileOp)
	return ok && t.Path == vt.Path && slices.Equal(t.Data, vt.Data)
}
func (*TmpfileOp) prefix() string { return "placing" }
func (t *TmpfileOp) String() string {
	return fmt.Sprintf("tmpfile %q (%d bytes)", t.Path, len(t.Data))
}
