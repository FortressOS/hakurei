package pipewire_test

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
	. "syscall"
	"testing"
	"time"

	"hakurei.app/container/stub"
	"hakurei.app/internal/pipewire"
)

func TestContext(t *testing.T) {
	t.Parallel()

	var (
		// Underlying connection stub holding test data.
		conn = stubUnixConn{samples: []stubUnixConnSample{
			{SYS_SENDMSG, samplePWContainer00, MSG_DONTWAIT | MSG_NOSIGNAL, nil, 0},
			{SYS_RECVMSG, samplePWContainer01, MSG_DONTWAIT | MSG_CMSG_CLOEXEC, nil, 0},
			{SYS_RECVMSG, "", MSG_DONTWAIT | MSG_CMSG_CLOEXEC, nil, EAGAIN},
			{SYS_SENDMSG, samplePWContainer03, MSG_DONTWAIT | MSG_NOSIGNAL, nil, 0},
			{SYS_RECVMSG, samplePWContainer04, MSG_DONTWAIT | MSG_CMSG_CLOEXEC, nil, 0},
			{SYS_RECVMSG, "", MSG_DONTWAIT | MSG_CMSG_CLOEXEC, nil, EAGAIN},
			{SYS_SENDMSG, samplePWContainer06, MSG_DONTWAIT | MSG_NOSIGNAL, []int{20, 21}, 0},
			{SYS_RECVMSG, samplePWContainer07, MSG_DONTWAIT | MSG_CMSG_CLOEXEC, nil, 0},
			{SYS_RECVMSG, "", MSG_DONTWAIT | MSG_CMSG_CLOEXEC, nil, EAGAIN},
		}}

		// Context instance under testing.
		ctx = pipewire.MustNew(&conn, pipewire.SPADict{
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
		})
	)

	var registry *pipewire.Registry
	const wantRegistryId = 2
	if r, err := ctx.GetRegistry(); err != nil {
		t.Fatalf("GetRegistry: error = %v", err)
	} else {
		if r.ID != wantRegistryId {
			t.Fatalf("GetRegistry: ID = %d, want %d", r.ID, wantRegistryId)
		}
		registry = r
	}
	if err := ctx.GetCore().Sync(); err != nil {
		t.Fatalf("Sync: error = %v", err)
	}

	wantCoreInfo0 := pipewire.CoreInfo{
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
	}

	wantClient0 := pipewire.Client{
		Info: &pipewire.ClientInfo{
			ID:         34,
			ChangeMask: pipewire.PW_CLIENT_CHANGE_MASK_PROPS,
			Properties: &pipewire.SPADict{
				{Key: "pipewire.protocol", Value: "protocol-native"},
				{Key: "core.name", Value: "pipewire-alice-1443"},
				{Key: "pipewire.sec.socket", Value: "pipewire-0-manager"},
				{Key: "pipewire.sec.pid", Value: "1443"},
				{Key: "pipewire.sec.uid", Value: "1000"},
				{Key: "pipewire.sec.gid", Value: "100"},
				{Key: "module.id", Value: "2"},
				{Key: "object.id", Value: "34"},
				{Key: "object.serial", Value: "34"},
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
				{Key: "pipewire.access", Value: "unrestricted"},
			},
		},
		Properties: pipewire.SPADict{
			{Key: "object.serial", Value: "34"},
			{Key: "module.id", Value: "2"},
			{Key: "pipewire.protocol", Value: "protocol-native"},
			{Key: "pipewire.sec.pid", Value: "1443"},
			{Key: "pipewire.sec.uid", Value: "1000"},
			{Key: "pipewire.sec.gid", Value: "100"},
			{Key: "pipewire.sec.socket", Value: "pipewire-0-manager"},
		},
	}

	wantRegistry0 := pipewire.Registry{
		ID: wantRegistryId,
		Objects: map[pipewire.Int]pipewire.RegistryGlobal{
			pipewire.PW_ID_CORE: {
				ID:          pipewire.PW_ID_CORE,
				Permissions: pipewire.PW_CORE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Core,
				Version:     pipewire.PW_VERSION_CORE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "0"},
					{Key: "core.name", Value: "pipewire-0"},
				},
			},

			1: {
				ID:          1,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "1"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-rt"},
				},
			},

			3: {
				ID:          3,
				Permissions: pipewire.PW_SECURITY_CONTEXT_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_SecurityContext,
				Version:     pipewire.PW_VERSION_SECURITY_CONTEXT,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "3"},
				},
			},

			2: {
				ID:          2,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "2"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-protocol-native"},
				},
			},

			5: {
				ID:          5,
				Permissions: pipewire.PW_PROFILER_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Profiler,
				Version:     pipewire.PW_VERSION_PROFILER,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "5"},
				},
			},

			4: {
				ID:          4,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "4"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-profiler"},
				},
			},

			6: {
				ID:          6,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "6"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-metadata"},
				},
			},

			7: {
				ID:          7,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "7"},
					{Key: "module.id", Value: "6"},
					{Key: "factory.name", Value: "metadata"},
					{Key: "factory.type.name", Value: pipewire.PW_TYPE_INTERFACE_Metadata},
					{Key: "factory.type.version", Value: "3"},
				},
			},

			8: {
				ID:          8,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "8"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-device-factory"},
				},
			},

			9: {
				ID:          9,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "9"},
					{Key: "module.id", Value: "8"},
					{Key: "factory.name", Value: "spa-device-factory"},
					{Key: "factory.type.name", Value: pipewire.PW_TYPE_INTERFACE_Device},
					{Key: "factory.type.version", Value: "3"},
				},
			},

			10: {
				ID:          10,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "10"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-node-factory"},
				},
			},

			11: {
				ID:          11,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "11"},
					{Key: "module.id", Value: "10"},
					{Key: "factory.name", Value: "spa-node-factory"},
					{Key: "factory.type.name", Value: pipewire.PW_TYPE_INTERFACE_Node},
					{Key: "factory.type.version", Value: "3"},
				},
			},

			12: {
				ID:          12,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "12"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-node"},
				},
			},

			13: {
				ID:          13,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "13"},
					{Key: "module.id", Value: "12"},
					{Key: "factory.name", Value: "client-node"},
					{Key: "factory.type.name", Value: pipewire.PW_TYPE_INTERFACE_ClientNode},
					{Key: "factory.type.version", Value: "6"},
				},
			},

			14: {
				ID:          14,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "14"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-device"},
				},
			},

			15: {
				ID:          15,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "15"},
					{Key: "module.id", Value: "14"},
					{Key: "factory.name", Value: "client-device"},
					{Key: "factory.type.name", Value: "Spa:Pointer:Interface:Device"},
					{Key: "factory.type.version", Value: "0"},
				},
			},

			16: {
				ID:          16,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "16"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-portal"},
				},
			},

			17: {
				ID:          17,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "17"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-access"},
				},
			},

			18: {
				ID:          18,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "18"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-adapter"},
				},
			},

			19: {
				ID:          19,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "19"},
					{Key: "module.id", Value: "18"},
					{Key: "factory.name", Value: "adapter"},
					{Key: "factory.type.name", Value: pipewire.PW_TYPE_INTERFACE_Node},
					{Key: "factory.type.version", Value: "3"},
				},
			},

			20: {
				ID:          20,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "20"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-link-factory"},
				},
			},

			21: {
				ID:          21,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "21"},
					{Key: "module.id", Value: "20"},
					{Key: "factory.name", Value: "link-factory"},
					{Key: "factory.type.name", Value: pipewire.PW_TYPE_INTERFACE_Link},
					{Key: "factory.type.version", Value: "3"},
				},
			},

			22: {
				ID:          22,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "22"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-session-manager"},
				},
			},

			23: {
				ID:          23,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "23"},
					{Key: "module.id", Value: "22"},
					{Key: "factory.name", Value: "client-endpoint"},
					{Key: "factory.type.name", Value: "PipeWire:Interface:ClientEndpoint"},
					{Key: "factory.type.version", Value: "0"},
				},
			},

			24: {
				ID:          24,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "24"},
					{Key: "module.id", Value: "22"},
					{Key: "factory.name", Value: "client-session"},
					{Key: "factory.type.name", Value: "PipeWire:Interface:ClientSession"},
					{Key: "factory.type.version", Value: "0"},
				},
			},

			25: {
				ID:          25,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "25"},
					{Key: "module.id", Value: "22"},
					{Key: "factory.name", Value: "session"},
					{Key: "factory.type.name", Value: "PipeWire:Interface:Session"},
					{Key: "factory.type.version", Value: "0"},
				},
			},

			26: {
				ID:          26,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "26"},
					{Key: "module.id", Value: "22"},
					{Key: "factory.name", Value: "endpoint"},
					{Key: "factory.type.name", Value: "PipeWire:Interface:Endpoint"},
					{Key: "factory.type.version", Value: "0"},
				},
			},

			27: {
				ID:          27,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "27"},
					{Key: "module.id", Value: "22"},
					{Key: "factory.name", Value: "endpoint-stream"},
					{Key: "factory.type.name", Value: "PipeWire:Interface:EndpointStream"},
					{Key: "factory.type.version", Value: "0"},
				},
			},

			28: {
				ID:          28,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "28"},
					{Key: "module.id", Value: "22"},
					{Key: "factory.name", Value: "endpoint-link"},
					{Key: "factory.type.name", Value: "PipeWire:Interface:EndpointLink"},
					{Key: "factory.type.version", Value: "0"},
				},
			},

			29: {
				ID:          29,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "29"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-x11-bell"},
				},
			},

			30: {
				ID:          30,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "30"},
					{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-jackdbus-detect"},
				},
			},

			31: {
				ID:          31,
				Permissions: pipewire.PW_PERM_RWXM, // why is this not PW_NODE_PERM_MASK?
				Type:        pipewire.PW_TYPE_INTERFACE_Node,
				Version:     pipewire.PW_VERSION_NODE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "31"},
					{Key: "factory.id", Value: "11"},
					{Key: "priority.driver", Value: "200000"},
					{Key: "node.name", Value: "Dummy-Driver"},
				},
			},

			32: {
				ID:          32,
				Permissions: pipewire.PW_PERM_RWXM, // why is this not PW_NODE_PERM_MASK?
				Type:        pipewire.PW_TYPE_INTERFACE_Node,
				Version:     pipewire.PW_VERSION_NODE,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "32"},
					{Key: "factory.id", Value: "11"},
					{Key: "priority.driver", Value: "190000"},
					{Key: "node.name", Value: "Freewheel-Driver"},
				},
			},

			33: {
				ID:          33,
				Permissions: pipewire.PW_METADATA_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Metadata,
				Version:     pipewire.PW_VERSION_METADATA,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "33"},
					{Key: "metadata.name", Value: "settings"},
				},
			},

			34: {
				ID:          34,
				Permissions: pipewire.PW_CLIENT_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Client,
				Version:     pipewire.PW_VERSION_CLIENT,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "34"},
					{Key: "module.id", Value: "2"},
					{Key: "pipewire.protocol", Value: "protocol-native"},
					{Key: "pipewire.sec.pid", Value: "1443"},
					{Key: "pipewire.sec.uid", Value: "1000"},
					{Key: "pipewire.sec.gid", Value: "100"},
					{Key: "pipewire.sec.socket", Value: "pipewire-0-manager"},
					{Key: "pipewire.access", Value: "unrestricted"},
					{Key: "application.name", Value: "pw-container"},
				},
			},

			35: {
				ID:          35,
				Permissions: pipewire.PW_CLIENT_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Client,
				Version:     pipewire.PW_VERSION_CLIENT,
				Properties: &pipewire.SPADict{
					{Key: "object.serial", Value: "35"},
					{Key: "module.id", Value: "2"},
					{Key: "pipewire.protocol", Value: "protocol-native"},
					{Key: "pipewire.sec.pid", Value: "1447"},
					{Key: "pipewire.sec.uid", Value: "1000"},
					{Key: "pipewire.sec.gid", Value: "100"},
					{Key: "pipewire.sec.socket", Value: "pipewire-0-manager"},
					{Key: "pipewire.access", Value: "unrestricted"},
					{Key: "application.name", Value: "WirePlumber"},
				},
			},
		},
	}

	if coreInfo := ctx.GetCore().Info; !reflect.DeepEqual(coreInfo, &wantCoreInfo0) {
		t.Fatalf("New: CoreInfo = %s, want %s", mustMarshalJSON(coreInfo), mustMarshalJSON(&wantCoreInfo0))
	}
	if client := ctx.GetClient(); !reflect.DeepEqual(client, &wantClient0) {
		t.Fatalf("New: Client = %s, want %s", mustMarshalJSON(client), mustMarshalJSON(&wantClient0))
	}
	if registry.ID != wantRegistry0.ID {
		t.Fatalf("GetRegistry: ID = %d, want %d", registry.ID, wantRegistry0.ID)
	}
	if !reflect.DeepEqual(registry.Objects, wantRegistry0.Objects) {
		t.Fatalf("GetRegistry: Objects = %s, want %s", mustMarshalJSON(registry.Objects), mustMarshalJSON(wantRegistry0.Objects))
	}

	var securityContext *pipewire.SecurityContext
	const wantSecurityContextId = 3
	if c, err := registry.GetSecurityContext(); err != nil {
		t.Fatalf("GetSecurityContext: error = %v", err)
	} else {
		if c.ID != wantSecurityContextId {
			t.Fatalf("GetSecurityContext: ID = %d, want %d", c.ID, wantSecurityContextId)
		}
		securityContext = c
	}
	if err := ctx.Roundtrip(); err != nil {
		t.Fatalf("Roundtrip: error = %v", err)
	}

	// none of these should change
	if coreInfo := ctx.GetCore().Info; !reflect.DeepEqual(coreInfo, &wantCoreInfo0) {
		t.Fatalf("Roundtrip: CoreInfo = %s, want %s", mustMarshalJSON(coreInfo), mustMarshalJSON(&wantCoreInfo0))
	}
	if client := ctx.GetClient(); !reflect.DeepEqual(client, &wantClient0) {
		t.Fatalf("Roundtrip: Client = %s, want %s", mustMarshalJSON(client), mustMarshalJSON(&wantClient0))
	}
	if registry.ID != wantRegistry0.ID {
		t.Fatalf("Roundtrip: ID = %d, want %d", registry.ID, wantRegistry0.ID)
	}
	if !reflect.DeepEqual(registry.Objects, wantRegistry0.Objects) {
		t.Fatalf("Roundtrip: Objects = %s, want %s", mustMarshalJSON(registry.Objects), mustMarshalJSON(wantRegistry0.Objects))
	}

	if err := securityContext.Create(21, 20, pipewire.SPADict{
		{Key: "pipewire.sec.engine", Value: "org.flatpak"},
		{Key: "pipewire.access", Value: "restricted"},
	}); err != nil {
		t.Fatalf("SecurityContext.Create: error = %v", err)
	}
	if err := ctx.GetCore().Sync(); err != nil {
		t.Fatalf("Sync: error = %v", err)
	}

	// none of these should change
	if coreInfo := ctx.GetCore().Info; !reflect.DeepEqual(coreInfo, &wantCoreInfo0) {
		t.Fatalf("Roundtrip: CoreInfo = %s, want %s", mustMarshalJSON(coreInfo), mustMarshalJSON(&wantCoreInfo0))
	}
	if client := ctx.GetClient(); !reflect.DeepEqual(client, &wantClient0) {
		t.Fatalf("Roundtrip: Client = %s, want %s", mustMarshalJSON(client), mustMarshalJSON(&wantClient0))
	}
	if registry.ID != wantRegistry0.ID {
		t.Fatalf("Roundtrip: ID = %d, want %d", registry.ID, wantRegistry0.ID)
	}
	if !reflect.DeepEqual(registry.Objects, wantRegistry0.Objects) {
		t.Fatalf("Roundtrip: Objects = %s, want %s", mustMarshalJSON(registry.Objects), mustMarshalJSON(wantRegistry0.Objects))
	}

	if err := ctx.Close(); err != nil {
		t.Fatalf("Close: error = %v", err)
	}
}

