package pipewire_test

import (
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestCoreHello(t *testing.T) {
	encodingTestCases[pipewire.CoreHello, *pipewire.CoreHello]{
		{"sample", []byte{
			0x10, 0, 0, 0,
			0xe, 0, 0, 0,
			4, 0, 0, 0,
			4, 0, 0, 0,
			4, 0, 0, 0,
			0, 0, 0, 0,
		}, pipewire.CoreHello{
			Version: pipewire.PW_VERSION_CORE,
		}, nil},
	}.run(t)
}
