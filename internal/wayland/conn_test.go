package wayland

import (
	"errors"
	"os"
	"reflect"
	"syscall"
	"testing"

	"hakurei.app/container/check"
)

func TestSecurityContextClose(t *testing.T) {
	t.Parallel()

	if err := (*SecurityContext)(nil).Close(); !reflect.DeepEqual(err, os.ErrInvalid) {
		t.Fatalf("Close: error = %v", err)
	}

	var ctx SecurityContext
	if err := syscall.Pipe2(ctx.closeFds[0:], syscall.O_CLOEXEC); err != nil {
		t.Fatalf("Pipe: error = %v", err)
	}
	t.Cleanup(func() { _ = syscall.Close(ctx.closeFds[0]); _ = syscall.Close(ctx.closeFds[1]) })

	if err := ctx.Close(); err != nil {
		t.Fatalf("Close: error = %v", err)
	}

	wantErr := errors.Join(syscall.EBADF, syscall.EBADF)
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
