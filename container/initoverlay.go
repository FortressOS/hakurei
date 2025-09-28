package container

import (
	"encoding/gob"
	"fmt"
	"slices"
	"strings"
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

const (
	// OverlayEphemeralUnexpectedUpper is set when [MountOverlayOp.Work] is nil
	// and [MountOverlayOp.Upper] holds an unexpected value.
	OverlayEphemeralUnexpectedUpper = iota
	// OverlayReadonlyLower is set when [MountOverlayOp.Lower] contains less than
	// two entries when mounting readonly.
	OverlayReadonlyLower
	// OverlayEmptyLower is set when [MountOverlayOp.Lower] has length of zero.
	OverlayEmptyLower
)

// OverlayArgumentError is returned for [MountOverlayOp] supplied with invalid argument.
type OverlayArgumentError struct {
	Type  uintptr
	Value string
}

func (e *OverlayArgumentError) Error() string {
	switch e.Type {
	case OverlayEphemeralUnexpectedUpper:
		return fmt.Sprintf("upperdir has unexpected value %q", e.Value)

	case OverlayReadonlyLower:
		return "readonly overlay requires at least two lowerdir"

	case OverlayEmptyLower:
		return "overlay requires at least one lowerdir"

	default:
		return fmt.Sprintf("invalid overlay argument error %#x", e.Type)
	}
}

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
	// If Work is nil and Upper holds the special value [AbsFHSRoot],
	// an ephemeral upperdir and workdir will be set up.
	//
	// If both Work and Upper are nil, upperdir and workdir is omitted and the overlay is mounted readonly.
	Upper *Absolute
	// formatted for [OptionOverlayUpperdir], resolved, prefixed and escaped during early
	upper string
	// The workdir needs to be an empty directory on the same filesystem as upperdir.
	Work *Absolute
	// formatted for [OptionOverlayWorkdir], resolved, prefixed and escaped during early
	work string

	ephemeral bool

	// used internally for mounting to the intermediate root
	noPrefix bool
}

func (o *MountOverlayOp) Valid() bool {
	if o == nil {
		return false
	}
	if o.Work != nil && o.Upper == nil {
		return false
	}
	if slices.Contains(o.Lower, nil) {
		return false
	}
	return o.Target != nil
}

func (o *MountOverlayOp) early(_ *setupState, k syscallDispatcher) error {
	if o.Work == nil && o.Upper != nil {
		switch o.Upper.String() {
		case FHSRoot: // ephemeral
			o.ephemeral = true // intermediate root not yet available

		default:
			return &OverlayArgumentError{OverlayEphemeralUnexpectedUpper, o.Upper.String()}
		}
	}
	// readonly handled in apply

	if !o.ephemeral {
		if o.Upper != o.Work && (o.Upper == nil || o.Work == nil) {
			// unreachable
			return OpStateError("overlay")
		}

		if o.Upper != nil {
			if v, err := k.evalSymlinks(o.Upper.String()); err != nil {
				return err
			} else {
				o.upper = EscapeOverlayDataSegment(toHost(v))
			}
		}

		if o.Work != nil {
			if v, err := k.evalSymlinks(o.Work.String()); err != nil {
				return err
			} else {
				o.work = EscapeOverlayDataSegment(toHost(v))
			}
		}
	}

	o.lower = make([]string, len(o.Lower))
	for i, a := range o.Lower { // nil checked in Valid
		if v, err := k.evalSymlinks(a.String()); err != nil {
			return err
		} else {
			o.lower[i] = EscapeOverlayDataSegment(toHost(v))
		}
	}
	return nil
}

func (o *MountOverlayOp) apply(state *setupState, k syscallDispatcher) error {
	target := o.Target.String()
	if !o.noPrefix {
		target = toSysroot(target)
	}
	if err := k.mkdirAll(target, state.ParentPerm); err != nil {
		return err
	}

	if o.ephemeral {
		var err error
		// these directories are created internally, therefore early (absolute, symlink, prefix, escape) is bypassed
		if o.upper, err = k.mkdirTemp(FHSRoot, intermediatePatternOverlayUpper); err != nil {
			return err
		}
		if o.work, err = k.mkdirTemp(FHSRoot, intermediatePatternOverlayWork); err != nil {
			return err
		}
	}

	options := make([]string, 0, 4)

	if o.upper == zeroString && o.work == zeroString { // readonly
		if len(o.Lower) < 2 {
			return &OverlayArgumentError{OverlayReadonlyLower, zeroString}
		}
		// "upperdir=" and "workdir=" may be omitted. In that case the overlay will be read-only
	} else {
		if len(o.Lower) == 0 {
			return &OverlayArgumentError{OverlayEmptyLower, zeroString}
		}
		options = append(options,
			OptionOverlayUpperdir+"="+o.upper,
			OptionOverlayWorkdir+"="+o.work)
	}
	options = append(options,
		OptionOverlayLowerdir+"="+strings.Join(o.lower, SpecialOverlayPath),
		OptionOverlayUserxattr)

	return k.mount(SourceOverlay, target, FstypeOverlay, 0, strings.Join(options, SpecialOverlayOption))
}

func (o *MountOverlayOp) Is(op Op) bool {
	vo, ok := op.(*MountOverlayOp)
	return ok && o.Valid() && vo.Valid() &&
		o.Target.Is(vo.Target) &&
		slices.EqualFunc(o.Lower, vo.Lower, func(a *Absolute, v *Absolute) bool { return a.Is(v) }) &&
		o.Upper.Is(vo.Upper) && o.Work.Is(vo.Work)
}
func (*MountOverlayOp) prefix() (string, bool) { return "mounting", true }
func (o *MountOverlayOp) String() string {
	return fmt.Sprintf("overlay on %q with %d layers", o.Target, len(o.Lower))
}
