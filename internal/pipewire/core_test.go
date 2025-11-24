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

func TestCoreSync(t *testing.T) {
	encodingTestCases[pipewire.CoreSync, *pipewire.CoreSync]{
		{"sample", []byte(sendmsg00Message03POD), pipewire.CoreSync{
			ID:       pipewire.PW_ID_CORE,
			Sequence: pipewire.CoreSyncSequenceOffset + 3,
		}, nil},
	}.run(t)
}

func TestCoreGetRegistry(t *testing.T) {
	encodingTestCases[pipewire.CoreGetRegistry, *pipewire.CoreGetRegistry]{
		{"sample", []byte(sendmsg00Message02POD), pipewire.CoreGetRegistry{
			Version: pipewire.PW_VERSION_REGISTRY,
			NewID:   2,
		}, nil},
	}.run(t)
}
