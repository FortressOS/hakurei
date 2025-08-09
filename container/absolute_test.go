package container

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func TestAbsoluteError(t *testing.T) {
	testCases := []struct {
		name string

		err error
		cmp error
		ok  bool
	}{
		{"EINVAL", new(AbsoluteError), syscall.EINVAL, true},
		{"not EINVAL", new(AbsoluteError), syscall.EBADE, false},
		{"ne val", new(AbsoluteError), &AbsoluteError{"etc"}, false},
		{"equals", &AbsoluteError{"etc"}, &AbsoluteError{"etc"}, true},
	}

	for _, tc := range testCases {
		if got := errors.Is(tc.err, tc.cmp); got != tc.ok {
			t.Errorf("Is: %v, want %v", got, tc.ok)
		}
	}

	t.Run("string", func(t *testing.T) {
		want := `path "etc" is not absolute`
		if got := (&AbsoluteError{"etc"}).Error(); got != want {
			t.Errorf("Error: %q, want %q", got, want)
		}
	})
}

func TestNewAbs(t *testing.T) {
	testCases := []struct {
		name string

		pathname string
		want     *Absolute
		wantErr  error
	}{
		{"good", "/etc", MustAbs("/etc"), nil},
		{"not absolute", "etc", nil, &AbsoluteError{"etc"}},
		{"zero", "", nil, &AbsoluteError{""}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewAbs(tc.pathname)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NewAbs: %#v, want %#v", got, tc.want)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("NewAbs: error = %v, want %v", err, tc.wantErr)
			}
		})
	}

	t.Run("must", func(t *testing.T) {
		defer func() {
			wantPanic := `path "etc" is not absolute`

			if r := recover(); r != wantPanic {
				t.Errorf("MustAbsolute: panic = %v; want %v", r, wantPanic)
			}
		}()

		MustAbs("etc")
	})
}

func TestAbsoluteString(t *testing.T) {
	t.Run("passthrough", func(t *testing.T) {
		pathname := "/etc"
		if got := (&Absolute{pathname}).String(); got != pathname {
			t.Errorf("String: %q, want %q", got, pathname)
		}
	})

	t.Run("zero", func(t *testing.T) {
		defer func() {
			wantPanic := "attempted use of zero Absolute"

			if r := recover(); r != wantPanic {
				t.Errorf("String: panic = %v, want %v", r, wantPanic)
			}
		}()

		panic(new(Absolute).String())
	})
}

type sCheck struct {
	Pathname *Absolute `json:"val"`
	Magic    int       `json:"magic"`
}

