package dbus_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/sandbox"
)

func TestNew(t *testing.T) {
	for _, tc := range [][2][2]string{
		{
			{"unix:path=/run/user/1971/bus", "/tmp/fortify.1971/1ca5d183ef4c99e74c3e544715f32702/bus"},
			{"unix:path=/run/dbus/system_bus_socket", "/tmp/fortify.1971/1ca5d183ef4c99e74c3e544715f32702/system_bus_socket"},
		},
		{
			{"unix:path=/run/user/1971/bus", "/tmp/fortify.1971/881ac3796ff3f3bf0a773824383187a0/bus"},
			{"unix:path=/run/dbus/system_bus_socket", "/tmp/fortify.1971/881ac3796ff3f3bf0a773824383187a0/system_bus_socket"},
		},
		{
			{"unix:path=/run/user/1971/bus", "/tmp/fortify.1971/3d1a5084520ef79c0c6a49a675bac701/bus"},
			{"unix:path=/run/dbus/system_bus_socket", "/tmp/fortify.1971/3d1a5084520ef79c0c6a49a675bac701/system_bus_socket"},
		},
		{
			{"unix:path=/run/user/1971/bus", "/tmp/fortify.1971/2a1639bab712799788ea0ff7aa280c35/bus"},
			{"unix:path=/run/dbus/system_bus_socket", "/tmp/fortify.1971/2a1639bab712799788ea0ff7aa280c35/system_bus_socket"},
		},
	} {
		t.Run("create instance for "+tc[0][0]+" and "+tc[1][0], func(t *testing.T) {
			if got := dbus.New(tc[0], tc[1]); !got.CompareTestNew(tc[0], tc[1]) {
				t.Errorf("New(%q, %q) = %v",
					tc[0], tc[1],
					got)
			}
		})
	}
}

func TestProxy_Seal(t *testing.T) {
	t.Run("double seal panic", func(t *testing.T) {
		defer func() {
			want := "dbus proxy sealed twice"
			if r := recover(); r != want {
				t.Errorf("Seal: panic = %q, want %q",
					r, want)
			}
		}()

		p := dbus.New([2]string{}, [2]string{})
		_ = p.Seal(dbus.NewConfig("", true, false), nil)
		_ = p.Seal(dbus.NewConfig("", true, false), nil)
	})

	ep := dbus.New([2]string{}, [2]string{})
	if err := ep.Seal(nil, nil); !errors.Is(err, dbus.ErrConfig) {
		t.Errorf("Seal(nil, nil) error = %v, want %v",
			err, dbus.ErrConfig)
	}

	for id, tc := range testCasePairs() {
		t.Run("create seal for "+id, func(t *testing.T) {
			p := dbus.New(tc[0].bus, tc[1].bus)
			if err := p.Seal(tc[0].c, tc[1].c); (errors.Is(err, syscall.EINVAL)) != tc[0].wantErr {
				t.Errorf("Seal(%p, %p) error = %v, wantErr %v",
					tc[0].c, tc[1].c,
					err, tc[0].wantErr)
				return
			}

			// rest of the tests happen for sealed instances
			if tc[0].wantErr {
				return
			}

			// build null-terminated string from wanted args
			want := new(strings.Builder)
			args := append(tc[0].want, tc[1].want...)
			for _, arg := range args {
				want.WriteString(arg)
				want.WriteByte('\x00')
			}

			wt := p.AccessTestProxySeal()
			got := new(strings.Builder)
			if _, err := wt.WriteTo(got); err != nil {
				t.Errorf("p.seal.WriteTo(): %v", err)
			}

			if want.String() != got.String() {
				t.Errorf("Seal(%p, %p) seal = %v, want %v",
					tc[0].c, tc[1].c,
					got.String(), want.String())
			}
		})
	}
}

func TestProxy_Start_Wait_Close_String(t *testing.T) {
	oldWaitDelay := helper.WaitDelay
	helper.WaitDelay = 16 * time.Second
	t.Cleanup(func() { helper.WaitDelay = oldWaitDelay })

	t.Run("sandbox", func(t *testing.T) {
		proxyName := dbus.ProxyName
		dbus.ProxyName = os.Args[0]
		t.Cleanup(func() { dbus.ProxyName = proxyName })
		testProxyStartWaitCloseString(t, true)
	})
	t.Run("direct", func(t *testing.T) { testProxyStartWaitCloseString(t, false) })
}

