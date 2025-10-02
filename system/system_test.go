package system

import (
	"errors"
	"os"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/system/internal/xcb"
)

func TestCriteria(t *testing.T) {
	testCases := []struct {
		name  string
		ec, t hst.Enablement
		want  bool
	}{
		{"nil", 0xff, hst.EWayland, true},
		{"nil user", 0xff, User, false},
		{"all", hst.EWayland | hst.EX11 | hst.EDBus | hst.EPulse | User | Process, Process, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var criteria *Criteria
			if tc.ec != 0xff {
				criteria = (*Criteria)(&tc.ec)
			}
			if got := criteria.hasType(tc.t); got != tc.want {
				t.Errorf("hasType: got %v, want %v",
					got, tc.want)
			}
		})
	}
}

func TestTypeString(t *testing.T) {
	testCases := []struct {
		e    hst.Enablement
		want string
	}{
		{hst.EWayland, hst.EWayland.String()},
		{hst.EX11, hst.EX11.String()},
		{hst.EDBus, hst.EDBus.String()},
		{hst.EPulse, hst.EPulse.String()},
		{User, "user"},
		{Process, "process"},
		{User | Process, "user, process"},
		{hst.EWayland | User | Process, "wayland, user, process"},
		{hst.EX11 | Process, "x11, process"},
	}

	for _, tc := range testCases {
		t.Run("label type string "+strconv.Itoa(int(tc.e)), func(t *testing.T) {
			if got := TypeString(tc.e); got != tc.want {
				t.Errorf("TypeString: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		t.Run("ctx", func(t *testing.T) {
			defer func() {
				want := "invalid call to New"
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}
			}()
			New(nil, container.NewMsg(nil), 0)
		})

		t.Run("msg", func(t *testing.T) {
			defer func() {
				want := "invalid call to New"
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}
			}()
			New(t.Context(), nil, 0)
		})

		t.Run("uid", func(t *testing.T) {
			defer func() {
				want := "invalid call to New"
				if r := recover(); r != want {
					t.Errorf("recover: %v, want %v", r, want)
				}
			}()
			New(t.Context(), container.NewMsg(nil), -1)
		})
	})

	sys := New(t.Context(), container.NewMsg(nil), 0xdeadbeef)
	if sys.ctx == nil {
		t.Error("New: ctx = nil")
	}
	if got := sys.UID(); got != 0xdeadbeef {
		t.Errorf("UID: %d", got)
	}
}

func TestEqual(t *testing.T) {
	testCases := []struct {
		name string
		sys  *I
		v    *I
		want bool
	}{
		{"simple UID",
			New(t.Context(), container.NewMsg(nil), 150),
			New(t.Context(), container.NewMsg(nil), 150),
			true},

		{"simple UID differ",
			New(t.Context(), container.NewMsg(nil), 150),
			New(t.Context(), container.NewMsg(nil), 151),
			false},

		{"simple UID nil",
			New(t.Context(), container.NewMsg(nil), 150),
			nil,
			false},

		{"op length mismatch",
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos"),
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos").
				Ensure(m("/run"), 0755),
			false},

		{"op value mismatch",
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos").
				Ensure(m("/run"), 0644),
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos").
				Ensure(m("/run"), 0755),
			false},

		{"op type mismatch",
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos").
				CopyFile(new([]byte), m("/home/ophestra/xdg/config/pulse/cookie"), 0, 256),
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos").
				Ensure(m("/run"), 0755),
			false},

		{"op equals",
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos").
				Ensure(m("/run"), 0755),
			New(t.Context(), container.NewMsg(nil), 150).
				ChangeHosts("chronos").
				Ensure(m("/run"), 0755),
			true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.sys.Equal(tc.v) != tc.want {
				t.Errorf("Equal: %v, want %v", !tc.want, tc.want)
			}
		})
	}
}

