package pipewire_test

import (
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestSecurityContextCreate(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.SecurityContextCreate, *pipewire.SecurityContextCreate]{
		{"sample", samplePWContainer[6][0][1], pipewire.SecurityContextCreate{
			ListenFd: 1 /* 21: duplicated from listen_fd */, CloseFd: 0, /* 20: duplicated from close_fd */
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_SEC_ENGINE, Value: "org.flatpak"},
				{Key: pipewire.PW_KEY_ACCESS, Value: "restricted"},
			},
		}, nil},
	}.run(t)
}