// stubUnixConnSample is sample data held by stubUnixConn.
type stubUnixConnSample struct {
	nr    uintptr
	iovec string
	flags uintptr
	files []int
	errno Errno
}

// stubUnixConn implements [pipewire.Conn] and checks the behaviour of [pipewire.Context].
type stubUnixConn struct {
	samples []stubUnixConnSample
	current int

	deadline *time.Time
}

// checkDeadline checks whether deadline is set reasonably.
func (conn *stubUnixConn) checkDeadline() error {
	if conn.deadline == nil || conn.deadline.Before(time.Now()) {
		return fmt.Errorf("invalid deadline %v", conn.deadline)
	}
	conn.deadline = nil
	return nil
}

// nextSample returns the current sample and increments the counter.
func (conn *stubUnixConn) nextSample(nr uintptr) (sample *stubUnixConnSample, wantOOB []byte, err error) {
	sample = &conn.samples[conn.current]
	conn.current++
	if sample.nr != nr {
		err = fmt.Errorf("unexpected syscall %d", SYS_SENDMSG)
		return
	}
	if len(sample.files) > 0 {
		wantOOB = UnixRights(sample.files...)
	}

	return
}

func (conn *stubUnixConn) ReadMsgUnix(b, oob []byte) (n, oobn, flags int, addr *net.UnixAddr, err error) {
	if conn.samples[conn.current-1].nr == SYS_SENDMSG {
		if err = conn.checkDeadline(); err != nil {
			return
		}
	}

	var (
		sample  *stubUnixConnSample
		wantOOB []byte
	)
	sample, wantOOB, err = conn.nextSample(SYS_RECVMSG)
	if err != nil {
		return
	}

	if copy(b, sample.iovec) != len(sample.iovec) {
		err = fmt.Errorf("insufficient iovec size %d, want at least %d", len(b), len(sample.iovec))
	}
	if copy(oob, wantOOB) != len(wantOOB) {
		err = fmt.Errorf("insufficient oob size %d, want at least %d", len(oob), len(wantOOB))
	}

	if sample.errno != 0 && sample.errno != EAGAIN {
		err = sample.errno
	}
	return len(sample.iovec), len(wantOOB), MSG_CMSG_CLOEXEC, nil, nil
}

