package helper_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"git.ophivana.moe/cat/fortify/helper"
)

func Test_argsFD_String(t *testing.T) {
	argsOnce.Do(prepareArgs)

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
}
