package pipewire_test

import (
	"testing"

	"hakurei.app/internal/pipewire"
)

func TestFooterCoreGeneration(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.Footer[pipewire.FooterCoreGeneration], *pipewire.Footer[pipewire.FooterCoreGeneration]]{

		/* recvmsg 0 */

		{"sample0", samplePWContainer[1][0][2], pipewire.Footer[pipewire.FooterCoreGeneration]{
			Opcode:  pipewire.FOOTER_CORE_OPCODE_GENERATION,
			Payload: pipewire.FooterCoreGeneration{RegistryGeneration: 0x22},
		}, nil},

		{"sample1", samplePWContainer[1][5][2], pipewire.Footer[pipewire.FooterCoreGeneration]{
			Opcode:  pipewire.FOOTER_CORE_OPCODE_GENERATION,
			Payload: pipewire.FooterCoreGeneration{RegistryGeneration: 0x23},
		}, nil},

		// happens on the last message, client footer sent in the next roundtrip
		{"sample2", samplePWContainer[1][42][2], pipewire.Footer[pipewire.FooterCoreGeneration]{
			Opcode:  pipewire.FOOTER_CORE_OPCODE_GENERATION,
			Payload: pipewire.FooterCoreGeneration{RegistryGeneration: 0x24},
		}, nil},
	}.run(t)

	encodingTestCases[pipewire.Footer[pipewire.FooterClientGeneration], *pipewire.Footer[pipewire.FooterClientGeneration]]{

		/* sendmsg 1 */

		{"sample0", samplePWContainer[3][0][2], pipewire.Footer[pipewire.FooterClientGeneration]{
			Opcode: pipewire.FOOTER_CORE_OPCODE_GENERATION,
			// triggered by difference in sample1, sample0 is overwritten in the same roundtrip
			Payload: pipewire.FooterClientGeneration{ClientGeneration: 0x23},
		}, nil},

		/* sendmsg 2 */

		{"sample1", samplePWContainer[6][0][2], pipewire.Footer[pipewire.FooterClientGeneration]{
			// triggered by difference in sample2, last footer in the previous roundtrip
			Opcode:  pipewire.FOOTER_CORE_OPCODE_GENERATION,
			Payload: pipewire.FooterClientGeneration{ClientGeneration: 0x24},
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
		}, nil},
	}.run(t)
}

func TestCoreDone(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreDone, *pipewire.CoreDone]{
		{"sample0", samplePWContainer[1][5][1], pipewire.CoreDone{
			ID:       -1,
			Sequence: 0,
		}, nil},

		// matches the Core::Sync sample
		{"sample1", samplePWContainer[1][41][1], pipewire.CoreDone{
			ID:       0,
			Sequence: pipewire.CoreSyncSequenceOffset + 3,
		}, nil},

		// matches the second Core::Sync sample
		{"sample2", samplePWContainer[7][0][1], pipewire.CoreDone{
			ID:       0,
			Sequence: pipewire.CoreSyncSequenceOffset + 6,
		}, nil},
	}.run(t)
}

func TestCorePing(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CorePing, *pipewire.CorePing]{
		// handmade sample
		{"sample", []byte{
			/* size: rest of data */ 0x20, 0, 0, 0,
			/* type: Struct */ byte(pipewire.SPA_TYPE_Struct), 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ byte(pipewire.SPA_TYPE_Int), 0, 0, 0,
			/* value: -1 */ 0xff, 0xff, 0xff, 0xff,
			/* padding */ 0, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ byte(pipewire.SPA_TYPE_Int), 0, 0, 0,
			/* value: 0 */ 0, 0, 0, 0,
			/* padding */ 0, 0, 0, 0,
		}, pipewire.CorePing{
			ID:       -1,
			Sequence: 0,
		}, nil},
	}.run(t)
}

