package system

import (
	"errors"
	"net"
	"os"
	"reflect"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/internal/hlog"
)

func TestOpError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		s    string
		is   error
		isF  error
		msg  string
	}{
		{"message", newOpErrorMessage("dbus", ErrDBusConfig,
			"attempted to create message bus proxy args without session bus config", false),
			"attempted to create message bus proxy args without session bus config",
			ErrDBusConfig, syscall.ENOTRECOVERABLE,
			"attempted to create message bus proxy args without session bus config"},

		{"apply", newOpError("tmpfile", syscall.EBADE, false),
			"apply tmpfile: invalid exchange",
			syscall.EBADE, syscall.EBADF,
			"cannot apply tmpfile: invalid exchange"},

		{"revert", newOpError("wayland", syscall.EBADF, true),
			"revert wayland: bad file descriptor",
			syscall.EBADF, syscall.EBADE,
			"cannot revert wayland: bad file descriptor"},

		{"path", newOpError("tmpfile", &os.PathError{Op: "stat", Path: "/run/dbus", Err: syscall.EISDIR}, false),
			"stat /run/dbus: is a directory",
			syscall.EISDIR, syscall.ENOTDIR,
			"cannot stat /run/dbus: is a directory"},

		{"net", newOpError("wayland", &net.OpError{Op: "dial", Net: "unix", Addr: &net.UnixAddr{Name: "/run/user/1000/wayland-1", Net: "unix"}, Err: syscall.ENOENT}, false),
			"dial unix /run/user/1000/wayland-1: no such file or directory",
			syscall.ENOENT, syscall.EPERM,
			"cannot dial unix /run/user/1000/wayland-1: no such file or directory"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("error", func(t *testing.T) {
				if got := tc.err.Error(); got != tc.s {
					t.Errorf("Error: %q, want %q", got, tc.s)
				}
			})

			t.Run("is", func(t *testing.T) {
				if !errors.Is(tc.err, tc.is) {
					t.Error("Is: unexpected false")
				}
				if errors.Is(tc.err, tc.isF) {
					t.Error("Is: unexpected true")
				}
			})

			t.Run("msg", func(t *testing.T) {
				if got, ok := container.GetErrorMessage(tc.err); !ok {
					if tc.msg != "" {
						t.Errorf("GetErrorMessage: err does not implement MessageError")
					}
					return
				} else if got != tc.msg {
					t.Errorf("GetErrorMessage: %q, want %q", got, tc.msg)
				}
			})
		})
	}

	t.Run("new", func(t *testing.T) {
		if err := newOpError("check", nil, false); err != nil {
			t.Errorf("newOpError: %v", err)
		}
		if err := newOpErrorMessage("check", nil, "", false); err != nil {
			t.Errorf("newOpErrorMessage: %v", err)
		}
	})
}

func TestSetOutput(t *testing.T) {
	oldmsg := msg
	t.Cleanup(func() { msg = oldmsg })
	msg = nil

	t.Run("nil", func(t *testing.T) {
		SetOutput(nil)
		if _, ok := msg.(*container.DefaultMsg); !ok {
			t.Errorf("SetOutput: %#v", msg)
		}
	})

	t.Run("hlog", func(t *testing.T) {
		SetOutput(hlog.Output{})
		if _, ok := msg.(hlog.Output); !ok {
			t.Errorf("SetOutput: %#v", msg)
		}
	})

	t.Run("reset", func(t *testing.T) {
		SetOutput(nil)
		if _, ok := msg.(*container.DefaultMsg); !ok {
			t.Errorf("SetOutput: %#v", msg)
		}
	})
}

func TestPrintJoinedError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want [][]any
	}{
		{"nil", nil, [][]any{{"not a joined error:", nil}}},
		{"single", errors.Join(syscall.EINVAL), [][]any{{"invalid argument"}}},

		{"unwrapped", syscall.EINVAL, [][]any{{"not a joined error:", syscall.EINVAL}}},
		{"unwrapped message", &OpError{
			Op:  "meow",
			Err: syscall.EBADFD,
		}, [][]any{
			{"cannot apply meow: file descriptor in bad state"},
		}},

		{"many", errors.Join(syscall.ENOTRECOVERABLE, syscall.ETIMEDOUT, syscall.EBADFD), [][]any{
			{"state not recoverable"},
			{"connection timed out"},
			{"file descriptor in bad state"},
		}},
		{"many message", errors.Join(
			&container.StartError{Step: "meow", Err: syscall.ENOMEM},
			&os.PathError{Op: "meow", Path: "/proc/nonexistent", Err: syscall.ENOSYS},
			&os.LinkError{Op: "link", Old: "/etc", New: "/proc/nonexistent", Err: syscall.ENOENT},
			&OpError{Op: "meow", Err: syscall.ENODEV, Revert: true},
		), [][]any{
			{"cannot meow: cannot allocate memory"},
			{"meow /proc/nonexistent: function not implemented"},
			{"link /etc /proc/nonexistent: no such file or directory"},
			{"cannot revert meow: no such device"},
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got [][]any
			printJoinedError(func(v ...any) { got = append(got, v) }, "not a joined error:", tc.err)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("printJoinedError: %#v, want %#v", got, tc.want)
			}
		})
	}
}

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
				ec   Enablement
				want bool
			}{
				{"nil", 0xff, ptc.et != User},
				{"self", ptc.et, true},
				{"all", EWayland | EX11 | EDBus | EPulse | User | Process, true},
				{"enablements", EWayland | EX11 | EDBus | EPulse, ptc.et != User && ptc.et != Process},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					var criteria *Criteria
					if tc.ec != 0xff {
						criteria = (*Criteria)(&tc.ec)
					}
					if got := criteria.hasType(o.Type()); got != tc.want {
						t.Errorf("hasType: got %v, want %v",
							got, tc.want)
					}
				})
			}
		})
	}
}
