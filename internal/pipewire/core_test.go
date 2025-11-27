package pipewire_test

import (
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestFooterCoreGeneration(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.Footer[pipewire.FooterCoreGeneration], *pipewire.Footer[pipewire.FooterCoreGeneration]]{
		{"sample", samplePWContainer[1][0][2], pipewire.Footer[pipewire.FooterCoreGeneration]{
			Opcode:  pipewire.FOOTER_CORE_OPCODE_GENERATION,
			Payload: pipewire.FooterCoreGeneration{RegistryGeneration: 0x22},
		}, nil},

		{"sample*", samplePWContainer[1][5][2], pipewire.Footer[pipewire.FooterCoreGeneration]{
			Opcode:  pipewire.FOOTER_CORE_OPCODE_GENERATION,
			Payload: pipewire.FooterCoreGeneration{RegistryGeneration: 0x23},
		}, nil},
	}.run(t)
}

func TestCoreInfo(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreInfo, *pipewire.CoreInfo]{
		{"sample", samplePWContainer[1][0][1], pipewire.CoreInfo{
			ID:         0,
			Cookie:     -2069267610,
			UserName:   "alice",
			HostName:   "nixos",
			Version:    "1.4.7",
			Name:       "pipewire-0",
			ChangeMask: pipewire.PW_CORE_CHANGE_MASK_PROPS,
			Properties: &pipewire.SPADict{
				{Key: "config.name", Value: "pipewire.conf"},
				{Key: "application.name", Value: "pipewire"},
				{Key: "application.process.binary", Value: "pipewire"},
				{Key: "application.language", Value: "en_US.UTF-8"},
				{Key: "application.process.id", Value: "1446"},
				{Key: "application.process.user", Value: "alice"},
				{Key: "application.process.host", Value: "nixos"},
				{Key: "window.x11.display", Value: ":0"},
				{Key: "cpu.vm.name", Value: "qemu"},
				{Key: "link.max-buffers", Value: "16"},
				{Key: "core.daemon", Value: "true"},
				{Key: "core.name", Value: "pipewire-0"},
				{Key: "default.clock.min-quantum", Value: "1024"},
				{Key: "cpu.max-align", Value: "32"},
				{Key: "default.clock.rate", Value: "48000"},
				{Key: "default.clock.quantum", Value: "1024"},
				{Key: "default.clock.max-quantum", Value: "2048"},
				{Key: "default.clock.quantum-limit", Value: "8192"},
				{Key: "default.clock.quantum-floor", Value: "4"},
				{Key: "default.video.width", Value: "640"},
				{Key: "default.video.height", Value: "480"},
				{Key: "default.video.rate.num", Value: "25"},
				{Key: "default.video.rate.denom", Value: "1"},
				{Key: "log.level", Value: "2"},
				{Key: "clock.power-of-two-quantum", Value: "true"},
				{Key: "mem.warn-mlock", Value: "false"},
				{Key: "mem.allow-mlock", Value: "true"},
				{Key: "settings.check-quantum", Value: "false"},
				{Key: "settings.check-rate", Value: "false"},
				{Key: "object.id", Value: "0"},
				{Key: "object.serial", Value: "0"}},
		}, nil},
	}.run(t)
}

func TestCoreDone(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreDone, *pipewire.CoreDone]{
		{"sample", samplePWContainer[1][5][1], pipewire.CoreDone{
			ID:       -1,
			Sequence: 0,
		}, nil},
	}.run(t)
}

func TestCoreBoundProps(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreBoundProps, *pipewire.CoreBoundProps]{
		{"sample", samplePWContainer[1][1][1], pipewire.CoreBoundProps{
			ID:       pipewire.PW_ID_CLIENT,
			GlobalID: 34,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "34"},
				{Key: "module.id", Value: "2"},
				{Key: "pipewire.protocol", Value: "protocol-native"},
				{Key: "pipewire.sec.pid", Value: "1443"},
				{Key: "pipewire.sec.uid", Value: "1000"},
				{Key: "pipewire.sec.gid", Value: "100"},
				{Key: "pipewire.sec.socket", Value: "pipewire-0-manager"}},
		}, nil},
	}.run(t)
}

func TestCoreHello(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreHello, *pipewire.CoreHello]{
		{"sample", samplePWContainer[0][0][1], pipewire.CoreHello{
			Version: pipewire.PW_VERSION_CORE,
		}, nil},
	}.run(t)
}

func TestCoreSync(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreSync, *pipewire.CoreSync]{
		{"sample", samplePWContainer[0][3][1], pipewire.CoreSync{
			ID:       pipewire.PW_ID_CORE,
			Sequence: pipewire.CoreSyncSequenceOffset + 3,
		}, nil},
	}.run(t)
}

func TestCoreGetRegistry(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreGetRegistry, *pipewire.CoreGetRegistry]{
		{"sample", samplePWContainer[0][2][1], pipewire.CoreGetRegistry{
			Version: pipewire.PW_VERSION_REGISTRY,
			NewID:   2,
		}, nil},
	}.run(t)
}
