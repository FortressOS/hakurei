package container

import (
	"errors"
	"io"
	"math"
	"os"
	"path"
	"reflect"
	"syscall"
	"testing"
	"unsafe"

	"hakurei.app/container/vfs"
)

func TestToSysroot(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{"", "/sysroot"},
		{"/", "/sysroot"},
		{"//etc///", "/sysroot/etc"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toSysroot(tc.name); got != tc.want {
				t.Errorf("toSysroot: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestToHost(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{"", "/host"},
		{"/", "/host"},
		{"//etc///", "/host/etc"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toHost(tc.name); got != tc.want {
				t.Errorf("toHost: %q, want %q", got, tc.want)
			}
		})
	}
}

// InternalToHostOvlEscape exports toHost passed to EscapeOverlayDataSegment.
func InternalToHostOvlEscape(s string) string { return EscapeOverlayDataSegment(toHost(s)) }

func TestCreateFile(t *testing.T) {
	t.Run("nonexistent", func(t *testing.T) {
		if err := createFile(path.Join(Nonexistent, ":3"), 0644, 0755, nil); !os.IsNotExist(err) {
			t.Errorf("createFile: error = %v", err)
		}
		if err := createFile(path.Join(Nonexistent), 0644, 0755, nil); !os.IsNotExist(err) {
			t.Errorf("createFile: error = %v", err)
		}
	})

	t.Run("touch", func(t *testing.T) {
		tempDir := t.TempDir()
		pathname := path.Join(tempDir, "empty")
		if err := createFile(pathname, 0644, 0755, nil); err != nil {
			t.Fatalf("createFile: error = %v", err)
		}
		if d, err := os.ReadFile(pathname); err != nil {
			t.Fatalf("ReadFile: error = %v", err)
		} else if len(d) != 0 {
			t.Fatalf("createFile: %q", string(d))
		}
	})

	t.Run("write", func(t *testing.T) {
		tempDir := t.TempDir()
		pathname := path.Join(tempDir, "zero")
		if err := createFile(pathname, 0644, 0755, []byte{0}); err != nil {
			t.Fatalf("createFile: error = %v", err)
		}
		if d, err := os.ReadFile(pathname); err != nil {
			t.Fatalf("ReadFile: error = %v", err)
		} else if string(d) != "\x00" {
			t.Fatalf("createFile: %q, want %q", string(d), "\x00")
		}
	})
}

func TestEnsureFile(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		if err := ensureFile(path.Join(t.TempDir(), "ensure"), 0644, 0755); err != nil {
			t.Errorf("ensureFile: error = %v", err)
		}
	})

	t.Run("stat", func(t *testing.T) {
		t.Run("inaccessible", func(t *testing.T) {
			tempDir := t.TempDir()
			pathname := path.Join(tempDir, "inaccessible")
			if f, err := os.Create(pathname); err != nil {
				t.Fatalf("Create: error = %v", err)
			} else {
				_ = f.Close()
			}

			if err := os.Chmod(tempDir, 0); err != nil {
				t.Fatalf("Chmod: error = %v", err)
			}
			if err := ensureFile(pathname, 0644, 0755); !errors.Is(err, syscall.EACCES) {
				t.Errorf("ensureFile: error = %v, want %v", err, syscall.EACCES)
			}
			if err := os.Chmod(tempDir, 0755); err != nil {
				t.Fatalf("Chmod: error = %v", err)
			}
		})

		t.Run("directory", func(t *testing.T) {
			if err := ensureFile(t.TempDir(), 0644, 0755); !errors.Is(err, syscall.EISDIR) {
				t.Errorf("ensureFile: error = %v, want %v", err, syscall.EISDIR)
			}
		})

		t.Run("ensure", func(t *testing.T) {
			tempDir := t.TempDir()
			pathname := path.Join(tempDir, "ensure")
			if f, err := os.Create(pathname); err != nil {
				t.Fatalf("Create: error = %v", err)
			} else {
				_ = f.Close()
			}

			if err := ensureFile(pathname, 0644, 0755); err != nil {
				t.Errorf("ensureFile: error = %v", err)
			}
		})
	})
}

