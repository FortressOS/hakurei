package container

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	. "syscall"
)

const (
	// intermediate root file name pattern for [MountOverlayOp.Upper];
	// remains after apply returns
	intermediatePatternOverlayUpper = "overlay.upper.*"
	// intermediate root file name pattern for [MountOverlayOp.Work];
	// remains after apply returns
	intermediatePatternOverlayWork = "overlay.work.*"
)

func init() { gob.Register(new(MountOverlayOp)) }

// Overlay appends an [Op] that mounts the overlay pseudo filesystem on [MountOverlayOp.Target].
func (f *Ops) Overlay(target, state, work *Absolute, layers ...*Absolute) *Ops {
	*f = append(*f, &MountOverlayOp{
		Target: target,
		Lower:  layers,
		Upper:  state,
		Work:   work,
	})
	return f
}

// OverlayEphemeral appends an [Op] that mounts the overlay pseudo filesystem on [MountOverlayOp.Target]
// with an ephemeral upperdir and workdir.
func (f *Ops) OverlayEphemeral(target *Absolute, layers ...*Absolute) *Ops {
	return f.Overlay(target, AbsFHSRoot, nil, layers...)
}

// OverlayReadonly appends an [Op] that mounts the overlay pseudo filesystem readonly on [MountOverlayOp.Target]
func (f *Ops) OverlayReadonly(target *Absolute, layers ...*Absolute) *Ops {
	return f.Overlay(target, nil, nil, layers...)
}

// MountOverlayOp mounts [FstypeOverlay] on container path Target.
type MountOverlayOp struct {
	Target *Absolute

	// Any filesystem, does not need to be on a writable filesystem.
	Lower []*Absolute
	// formatted for [OptionOverlayLowerdir], resolved, prefixed and escaped during early
	lower []string
	// The upperdir is normally on a writable filesystem.
	//
	// If Work is nil and Upper holds the special value [FHSRoot],
	// an ephemeral upperdir and workdir will be set up.
	//
	// If both Work and Upper are empty strings, upperdir and workdir is omitted and the overlay is mounted readonly.
	Upper *Absolute
	// formatted for [OptionOverlayUpperdir], resolved, prefixed and escaped during early
	upper string
	// The workdir needs to be an empty directory on the same filesystem as upperdir.
	Work *Absolute
	// formatted for [OptionOverlayWorkdir], resolved, prefixed and escaped during early
	work string

	ephemeral bool
}

func (o *MountOverlayOp) early(*setupState) error {
	if o.Work == nil && o.Upper != nil {
		switch o.Upper.String() {
		case FHSRoot: // ephemeral
			o.ephemeral = true // intermediate root not yet available

		default:
			return msg.WrapErr(EINVAL, fmt.Sprintf("upperdir has unexpected value %q", o.Upper))
		}
	}
	// readonly handled in apply

	if !o.ephemeral {
		if o.Upper != o.Work && (o.Upper == nil || o.Work == nil) {
			// unreachable
			return msg.WrapErr(ENOTRECOVERABLE, "impossible overlay state reached")
		}

		if o.Upper != nil {
			if v, err := filepath.EvalSymlinks(o.Upper.String()); err != nil {
				return wrapErrSelf(err)
			} else {
				o.upper = EscapeOverlayDataSegment(toHost(v))
			}
		}

		if o.Work != nil {
			if v, err := filepath.EvalSymlinks(o.Work.String()); err != nil {
				return wrapErrSelf(err)
			} else {
				o.work = EscapeOverlayDataSegment(toHost(v))
			}
		}
	}

	o.lower = make([]string, len(o.Lower))
	for i, a := range o.Lower {
		if a == nil {
			return EBADE
		}

		if v, err := filepath.EvalSymlinks(a.String()); err != nil {
			return wrapErrSelf(err)
		} else {
			o.lower[i] = EscapeOverlayDataSegment(toHost(v))
		}
	}
	return nil
}

func (o *MountOverlayOp) apply(state *setupState) error {
	if o.Target == nil {
		return EBADE
	}
	target := toSysroot(o.Target.String())
	if err := os.MkdirAll(target, state.ParentPerm); err != nil {
		return wrapErrSelf(err)
	}

	if o.ephemeral {
		var err error
		// these directories are created internally, therefore early (absolute, symlink, prefix, escape) is bypassed
		if o.upper, err = os.MkdirTemp(FHSRoot, intermediatePatternOverlayUpper); err != nil {
			return wrapErrSelf(err)
		}
		if o.work, err = os.MkdirTemp(FHSRoot, intermediatePatternOverlayWork); err != nil {
			return wrapErrSelf(err)
		}
	}

	options := make([]string, 0, 4)

	if o.upper == zeroString && o.work == zeroString { // readonly
		if len(o.Lower) < 2 {
			return msg.WrapErr(EINVAL, "readonly overlay requires at least two lowerdir")
		}
		// "upperdir=" and "workdir=" may be omitted. In that case the overlay will be read-only
	} else {
		if len(o.Lower) == 0 {
			return msg.WrapErr(EINVAL, "overlay requires at least one lowerdir")
		}
		options = append(options,
			OptionOverlayUpperdir+"="+o.upper,
			OptionOverlayWorkdir+"="+o.work)
	}
	options = append(options,
		OptionOverlayLowerdir+"="+strings.Join(o.lower, SpecialOverlayPath),
		OptionOverlayUserxattr)

	return wrapErrSuffix(Mount(SourceOverlay, target, FstypeOverlay, 0, strings.Join(options, SpecialOverlayOption)),
		fmt.Sprintf("cannot mount overlay on %q:", o.Target))
}

func (o *MountOverlayOp) Is(op Op) bool {
	vo, ok := op.(*MountOverlayOp)
	return ok &&
		o.Target == vo.Target &&
		slices.Equal(o.Lower, vo.Lower) &&
		o.Upper == vo.Upper &&
		o.Work == vo.Work
}
func (*MountOverlayOp) prefix() string { return "mounting" }
func (o *MountOverlayOp) String() string {
	return fmt.Sprintf("overlay on %q with %d layers", o.Target, len(o.Lower))
}
