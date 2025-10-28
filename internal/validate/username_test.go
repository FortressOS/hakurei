package validate_test

import (
	"strings"
	"testing"

	"hakurei.app/internal/validate"
)

func TestIsValidUsername(t *testing.T) {
	t.Parallel()

	t.Run("long", func(t *testing.T) {
		if validate.IsValidUsername(strings.Repeat("a", validate.Sysconf(validate.SC_LOGIN_NAME_MAX))) {
			t.Errorf("IsValidUsername unexpected true")
		}
	})

	t.Run("regexp", func(t *testing.T) {
		if validate.IsValidUsername("0") {
			t.Errorf("IsValidUsername unexpected true")
		}
	})

	t.Run("valid", func(t *testing.T) {
		if !validate.IsValidUsername("alice") {
			t.Errorf("IsValidUsername unexpected false")
		}
	})
}
