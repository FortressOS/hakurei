package container_test

import (
	"errors"
	"os"
	"slices"
	"strconv"
	"syscall"
	"testing"

	"hakurei.app/container"
)

func TestSetupReceive(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		const key = "TEST_ENV_NOT_SET"
		{
			v, ok := os.LookupEnv(key)
			t.Cleanup(func() {
				if ok {
					if err := os.Setenv(key, v); err != nil {
						t.Fatalf("Setenv: error = %v", err)
					}
				} else {
					if err := os.Unsetenv(key); err != nil {
						t.Fatalf("Unsetenv: error = %v", err)
					}
				}
			})
		}

		if _, err := container.Receive(key, nil, nil); !errors.Is(err, container.ErrNotSet) {
			t.Errorf("Receive: error = %v, want %v", err, container.ErrNotSet)
		}
	})

	t.Run("format", func(t *testing.T) {
		const key = "TEST_ENV_FORMAT"
		t.Setenv(key, "")

		if _, err := container.Receive(key, nil, nil); !errors.Is(err, container.ErrFdFormat) {
			t.Errorf("Receive: error = %v, want %v", err, container.ErrFdFormat)
		}
	})

	t.Run("range", func(t *testing.T) {
		const key = "TEST_ENV_RANGE"
		t.Setenv(key, "-1")

		if _, err := container.Receive(key, nil, nil); !errors.Is(err, syscall.EBADF) {
			t.Errorf("Receive: error = %v, want %v", err, syscall.EBADF)
		}
	})

	t.Run("setup receive", func(t *testing.T) {
		check := func(t *testing.T, useNilFp bool) {
			const key = "TEST_SETUP_RECEIVE"
			payload := []int{syscall.MS_MGC_VAL, syscall.MS_MGC_MSK, syscall.MS_ASYNC, syscall.MS_ACTIVE}

			encoderDone := make(chan error, 1)
			extraFiles := make([]*os.File, 0, 1)
			if fd, encoder, err := container.Setup(&extraFiles); err != nil {
				t.Fatalf("Setup: error = %v", err)
			} else if fd != 3 {
				t.Fatalf("Setup: fd = %d, want 3", fd)
			} else {
				go func() { encoderDone <- encoder.Encode(payload) }()
			}

			if len(extraFiles) != 1 {
				t.Fatalf("extraFiles: len = %v, want 1", len(extraFiles))
			}

			var dupFd int
			if fd, err := syscall.Dup(int(extraFiles[0].Fd())); err != nil {
				t.Fatalf("Dup: error = %v", err)
			} else {
				syscall.CloseOnExec(fd)
				dupFd = fd
				t.Setenv(key, strconv.Itoa(fd))
			}

			var (
				gotPayload []int
				fp         **os.File
			)
			if !useNilFp {
				fp = new(*os.File)
			}
			var closeFile func() error
			if f, err := container.Receive(key, &gotPayload, fp); err != nil {
				t.Fatalf("Receive: error = %v", err)
			} else {
				closeFile = f

				if !slices.Equal(payload, gotPayload) {
					t.Errorf("Receive: %#v, want %#v", gotPayload, payload)
				}
			}
			if !useNilFp {
				if name := (*fp).Name(); name != "setup" {
					t.Errorf("Name: %s, want setup", name)
				}
				if fd := int((*fp).Fd()); fd != dupFd {
					t.Errorf("Fd: %d, want %d", fd, dupFd)
				}
			}

			if err := <-encoderDone; err != nil {
				t.Errorf("Encode: error = %v", err)
			}

			if closeFile != nil {
				if err := closeFile(); err != nil {
					t.Errorf("Close: error = %v", err)
				}
			}
		}

		t.Run("fp", func(t *testing.T) { check(t, false) })
		t.Run("nil", func(t *testing.T) { check(t, true) })
	})
}