func (conn *stubUnixConn) WriteMsgUnix(b, oob []byte, addr *net.UnixAddr) (n, oobn int, err error) {
	if addr != nil {
		err = fmt.Errorf("WriteMsgUnix called with non-nil addr: %#v", addr)
		return
	}
	if err = conn.checkDeadline(); err != nil {
		return
	}

	var (
		sample  *stubUnixConnSample
		wantOOB []byte
	)
	sample, wantOOB, err = conn.nextSample(SYS_SENDMSG)
	if err != nil {
		return
	}

	if string(b) != sample.iovec {
		err = fmt.Errorf("iovec: %#v, want %#v", b, []byte(sample.iovec))
		return
	}
	if string(oob[:len(wantOOB)]) != string(wantOOB) {
		err = fmt.Errorf("oob: %#v, want %#v", oob[:len(wantOOB)], wantOOB)
		return
	}
	return len(sample.iovec), len(wantOOB), nil
}

func (conn *stubUnixConn) SetDeadline(t time.Time) error { conn.deadline = &t; return nil }

func (conn *stubUnixConn) Close() error {
	if conn.current != len(conn.samples) {
		return fmt.Errorf("consumed %d samples, want %d", conn.current, len(conn.samples))
	}
	return nil
}

var (
	//go:embed testdata/pw-container-00-sendmsg
	samplePWContainer00 string
	//go:embed testdata/pw-container-01-recvmsg
	samplePWContainer01 string
	//go:embed testdata/pw-container-03-sendmsg
	samplePWContainer03 string
	//go:embed testdata/pw-container-04-recvmsg
	samplePWContainer04 string
	//go:embed testdata/pw-container-06-sendmsg
	samplePWContainer06 string
	//go:embed testdata/pw-container-07-recvmsg
	samplePWContainer07 string

	// samplePWContainer is a collection of messages from the pw-container sample.
	samplePWContainer = [...][][3][]byte{
		splitMessages(samplePWContainer00),
		splitMessages(samplePWContainer01),
		nil,
		splitMessages(samplePWContainer03),
		splitMessages(samplePWContainer04),
		nil,
		splitMessages(samplePWContainer06),
		splitMessages(samplePWContainer07),
		nil,
	}
)

