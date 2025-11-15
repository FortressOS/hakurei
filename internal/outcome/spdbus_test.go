package outcome

import (
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/stub"
	"hakurei.app/hst"
	"hakurei.app/internal/acl"
	"hakurei.app/internal/dbus"
	"hakurei.app/internal/helper"
	"hakurei.app/internal/system"
	"hakurei.app/message"
)

func TestSpDBusOp(t *testing.T) {
	config := hst.Template()

	checkOpBehaviour(t, []opBehaviourTestCase{
		{"not enabled", func(bool, bool) outcomeOp {
			return new(spDBusOp)
		}, func() *hst.Config {
			c := hst.Template()
			*c.Enablements = 0
			return c
		}, nil, nil, nil, nil, errNotEnabled, nil, nil, nil, nil, nil},

		{"invalid", func(bool, bool) outcomeOp {
			return new(spDBusOp)
		}, func() *hst.Config {
			c := hst.Template()
			c.SessionBus.Talk[0] += "\x00"
			c.SystemBus = nil
			return c
		}, nil, []stub.Call{
			call("dbusAddress", stub.ExpectArgs{}, [2]string{
				"unix:path=/run/user/1000/bus",
				"unix:path=/var/run/dbus/system_bus_socket",
			}, nil),
		}, nil, sysUsesInstance(nil), &system.OpError{
			Op:     "dbus",
			Err:    syscall.EINVAL,
			Msg:    "message bus proxy configuration contains NUL byte",
			Revert: false,
		}, nil, nil, nil, nil, nil},

		{"success default", func(bool, bool) outcomeOp {
			return new(spDBusOp)
		}, func() *hst.Config {
			c := hst.Template()
			c.SessionBus, c.SystemBus = nil, nil
			return c
		}, nil, []stub.Call{
			call("dbusAddress", stub.ExpectArgs{}, [2]string{
				"unix:path=/run/user/1000/bus",
				"unix:path=/var/run/dbus/system_bus_socket",
			}, nil),
			call("isVerbose", stub.ExpectArgs{}, true, nil),
			call("verbose", stub.ExpectArgs{[]any{"session bus proxy:", []string{
				"unix:path=/run/user/1000/bus",
				wantInstancePrefix + "/bus",
				"--filter",
				"--talk=org.freedesktop.DBus",
				"--talk=org.freedesktop.Notifications",
				"--own=org.chromium.Chromium.*",
				"--own=org.mpris.MediaPlayer2.org.chromium.Chromium.*",
				"--call=org.freedesktop.portal.*=*",
				"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*",
			}}}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"message bus proxy final args:", helper.MustNewCheckedArgs(
				"unix:path=/run/user/1000/bus",
				wantInstancePrefix+"/bus",
				"--filter",
				"--talk=org.freedesktop.DBus",
				"--talk=org.freedesktop.Notifications",
				"--own=org.chromium.Chromium.*",
				"--own=org.mpris.MediaPlayer2.org.chromium.Chromium.*",
				"--call=org.freedesktop.portal.*=*",
				"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*",
			)}}, nil, nil),
		}, func() *system.I {
			sys := system.New(panicMsgContext{}, message.New(nil), checkExpectUid)
			sys.Ephemeral(system.Process, m(wantInstancePrefix), 0711)
			if err := sys.ProxyDBus(
				dbus.NewConfig(config.ID, true, true), nil,
				dbus.ProxyPair{"unix:path=/run/user/1000/bus", wantInstancePrefix + "/bus"},
				dbus.ProxyPair{"unix:path=/var/run/dbus/system_bus_socket", wantInstancePrefix + "/system_bus_socket"},
			); err != nil {
				t.Fatalf("cannot prepare sys: %v", err)
			}
			sys.UpdatePerm(m(wantInstancePrefix+"/bus"), acl.Read, acl.Write)
			return sys
		}(), sysUsesInstance(nil), nil, insertsOps(afterSpRuntimeOp(nil)), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m(wantInstancePrefix+"/bus"),
					m("/run/user/1000/bus"), 0),
		}, paramsWantEnv(config, map[string]string{
			"DBUS_SESSION_BUS_ADDRESS": "unix:path=/run/user/1000/bus",
		}, nil), nil},

		{"success", func(isShim, _ bool) outcomeOp {
			if !isShim {
				return new(spDBusOp)
			}
			return &spDBusOp{ProxySystem: true}
		}, hst.Template, nil, []stub.Call{
			call("dbusAddress", stub.ExpectArgs{}, [2]string{
				"unix:path=/run/user/1000/bus",
				"unix:path=/var/run/dbus/system_bus_socket",
			}, nil),
			call("isVerbose", stub.ExpectArgs{}, true, nil),
			call("verbose", stub.ExpectArgs{[]any{"session bus proxy:", []string{
				"unix:path=/run/user/1000/bus",
				wantInstancePrefix + "/bus",
				"--filter",
				"--talk=org.freedesktop.Notifications",
				"--talk=org.freedesktop.FileManager1",
				"--talk=org.freedesktop.ScreenSaver",
				"--talk=org.freedesktop.secrets",
				"--talk=org.kde.kwalletd5",
				"--talk=org.kde.kwalletd6",
				"--talk=org.gnome.SessionManager",
				"--own=org.chromium.Chromium.*",
				"--own=org.mpris.MediaPlayer2.org.chromium.Chromium.*",
				"--own=org.mpris.MediaPlayer2.chromium.*",
				"--call=org.freedesktop.portal.*=*",
				"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*",
			}}}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"system bus proxy:", []string{
				"unix:path=/var/run/dbus/system_bus_socket",
				wantInstancePrefix + "/system_bus_socket",
				"--filter",
				"--talk=org.bluez",
				"--talk=org.freedesktop.Avahi",
				"--talk=org.freedesktop.UPower",
			}}}, nil, nil),
			call("verbose", stub.ExpectArgs{[]any{"message bus proxy final args:", helper.MustNewCheckedArgs(
				"unix:path=/run/user/1000/bus",
				wantInstancePrefix+"/bus",
				"--filter",
				"--talk=org.freedesktop.Notifications",
				"--talk=org.freedesktop.FileManager1",
				"--talk=org.freedesktop.ScreenSaver",
				"--talk=org.freedesktop.secrets",
				"--talk=org.kde.kwalletd5",
				"--talk=org.kde.kwalletd6",
				"--talk=org.gnome.SessionManager",
				"--own=org.chromium.Chromium.*",
				"--own=org.mpris.MediaPlayer2.org.chromium.Chromium.*",
				"--own=org.mpris.MediaPlayer2.chromium.*",
				"--call=org.freedesktop.portal.*=*",
				"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*",

				"unix:path=/var/run/dbus/system_bus_socket",
				wantInstancePrefix+"/system_bus_socket",
				"--filter",
				"--talk=org.bluez",
				"--talk=org.freedesktop.Avahi",
				"--talk=org.freedesktop.UPower",
			)}}, nil, nil),
		}, func() *system.I {
			sys := system.New(panicMsgContext{}, message.New(nil), checkExpectUid)
			sys.Ephemeral(system.Process, m(wantInstancePrefix), 0711)
			if err := sys.ProxyDBus(
				config.SessionBus, config.SystemBus,
				dbus.ProxyPair{"unix:path=/run/user/1000/bus", wantInstancePrefix + "/bus"},
				dbus.ProxyPair{"unix:path=/var/run/dbus/system_bus_socket", wantInstancePrefix + "/system_bus_socket"},
			); err != nil {
				t.Fatalf("cannot prepare sys: %v", err)
			}
			sys.UpdatePerm(m(wantInstancePrefix+"/bus"), acl.Read, acl.Write).
				UpdatePerm(m(wantInstancePrefix+"/system_bus_socket"), acl.Read, acl.Write)
			return sys
		}(), sysUsesInstance(nil), nil, insertsOps(afterSpRuntimeOp(nil)), []stub.Call{
			// this op configures the container state and does not make calls during toContainer
		}, &container.Params{
			Ops: new(container.Ops).
				Bind(m(wantInstancePrefix+"/bus"),
					m("/run/user/1000/bus"), 0).
				Bind(m(wantInstancePrefix+"/system_bus_socket"),
					m("/var/run/dbus/system_bus_socket"), 0),
		}, paramsWantEnv(config, map[string]string{
			"DBUS_SESSION_BUS_ADDRESS": "unix:path=/run/user/1000/bus",
			"DBUS_SYSTEM_BUS_ADDRESS":  "unix:path=/var/run/dbus/system_bus_socket",
		}, nil), nil},
	})
}
