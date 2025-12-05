package pipewire_test

import (
	"fmt"
	"reflect"
	"strconv"
	. "syscall"
	"testing"

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
			{Key: pipewire.PW_KEY_REMOTE_INTENTION, Value: "manager"},
			{Key: pipewire.PW_KEY_APP_NAME, Value: "pw-container"},
			{Key: pipewire.PW_KEY_APP_PROCESS_BINARY, Value: "pw-container"},
			{Key: pipewire.PW_KEY_APP_LANGUAGE, Value: "en_US.UTF-8"},
			{Key: pipewire.PW_KEY_APP_PROCESS_ID, Value: "1443"},
			{Key: pipewire.PW_KEY_APP_PROCESS_USER, Value: "alice"},
			{Key: pipewire.PW_KEY_APP_PROCESS_HOST, Value: "nixos"},
			{Key: pipewire.PW_KEY_APP_PROCESS_SESSION_ID, Value: "1"},
			{Key: pipewire.PW_KEY_WINDOW_X11_DISPLAY, Value: ":0"},
			{Key: "cpu.vm.name", Value: "qemu"},
			{Key: "log.level", Value: "0"},
			{Key: pipewire.PW_KEY_CPU_MAX_ALIGN, Value: "32"},
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
			{Key: pipewire.PW_KEY_CORE_VERSION, Value: "1.4.7"},
			{Key: pipewire.PW_KEY_CORE_NAME, Value: "pipewire-alice-1443"},
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
			{Key: pipewire.PW_KEY_CONFIG_NAME, Value: "pipewire.conf"},
			{Key: pipewire.PW_KEY_APP_NAME, Value: "pipewire"},
			{Key: pipewire.PW_KEY_APP_PROCESS_BINARY, Value: "pipewire"},
			{Key: pipewire.PW_KEY_APP_LANGUAGE, Value: "en_US.UTF-8"},
			{Key: pipewire.PW_KEY_APP_PROCESS_ID, Value: "1446"},
			{Key: pipewire.PW_KEY_APP_PROCESS_USER, Value: "alice"},
			{Key: pipewire.PW_KEY_APP_PROCESS_HOST, Value: "nixos"},
			{Key: pipewire.PW_KEY_WINDOW_X11_DISPLAY, Value: ":0"},
			{Key: "cpu.vm.name", Value: "qemu"},
			{Key: "link.max-buffers", Value: "16"},
			{Key: pipewire.PW_KEY_CORE_DAEMON, Value: "true"},
			{Key: pipewire.PW_KEY_CORE_NAME, Value: "pipewire-0"},
			{Key: "default.clock.min-quantum", Value: "1024"},
			{Key: pipewire.PW_KEY_CPU_MAX_ALIGN, Value: "32"},
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
			{Key: pipewire.PW_KEY_OBJECT_ID, Value: "0"},
			{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "0"}},
	}

	wantClient0 := pipewire.Client{
		Info: &pipewire.ClientInfo{
			ID:         34,
			ChangeMask: pipewire.PW_CLIENT_CHANGE_MASK_PROPS,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_PROTOCOL, Value: "protocol-native"},
				{Key: pipewire.PW_KEY_CORE_NAME, Value: "pipewire-alice-1443"},
				{Key: pipewire.PW_KEY_SEC_SOCKET, Value: "pipewire-0-manager"},
				{Key: pipewire.PW_KEY_SEC_PID, Value: "1443"},
				{Key: pipewire.PW_KEY_SEC_UID, Value: "1000"},
				{Key: pipewire.PW_KEY_SEC_GID, Value: "100"},
				{Key: pipewire.PW_KEY_MODULE_ID, Value: "2"},
				{Key: pipewire.PW_KEY_OBJECT_ID, Value: "34"},
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "34"},
				{Key: pipewire.PW_KEY_REMOTE_INTENTION, Value: "manager"},
				{Key: pipewire.PW_KEY_APP_NAME, Value: "pw-container"},
				{Key: pipewire.PW_KEY_APP_PROCESS_BINARY, Value: "pw-container"},
				{Key: pipewire.PW_KEY_APP_LANGUAGE, Value: "en_US.UTF-8"},
				{Key: pipewire.PW_KEY_APP_PROCESS_ID, Value: "1443"},
				{Key: pipewire.PW_KEY_APP_PROCESS_USER, Value: "alice"},
				{Key: pipewire.PW_KEY_APP_PROCESS_HOST, Value: "nixos"},
				{Key: pipewire.PW_KEY_APP_PROCESS_SESSION_ID, Value: "1"},
				{Key: pipewire.PW_KEY_WINDOW_X11_DISPLAY, Value: ":0"},
				{Key: "cpu.vm.name", Value: "qemu"},
				{Key: "log.level", Value: "0"},
				{Key: pipewire.PW_KEY_CPU_MAX_ALIGN, Value: "32"},
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
				{Key: pipewire.PW_KEY_CORE_VERSION, Value: "1.4.7"},
				{Key: pipewire.PW_KEY_ACCESS, Value: "unrestricted"},
			},
		},
		Properties: pipewire.SPADict{
			{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "34"},
			{Key: pipewire.PW_KEY_MODULE_ID, Value: "2"},
			{Key: pipewire.PW_KEY_PROTOCOL, Value: "protocol-native"},
			{Key: pipewire.PW_KEY_SEC_PID, Value: "1443"},
			{Key: pipewire.PW_KEY_SEC_UID, Value: "1000"},
			{Key: pipewire.PW_KEY_SEC_GID, Value: "100"},
			{Key: pipewire.PW_KEY_SEC_SOCKET, Value: "pipewire-0-manager"},
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
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "0"},
					{Key: pipewire.PW_KEY_CORE_NAME, Value: "pipewire-0"},
				},
			},

			1: {
				ID:          1,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "1"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-rt"},
				},
			},

			3: {
				ID:          3,
				Permissions: pipewire.PW_SECURITY_CONTEXT_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_SecurityContext,
				Version:     pipewire.PW_VERSION_SECURITY_CONTEXT,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "3"},
				},
			},

			2: {
				ID:          2,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "2"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-protocol-native"},
				},
			},

			5: {
				ID:          5,
				Permissions: pipewire.PW_PROFILER_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Profiler,
				Version:     pipewire.PW_VERSION_PROFILER,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "5"},
				},
			},

			4: {
				ID:          4,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "4"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-profiler"},
				},
			},

			6: {
				ID:          6,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "6"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-metadata"},
				},
			},

			7: {
				ID:          7,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "7"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "6"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "metadata"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: pipewire.PW_TYPE_INTERFACE_Metadata},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "3"},
				},
			},

			8: {
				ID:          8,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "8"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-device-factory"},
				},
			},

			9: {
				ID:          9,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "9"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "8"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "spa-device-factory"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: pipewire.PW_TYPE_INTERFACE_Device},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "3"},
				},
			},

			10: {
				ID:          10,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "10"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-node-factory"},
				},
			},

			11: {
				ID:          11,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "11"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "10"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "spa-node-factory"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: pipewire.PW_TYPE_INTERFACE_Node},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "3"},
				},
			},

			12: {
				ID:          12,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "12"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-node"},
				},
			},

			13: {
				ID:          13,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "13"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "12"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "client-node"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: pipewire.PW_TYPE_INTERFACE_ClientNode},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "6"},
				},
			},

			14: {
				ID:          14,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "14"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-device"},
				},
			},

			15: {
				ID:          15,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "15"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "14"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "client-device"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: "Spa:Pointer:Interface:Device"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "0"},
				},
			},

			16: {
				ID:          16,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "16"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-portal"},
				},
			},

			17: {
				ID:          17,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "17"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-access"},
				},
			},

			18: {
				ID:          18,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "18"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-adapter"},
				},
			},

			19: {
				ID:          19,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "19"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "18"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "adapter"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: pipewire.PW_TYPE_INTERFACE_Node},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "3"},
				},
			},

			20: {
				ID:          20,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "20"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-link-factory"},
				},
			},

			21: {
				ID:          21,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "21"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "20"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "link-factory"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: pipewire.PW_TYPE_INTERFACE_Link},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "3"},
				},
			},

			22: {
				ID:          22,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "22"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-session-manager"},
				},
			},

			23: {
				ID:          23,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "23"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "22"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "client-endpoint"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: "PipeWire:Interface:ClientEndpoint"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "0"},
				},
			},

			24: {
				ID:          24,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "24"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "22"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "client-session"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: "PipeWire:Interface:ClientSession"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "0"},
				},
			},

			25: {
				ID:          25,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "25"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "22"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "session"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: "PipeWire:Interface:Session"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "0"},
				},
			},

			26: {
				ID:          26,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "26"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "22"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "endpoint"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: "PipeWire:Interface:Endpoint"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "0"},
				},
			},

			27: {
				ID:          27,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "27"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "22"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "endpoint-stream"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: "PipeWire:Interface:EndpointStream"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "0"},
				},
			},

			28: {
				ID:          28,
				Permissions: pipewire.PW_FACTORY_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Factory,
				Version:     pipewire.PW_VERSION_FACTORY,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "28"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "22"},
					{Key: pipewire.PW_KEY_FACTORY_NAME, Value: "endpoint-link"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_NAME, Value: "PipeWire:Interface:EndpointLink"},
					{Key: pipewire.PW_KEY_FACTORY_TYPE_VERSION, Value: "0"},
				},
			},

			29: {
				ID:          29,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "29"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-x11-bell"},
				},
			},

			30: {
				ID:          30,
				Permissions: pipewire.PW_MODULE_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Module,
				Version:     pipewire.PW_VERSION_MODULE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "30"},
					{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-jackdbus-detect"},
				},
			},

			31: {
				ID:          31,
				Permissions: pipewire.PW_PERM_RWXM, // why is this not PW_NODE_PERM_MASK?
				Type:        pipewire.PW_TYPE_INTERFACE_Node,
				Version:     pipewire.PW_VERSION_NODE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "31"},
					{Key: pipewire.PW_KEY_FACTORY_ID, Value: "11"},
					{Key: pipewire.PW_KEY_PRIORITY_DRIVER, Value: "200000"},
					{Key: pipewire.PW_KEY_NODE_NAME, Value: "Dummy-Driver"},
				},
			},

			32: {
				ID:          32,
				Permissions: pipewire.PW_PERM_RWXM, // why is this not PW_NODE_PERM_MASK?
				Type:        pipewire.PW_TYPE_INTERFACE_Node,
				Version:     pipewire.PW_VERSION_NODE,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "32"},
					{Key: pipewire.PW_KEY_FACTORY_ID, Value: "11"},
					{Key: pipewire.PW_KEY_PRIORITY_DRIVER, Value: "190000"},
					{Key: pipewire.PW_KEY_NODE_NAME, Value: "Freewheel-Driver"},
				},
			},

			33: {
				ID:          33,
				Permissions: pipewire.PW_METADATA_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Metadata,
				Version:     pipewire.PW_VERSION_METADATA,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "33"},
					{Key: "metadata.name", Value: "settings"},
				},
			},

			34: {
				ID:          34,
				Permissions: pipewire.PW_CLIENT_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Client,
				Version:     pipewire.PW_VERSION_CLIENT,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "34"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "2"},
					{Key: pipewire.PW_KEY_PROTOCOL, Value: "protocol-native"},
					{Key: pipewire.PW_KEY_SEC_PID, Value: "1443"},
					{Key: pipewire.PW_KEY_SEC_UID, Value: "1000"},
					{Key: pipewire.PW_KEY_SEC_GID, Value: "100"},
					{Key: pipewire.PW_KEY_SEC_SOCKET, Value: "pipewire-0-manager"},
					{Key: pipewire.PW_KEY_ACCESS, Value: "unrestricted"},
					{Key: pipewire.PW_KEY_APP_NAME, Value: "pw-container"},
				},
			},

			35: {
				ID:          35,
				Permissions: pipewire.PW_CLIENT_PERM_MASK,
				Type:        pipewire.PW_TYPE_INTERFACE_Client,
				Version:     pipewire.PW_VERSION_CLIENT,
				Properties: &pipewire.SPADict{
					{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "35"},
					{Key: pipewire.PW_KEY_MODULE_ID, Value: "2"},
					{Key: pipewire.PW_KEY_PROTOCOL, Value: "protocol-native"},
					{Key: pipewire.PW_KEY_SEC_PID, Value: "1447"},
					{Key: pipewire.PW_KEY_SEC_UID, Value: "1000"},
					{Key: pipewire.PW_KEY_SEC_GID, Value: "100"},
					{Key: pipewire.PW_KEY_SEC_SOCKET, Value: "pipewire-0-manager"},
					{Key: pipewire.PW_KEY_ACCESS, Value: "unrestricted"},
					{Key: pipewire.PW_KEY_APP_NAME, Value: "WirePlumber"},
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
		{Key: pipewire.PW_KEY_SEC_ENGINE, Value: "org.flatpak"},
		{Key: pipewire.PW_KEY_ACCESS, Value: "restricted"},
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
	flags int
	files []int
	errno Errno
}

// stubUnixConn implements [pipewire.Conn] and checks the behaviour of [pipewire.Context].
type stubUnixConn struct {
	samples []stubUnixConnSample
	current int
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

func (conn *stubUnixConn) Recvmsg(p, oob []byte, flags int) (n, oobn, recvflags int, err error) {
	var (
		sample  *stubUnixConnSample
		wantOOB []byte
	)
	sample, wantOOB, err = conn.nextSample(SYS_RECVMSG)
	if err != nil {
		return
	}

	if n = copy(p, sample.iovec); n != len(sample.iovec) {
		err = fmt.Errorf("insufficient iovec size %d, want at least %d", len(p), len(sample.iovec))
		return
	}
	if oobn = copy(oob, wantOOB); oobn != len(wantOOB) {
		err = fmt.Errorf("insufficient oob size %d, want at least %d", len(oob), len(wantOOB))
		return
	}
	if flags != sample.flags {
		err = fmt.Errorf("flags = %#x, want %#x", flags, sample.flags)
		return
	}

	recvflags = MSG_CMSG_CLOEXEC
	if sample.errno != 0 {
		err = sample.errno
		if n != 0 {
			panic("invalid recvmsg: n = " + strconv.Itoa(n))
		}
		n = -1
	}
	return
}

func (conn *stubUnixConn) Sendmsg(p, oob []byte, flags int) (n int, err error) {
	var (
		sample  *stubUnixConnSample
		wantOOB []byte
	)
	sample, wantOOB, err = conn.nextSample(SYS_SENDMSG)
	if err != nil {
		return
	}

	if string(p) != sample.iovec {
		err = fmt.Errorf("iovec: %#v, want %#v", p, []byte(sample.iovec))
		return
	}
	if string(oob[:len(wantOOB)]) != string(wantOOB) {
		err = fmt.Errorf("oob: %#v, want %#v", oob[:len(wantOOB)], wantOOB)
		return
	}
	if flags != sample.flags {
		err = fmt.Errorf("flags = %#x, want %#x", flags, sample.flags)
		return
	}

	n = len(sample.iovec)
	if sample.errno != 0 {
		err = sample.errno
	}
	return
}

func (conn *stubUnixConn) Close() error {
	if conn.current != len(conn.samples) {
		return fmt.Errorf("consumed %d samples, want %d", conn.current, len(conn.samples))
	}
	return nil
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
		{"UnexpectedFilesError", pipewire.UnexpectedFilesError(1 << 4), "server message headers claim to have sent more files than actually received"},
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
