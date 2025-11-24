package pipewire_test

import (
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestCoreHello(t *testing.T) {
	encodingTestCases[pipewire.CoreHello, *pipewire.CoreHello]{
		{"sample", []byte(sendmsg00Message00POD), pipewire.CoreHello{
			Version: pipewire.PW_VERSION_CORE,
		}, nil},
	}.run(t)
}
