package dbus_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"hakurei.app/container"
	"hakurei.app/helper"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
	"hakurei.app/system/dbus"
)

func TestFinalise(t *testing.T) {
	if _, err := dbus.Finalise(dbus.ProxyPair{}, dbus.ProxyPair{}, nil, nil); !errors.Is(err, syscall.EBADE) {
		t.Errorf("Finalise: error = %v, want %v",
			err, syscall.EBADE)
	}

	for id, tc := range testCasePairs() {
		t.Run("create final for "+id, func(t *testing.T) {
			var wt io.WriterTo
			if v, err := dbus.Finalise(tc[0].bus, tc[1].bus, tc[0].c, tc[1].c); (errors.Is(err, syscall.EINVAL)) != tc[0].wantErr {
				t.Errorf("Finalise: error = %v, wantErr %v",
					err, tc[0].wantErr)
				return
			} else {
				wt = v
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
				want.WriteByte(0)
			}

			got := new(strings.Builder)
			if _, err := wt.WriteTo(got); err != nil {
				t.Errorf("WriteTo: error = %v", err)
			}

			if want.String() != got.String() {
				t.Errorf("Seal: %q, want %q",
					got.String(), want.String())
			}
		})
	}
}

func TestProxyStartWaitCloseString(t *testing.T) {
	oldWaitDelay := helper.WaitDelay
	helper.WaitDelay = 16 * time.Second
	t.Cleanup(func() { helper.WaitDelay = oldWaitDelay })

	t.Run("sandbox", func(t *testing.T) {
		proxyName := dbus.ProxyName
		dbus.ProxyName = os.Args[0]
		t.Cleanup(func() { dbus.ProxyName = proxyName })
		testProxyFinaliseStartWaitCloseString(t, true)
	})
	t.Run("direct", func(t *testing.T) { testProxyFinaliseStartWaitCloseString(t, false) })
}

func testProxyFinaliseStartWaitCloseString(t *testing.T, useSandbox bool) {
	var p *dbus.Proxy

	t.Run("string for nil proxy", func(t *testing.T) {
		want := "(invalid dbus proxy)"
		if got := p.String(); got != want {
			t.Errorf("String: %q, want %q",
				got, want)
		}
	})

	t.Run("invalid start", func(t *testing.T) {
		if !useSandbox {
			p = dbus.NewDirect(t.Context(), nil, nil)
		} else {
			p = dbus.New(t.Context(), nil, nil)
		}

		if err := p.Start(); !errors.Is(err, syscall.ENOTRECOVERABLE) {
			t.Errorf("Start: error = %q, wantErr %q",
				err, syscall.ENOTRECOVERABLE)
			return
		}
	})

	for id, tc := range testCasePairs() {
		// this test does not test errors
		if tc[0].wantErr {
			continue
		}

		t.Run("proxy for "+id, func(t *testing.T) {
			var final *dbus.Final
			t.Run("finalise", func(t *testing.T) {
				if v, err := dbus.Finalise(tc[0].bus, tc[1].bus, tc[0].c, tc[1].c); err != nil {
					t.Errorf("Finalise: error = %v, wantErr %v",
						err, tc[0].wantErr)
					return
				} else {
					final = v
				}
			})

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			if !useSandbox {
				p = dbus.NewDirect(ctx, final, nil)
			} else {
				p = dbus.New(ctx, final, nil)
			}

			p.CommandContext = func(ctx context.Context) (cmd *exec.Cmd) {
				return exec.CommandContext(ctx, os.Args[0], "-test.v",
					"-test.run=TestHelperInit", "--", "init")
			}
			p.CmdF = func(v any) {
				if useSandbox {
					z := v.(*container.Container)
					if z.Args[0] != dbus.ProxyName {
						panic(fmt.Sprintf("unexpected argv0 %q", os.Args[0]))
					}
					z.Args = append([]string{os.Args[0], "-test.run=TestHelperStub", "--"}, z.Args[1:]...)
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

			t.Run("invalid wait", func(t *testing.T) {
				wantErr := "dbus: not started"
				if err := p.Wait(); err == nil || err.Error() != wantErr {
					t.Errorf("Wait: error = %v, wantErr %v",
						err, wantErr)
				}
			})

			t.Run("string", func(t *testing.T) {
				want := "(unused dbus proxy)"
				if got := p.String(); got != want {
					t.Errorf("String: %q, want %q",
						got, want)
					return
				}
			})

			t.Run("start", func(t *testing.T) {
				if err := p.Start(); err != nil {
					t.Fatalf("Start: error = %v",
						err)
				}

				t.Run("string", func(t *testing.T) {
					wantSubstr := fmt.Sprintf("%s -test.run=TestHelperStub -- --args=3 --fd=4", os.Args[0])
					if useSandbox {
						wantSubstr = fmt.Sprintf(`argv: ["%s" "-test.run=TestHelperStub" "--" "--args=3" "--fd=4"], filter: true, rules: 0, flags: 0x1, presets: 0xf`, os.Args[0])
					}
					if got := p.String(); !strings.Contains(got, wantSubstr) {
						t.Errorf("String: %q, want %q",
							got, wantSubstr)
						return
					}
				})

				t.Run("wait", func(t *testing.T) {
					done := make(chan struct{})
					go func() {
						if err := p.Wait(); err != nil {
							t.Errorf("Wait: error = %v\noutput: %s",
								err, output.String())
						}
						close(done)
					}()
					p.Close()
					<-done
				})
			})
		})
	}
}

func TestHelperInit(t *testing.T) {
	if len(os.Args) != 5 || os.Args[4] != "init" {
		return
	}
	container.SetOutput(hlog.Output{})
	container.Init(hlog.Prepare, internal.InstallOutput)
}
