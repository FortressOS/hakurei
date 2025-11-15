package system

import (
	"context"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/dbus"
	"hakurei.app/internal/helper"
)

func TestDBusProxyOp(t *testing.T) {
	t.Parallel()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"dbusProxyStart", 0xdead, 0xff, &dbusProxyOp{
			final:  dbusNewFinalSample(4),
			out:    new(linePrefixWriter), // panics on write
			system: true,
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"session bus proxy on %q for upstream %q", []any{"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus", "unix:path=/run/user/1000/bus"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"system bus proxy on %q for upstream %q", []any{"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket", "unix:path=/run/dbus/system_bus_socket"}}, nil, nil),
			call("dbusProxyStart", stub.ExpectArgs{dbusNewFinalSample(4)}, nil, stub.UniqueError(2)),
		}, &OpError{
			Op: "dbus", Err: stub.UniqueError(2),
			Msg: "cannot start message bus proxy: unique error 2 injected by the test suite",
		}, nil, nil},

		{"dbusProxyWait", 0xdead, 0xff, &dbusProxyOp{
			final: dbusNewFinalSample(3),
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"session bus proxy on %q for upstream %q", []any{"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus", "unix:path=/run/user/1000/bus"}}, nil, nil),
			call("dbusProxyStart", stub.ExpectArgs{dbusNewFinalSample(3)}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"starting message bus proxy", ignoreValue{}}}, nil, nil),
		}, nil, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"terminating message bus proxy"}}, nil, nil),
			call("dbusProxyClose", stub.ExpectArgs{dbusNewFinalSample(3)}, nil, nil),
			call("dbusProxyWait", stub.ExpectArgs{dbusNewFinalSample(3)}, nil, stub.UniqueError(1)),
			call("verbose", stub.ExpectArgs{[]any{"message bus proxy exit"}}, nil, nil),
		}, &OpError{
			Op: "dbus", Err: stub.UniqueError(1), Revert: true,
			Msg: "message bus proxy error: unique error 1 injected by the test suite",
		}},

		{"success dbusProxyWait cancel", 0xdead, 0xff, &dbusProxyOp{
			final:  dbusNewFinalSample(2),
			system: true,
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"session bus proxy on %q for upstream %q", []any{"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus", "unix:path=/run/user/1000/bus"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"system bus proxy on %q for upstream %q", []any{"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket", "unix:path=/run/dbus/system_bus_socket"}}, nil, nil),
			call("dbusProxyStart", stub.ExpectArgs{dbusNewFinalSample(2)}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"starting message bus proxy", ignoreValue{}}}, nil, nil),
		}, nil, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"terminating message bus proxy"}}, nil, nil),
			call("dbusProxyClose", stub.ExpectArgs{dbusNewFinalSample(2)}, nil, nil),
			call("dbusProxyWait", stub.ExpectArgs{dbusNewFinalSample(2)}, nil, context.Canceled),
			call("verbose", stub.ExpectArgs{[]any{"message bus proxy canceled upstream"}}, nil, nil),
		}, nil},

		{"success", 0xdead, 0xff, &dbusProxyOp{
			final:  dbusNewFinalSample(1),
			system: true,
		}, []stub.Call{
			call("verbosef", stub.ExpectArgs{"session bus proxy on %q for upstream %q", []any{"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus", "unix:path=/run/user/1000/bus"}}, nil, nil),
			call("verbosef", stub.ExpectArgs{"system bus proxy on %q for upstream %q", []any{"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket", "unix:path=/run/dbus/system_bus_socket"}}, nil, nil),
			call("dbusProxyStart", stub.ExpectArgs{dbusNewFinalSample(1)}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"starting message bus proxy", ignoreValue{}}}, nil, nil),
		}, nil, []stub.Call{
			call("verbose", stub.ExpectArgs{[]any{"terminating message bus proxy"}}, nil, nil),
			call("dbusProxyClose", stub.ExpectArgs{dbusNewFinalSample(1)}, nil, nil),
			call("dbusProxyWait", stub.ExpectArgs{dbusNewFinalSample(1)}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"message bus proxy exit"}}, nil, nil),
		}, nil},
	})

	checkOpsBuilder(t, "ProxyDBus", []opsBuilderTestCase{
		{"nil session", 0xcafe, func(t *testing.T, sys *I) {
			wantErr := &OpError{
				Op: "dbus", Err: ErrDBusConfig,
				Msg: "attempted to create message bus proxy args without session bus config",
			}
			if err := sys.ProxyDBus(nil, new(hst.BusConfig), dbus.ProxyPair{}, dbus.ProxyPair{}); !reflect.DeepEqual(err, wantErr) {
				t.Errorf("ProxyDBus: error = %v, want %v", err, wantErr)
			}
		}, nil, stub.Expect{}},

		{"dbusFinalise NUL", 0xcafe, func(_ *testing.T, sys *I) {
			defer func() {
				want := "message bus proxy configuration contains NUL byte"
				if r := recover(); r != want {
					t.Errorf("MustProxyDBus: panic = %v, want %v", r, want)
				}
			}()

			sys.MustProxyDBus(
				&hst.BusConfig{
					// use impossible value here as an implicit assert that it goes through the stub
					Talk: []string{"session\x00"}, Filter: true,
				}, &hst.BusConfig{
					// use impossible value here as an implicit assert that it goes through the stub
					Talk: []string{"system\x00"}, Filter: true,
				}, dbus.ProxyPair{
					"unix:path=/run/user/1000/bus",
					"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus",
				}, dbus.ProxyPair{
					"unix:path=/run/dbus/system_bus_socket",
					"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket",
				})
		}, nil, stub.Expect{Calls: []stub.Call{
			call("dbusFinalise", stub.ExpectArgs{
				dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus"},
				dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket"},
				&hst.BusConfig{Talk: []string{"session\x00"}, Filter: true},
				&hst.BusConfig{Talk: []string{"system\x00"}, Filter: true},
			}, (*dbus.Final)(nil), syscall.EINVAL),
		}}},

		{"dbusFinalise", 0xcafe, func(_ *testing.T, sys *I) {
			wantErr := &OpError{
				Op: "dbus", Err: stub.UniqueError(0),
				Msg: "cannot finalise message bus proxy: unique error 0 injected by the test suite",
			}
			if err := sys.ProxyDBus(
				&hst.BusConfig{
					// use impossible value here as an implicit assert that it goes through the stub
					Talk: []string{"session\x00"}, Filter: true,
				}, &hst.BusConfig{
					// use impossible value here as an implicit assert that it goes through the stub
					Talk: []string{"system\x00"}, Filter: true,
				}, dbus.ProxyPair{
					"unix:path=/run/user/1000/bus",
					"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus",
				}, dbus.ProxyPair{
					"unix:path=/run/dbus/system_bus_socket",
					"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket",
				}); !reflect.DeepEqual(err, wantErr) {
				t.Errorf("ProxyDBus: error = %v", err)
			}
		}, nil, stub.Expect{Calls: []stub.Call{
			call("dbusFinalise", stub.ExpectArgs{
				dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus"},
				dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket"},
				&hst.BusConfig{Talk: []string{"session\x00"}, Filter: true},
				&hst.BusConfig{Talk: []string{"system\x00"}, Filter: true},
			}, (*dbus.Final)(nil), stub.UniqueError(0)),
		}}},

		{"full", 0xcafe, func(_ *testing.T, sys *I) {
			sys.MustProxyDBus(
				&hst.BusConfig{
					// use impossible value here as an implicit assert that it goes through the stub
					Talk: []string{"session\x00"}, Filter: true,
				}, &hst.BusConfig{
					// use impossible value here as an implicit assert that it goes through the stub
					Talk: []string{"system\x00"}, Filter: true,
				}, dbus.ProxyPair{
					"unix:path=/run/user/1000/bus",
					"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus",
				}, dbus.ProxyPair{
					"unix:path=/run/dbus/system_bus_socket",
					"/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket",
				})
		}, []Op{
			&dbusProxyOp{
				final:  dbusNewFinalSample(0),
				system: true,
			},
		}, stub.Expect{Calls: []stub.Call{
			call("dbusFinalise", stub.ExpectArgs{
				dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus"},
				dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket"},
				&hst.BusConfig{Talk: []string{"session\x00"}, Filter: true},
				&hst.BusConfig{Talk: []string{"system\x00"}, Filter: true},
			}, dbusNewFinalSample(0), nil),
			call("isVerbose", stub.ExpectArgs{}, true, nil),
			call("verbose", stub.ExpectArgs{[]any{"session bus proxy:", []string{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus", "--filter", "--talk=session\x00"}}}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"system bus proxy:", []string{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket", "--filter", "--talk=system\x00"}}}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"message bus proxy final args:", helper.MustNewCheckedArgs("unique", "value", "0", "injected", "by", "the", "test", "suite")}}, nil, nil),
		}}},
	})

	checkOpIs(t, []opIsTestCase{
		{"nil", (*dbusProxyOp)(nil), (*dbusProxyOp)(nil), false},
		{"zero", new(dbusProxyOp), new(dbusProxyOp), false},

		{"system differs", &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: false,
		}, &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, false},

		{"wt differs", &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1001/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, false},

		{"final system upstream differs", &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket\x00"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, false},

		{"final session upstream differs", &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1001/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, false},

		{"final system differs", &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.1/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, false},

		{"final session differs", &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1001/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, false},

		{"equals", &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, &dbusProxyOp{final: &dbus.Final{
			Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus"},
			System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket"},

			SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
			SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"unix", "/run/dbus/system_bus_socket"}}}},

			WriterTo: helper.MustNewCheckedArgs(
				"--filter", "unix:path=/run/user/1000/bus", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/bus",
				"--filter", "unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/b186c281d9e83a39afdc66d964ef99c6/system_bus_socket",
			),
		}, system: true,
		}, true},
	})

	checkOpMeta(t, []opMetaTestCase{
		{"dbus", new(dbusProxyOp),
			Process, "/proc/nonexistent",
			"(invalid dbus proxy)"},
	})
}

