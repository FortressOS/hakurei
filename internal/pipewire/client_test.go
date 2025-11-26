package pipewire_test

import (
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestClientInfo(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.ClientInfo, *pipewire.ClientInfo]{
		{"sample", []byte(recvmsg00Message02POD), pipewire.ClientInfo{
			ID:         34,
			ChangeMask: pipewire.PW_CLIENT_CHANGE_MASK_PROPS,
			Props: &pipewire.SPADict{NItems: 9, Items: []pipewire.SPADictItem{
				{Key: "pipewire.protocol", Value: "protocol-native"},
				{Key: "core.name", Value: "pipewire-0"},
				{Key: "pipewire.sec.socket", Value: "pipewire-0-manager"},
				{Key: "pipewire.sec.pid", Value: "1443"},
				{Key: "pipewire.sec.uid", Value: "1000"},
				{Key: "pipewire.sec.gid", Value: "100"},
				{Key: "module.id", Value: "2"},
				{Key: "object.id", Value: "34"},
				{Key: "object.serial", Value: "34"},
			}}}, nil},
	}.run(t)
}

func TestClientUpdateProperties(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.ClientUpdateProperties, *pipewire.ClientUpdateProperties]{
		{"sample", []byte(sendmsg00Message01POD), pipewire.ClientUpdateProperties{Props: &pipewire.SPADict{NItems: 0x1e, Items: []pipewire.SPADictItem{
			{Key: "remote.intention", Value: "manager"},
			{Key: "application.name", Value: "pw-container"},
			{Key: "application.process.binary", Value: "pw-container"},
			{Key: "application.language", Value: "en_US.UTF-8"},
			{Key: "application.process.id", Value: "1443"},
			{Key: "application.process.user", Value: "alice"},
			{Key: "application.process.host", Value: "nixos"},
			{Key: "application.process.session-id", Value: "1"},
			{Key: "window.x11.display", Value: ":0"},
			{Key: "cpu.vm.name", Value: "qemu"},
			{Key: "log.level", Value: "0"},
			{Key: "cpu.max-align", Value: "32"},
			{Key: "default.clock.rate", Value: "48000"},
			{Key: "default.clock.quantum", Value: "1024"},
			{Key: "default.clock.min-quantum", Value: "32"},
			{Key: "default.clock.max-quantum", Value: "2048"},
			{Key: "default.clock.quantum-limit", Value: "8192"},
			{Key: "default.clock.quantum-floor", Value: "4"},
			{Key: "default.video.width", Value: "640"},
			{Key: "default.video.height", Value: "480"},
			{Key: "default.video.rate.num", Value: "25"},
			{Key: "default.video.rate.denom", Value: "1"},
			{Key: "clock.power-of-two-quantum", Value: "true"},
			{Key: "link.max-buffers", Value: "64"},
			{Key: "mem.warn-mlock", Value: "false"},
			{Key: "mem.allow-mlock", Value: "true"},
			{Key: "settings.check-quantum", Value: "false"},
			{Key: "settings.check-rate", Value: "false"},
			{Key: "core.version", Value: "1.4.7"},
			{Key: "core.name", Value: "pipewire-alice-1443"},
		}}}, nil},
	}.run(t)
}
