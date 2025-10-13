package helper_test

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"

	"hakurei.app/helper"
)

func TestArgsString(t *testing.T) {
	t.Parallel()
	wantString := strings.Join(wantArgs, " ")
	if got := argsWt.(fmt.Stringer).String(); got != wantString {
		t.Errorf("String: %q, want %q", got, wantString)
	}
}

func TestNewCheckedArgs(t *testing.T) {
	t.Parallel()

	args := []string{"\x00"}
	if _, err := helper.NewCheckedArgs(args...); !errors.Is(err, syscall.EINVAL) {
		t.Errorf("NewCheckedArgs: error = %v, wantErr %v", err, syscall.EINVAL)
	}

	t.Run("must panic", func(t *testing.T) {
		t.Parallel()

		badPayload := []string{"\x00"}
		defer func() {
			wantPanic := "invalid argument"
			if r := recover(); r != wantPanic {
				t.Errorf("MustNewCheckedArgs: panic = %v, wantPanic %v", r, wantPanic)
			}
		}()
		helper.MustNewCheckedArgs(badPayload...)
	})
}
