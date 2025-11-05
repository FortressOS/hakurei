package hst_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/hst"
)

func TestFilesystemConfigJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want hst.FilesystemConfigJSON

		wantErr     error
		data, sData string
	}{
		{"nil", hst.FilesystemConfigJSON{FilesystemConfig: nil}, hst.ErrFSNull,
			`null`, `{"fs":null,"magic":3236757504}`},

		{"bad type", hst.FilesystemConfigJSON{FilesystemConfig: stubFS{"cat"}},
			hst.FSTypeError("cat"),
			`{"type":"cat","meow":true}`, `{"fs":{"type":"cat","meow":true},"magic":3236757504}`},

		{"bad impl bind", hst.FilesystemConfigJSON{FilesystemConfig: stubFS{"bind"}},
			hst.FSImplError{Value: stubFS{"bind"}},
			"\x00", "\x00"},

		{"bad impl ephemeral", hst.FilesystemConfigJSON{FilesystemConfig: stubFS{"ephemeral"}},
			hst.FSImplError{Value: stubFS{"ephemeral"}},
			"\x00", "\x00"},

		{"bad impl overlay", hst.FilesystemConfigJSON{FilesystemConfig: stubFS{"overlay"}},
			hst.FSImplError{Value: stubFS{"overlay"}},
			"\x00", "\x00"},

		{"bind", hst.FilesystemConfigJSON{
			FilesystemConfig: &hst.FSBind{
				Target:   m("/etc"),
				Source:   m("/mnt/etc"),
				Optional: true,
			},
		}, nil,
			`{"type":"bind","dst":"/etc","src":"/mnt/etc","optional":true}`,
			`{"fs":{"type":"bind","dst":"/etc","src":"/mnt/etc","optional":true},"magic":3236757504}`},

		{"ephemeral", hst.FilesystemConfigJSON{
			FilesystemConfig: &hst.FSEphemeral{
				Target: m("/run/user/65534"),
				Write:  true,
				Size:   1 << 10,
				Perm:   0700,
			},
		}, nil,
			`{"type":"ephemeral","dst":"/run/user/65534","write":true,"size":1024,"perm":448}`,
			`{"fs":{"type":"ephemeral","dst":"/run/user/65534","write":true,"size":1024,"perm":448},"magic":3236757504}`},

		{"overlay", hst.FilesystemConfigJSON{
			FilesystemConfig: &hst.FSOverlay{
				Target: m("/nix/store"),
				Lower:  ms("/mnt-root/nix/.ro-store"),
				Upper:  m("/mnt-root/nix/.rw-store/upper"),
				Work:   m("/mnt-root/nix/.rw-store/work"),
			},
		}, nil,
			`{"type":"overlay","dst":"/nix/store","lower":["/mnt-root/nix/.ro-store"],"upper":"/mnt-root/nix/.rw-store/upper","work":"/mnt-root/nix/.rw-store/work"}`,
			`{"fs":{"type":"overlay","dst":"/nix/store","lower":["/mnt-root/nix/.ro-store"],"upper":"/mnt-root/nix/.rw-store/upper","work":"/mnt-root/nix/.rw-store/work"},"magic":3236757504}`},

		{"link", hst.FilesystemConfigJSON{
			FilesystemConfig: &hst.FSLink{
				Target:      m("/run/current-system"),
				Linkname:    "/run/current-system",
				Dereference: true,
			},
		}, nil,
			`{"type":"link","dst":"/run/current-system","linkname":"/run/current-system","dereference":true}`,
			`{"fs":{"type":"link","dst":"/run/current-system","linkname":"/run/current-system","dereference":true},"magic":3236757504}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("marshal", func(t *testing.T) {
				t.Parallel()
				wantErr := tc.wantErr
				if errors.As(wantErr, new(hst.FSTypeError)) {
					// for unsupported implementation tc
					wantErr = hst.FSImplError{Value: stubFS{"cat"}}
				}

				{
					d, err := json.Marshal(&tc.want)
					if !errors.Is(err, wantErr) {
						t.Errorf("Marshal: error = %v, want %v", err, wantErr)
					}
					if wantErr != nil {
						goto checkSMarshal
					}
					if string(d) != tc.data {
						t.Errorf("Marshal:\n%s\nwant:\n%s", string(d), tc.data)
					}
				}

			checkSMarshal:
				{
					d, err := json.Marshal(&sCheck{tc.want, syscall.MS_MGC_VAL})
					if !errors.Is(err, wantErr) {
						t.Errorf("Marshal: error = %v, want %v", err, wantErr)
					}
					if wantErr != nil {
						return
					}
					if string(d) != tc.sData {
						t.Errorf("Marshal:\n%s\nwant:\n%s", string(d), tc.sData)
					}
				}
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()
				if tc.data == "\x00" && tc.sData == "\x00" {
					if errors.As(tc.wantErr, new(hst.FSImplError)) {
						// this error is only returned on marshal
						return
					}
				}

				{
					var got hst.FilesystemConfigJSON
					err := json.Unmarshal([]byte(tc.data), &got)
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("Unmarshal: error = %v, want %v", err, tc.wantErr)
					}
					if tc.wantErr != nil {
						goto checkSUnmarshal
					}
					if !reflect.DeepEqual(&tc.want, &got) {
						t.Errorf("Unmarshal: %#v, want %#v", &tc.want, &got)
					}
				}

			checkSUnmarshal:
				{
					var got sCheck
					err := json.Unmarshal([]byte(tc.sData), &got)
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("Unmarshal: error = %v, want %v", err, tc.wantErr)
					}
					if tc.wantErr != nil {
						return
					}
					want := sCheck{tc.want, syscall.MS_MGC_VAL}
					if !reflect.DeepEqual(&got, &want) {
						t.Errorf("Unmarshal: %#v, want %#v", &got, &want)
					}
				}
			})
		})
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		if got := (*hst.FilesystemConfigJSON).Valid(nil); got {
			t.Errorf("Valid: %v, want false", got)
		}

		if got := new(hst.FilesystemConfigJSON).Valid(); got {
			t.Errorf("Valid: %v, want false", got)
		}

		if got := (&hst.FilesystemConfigJSON{FilesystemConfig: &hst.FSBind{Source: m("/etc")}}).Valid(); !got {
			t.Errorf("Valid: %v, want true", got)
		}
	})

	t.Run("passthrough", func(t *testing.T) {
		t.Parallel()
		if err := new(hst.FilesystemConfigJSON).UnmarshalJSON(make([]byte, 0)); err == nil {
			t.Errorf("UnmarshalJSON: error = %v", err)
		}
	})
}

func TestFSErrors(t *testing.T) {
	t.Parallel()

	t.Run("type", func(t *testing.T) {
		t.Parallel()
		want := `invalid filesystem type "cat"`
		if got := hst.FSTypeError("cat").Error(); got != want {
			t.Errorf("Error: %q, want %q", got, want)
		}
	})

	t.Run("impl", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			val  hst.FilesystemConfig
			want string
		}{
			{"nil", nil, "implementation nil not supported"},
			{"stub", stubFS{"cat"}, "implementation stubFS not supported"},
			{"*stub", &stubFS{"cat"}, "implementation *stubFS not supported"},
			{"(*stub)(nil)", (*stubFS)(nil), "implementation *stubFS not supported"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := hst.FSImplError{Value: tc.val}
				if got := err.Error(); got != tc.want {
					t.Errorf("Error: %q, want %q", got, tc.want)
				}
			})
		}
	})
}

type stubFS struct {
	typeName string
}

func (s stubFS) Valid() bool             { return false }
func (s stubFS) Path() *check.Absolute   { panic("unreachable") }
func (s stubFS) Host() []*check.Absolute { panic("unreachable") }
func (s stubFS) Apply(*hst.ApplyState)   { panic("unreachable") }
func (s stubFS) String() string          { return "<invalid " + s.typeName + ">" }

type sCheck struct {
	FS    hst.FilesystemConfigJSON `json:"fs"`
	Magic uint64                   `json:"magic"`
}

type fsTestCase struct {
	name  string
	fs    hst.FilesystemConfig
	valid bool
	ops   container.Ops
	path  *check.Absolute
	host  []*check.Absolute
	str   string
}

func checkFs(t *testing.T, testCases []fsTestCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("valid", func(t *testing.T) {
				t.Parallel()
				if got := tc.fs.Valid(); got != tc.valid {
					t.Errorf("Valid: %v, want %v", got, tc.valid)
				}
			})

			t.Run("ops", func(t *testing.T) {
				t.Parallel()
				ops := new(container.Ops)
				tc.fs.Apply(&hst.ApplyState{AutoEtcPrefix: ":3", Ops: opsAdapter{ops}})
				if !reflect.DeepEqual(ops, &tc.ops) {
					gotString := new(strings.Builder)
					for _, op := range *ops {
						gotString.WriteString("\n" + op.String())
					}
					wantString := new(strings.Builder)
					for _, op := range tc.ops {
						wantString.WriteString("\n" + op.String())
					}
					t.Errorf("Apply: %s, want %s", gotString, wantString)
				}
			})

			t.Run("path", func(t *testing.T) {
				t.Parallel()
				if got := tc.fs.Path(); !reflect.DeepEqual(got, tc.path) {
					t.Errorf("Target: %q, want %q", got, tc.path)
				}
			})

			t.Run("host", func(t *testing.T) {
				t.Parallel()
				if got := tc.fs.Host(); !reflect.DeepEqual(got, tc.host) {
					t.Errorf("Host: %q, want %q", got, tc.host)
				}
			})

			t.Run("string", func(t *testing.T) {
				t.Parallel()
				if tc.str == "\x00" {
					return
				}

				if got := tc.fs.String(); got != tc.str {
					t.Errorf("String: %q, want %q", got, tc.str)
				}
			})
		})
	}
}

type opsAdapter struct{ *container.Ops }

func (p opsAdapter) Tmpfs(target *check.Absolute, size int, perm os.FileMode) hst.Ops {
	return opsAdapter{p.Ops.Tmpfs(target, size, perm)}
}

func (p opsAdapter) Readonly(target *check.Absolute, perm os.FileMode) hst.Ops {
	return opsAdapter{p.Ops.Readonly(target, perm)}
}

func (p opsAdapter) Bind(source, target *check.Absolute, flags int) hst.Ops {
	return opsAdapter{p.Ops.Bind(source, target, flags)}
}

func (p opsAdapter) Overlay(target, state, work *check.Absolute, layers ...*check.Absolute) hst.Ops {
	return opsAdapter{p.Ops.Overlay(target, state, work, layers...)}
}

func (p opsAdapter) OverlayReadonly(target *check.Absolute, layers ...*check.Absolute) hst.Ops {
	return opsAdapter{p.Ops.OverlayReadonly(target, layers...)}
}

func (p opsAdapter) Link(target *check.Absolute, linkName string, dereference bool) hst.Ops {
	return opsAdapter{p.Ops.Link(target, linkName, dereference)}
}

func (p opsAdapter) Root(host *check.Absolute, flags int) hst.Ops {
	return opsAdapter{p.Ops.Root(host, flags)}
}

func (p opsAdapter) Etc(host *check.Absolute, prefix string) hst.Ops {
	return opsAdapter{p.Ops.Etc(host, prefix)}
}

func m(pathname string) *check.Absolute { return check.MustAbs(pathname) }
func ms(pathnames ...string) []*check.Absolute {
	as := make([]*check.Absolute, len(pathnames))
	for i, pathname := range pathnames {
		as[i] = check.MustAbs(pathname)
	}
	return as
}