// splitMessages splits concatenated messages into groups of
// header, payload, footer of each individual message.
// splitMessages panics on any decoding error.
func splitMessages(iovec string) (messages [][3][]byte) {
	data := []byte(iovec)
	messages = make([][3][]byte, 0, 1<<7)

	var header pipewire.Header
	for len(data) != 0 {
		if err := header.UnmarshalBinary(data[:pipewire.SizeHeader]); err != nil {
			panic(err)
		}
		size := pipewire.SizePrefix + binary.NativeEndian.Uint32(data[pipewire.SizeHeader:])
		messages = append(messages, [3][]byte{
			data[:pipewire.SizeHeader],
			data[pipewire.SizeHeader : pipewire.SizeHeader+size],
			data[pipewire.SizeHeader+size : pipewire.SizeHeader+header.Size],
		})
		data = data[pipewire.SizeHeader+header.Size:]
	}
	return
}

func TestContextErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want string
	}{
		{"ProxyConsumeError invalid", pipewire.ProxyConsumeError{}, "invalid proxy consume error"},
		{"ProxyConsumeError single", pipewire.ProxyConsumeError{
			stub.UniqueError(0),
		}, "unique error 0 injected by the test suite"},
		{"ProxyConsumeError multiple", pipewire.ProxyConsumeError{
			stub.UniqueError(1),
			stub.UniqueError(2),
			stub.UniqueError(3),
			stub.UniqueError(4),
			stub.UniqueError(5),
			stub.UniqueError(6),
			stub.UniqueError(7),
		}, "unique error 1 injected by the test suite; 7 additional proxy errors occurred after this point"},

		{"ProxyFatalError", &pipewire.ProxyFatalError{
			Err: stub.UniqueError(8),
		}, "unique error 8 injected by the test suite"},
		{"ProxyFatalError proxy errors", &pipewire.ProxyFatalError{
			Err:       stub.UniqueError(9),
			ProxyErrs: make([]error, 1<<4),
		}, "unique error 9 injected by the test suite; 16 additional proxy errors occurred before this point"},

		{"UnexpectedFileCountError", &pipewire.UnexpectedFileCountError{0, -1}, "received -1 files instead of the expected 0"},
		{"UnacknowledgedProxyError", make(pipewire.UnacknowledgedProxyError, 1<<4), "server did not acknowledge 16 proxies"},
		{"DanglingFilesError", make(pipewire.DanglingFilesError, 1<<4), "received 16 dangling files"},
		{"UnexpectedFilesError", pipewire.UnexpectedFilesError(1 << 4), "server message headers claim to have sent more than 16 files"},
		{"UnexpectedSequenceError", pipewire.UnexpectedSequenceError(1 << 4), "unexpected seq 16"},
		{"UnsupportedFooterOpcodeError", pipewire.UnsupportedFooterOpcodeError(1 << 4), "unsupported footer opcode 16"},

		{"RoundtripUnexpectedEOFError ErrRoundtripEOFHeader", pipewire.ErrRoundtripEOFHeader, "unexpected EOF decoding message header"},
		{"RoundtripUnexpectedEOFError ErrRoundtripEOFBody", pipewire.ErrRoundtripEOFBody, "unexpected EOF establishing message body bounds"},
		{"RoundtripUnexpectedEOFError ErrRoundtripEOFFooter", pipewire.ErrRoundtripEOFFooter, "unexpected EOF establishing message footer bounds"},
		{"RoundtripUnexpectedEOFError ErrRoundtripEOFFooterOpcode", pipewire.ErrRoundtripEOFFooterOpcode, "unexpected EOF decoding message footer opcode"},
		{"RoundtripUnexpectedEOFError invalid", pipewire.RoundtripUnexpectedEOFError(0xbad), "unexpected EOF"},

		{"UnsupportedOpcodeError", &pipewire.UnsupportedOpcodeError{
			Opcode:    0xff,
			Interface: pipewire.PW_TYPE_INFO_INTERFACE_BASE + "Invalid",
		}, "unsupported PipeWire:Interface:Invalid opcode 255"},

		{"UnknownIdError", &pipewire.UnknownIdError{
			Id:   -1,
			Data: "\x00",
		}, "unknown proxy id -1"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error: %q, want %q", got, tc.want)
			}
		})
	}
}
