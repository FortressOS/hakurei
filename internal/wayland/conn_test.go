package wayland

import (
	"errors"
	"os"
	"path"
	"reflect"
	"syscall"
	"testing"

	"hakurei.app/container/check"
)

func TestSecurityContextClose(t *testing.T) {
	// do not parallel: fd test not thread safe

	if err := (*SecurityContext)(nil).Close(); !reflect.DeepEqual(err, os.ErrInvalid) {
		t.Fatalf("Close: error = %v", err)
	}

	var ctx SecurityContext
	if f, err := os.Create(path.Join(t.TempDir(), "remove")); err != nil {
		t.Fatal(err)
	} else {
		ctx.bindPath = check.MustAbs(f.Name())
	}
	if err := syscall.Pipe2(ctx.closeFds[0:], syscall.O_CLOEXEC); err != nil {
		t.Fatalf("Pipe: error = %v", err)
	}
	t.Cleanup(func() { _ = syscall.Close(ctx.closeFds[0]); _ = syscall.Close(ctx.closeFds[1]) })

	if err := ctx.Close(); err != nil {
		t.Fatalf("Close: error = %v", err)
	} else if _, err = os.Stat(ctx.bindPath.String()); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Did not remove %q", ctx.bindPath)
	}

	wantErr := &Error{Cause: RCleanup, Path: ctx.bindPath.String(), Errno: errors.Join(syscall.EBADF, syscall.EBADF, &os.PathError{
		Op:   "remove",
		Path: ctx.bindPath.String(),
		Err:  syscall.ENOENT,
	})}
	if err := ctx.Close(); !reflect.DeepEqual(err, wantErr) {
		t.Fatalf("Close: error = %#v, want %#v", err, wantErr)
	}
}

func TestNewEnsure(t *testing.T) {
	existingDirPath := check.MustAbs(t.TempDir()).Append("dir")
	if err := os.MkdirAll(existingDirPath.String(), 0700); err != nil {
		t.Fatal(err)
	}
	nonexistent := check.MustAbs("/proc/nonexistent")

	wantErr := &Error{RCreate, existingDirPath.String(), nonexistent.String(), &os.PathError{
		Op:   "open",
		Path: existingDirPath.String(),
		Err:  syscall.EISDIR,
	}}
	if _, err := New(
		nonexistent,
		existingDirPath, "", "",
	); !reflect.DeepEqual(err, wantErr) {
		t.Fatalf("New: error = %#v, want %#v", err, wantErr)
	}
}
