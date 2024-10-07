package helper_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"git.ophivana.moe/cat/fortify/helper"
)

func Test_argsFD_String(t *testing.T) {
	wantString := strings.Join(want, " ")
	if got := argsWt.(fmt.Stringer).String(); got != wantString {
		t.Errorf("String(): got %v; want %v",
			got, wantString)
	}
}

func TestNewCheckedArgs(t *testing.T) {
	args := []string{"\x00"}
	if _, err := helper.NewCheckedArgs(args); !errors.Is(err, helper.ErrContainsNull) {
		t.Errorf("NewCheckedArgs(%q) error = %v, wantErr %v",
			args,
			err, helper.ErrContainsNull)
	}

	t.Run("must panic", func(t *testing.T) {
		badPayload := []string{"\x00"}
		defer func() {
			wantPanic := "argument contains null character"
			if r := recover(); r != wantPanic {
				t.Errorf("MustNewCheckedArgs(%q) panic = %v, wantPanic %v",
					badPayload,
					r, wantPanic)
			}
		}()
		helper.MustNewCheckedArgs(badPayload)
	})
}