func TestCommitRevert(t *testing.T) {
	testCases := []struct {
		name string
		f    func(sys *I)
		ec   hst.Enablement

		commit        []stub.Call
		wantErrCommit error

		revert        []stub.Call
		wantErrRevert error
	}{
		{"apply xhost partial mkdir", func(sys *I) {
			sys.
				Ephemeral(Process, m("/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"), 0711).
				ChangeHosts("chronos")
		}, 0xff, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, stub.UniqueError(2)),
			call("verbosef", stub.ExpectArgs{"commit faulted after %d ops, rolling back partial commit", []any{1}}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"destroying ephemeral directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"}, nil, stub.UniqueError(3)),
			call("println", stub.ExpectArgs{[]any{"cannot revert mkdir: unique error 3 injected by the test suite"}}, nil, nil),
		}, &OpError{Op: "xhost", Err: stub.UniqueError(2)}, nil, nil},

		{"apply xhost", func(sys *I) {
			sys.
				Ephemeral(Process, m("/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"), 0711).
				ChangeHosts("chronos")
		}, 0xff, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, stub.UniqueError(2)),
			call("verbosef", stub.ExpectArgs{"commit faulted after %d ops, rolling back partial commit", []any{1}}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"destroying ephemeral directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"}, nil, nil),
		}, &OpError{Op: "xhost", Err: stub.UniqueError(2)}, nil, nil},

		{"revert multi", func(sys *I) {
			sys.
				Ephemeral(Process, m("/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"), 0711).
				ChangeHosts("chronos")
		}, 0xff, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"deleting entry %s from X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeDelete), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, stub.UniqueError(1)),
			call("verbose", stub.ExpectArgs{[]any{"destroying ephemeral directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"}, nil, stub.UniqueError(0)),
		}, errors.Join(
			&OpError{Op: "xhost", Err: stub.UniqueError(1), Revert: true},
			&OpError{Op: "mkdir", Err: stub.UniqueError(0), Revert: true})},

		{"success", func(sys *I) {
			sys.
				Ephemeral(Process, m("/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"), 0711).
				ChangeHosts("chronos")
		}, 0xff, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"ensuring directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("mkdir", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", os.FileMode(0711)}, nil, nil),
			call("verbosef", stub.ExpectArgs{"inserting entry %s to X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeInsert), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, nil),
		}, nil, []stub.Call{
			call("verbosef", stub.ExpectArgs{"deleting entry %s from X11", []any{xhostOp("chronos")}}, nil, nil),
			call("xcbChangeHosts", stub.ExpectArgs{xcb.HostMode(xcb.HostModeDelete), xcb.Family(xcb.FamilyServerInterpreted), "localuser\x00chronos"}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"destroying ephemeral directory", &mkdirOp{Process, "/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9", 0711, true}}}, nil, nil),
			call("remove", stub.ExpectArgs{"/tmp/hakurei.0/f2f3bcd492d0266438fa9bf164fe90d9"}, nil, nil),
		}, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ec *Criteria
			if tc.ec != 0xff {
				ec = (*Criteria)(&tc.ec)
			}

			sys, s := InternalNew(t, stub.Expect{Calls: slices.Concat(tc.commit, []stub.Call{{Name: stub.CallSeparator}}, tc.revert)}, 0xbad)
			defer stub.HandleExit(t)
			tc.f(sys)
			errCommit := sys.Commit()
			s.Expects(stub.CallSeparator)
			if !reflect.DeepEqual(errCommit, tc.wantErrCommit) {
				t.Errorf("Commit: error = %v, want %v", errCommit, tc.wantErrCommit)
			}
			if errCommit != nil {
				goto out
			}

			if err := sys.Revert(ec); !reflect.DeepEqual(err, tc.wantErrRevert) {
				t.Errorf("Revert: error = %v, want %v", err, tc.wantErrRevert)
			}

		out:
			s.VisitIncomplete(func(s *stub.Stub[syscallDispatcher]) {
				count := s.Pos() - 1 // separator
				if count < len(tc.commit) {
					t.Errorf("Commit: %d calls, want %d", count, len(tc.commit))
				} else {
					t.Errorf("Revert: %d calls, want %d", count-len(tc.commit), len(tc.revert))
				}
			})
		})
	}

	t.Run("panic", func(t *testing.T) {
		t.Run("committed", func(t *testing.T) {
			defer func() {
				want := "attempting to commit twice"
				if r := recover(); r != want {
					t.Errorf("Commit: panic = %v, want %v", r, want)
				}
			}()
			_ = (&I{committed: true}).Commit()
		})

		t.Run("reverted", func(t *testing.T) {
			defer func() {
				want := "attempting to revert twice"
				if r := recover(); r != want {
					t.Errorf("Revert: panic = %v, want %v", r, want)
				}
			}()
			_ = (&I{reverted: true}).Revert(nil)
		})
	})
}

func TestNop(t *testing.T) {
	// these do nothing
	new(noCopy).Unlock()
	new(noCopy).Lock()
}

func m(pathname string) *container.Absolute { return container.MustAbs(pathname) }