func TestCoreError(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreError, *pipewire.CoreError]{
		// captured from pw-cli
		{"pw-cli", []byte{
			/* size: rest of data */ 0x58, 0, 0, 0,
			/* type: Struct */ 0xe, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ 4, 0, 0, 0,
			/* value: 2 */ 2, 0, 0, 0,
			/* padding */ 0, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ 4, 0, 0, 0,
			/* value: 0x67 */ 0x67, 0, 0, 0,
			/* padding */ 0, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ 4, 0, 0, 0,
			/* value: -1 */ 0xff, 0xff, 0xff, 0xff,
			/* padding */ 0, 0, 0, 0,

			/* size: 0x1b bytes */ 0x1b, 0, 0, 0,
			/*type: String*/ 8, 0, 0, 0,

			// value: "no permission to destroy 0\x00"
			0x6e, 0x6f, 0x20, 0x70,
			0x65, 0x72, 0x6d, 0x69,
			0x73, 0x73, 0x69, 0x6f,
			0x6e, 0x20, 0x74, 0x6f,
			0x20, 0x64, 0x65, 0x73,
			0x74, 0x72, 0x6f, 0x79,
			0x20, 0x30, 0,

			/* padding */ 0, 0, 0, 0, 0,
		}, pipewire.CoreError{
			ID:       2,
			Sequence: 0x67,
			Result:   -1,
			Message:  "no permission to destroy 0",
		}, nil},
	}.run(t)
}

func TestCoreBoundProps(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreBoundProps, *pipewire.CoreBoundProps]{

		/* recvmsg 0 */

		{"sample0", samplePWContainer[1][1][1], pipewire.CoreBoundProps{
			ID:       pipewire.PW_ID_CLIENT,
			GlobalID: 34,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "34"},
				{Key: pipewire.PW_KEY_MODULE_ID, Value: "2"},
				{Key: pipewire.PW_KEY_PROTOCOL, Value: "protocol-native"},
				{Key: pipewire.PW_KEY_SEC_PID, Value: "1443"},
				{Key: pipewire.PW_KEY_SEC_UID, Value: "1000"},
				{Key: pipewire.PW_KEY_SEC_GID, Value: "100"},
				{Key: pipewire.PW_KEY_SEC_SOCKET, Value: "pipewire-0-manager"}},
		}, nil},

		/* recvmsg 1 */

		{"sample1", samplePWContainer[4][0][1], pipewire.CoreBoundProps{
			ID:       3,
			GlobalID: 3,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "3"},
			},
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
		{"sample0", samplePWContainer[0][3][1], pipewire.CoreSync{
			ID:       0,
			Sequence: pipewire.CoreSyncSequenceOffset + 3,
		}, nil},

		{"sample1", samplePWContainer[6][1][1], pipewire.CoreSync{
			ID:       0,
			Sequence: pipewire.CoreSyncSequenceOffset + 6,
		}, nil},
	}.run(t)
}

func TestCorePong(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CorePong, *pipewire.CorePong]{
		// handmade sample
		{"sample", []byte{
			/* size: rest of data */ 0x20, 0, 0, 0,
			/* type: Struct */ byte(pipewire.SPA_TYPE_Struct), 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ byte(pipewire.SPA_TYPE_Int), 0, 0, 0,
			/* value: -1 */ 0xff, 0xff, 0xff, 0xff,
			/* padding */ 0, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ byte(pipewire.SPA_TYPE_Int), 0, 0, 0,
			/* value: 0 */ 0, 0, 0, 0,
			/* padding */ 0, 0, 0, 0,
		}, pipewire.CorePong{
			ID:       -1,
			Sequence: 0,
		}, nil},
	}.run(t)
}

func TestCoreGetRegistry(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.CoreGetRegistry, *pipewire.CoreGetRegistry]{
		{"sample", samplePWContainer[0][2][1], pipewire.CoreGetRegistry{
			Version: pipewire.PW_VERSION_REGISTRY,
			// this ends up as the Id of PW_TYPE_INTERFACE_Registry
			NewID: 2,
		}, nil},
	}.run(t)
}