func dbusNewFinalSample(v int) *dbus.Final {
	return &dbus.Final{
		Session: dbus.ProxyPair{"unix:path=/run/user/1000/bus", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/bus"},
		System:  dbus.ProxyPair{"unix:path=/run/dbus/system_bus_socket", "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f/system_bus_socket"},

		SessionUpstream: []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/user/1000/bus"}}}},
		SystemUpstream:  []dbus.AddrEntry{{Method: "unix", Values: [][2]string{{"path", "/run/dbus/system_bus_socket"}}}},

		WriterTo: helper.MustNewCheckedArgs("unique", "value", strconv.Itoa(v), "injected", "by", "the", "test", "suite"),
	}
}

func TestLinePrefixWriter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prefix  string
		f       func(w func(s string))
		wantErr []error
		wantPt  []string
		want    []string
		wantExt []string
		wantBuf string
	}{
		{"nop", "(nop) ", func(func(string)) {}, nil, nil, nil, nil, ""},

		{"partial", "(partial) ", func(w func(string)) {
			w("C-65533: -> ")
		}, nil, nil, nil, []string{
			"*(partial) C-65533: -> ",
		}, "C-65533: -> "},

		{"break", "(break) ", func(w func(string)) {
			w("C-65533: -> ")
			w("org.freedesktop.DBus fake ListNames\n")
		}, nil, nil, []string{
			"C-65533: -> org.freedesktop.DBus fake ListNames",
		}, nil, ""},

		{"break pt", "(break pt) ", func(w func(string)) {
			w("init: ")
			w("received setup parameters\n")
		}, nil, []string{
			"init: received setup parameters",
		}, nil, nil, ""},

		{"threshold", "(threshold) ", func(w func(s string)) {
			w(string(make([]byte, lpwSizeThreshold)))
			w("\n")
		}, []error{nil, syscall.ENOMEM}, nil, nil, []string{
			"*(threshold) " + string(make([]byte, lpwSizeThreshold)),
			"+(threshold) write threshold reached, output may be incomplete",
		}, string(make([]byte, lpwSizeThreshold))},

		{"threshold multi", "(threshold multi) ", func(w func(s string)) {
			w(":3\n")
			w(string(make([]byte, lpwSizeThreshold-3)))
			w("\n")
		}, []error{nil, nil, syscall.ENOMEM}, nil, []string{
			":3",
		}, []string{
			"*(threshold multi) " + string(make([]byte, lpwSizeThreshold-3)),
			"+(threshold multi) write threshold reached, output may be incomplete",
		}, string(make([]byte, lpwSizeThreshold-3))},

		{"threshold multi partial", "(threshold multi partial) ", func(w func(s string)) {
			w(":3\n")
			w(string(make([]byte, lpwSizeThreshold-2)))
			w("dropped\n")
		}, []error{nil, nil, syscall.ENOMEM}, nil, []string{
			":3",
		}, []string{
			"*(threshold multi partial) " + string(make([]byte, lpwSizeThreshold-2)),
			"+(threshold multi partial) write threshold reached, output may be incomplete",
		}, string(make([]byte, lpwSizeThreshold-2))},

		{"threshold exact", "(threshold exact) ", func(w func(s string)) {
			w(string(make([]byte, lpwSizeThreshold-1)))
			w("\n")
		}, nil, nil, []string{
			string(make([]byte, lpwSizeThreshold-1)),
		}, []string{
			"+(threshold exact) write threshold reached, output may be incomplete",
		}, ""},

		{"sample", "(dbus) ", func(w func(s string)) {
			w("init: received setup parameters\n")
			w(`init: mounting "/nix/store/5gml2l2cj28yvyfyzblzjy1laqpxmyzd-libselinux-3.8.1/lib" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/5gml2l2cj28yvyfyzblzjy1laqpxmyzd-libselinux-3.8.1/lib" on "/sysroot/nix/store/5gml2l2cj28yvyfyzblzjy1laqpxmyzd-libselinux-3.8.1/lib" flags 0x4005` + "\n")
			w(`init: mounting "/nix/store/bcs094l67dlbqf7idxxbljp293zms9mh-util-linux-minimal-2.41-lib/lib" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/bcs094l67dlbqf7idxxbljp293zms9mh-util-linux-minimal-2.41-lib/lib" on "/sysroot/nix/store/bcs094l67dlbqf7idxxbljp293zms9mh-util-linux-minimal-2.41-lib/lib" flags 0x4005` + "\n")
			w(`init: mounting "/nix/store/jl19fdc7gdxqz9a1s368r9d15vpirnqy-zlib-1.3.1/lib" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/jl19fdc7gdxqz9a1s368r9d15vpirnqy-zlib-1.3.1/lib" on "/sysroot/nix/store/jl19fdc7gdxqz9a1s368r9d15vpirnqy-zlib-1.3.1/lib" flags 0x4005` + "\n")
			w(`init: mounting "/nix/store/rnn29mhynsa4ncmk0fkcrdr29n0j20l4-libffi-3.4.8/lib" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/rnn29mhynsa4ncmk0fkcrdr29n0j20l4-libffi-3.4.8/lib" on "/sysroot/nix/store/rnn29mhynsa4ncmk0fkcrdr29n0j20l4-libffi-3.4.8/lib" flags 0x4005` + "\n")
			w(`init: mounting "/nix/store/vvp8hlss3d5q6hn0cifq04jrpnp6bini-pcre2-10.44/lib" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/vvp8hlss3d5q6hn0cifq04jrpnp6bini-pcre2-10.44/lib" on "/sysroot/nix/store/vvp8hlss3d5q6hn0cifq04jrpnp6bini-pcre2-10.44/lib" flags 0x4005` + "\n")
			w(`init: mounting "/nix/store/y3nxdc2x8hwivppzgx5hkrhacsh87l21-glib-2.84.3/lib" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/y3nxdc2x8hwivppzgx5hkrhacsh87l21-glib-2.84.3/lib" on "/sysroot/nix/store/y3nxdc2x8hwivppzgx5hkrhacsh87l21-glib-2.84.3/lib" flags 0x4005` + "\n")
			w(`init: mounting "/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" on "/sysroot/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" flags 0x4005` + "\n")
			w(`init: mounting "/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib64" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" on "/sysroot/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib64" flags 0x4005` + "\n")
			w(`init: mounting "/run/user/1000" flags 0x0` + "\n")
			w(`init: resolved "/host/run/user/1000" on "/sysroot/run/user/1000" flags 0x4005` + "\n")
			w(`init: mounting "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f" flags 0x2` + "\n")
			w(`init: resolved "/host/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f" on "/sysroot/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f" flags 0x4004` + "\n")
			w(`init: mounting "/nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin" flags 0x0` + "\n")
			w(`init: resolved "/host/nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin" on "/sysroot/nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin" flags 0x4005` + "\n")
			w("init: resolving presets 0xf\n")
			w("init: 68 filter rules loaded\n")
			w("init: starting initial program /nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin/xdg-dbus-proxy\n")
			w("C1: -> org.freedesktop.DBus call org.freedesktop.DBus.Hello at /org/freedesktop/DBus\n")
			w("C-65536: -> org.freedesktop.DBus fake wildcarded AddMatch for org.freedesktop.portal\n")
			w("C-65535: -> org.freedesktop.DBus fake AddMatch for org.freedesktop.Notifications\n")
			w("C-65534: -> org.freedesktop.DBus fake GetNameOwner for org.freedesktop.Notifications\n")
			w("C-65533: -> org.freedesktop.DBus fake ListNames\n")
			w("B1: <- org.freedesktop.DBus return from C1\n")
			w("B2: <- org.freedesktop.DBus signal org.freedesktop.DBus.NameAcquired at /org/freedesktop/DBus\n")
			w("B3: <- org.freedesktop.DBus return from C-65536\n")
			w("*SKIPPED*\n")
			w("B4: <- org.freedesktop.DBus return from C-65535\n")
			w("*SKIPPED*\n")
			w("B5: <- org.freedesktop.DBus return error org.freedesktop.DBus.Error.NameHasNoOwner from C-65534\n")
			w("*SKIPPED*\n")
			w("B6: <- org.freedesktop.DBus return from C-65533\n")
			w("C-65532: -> org.freedesktop.DBus fake GetNameOwner for org.freedesktop.DBus\n")
			w("*SKIPPED*\n")
			w("B7: <- org.freedesktop.DBus return from C-65532\n")
			w("*SKIPPED*\n")
			w("C2: -> org.freedesktop.DBus call org.freedesktop.DBus.AddMatch at /org/freedesktop/DBus\n")
			w("C3: -> org.freedesktop.DBus call org.freedesktop.DBus.GetNameOwner at /org/freedesktop/DBus\n")
			w("C4: -> org.freedesktop.DBus call org.freedesktop.DBus.AddMatch at /org/freedesktop/DBus\n")
			w("C5: -> org.freedesktop.DBus call org.freedesktop.DBus.StartServiceByName at /org/freedesktop/DBus\n")
			w("B8: <- org.freedesktop.DBus return from C2\n")
			w("B9: <- org.freedesktop.DBus return error org.freedesktop.DBus.Error.NameHasNoOwner from C3\n")
			w("B10: <- org.freedesktop.DBus return from C4\n")
			w("B12: <- org.freedesktop.DBus signal org.freedesktop.DBus.NameOwnerChanged at /org/freedesktop/DBus\n")
			w("B11: <- org.freedesktop.DBus return from C5\n")
			w("C6: -> org.freedesktop.DBus call org.freedesktop.DBus.GetNameOwner at /org/freedesktop/DBus\n")
			w("B13: <- org.freedesktop.DBus return from C6\n")
			w("C7: -> :1.4 call org.freedesktop.Notifications.GetServerInformation at /org/freedesktop/Notifications\n")
			w("B4: <- :1.4 return from C7\n")
			w("C8: -> :1.4 call org.freedesktop.Notifications.GetServerInformation at /org/freedesktop/Notifications\n")
			w("B5: <- :1.4 return from C8\n")
			w("C9: -> :1.4 call org.freedesktop.Notifications.Notify at /org/freedesktop/Notifications\n")
			w("B6: <- :1.4 return from C9\n")
			w("C10: -> org.freedesktop.DBus call org.freedesktop.DBus.RemoveMatch at /org/freedesktop/DBus\n")
			w("C11: -> org.freedesktop.DBus call org.freedesktop.DBus.RemoveMatch at /org/freedesktop/DBus\n")
			w("B14: <- org.freedesktop.DBus return from C10\n")
			w("B15: <- org.freedesktop.DBus return from C11\n")
			w("init: initial process exited with code 0\n")
		}, nil, []string{
			"init: received setup parameters",
			`init: mounting "/nix/store/5gml2l2cj28yvyfyzblzjy1laqpxmyzd-libselinux-3.8.1/lib" flags 0x0`,
			`init: resolved "/host/nix/store/5gml2l2cj28yvyfyzblzjy1laqpxmyzd-libselinux-3.8.1/lib" on "/sysroot/nix/store/5gml2l2cj28yvyfyzblzjy1laqpxmyzd-libselinux-3.8.1/lib" flags 0x4005`,
			`init: mounting "/nix/store/bcs094l67dlbqf7idxxbljp293zms9mh-util-linux-minimal-2.41-lib/lib" flags 0x0`,
			`init: resolved "/host/nix/store/bcs094l67dlbqf7idxxbljp293zms9mh-util-linux-minimal-2.41-lib/lib" on "/sysroot/nix/store/bcs094l67dlbqf7idxxbljp293zms9mh-util-linux-minimal-2.41-lib/lib" flags 0x4005`,
			`init: mounting "/nix/store/jl19fdc7gdxqz9a1s368r9d15vpirnqy-zlib-1.3.1/lib" flags 0x0`,
			`init: resolved "/host/nix/store/jl19fdc7gdxqz9a1s368r9d15vpirnqy-zlib-1.3.1/lib" on "/sysroot/nix/store/jl19fdc7gdxqz9a1s368r9d15vpirnqy-zlib-1.3.1/lib" flags 0x4005`,
			`init: mounting "/nix/store/rnn29mhynsa4ncmk0fkcrdr29n0j20l4-libffi-3.4.8/lib" flags 0x0`,
			`init: resolved "/host/nix/store/rnn29mhynsa4ncmk0fkcrdr29n0j20l4-libffi-3.4.8/lib" on "/sysroot/nix/store/rnn29mhynsa4ncmk0fkcrdr29n0j20l4-libffi-3.4.8/lib" flags 0x4005`,
			`init: mounting "/nix/store/vvp8hlss3d5q6hn0cifq04jrpnp6bini-pcre2-10.44/lib" flags 0x0`,
			`init: resolved "/host/nix/store/vvp8hlss3d5q6hn0cifq04jrpnp6bini-pcre2-10.44/lib" on "/sysroot/nix/store/vvp8hlss3d5q6hn0cifq04jrpnp6bini-pcre2-10.44/lib" flags 0x4005`,
			`init: mounting "/nix/store/y3nxdc2x8hwivppzgx5hkrhacsh87l21-glib-2.84.3/lib" flags 0x0`,
			`init: resolved "/host/nix/store/y3nxdc2x8hwivppzgx5hkrhacsh87l21-glib-2.84.3/lib" on "/sysroot/nix/store/y3nxdc2x8hwivppzgx5hkrhacsh87l21-glib-2.84.3/lib" flags 0x4005`,
			`init: mounting "/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" flags 0x0`,
			`init: resolved "/host/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" on "/sysroot/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" flags 0x4005`,
			`init: mounting "/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib64" flags 0x0`,
			`init: resolved "/host/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib" on "/sysroot/nix/store/zdpby3l6azi78sl83cpad2qjpfj25aqx-glibc-2.40-66/lib64" flags 0x4005`,
			`init: mounting "/run/user/1000" flags 0x0`,
			`init: resolved "/host/run/user/1000" on "/sysroot/run/user/1000" flags 0x4005`,
			`init: mounting "/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f" flags 0x2`,
			`init: resolved "/host/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f" on "/sysroot/tmp/hakurei.0/99dd71ee2146369514e0d10783368f8f" flags 0x4004`,
			`init: mounting "/nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin" flags 0x0`,
			`init: resolved "/host/nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin" on "/sysroot/nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin" flags 0x4005`,
			"init: resolving presets 0xf",
			"init: 68 filter rules loaded",
			"init: starting initial program /nix/store/d2divmq2d897amikcwpdx7zrbpddxxcl-xdg-dbus-proxy-0.1.6/bin/xdg-dbus-proxy",

			"init: initial process exited with code 0",
		}, []string{
			"C1: -> org.freedesktop.DBus call org.freedesktop.DBus.Hello at /org/freedesktop/DBus",
			"C-65536: -> org.freedesktop.DBus fake wildcarded AddMatch for org.freedesktop.portal",
			"C-65535: -> org.freedesktop.DBus fake AddMatch for org.freedesktop.Notifications",
			"C-65534: -> org.freedesktop.DBus fake GetNameOwner for org.freedesktop.Notifications",
			"C-65533: -> org.freedesktop.DBus fake ListNames",
			"B1: <- org.freedesktop.DBus return from C1",
			"B2: <- org.freedesktop.DBus signal org.freedesktop.DBus.NameAcquired at /org/freedesktop/DBus",
			"B3: <- org.freedesktop.DBus return from C-65536",
			"*SKIPPED*",
			"B4: <- org.freedesktop.DBus return from C-65535",
			"*SKIPPED*",
			"B5: <- org.freedesktop.DBus return error org.freedesktop.DBus.Error.NameHasNoOwner from C-65534",
			"*SKIPPED*",
			"B6: <- org.freedesktop.DBus return from C-65533",
			"C-65532: -> org.freedesktop.DBus fake GetNameOwner for org.freedesktop.DBus",
			"*SKIPPED*",
			"B7: <- org.freedesktop.DBus return from C-65532",
			"*SKIPPED*",
			"C2: -> org.freedesktop.DBus call org.freedesktop.DBus.AddMatch at /org/freedesktop/DBus",
			"C3: -> org.freedesktop.DBus call org.freedesktop.DBus.GetNameOwner at /org/freedesktop/DBus",
			"C4: -> org.freedesktop.DBus call org.freedesktop.DBus.AddMatch at /org/freedesktop/DBus",
			"C5: -> org.freedesktop.DBus call org.freedesktop.DBus.StartServiceByName at /org/freedesktop/DBus",
			"B8: <- org.freedesktop.DBus return from C2",
			"B9: <- org.freedesktop.DBus return error org.freedesktop.DBus.Error.NameHasNoOwner from C3",
			"B10: <- org.freedesktop.DBus return from C4",
			"B12: <- org.freedesktop.DBus signal org.freedesktop.DBus.NameOwnerChanged at /org/freedesktop/DBus",
			"B11: <- org.freedesktop.DBus return from C5",
			"C6: -> org.freedesktop.DBus call org.freedesktop.DBus.GetNameOwner at /org/freedesktop/DBus",
			"B13: <- org.freedesktop.DBus return from C6",
			"C7: -> :1.4 call org.freedesktop.Notifications.GetServerInformation at /org/freedesktop/Notifications",
			"B4: <- :1.4 return from C7",
			"C8: -> :1.4 call org.freedesktop.Notifications.GetServerInformation at /org/freedesktop/Notifications",
			"B5: <- :1.4 return from C8",
			"C9: -> :1.4 call org.freedesktop.Notifications.Notify at /org/freedesktop/Notifications",
			"B6: <- :1.4 return from C9",
			"C10: -> org.freedesktop.DBus call org.freedesktop.DBus.RemoveMatch at /org/freedesktop/DBus",
			"C11: -> org.freedesktop.DBus call org.freedesktop.DBus.RemoveMatch at /org/freedesktop/DBus",
			"B14: <- org.freedesktop.DBus return from C10",
			"B15: <- org.freedesktop.DBus return from C11",
		}, nil, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotPt := make([]string, 0, len(tc.wantPt))
			out := &linePrefixWriter{
				prefix: tc.prefix,
				println: func(v ...any) {
					if len(v) != 1 {
						t.Fatalf("invalid call to println: %#v", v)
					}
					gotPt = append(gotPt, v[0].(string))
				},
				buf: new(strings.Builder),
			}

			var pos int
			tc.f(func(s string) {
				_, err := out.Write([]byte(s))
				if tc.wantErr != nil {
					if !reflect.DeepEqual(err, tc.wantErr[pos]) {
						t.Fatalf("Write: error = %v, want %v", err, tc.wantErr[pos])
					}
				} else if err != nil {
					t.Fatalf("Write: unexpected error: %v", err)
					return
				}
				pos++
			})

			if !slices.Equal(out.msg, tc.want) {
				t.Errorf("msg: %#v, want %#v", out.msg, tc.want)
			}

			if out.buf.String() != tc.wantBuf {
				t.Errorf("buf: %q, want %q", out.buf, tc.wantBuf)
			}

			wantPt := make([]string, len(tc.wantPt))
			for i, m := range tc.wantPt {
				wantPt[i] = tc.prefix + m
			}
			if !slices.Equal(gotPt, wantPt) {
				t.Errorf("passthrough: %#v, want %#v", gotPt, wantPt)
			}

			wantDump := make([]string, len(tc.want)+len(tc.wantExt))
			for i, want := range tc.want {
				wantDump[i] = tc.prefix + want
			}
			for i, want := range tc.wantExt {
				wantDump[len(tc.want)+i] = want
			}
			t.Run("dump", func(t *testing.T) {
				got := make([]string, 0, len(wantDump))
				out.println = func(v ...any) {
					if len(v) != 1 {
						t.Fatalf("Dump: invalid call to println: %#v", v)
					}
					got = append(got, v[0].(string))
				}
				out.Dump()

				if !slices.Equal(got, wantDump) {
					t.Errorf("Dump: %#v, want %#v", got, wantDump)
				}
			})
		})
	}
}
