package verbose_test

import (
	"testing"

	"git.ophivana.moe/security/fortify/internal/verbose"
)

func TestGetSet(t *testing.T) {
	verbose.Set(false)
	if verbose.Get() {
		t.Errorf("Get() = true, want false")
	}

	verbose.Set(true)
	if !verbose.Get() {
		t.Errorf("Get() = false, want true")
	}

}