func TestRegistryGlobal(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.RegistryGlobal, *pipewire.RegistryGlobal]{
		{"sample0", samplePWContainer[1][6][1], pipewire.RegistryGlobal{
			ID:          pipewire.PW_ID_CORE,
			Permissions: pipewire.PW_CORE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Core,
			Version:     pipewire.PW_VERSION_CORE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "0"},
				{Key: pipewire.PW_KEY_CORE_NAME, Value: "pipewire-0"},
			},
		}, nil},

		{"sample1", samplePWContainer[1][7][1], pipewire.RegistryGlobal{
			ID:          1,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "1"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-rt"},
			},
		}, nil},

		{"sample2", samplePWContainer[1][8][1], pipewire.RegistryGlobal{
			ID:          3,
			Permissions: pipewire.PW_SECURITY_CONTEXT_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_SecurityContext,
			Version:     pipewire.PW_VERSION_SECURITY_CONTEXT,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "3"},
			},
		}, nil},

		{"sample3", samplePWContainer[1][9][1], pipewire.RegistryGlobal{
			ID:          2,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "2"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-protocol-native"},
			},
		}, nil},

		{"sample4", samplePWContainer[1][10][1], pipewire.RegistryGlobal{
			ID:          5,
			Permissions: pipewire.PW_PROFILER_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Profiler,
			Version:     pipewire.PW_VERSION_PROFILER,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "5"},
			},
		}, nil},

		{"sample5", samplePWContainer[1][11][1], pipewire.RegistryGlobal{
			ID:          4,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "4"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-profiler"},
			},
		}, nil},

		{"sample6", samplePWContainer[1][12][1], pipewire.RegistryGlobal{
			ID:          6,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "6"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-metadata"},
			},
		}, nil},

		{"sample7", samplePWContainer[1][13][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample8", samplePWContainer[1][14][1], pipewire.RegistryGlobal{
			ID:          8,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "8"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-device-factory"},
			},
		}, nil},

		{"sample9", samplePWContainer[1][15][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample10", samplePWContainer[1][16][1], pipewire.RegistryGlobal{
			ID:          10,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "10"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-node-factory"},
			},
		}, nil},

		{"sample11", samplePWContainer[1][17][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample12", samplePWContainer[1][18][1], pipewire.RegistryGlobal{
			ID:          12,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "12"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-node"},
			},
		}, nil},

		{"sample13", samplePWContainer[1][19][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample14", samplePWContainer[1][20][1], pipewire.RegistryGlobal{
			ID:          14,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "14"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-device"},
			},
		}, nil},

		{"sample15", samplePWContainer[1][21][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample16", samplePWContainer[1][22][1], pipewire.RegistryGlobal{
			ID:          16,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "16"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-portal"},
			},
		}, nil},

		{"sample17", samplePWContainer[1][23][1], pipewire.RegistryGlobal{
			ID:          17,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "17"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-access"},
			},
		}, nil},

		{"sample18", samplePWContainer[1][24][1], pipewire.RegistryGlobal{
			ID:          18,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "18"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-adapter"},
			},
		}, nil},

		{"sample19", samplePWContainer[1][25][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample20", samplePWContainer[1][26][1], pipewire.RegistryGlobal{
			ID:          20,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "20"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-link-factory"},
			},
		}, nil},

		{"sample21", samplePWContainer[1][27][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample22", samplePWContainer[1][28][1], pipewire.RegistryGlobal{
			ID:          22,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "22"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-session-manager"},
			},
		}, nil},

		{"sample23", samplePWContainer[1][29][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample24", samplePWContainer[1][30][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample25", samplePWContainer[1][31][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample26", samplePWContainer[1][32][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample27", samplePWContainer[1][33][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample28", samplePWContainer[1][34][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample29", samplePWContainer[1][35][1], pipewire.RegistryGlobal{
			ID:          29,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "29"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-x11-bell"},
			},
		}, nil},

		{"sample30", samplePWContainer[1][36][1], pipewire.RegistryGlobal{
			ID:          30,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "30"},
				{Key: pipewire.PW_KEY_MODULE_NAME, Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-jackdbus-detect"},
			},
		}, nil},

		{"sample31", samplePWContainer[1][37][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample32", samplePWContainer[1][38][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample33", samplePWContainer[1][39][1], pipewire.RegistryGlobal{
			ID:          33,
			Permissions: pipewire.PW_METADATA_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Metadata,
			Version:     pipewire.PW_VERSION_METADATA,
			Properties: &pipewire.SPADict{
				{Key: pipewire.PW_KEY_OBJECT_SERIAL, Value: "33"},
				{Key: "metadata.name", Value: "settings"},
			},
		}, nil},

		{"sample34", samplePWContainer[1][40][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample35", samplePWContainer[1][42][1], pipewire.RegistryGlobal{
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
		}, nil},
	}.run(t)
}

func TestRegistryBind(t *testing.T) {
	t.Parallel()

	encodingTestCases[pipewire.RegistryBind, *pipewire.RegistryBind]{
		{"sample", samplePWContainer[3][0][1], pipewire.RegistryBind{
			ID:      3,
			Type:    pipewire.PW_TYPE_INTERFACE_SecurityContext,
			Version: pipewire.PW_VERSION_SECURITY_CONTEXT,
			NewID:   3, // registry takes up 2
		}, nil},
	}.run(t)
}
