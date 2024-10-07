package dbus_test

import (
	"errors"
	"strings"
	"testing"

	"git.ophivana.moe/cat/fortify/dbus"
	"git.ophivana.moe/cat/fortify/helper"
)

const (
	binPath = "/usr/bin/bwrap"
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
			if got := dbus.New(binPath, tc[0], tc[1]); !got.CompareTestNew(binPath, tc[0], tc[1]) {
				t.Errorf("New(%q, %q, %q) = %v",
					binPath, tc[0], tc[1],
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

		p := dbus.New(binPath, [2]string{}, [2]string{})
		_ = p.Seal(dbus.NewConfig("", true, false), nil)
		_ = p.Seal(dbus.NewConfig("", true, false), nil)
	})

	ep := dbus.New(binPath, [2]string{}, [2]string{})
	if err := ep.Seal(nil, nil); !errors.Is(err, dbus.ErrConfig) {
		t.Errorf("Seal(nil, nil) error = %v, want %v",
			err, dbus.ErrConfig)
	}

	for id, tc := range testCasePairs() {
		t.Run("create seal for "+id, func(t *testing.T) {
			p := dbus.New(binPath, tc[0].bus, tc[1].bus)
			if err := p.Seal(tc[0].c, tc[1].c); (errors.Is(err, helper.ErrContainsNull)) != tc[0].wantErr {
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
			helper.InternalReplaceExecCommand(t)
			p := dbus.New(binPath, tc[0].bus, tc[1].bus)

			t.Run("unsealed behaviour of "+id, func(t *testing.T) {
				t.Run("unsealed string of "+id, func(t *testing.T) {
					want := "(unsealed dbus proxy)"
					if got := p.String(); got != want {
						t.Errorf("String() = %v, want %v",
							got, want)
						return
					}
				})

				t.Run("unsealed wait of "+id, func(t *testing.T) {
					wantErr := "proxy not started"
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

			t.Run("sealed behaviour of "+id, func(t *testing.T) {
				want := strings.Join(append(tc[0].want, tc[1].want...), " ")
				if got := p.String(); got != want {
					t.Errorf("String() = %v, want %v",
						got, want)
					return
				}

				t.Run("sealed start of "+id, func(t *testing.T) {
					if err := p.Start(nil, nil); err != nil {
						t.Errorf("Start(nil, nil) error = %v",
							err)
					}

					t.Run("started string of "+id, func(t *testing.T) {
						wantSubstr := binPath + " --args=3"
						if got := p.String(); !strings.Contains(got, wantSubstr) {
							t.Errorf("String() = %v, want %v",
								p.String(), wantSubstr)
							return
						}
					})

					t.Run("sealed closing of "+id+" without status", func(t *testing.T) {
						wantPanic := "attempted to close helper with no status pipe"
						defer func() {
							if r := recover(); r != wantPanic {
								t.Errorf("Close() panic = %v, wantPanic %v",
									r, wantPanic)
							}
						}()

						if err := p.Close(); err != nil {
							t.Errorf("Close() error = %v",
								err)
						}
					})

					t.Run("started wait of "+id, func(t *testing.T) {
						if err := p.Wait(); err != nil {
							t.Errorf("Wait() error = %v",
								err)
						}
					})
				})
			})
		})
	}
}
