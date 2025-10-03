package app

import (
	"strings"
	"testing"
)

func TestIsValidUsername(t *testing.T) {
	t.Run("long", func(t *testing.T) {
		if isValidUsername(strings.Repeat("a", sysconf(_SC_LOGIN_NAME_MAX))) {
			t.Errorf("isValidUsername unexpected true")
		}
	})

	t.Run("regexp", func(t *testing.T) {
		if isValidUsername("0") {
			t.Errorf("isValidUsername unexpected true")
		}
	})

	t.Run("valid", func(t *testing.T) {
		if !isValidUsername("alice") {
			t.Errorf("isValidUsername unexpected false")
		}
	})
}