func testProxyStartWaitCloseString(t *testing.T, useSandbox bool) {
	for id, tc := range testCasePairs() {
		// this test does not test errors
		if tc[0].wantErr {
			continue
		}

		t.Run("string for nil proxy", func(t *testing.T) {
			var p *dbus.Proxy
			want := "(invalid dbus proxy)"
			if got := p.String(); got != want {
				t.Errorf("String() = %v, want %v",
					got, want)
			}
		})

		t.Run("proxy for "+id, func(t *testing.T) {
			p := dbus.New(tc[0].bus, tc[1].bus)
			p.CommandContext = func(ctx context.Context) (cmd *exec.Cmd) {
				return exec.CommandContext(ctx, os.Args[0], "-test.v",
					"-test.run=TestHelperInit", "--", "init")
			}
			p.CmdF = func(v any) {
				if useSandbox {
					container := v.(*sandbox.Container)
					if container.Args[0] != dbus.ProxyName {
						panic(fmt.Sprintf("unexpected argv0 %q", os.Args[0]))
					}
					container.Args = append([]string{os.Args[0], "-test.run=TestHelperStub", "--"}, container.Args[1:]...)
				} else {
					cmd := v.(*exec.Cmd)
					if cmd.Args[0] != dbus.ProxyName {
						panic(fmt.Sprintf("unexpected argv0 %q", os.Args[0]))
					}
					cmd.Err = nil
					cmd.Path = os.Args[0]
					cmd.Args = append([]string{os.Args[0], "-test.run=TestHelperStub", "--"}, cmd.Args[1:]...)
				}
			}
			p.FilterF = func(v []byte) []byte { return bytes.SplitN(v, []byte("TestHelperInit\n"), 2)[1] }
			output := new(strings.Builder)

			t.Run("unsealed", func(t *testing.T) {
				t.Run("string", func(t *testing.T) {
					want := "(unsealed dbus proxy)"
					if got := p.String(); got != want {
						t.Errorf("String() = %v, want %v",
							got, want)
						return
					}
				})

				t.Run("start", func(t *testing.T) {
					want := "proxy not sealed"
					if err := p.Start(context.Background(), nil, useSandbox); err == nil || err.Error() != want {
						t.Errorf("Start() error = %v, wantErr %q",
							err, errors.New(want))
						return
					}
				})

				t.Run("wait", func(t *testing.T) {
					wantErr := "dbus: not started"
					if err := p.Wait(); err == nil || err.Error() != wantErr {
						t.Errorf("Wait() error = %v, wantErr %v",
							err, wantErr)
					}
				})
			})

			t.Run("seal with "+id, func(t *testing.T) {
				if err := p.Seal(tc[0].c, tc[1].c); err != nil {
					t.Errorf("Seal(%p, %p) error = %v, wantErr %v",
						tc[0].c, tc[1].c,
						err, tc[0].wantErr)
					return
				}
			})

			t.Run("sealed", func(t *testing.T) {
				want := strings.Join(append(tc[0].want, tc[1].want...), " ")
				if got := p.String(); got != want {
					t.Errorf("String() = %v, want %v",
						got, want)
					return
				}

				t.Run("start", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					if err := p.Start(ctx, output, useSandbox); err != nil {
						t.Fatalf("Start(nil, nil) error = %v",
							err)
					}

					t.Run("string", func(t *testing.T) {
						wantSubstr := fmt.Sprintf("%s -test.run=TestHelperStub -- --args=3 --fd=4", os.Args[0])
						if useSandbox {
							wantSubstr = fmt.Sprintf(`argv: ["%s" "-test.run=TestHelperStub" "--" "--args=3" "--fd=4"], flags: 0x0, seccomp: 0x3e`, os.Args[0])
						}
						if got := p.String(); !strings.Contains(got, wantSubstr) {
							t.Errorf("String() = %v, want %v",
								p.String(), wantSubstr)
							return
						}
					})

					t.Run("wait", func(t *testing.T) {
						p.Close()
						if err := p.Wait(); err != nil {
							t.Errorf("Wait() error = %v\noutput: %s",
								err, output.String())
						}
					})
				})
			})
		})
	}
}

func TestHelperInit(t *testing.T) {
	if len(os.Args) != 5 || os.Args[4] != "init" {
		return
	}
	sandbox.SetOutput(fmsg.Output{})
	sandbox.Init(fmsg.Prepare, internal.InstallFmsg)
}