func TestProcPaths(t *testing.T) {
	t.Run("host", func(t *testing.T) {
		t.Run("stdout", func(t *testing.T) {
			want := "/host/proc/self/fd/1"
			if got := hostProc.stdout(); got != want {
				t.Errorf("stdout: %q, want %q", got, want)
			}
		})
		t.Run("fd", func(t *testing.T) {
			want := "/host/proc/self/fd/9223372036854775807"
			if got := hostProc.fd(math.MaxInt64); got != want {
				t.Errorf("stdout: %q, want %q", got, want)
			}
		})
	})

	t.Run("mountinfo", func(t *testing.T) {
		t.Run("nonexistent", func(t *testing.T) {
			nonexistentProc := newProcPaths(t.TempDir())
			if err := nonexistentProc.mountinfo(func(*vfs.MountInfoDecoder) error { return syscall.EINVAL }); !os.IsNotExist(err) {
				t.Errorf("mountinfo: error = %v", err)
			}
		})

		t.Run("sample", func(t *testing.T) {
			tempDir := t.TempDir()
			if err := os.MkdirAll(path.Join(tempDir, "proc/self"), 0755); err != nil {
				t.Fatalf("MkdirAll: error = %v", err)
			}

			t.Run("clean", func(t *testing.T) {
				if err := os.WriteFile(path.Join(tempDir, "proc/self/mountinfo"), []byte(`15 20 0:3 / /proc rw,relatime - proc /proc rw
16 20 0:15 / /sys rw,relatime - sysfs /sys rw
17 20 0:5 / /dev rw,relatime - devtmpfs udev rw,size=1983516k,nr_inodes=495879,mode=755`), 0644); err != nil {
					t.Fatalf("WriteFile: error = %v", err)
				}

				var mountInfo *vfs.MountInfo
				if err := newProcPaths(tempDir).mountinfo(func(d *vfs.MountInfoDecoder) error { return d.Decode(&mountInfo) }); err != nil {
					t.Fatalf("mountinfo: error = %v", err)
				}

				wantMountInfo := &vfs.MountInfo{Next: &vfs.MountInfo{Next: &vfs.MountInfo{
					MountInfoEntry: vfs.MountInfoEntry{ID: 17, Parent: 20, Devno: vfs.DevT{0, 5}, Root: "/", Target: "/dev", VfsOptstr: "rw,relatime", OptFields: []string{}, FsType: "devtmpfs", Source: "udev", FsOptstr: "rw,size=1983516k,nr_inodes=495879,mode=755"}},
					MountInfoEntry: vfs.MountInfoEntry{ID: 16, Parent: 20, Devno: vfs.DevT{0, 15}, Root: "/", Target: "/sys", VfsOptstr: "rw,relatime", OptFields: []string{}, FsType: "sysfs", Source: "/sys", FsOptstr: "rw"}},
					MountInfoEntry: vfs.MountInfoEntry{ID: 15, Parent: 20, Devno: vfs.DevT{0, 3}, Root: "/", Target: "/proc", VfsOptstr: "rw,relatime", OptFields: []string{}, FsType: "proc", Source: "/proc", FsOptstr: "rw"},
				}
				if !reflect.DeepEqual(mountInfo, wantMountInfo) {
					t.Errorf("Decode: %#v, want %#v", mountInfo, wantMountInfo)
				}
			})

			t.Run("closed", func(t *testing.T) {
				if err := newProcPaths(tempDir).mountinfo(func(d *vfs.MountInfoDecoder) error {
					v := reflect.ValueOf(d).Elem().FieldByName("s").Elem().FieldByName("r")
					v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr()))
					if f, ok := v.Elem().Interface().(io.ReadCloser); !ok {
						t.Fatal("implementation of bufio.Scanner no longer compatible with this fault injection")
						return syscall.ENOTRECOVERABLE
					} else {
						return f.Close()
					}
				}); !errors.Is(err, os.ErrClosed) {
					t.Errorf("mountinfo: error = %v, want %v", err, os.ErrClosed)
				}
			})

			t.Run("malformed", func(t *testing.T) {
				path.Join(tempDir, "proc/self/mountinfo")
				if err := os.WriteFile(path.Join(tempDir, "proc/self/mountinfo"), []byte{0}, 0644); err != nil {
					t.Fatalf("WriteFile: error = %v", err)
				}

				if err := newProcPaths(tempDir).mountinfo(func(d *vfs.MountInfoDecoder) error { return d.Decode(new(*vfs.MountInfo)) }); !errors.Is(err, vfs.ErrMountInfoFields) {
					t.Fatalf("mountinfo: error = %v, want %v", err, vfs.ErrMountInfoFields)
				}
			})
		})
	})
}
