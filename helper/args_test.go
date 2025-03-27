package helper_test

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"

	"git.gensokyo.uk/security/fortify/helper"
)

func TestArgsString(t *testing.T) {
	wantString := strings.Join(wantArgs, " ")
	if got := argsWt.(fmt.Stringer).String(); got != wantString {
		t.Errorf("String: %q, want %q",
			got, wantString)
	}
}

func TestNewCheckedArgs(t *testing.T) {
	args := []string{"\x00"}
	if _, err := helper.NewCheckedArgs(args); !errors.Is(err, syscall.EINVAL) {
		t.Errorf("NewCheckedArgs: error = %v, wantErr %v",
			err, syscall.EINVAL)
	}

	t.Run("must panic", func(t *testing.T) {
		badPayload := []string{"\x00"}
		defer func() {
			wantPanic := "invalid argument"
			if r := recover(); r != wantPanic {
				t.Errorf("MustNewCheckedArgs: panic = %v, wantPanic %v",
					r, wantPanic)
			}
		}()
		helper.MustNewCheckedArgs(badPayload)
	})
}