func TestCodecAbsolute(t *testing.T) {
	testCases := []struct {
		name string
		a    *Absolute

		wantErr error

		gob, sGob   string
		json, sJson string
	}{
		{"good", MustAbs("/etc"),
			nil,
			"\t\x7f\x05\x01\x02\xff\x82\x00\x00\x00\b\xff\x80\x00\x04/etc",
			",\xff\x83\x03\x01\x01\x06sCheck\x01\xff\x84\x00\x01\x02\x01\bPathname\x01\xff\x80\x00\x01\x05Magic\x01\x04\x00\x00\x00\t\x7f\x05\x01\x02\xff\x82\x00\x00\x00\x10\xff\x84\x01\x04/etc\x01\xfb\x01\x81\xda\x00\x00\x00",

			`"/etc"`, `{"val":"/etc","magic":3236757504}`},
		{"not absolute", nil,
			&AbsoluteError{"etc"},
			"\t\x7f\x05\x01\x02\xff\x82\x00\x00\x00\a\xff\x80\x00\x03etc",
			",\xff\x83\x03\x01\x01\x06sCheck\x01\xff\x84\x00\x01\x02\x01\bPathname\x01\xff\x80\x00\x01\x05Magic\x01\x04\x00\x00\x00\t\x7f\x05\x01\x02\xff\x82\x00\x00\x00\x0f\xff\x84\x01\x03etc\x01\xfb\x01\x81\xda\x00\x00\x00",

			`"etc"`, `{"val":"etc","magic":3236757504}`},
		{"zero", nil,
			new(AbsoluteError),
			"\t\x7f\x05\x01\x02\xff\x82\x00\x00\x00\x04\xff\x80\x00\x00",
			",\xff\x83\x03\x01\x01\x06sCheck\x01\xff\x84\x00\x01\x02\x01\bPathname\x01\xff\x80\x00\x01\x05Magic\x01\x04\x00\x00\x00\t\x7f\x05\x01\x02\xff\x82\x00\x00\x00\f\xff\x84\x01\x00\x01\xfb\x01\x81\xda\x00\x00\x00",
			`""`, `{"val":"","magic":3236757504}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("gob", func(t *testing.T) {
				t.Run("encode", func(t *testing.T) {
					// encode is unchecked
					if errors.Is(tc.wantErr, syscall.EINVAL) {
						return
					}

					{
						buf := new(bytes.Buffer)
						err := gob.NewEncoder(buf).Encode(tc.a)
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Encode: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							goto checkSEncode
						}
						if buf.String() != tc.gob {
							t.Errorf("Encode:\n%q\nwant:\n%q", buf.String(), tc.gob)
						}
					}

				checkSEncode:
					{
						buf := new(bytes.Buffer)
						err := gob.NewEncoder(buf).Encode(&sCheck{tc.a, syscall.MS_MGC_VAL})
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Encode: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							return
						}
						if buf.String() != tc.sGob {
							t.Errorf("Encode:\n%q\nwant:\n%q", buf.String(), tc.sGob)
						}
					}
				})

				t.Run("decode", func(t *testing.T) {
					{
						var gotA *Absolute
						err := gob.NewDecoder(strings.NewReader(tc.gob)).Decode(&gotA)
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Decode: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							goto checkSDecode
						}
						if !reflect.DeepEqual(tc.a, gotA) {
							t.Errorf("Decode: %#v, want %#v", tc.a, gotA)
						}
					}

				checkSDecode:
					{
						var gotSCheck sCheck
						err := gob.NewDecoder(strings.NewReader(tc.sGob)).Decode(&gotSCheck)
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Decode: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							return
						}
						want := sCheck{tc.a, syscall.MS_MGC_VAL}
						if !reflect.DeepEqual(gotSCheck, want) {
							t.Errorf("Decode: %#v, want %#v", gotSCheck, want)
						}
					}
				})

			})

			t.Run("json", func(t *testing.T) {
				t.Run("marshal", func(t *testing.T) {
					// marshal is unchecked
					if errors.Is(tc.wantErr, syscall.EINVAL) {
						return
					}

					{
						d, err := json.Marshal(tc.a)
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Marshal: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							goto checkSMarshal
						}
						if string(d) != tc.json {
							t.Errorf("Marshal:\n%s\nwant:\n%s", string(d), tc.json)
						}
					}

				checkSMarshal:
					{
						d, err := json.Marshal(&sCheck{tc.a, syscall.MS_MGC_VAL})
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Marshal: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							return
						}
						if string(d) != tc.sJson {
							t.Errorf("Marshal:\n%s\nwant:\n%s", string(d), tc.sJson)
						}
					}
				})

				t.Run("unmarshal", func(t *testing.T) {
					{
						var gotA *Absolute
						err := json.Unmarshal([]byte(tc.json), &gotA)
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Unmarshal: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							goto checkSUnmarshal
						}
						if !reflect.DeepEqual(tc.a, gotA) {
							t.Errorf("Unmarshal: %#v, want %#v", tc.a, gotA)
						}
					}

				checkSUnmarshal:
					{
						var gotSCheck sCheck
						err := json.Unmarshal([]byte(tc.sJson), &gotSCheck)
						if !errors.Is(err, tc.wantErr) {
							t.Errorf("Unmarshal: error = %v, want %v", err, tc.wantErr)
						}
						if tc.wantErr != nil {
							return
						}
						want := sCheck{tc.a, syscall.MS_MGC_VAL}
						if !reflect.DeepEqual(gotSCheck, want) {
							t.Errorf("Unmarshal: %#v, want %#v", gotSCheck, want)
						}
					}
				})
			})
		})
	}

	t.Run("json passthrough", func(t *testing.T) {
		wantErr := "invalid character ':' looking for beginning of value"
		if err := new(Absolute).UnmarshalJSON([]byte(":3")); err == nil || err.Error() != wantErr {
			t.Errorf("UnmarshalJSON: error = %v, want %s", err, wantErr)
		}
	})
}

func TestAbsoluteWrap(t *testing.T) {
	t.Run("join", func(t *testing.T) {
		want := "/etc/nix/nix.conf"
		if got := MustAbs("/etc").Append("nix", "nix.conf"); got.String() != want {
			t.Errorf("Append: %q, want %q", got, want)
		}
	})

	t.Run("dir", func(t *testing.T) {
		want := "/"
		if got := MustAbs("/etc").Dir(); got.String() != want {
			t.Errorf("Dir: %q, want %q", got, want)
		}
	})

	t.Run("sort", func(t *testing.T) {
		want := []*Absolute{MustAbs("/etc"), MustAbs("/proc"), MustAbs("/sys")}
		got := []*Absolute{MustAbs("/proc"), MustAbs("/sys"), MustAbs("/etc")}
		SortAbs(got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("SortAbs: %#v, want %#v", got, want)
		}
	})

	t.Run("compact", func(t *testing.T) {
		want := []*Absolute{MustAbs("/etc"), MustAbs("/proc"), MustAbs("/sys")}
		if got := CompactAbs([]*Absolute{MustAbs("/etc"), MustAbs("/proc"), MustAbs("/proc"), MustAbs("/sys")}); !reflect.DeepEqual(got, want) {
			t.Errorf("CompactAbs: %#v, want %#v", got, want)
		}
	})
}
