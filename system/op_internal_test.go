package system

import "testing"

type tcOp struct {
	et   Enablement
	path string
}

// test an instance of the Op interface
func (ptc tcOp) test(t *testing.T, gotOps []Op, wantOps []Op, fn string) {
	if len(gotOps) != len(wantOps) {
		t.Errorf("%s: inserted %v Ops, want %v", fn,
			len(gotOps), len(wantOps))
		return
	}

	t.Run("path", func(t *testing.T) {
		if len(gotOps) > 0 {
			if got := gotOps[0].Path(); got != ptc.path {
				t.Errorf("Path() = %q, want %q",
					got, ptc.path)
				return
			}
		}
	})

	for i := range gotOps {
		o := gotOps[i]

		t.Run("is", func(t *testing.T) {
			if !o.Is(o) {
				t.Errorf("Is returned false on self")
				return
			}
			if !o.Is(wantOps[i]) {
				t.Errorf("%s: inserted %#v, want %#v",
					fn,
					o, wantOps[i])
				return
			}
		})

		t.Run("criteria", func(t *testing.T) {
			testCases := []struct {
				name string
				ec   *Criteria
				want bool
			}{
				{"nil", newCriteria(), ptc.et != User},
				{"self", newCriteria(ptc.et), true},
				{"all", newCriteria(EWayland, EX11, EDBus, EPulse, User, Process), true},
				{"enablements", newCriteria(EWayland, EX11, EDBus, EPulse), ptc.et != User && ptc.et != Process},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					if got := tc.ec.hasType(o); got != tc.want {
						t.Errorf("hasType: got %v, want %v",
							got, tc.want)
					}
				})
			}
		})
	}
}

func newCriteria(labels ...Enablement) *Criteria {
	ec := new(Criteria)
	if len(labels) == 0 {
		return ec
	}

	ec.Enablements = new(Enablements)
	for _, e := range labels {
		ec.Set(e)
	}
	return ec
}
