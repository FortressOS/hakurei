package state

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"hakurei.app/hst"
)

func TestEntryHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data [entryHeaderSize]byte
		et   hst.Enablement
		err  error
	}{
		{"complement mismatch", [entryHeaderSize]byte{0x00, 0xff, 0xca, 0xfe, 0x00, 0x00,
			0x0a, 0xf6}, 0,
			errors.New("header enablement value is inconsistent")},
		{"unexpected revision", [entryHeaderSize]byte{0x00, 0xff, 0xca, 0xfe, 0xff, 0xff}, 0,
			errors.New("unexpected revision ffff")},
		{"invalid header", [entryHeaderSize]byte{0x00, 0xfe, 0xca, 0xfe}, 0,
			errors.New("invalid header 00fecafe")},

		{"success high", [entryHeaderSize]byte{0x00, 0xff, 0xca, 0xfe, 0x00, 0x00,
			0xff, 0x00}, 0xff, nil},
		{"success", [entryHeaderSize]byte{0x00, 0xff, 0xca, 0xfe, 0x00, 0x00,
			0x09, 0xf6}, hst.EWayland | hst.EPulse, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("encode", func(t *testing.T) {
				if tc.err != nil {
					return
				}
				t.Parallel()

				if got := entryHeaderEncode(tc.et); *got != tc.data {
					t.Errorf("entryHeaderEncode: %x, want %x", *got, tc.data)
				}

				t.Run("write", func(t *testing.T) {
					var buf bytes.Buffer
					if err := entryWriteHeader(&buf, tc.et); err != nil {
						t.Fatalf("entryWriteHeader: error = %v", err)
					}
					if got := ([entryHeaderSize]byte)(buf.Bytes()); got != tc.data {
						t.Errorf("entryWriteHeader: %x, want %x", got, tc.data)
					}
				})
			})

			t.Run("decode", func(t *testing.T) {
				t.Parallel()

				got, err := entryHeaderDecode(&tc.data)
				if !reflect.DeepEqual(err, tc.err) {
					t.Fatalf("entryHeaderDecode: error = %#v, want %#v", err, tc.err)
				}
				if err != nil {
					return
				}
				if got != tc.et {
					t.Errorf("entryHeaderDecode: et = %q, want %q", got, tc.et)
				}

				if got, err = entryReadHeader(bytes.NewReader(tc.data[:])); err != nil {
					t.Fatalf("entryReadHeader: error = %#v", err)
				} else if got != tc.et {
					t.Errorf("entryReadHeader: et = %q, want %q", got, tc.et)
				}
			})
		})
	}
}

func TestEntrySizeError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want string
	}{
		{"size only", &EntrySizeError{Size: 0xdeadbeef},
			`state entry file is too short`},
		{"full", &EntrySizeError{Name: "nonexistent", Size: 0xdeadbeef},
			`state entry file "nonexistent" is too short`},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error: %s, want %s", got, tc.want)
			}
		})
	}
}

func TestEntryCheckFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		fi   os.FileInfo
		err  error
	}{
		{"dir", &stubFi{name: "dir", isDir: true},
			syscall.EISDIR},
		{"short", stubFi{name: "short", size: 8},
			&EntrySizeError{Name: "short", Size: 8}},
		{"success", stubFi{size: 9}, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := entryCheckFile(tc.fi); !reflect.DeepEqual(err, tc.err) {
				t.Errorf("entryCheckFile: error = %#v, want %#v", err, tc.err)
			}
		})
	}
}

func TestEntryReadHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		newR func() io.Reader
		err  error
	}{
		{"eof", func() io.Reader { return bytes.NewReader([]byte{}) }, io.EOF},
		{"short", func() io.Reader { return bytes.NewReader([]byte{0}) }, &EntrySizeError{Size: 1}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := entryReadHeader(tc.newR()); !reflect.DeepEqual(err, tc.err) {
				t.Errorf("entryReadHeader: error = %#v, want %#v", err, tc.err)
			}
		})
	}
}

// stubFi partially implements [os.FileInfo] using hardcoded values.
type stubFi struct {
	name  string
	size  int64
	isDir bool
}

func (fi stubFi) Name() string {
	if fi.name == "" {
		panic("unreachable")
	}
	return fi.name
}

func (fi stubFi) Size() int64 {
	if fi.size < 0 {
		panic("unreachable")
	}
	return fi.size
}

func (fi stubFi) IsDir() bool { return fi.isDir }

func (fi stubFi) Mode() fs.FileMode  { panic("unreachable") }
func (fi stubFi) ModTime() time.Time { panic("unreachable") }
func (fi stubFi) Sys() any           { panic("unreachable") }
