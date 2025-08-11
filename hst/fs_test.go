package hst_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/container"
	"hakurei.app/hst"
)

func TestFilesystemConfigJSON(t *testing.T) {
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
			hst.FSImplError{
				Type:  "bind",
				Value: stubFS{"bind"},
			},
			"\x00", "\x00"},

		{"bad impl ephemeral", hst.FilesystemConfigJSON{FilesystemConfig: stubFS{"ephemeral"}},
			hst.FSImplError{
				Type:  "ephemeral",
				Value: stubFS{"ephemeral"},
			},
			"\x00", "\x00"},

		{"bind", hst.FilesystemConfigJSON{
			FilesystemConfig: &hst.FSBind{
				Dst:      m("/etc"),
				Src:      m("/mnt/etc"),
				Optional: true,
			},
		}, nil,
			`{"type":"bind","dst":"/etc","src":"/mnt/etc","optional":true}`,
			`{"fs":{"type":"bind","dst":"/etc","src":"/mnt/etc","optional":true},"magic":3236757504}`},

		{"ephemeral", hst.FilesystemConfigJSON{
			FilesystemConfig: &hst.FSEphemeral{
				Dst:   m("/run/user/65534"),
				Write: true,
				Size:  1 << 10,
				Perm:  0700,
			},
		}, nil,
			`{"type":"ephemeral","dst":"/run/user/65534","write":true,"size":1024,"perm":448}`,
			`{"fs":{"type":"ephemeral","dst":"/run/user/65534","write":true,"size":1024,"perm":448},"magic":3236757504}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("marshal", func(t *testing.T) {
				{
					d, err := json.Marshal(&tc.want)
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("Marshal: error = %v, want %v", err, tc.wantErr)
					}
					if tc.wantErr != nil {
						goto checkSMarshal
					}
					if string(d) != tc.data {
						t.Errorf("Marshal:\n%s\nwant:\n%s", string(d), tc.data)
					}
				}

			checkSMarshal:
				{
					d, err := json.Marshal(&sCheck{tc.want, syscall.MS_MGC_VAL})
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("Marshal: error = %v, want %v", err, tc.wantErr)
					}
					if tc.wantErr != nil {
						return
					}
					if string(d) != tc.sData {
						t.Errorf("Marshal:\n%s\nwant:\n%s", string(d), tc.sData)
					}
				}
			})

			t.Run("unmarshal", func(t *testing.T) {
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
		if got := (*hst.FilesystemConfigJSON).Valid(nil); got {
			t.Errorf("Valid: %v, want false", got)
		}

		if got := new(hst.FilesystemConfigJSON).Valid(); got {
			t.Errorf("Valid: %v, want false", got)
		}

		if got := (&hst.FilesystemConfigJSON{FilesystemConfig: new(hst.FSBind)}).Valid(); !got {
			t.Errorf("Valid: %v, want true", got)
		}
	})

	t.Run("passthrough", func(t *testing.T) {
		if err := new(hst.FilesystemConfigJSON).UnmarshalJSON(make([]byte, 0)); err == nil {
			t.Errorf("UnmarshalJSON: error = %v", err)
		}
	})
}

func TestFSErrors(t *testing.T) {
	t.Run("type", func(t *testing.T) {
		want := `invalid filesystem type "cat"`
		if got := hst.FSTypeError("cat").Error(); got != want {
			t.Errorf("Error: %q, want %q", got, want)
		}
	})

	t.Run("impl", func(t *testing.T) {
		testCases := []struct {
			name string
			val  hst.FilesystemConfig
			want string
		}{
			{"nil", nil, "implementation nil is not cat"},
			{"stub", stubFS{"cat"}, "implementation stubFS is not cat"},
			{"*stub", &stubFS{"cat"}, "implementation *stubFS is not cat"},
			{"(*stub)(nil)", (*stubFS)(nil), "implementation *stubFS is not cat"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := hst.FSImplError{Type: "cat", Value: tc.val}
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

func (s stubFS) Type() string                { return s.typeName }
func (s stubFS) Target() *container.Absolute { panic("unreachable") }
func (s stubFS) Host() []*container.Absolute { panic("unreachable") }
func (s stubFS) Apply(*container.Ops)        { panic("unreachable") }
func (s stubFS) String() string              { return "<invalid " + s.typeName + ">" }

type sCheck struct {
	FS    hst.FilesystemConfigJSON `json:"fs"`
	Magic int                      `json:"magic"`
}

type fsTestCase struct {
	name   string
	fs     hst.FilesystemConfig
	ops    container.Ops
	target *container.Absolute
	host   []*container.Absolute
	str    string
}

func checkFs(t *testing.T, fstype string, testCases []fsTestCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fs.Type(); got != fstype {
				t.Errorf("Type: %q, want %q", got, fstype)
			}

			t.Run("ops", func(t *testing.T) {
				ops := new(container.Ops)
				tc.fs.Apply(ops)
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

			t.Run("target", func(t *testing.T) {
				if got := tc.fs.Target(); !reflect.DeepEqual(got, tc.target) {
					t.Errorf("Target: %q, want %q", got, tc.target)
				}
			})

			t.Run("host", func(t *testing.T) {
				if got := tc.fs.Host(); !reflect.DeepEqual(got, tc.host) {
					t.Errorf("Host: %q, want %q", got, tc.host)
				}
			})

			t.Run("string", func(t *testing.T) {
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

func m(pathname string) *container.Absolute { return container.MustAbs(pathname) }
func ms(pathnames ...string) []*container.Absolute {
	as := make([]*container.Absolute, len(pathnames))
	for i, pathname := range pathnames {
		as[i] = container.MustAbs(pathname)
	}
	return as
}
