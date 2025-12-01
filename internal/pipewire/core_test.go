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
			/* type: Struct */ pipewire.SPA_TYPE_Struct, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ pipewire.SPA_TYPE_Int, 0, 0, 0,
			/* value: -1 */ 0xff, 0xff, 0xff, 0xff,
			/* padding */ 0, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ pipewire.SPA_TYPE_Int, 0, 0, 0,
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
				{Key: "object.serial", Value: "34"},
				{Key: "module.id", Value: "2"},
				{Key: "pipewire.protocol", Value: "protocol-native"},
				{Key: "pipewire.sec.pid", Value: "1443"},
				{Key: "pipewire.sec.uid", Value: "1000"},
				{Key: "pipewire.sec.gid", Value: "100"},
				{Key: "pipewire.sec.socket", Value: "pipewire-0-manager"}},
		}, nil},

		/* recvmsg 1 */

		{"sample1", samplePWContainer[4][0][1], pipewire.CoreBoundProps{
			ID:       3,
			GlobalID: 3,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "3"},
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
			/* type: Struct */ pipewire.SPA_TYPE_Struct, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ pipewire.SPA_TYPE_Int, 0, 0, 0,
			/* value: -1 */ 0xff, 0xff, 0xff, 0xff,
			/* padding */ 0, 0, 0, 0,

			/* size: 4 bytes */ 4, 0, 0, 0,
			/* type: Int */ pipewire.SPA_TYPE_Int, 0, 0, 0,
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
				{Key: "object.serial", Value: "0"},
				{Key: "core.name", Value: "pipewire-0"},
			},
		}, nil},

		{"sample1", samplePWContainer[1][7][1], pipewire.RegistryGlobal{
			ID:          1,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "1"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-rt"},
			},
		}, nil},

		{"sample2", samplePWContainer[1][8][1], pipewire.RegistryGlobal{
			ID:          3,
			Permissions: pipewire.PW_SECURITY_CONTEXT_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_SecurityContext,
			Version:     pipewire.PW_VERSION_SECURITY_CONTEXT,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "3"},
			},
		}, nil},

		{"sample3", samplePWContainer[1][9][1], pipewire.RegistryGlobal{
			ID:          2,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "2"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-protocol-native"},
			},
		}, nil},

		{"sample4", samplePWContainer[1][10][1], pipewire.RegistryGlobal{
			ID:          5,
			Permissions: pipewire.PW_PROFILER_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Profiler,
			Version:     pipewire.PW_VERSION_PROFILER,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "5"},
			},
		}, nil},

		{"sample5", samplePWContainer[1][11][1], pipewire.RegistryGlobal{
			ID:          4,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "4"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-profiler"},
			},
		}, nil},

		{"sample6", samplePWContainer[1][12][1], pipewire.RegistryGlobal{
			ID:          6,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "6"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-metadata"},
			},
		}, nil},

		{"sample7", samplePWContainer[1][13][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample8", samplePWContainer[1][14][1], pipewire.RegistryGlobal{
			ID:          8,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "8"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-device-factory"},
			},
		}, nil},

		{"sample9", samplePWContainer[1][15][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample10", samplePWContainer[1][16][1], pipewire.RegistryGlobal{
			ID:          10,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "10"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-spa-node-factory"},
			},
		}, nil},

		{"sample11", samplePWContainer[1][17][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample12", samplePWContainer[1][18][1], pipewire.RegistryGlobal{
			ID:          12,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "12"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-node"},
			},
		}, nil},

		{"sample13", samplePWContainer[1][19][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample14", samplePWContainer[1][20][1], pipewire.RegistryGlobal{
			ID:          14,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "14"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-client-device"},
			},
		}, nil},

		{"sample15", samplePWContainer[1][21][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample16", samplePWContainer[1][22][1], pipewire.RegistryGlobal{
			ID:          16,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "16"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-portal"},
			},
		}, nil},

		{"sample17", samplePWContainer[1][23][1], pipewire.RegistryGlobal{
			ID:          17,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "17"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-access"},
			},
		}, nil},

		{"sample18", samplePWContainer[1][24][1], pipewire.RegistryGlobal{
			ID:          18,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "18"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-adapter"},
			},
		}, nil},

		{"sample19", samplePWContainer[1][25][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample20", samplePWContainer[1][26][1], pipewire.RegistryGlobal{
			ID:          20,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "20"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-link-factory"},
			},
		}, nil},

		{"sample21", samplePWContainer[1][27][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample22", samplePWContainer[1][28][1], pipewire.RegistryGlobal{
			ID:          22,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "22"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-session-manager"},
			},
		}, nil},

		{"sample23", samplePWContainer[1][29][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample24", samplePWContainer[1][30][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample25", samplePWContainer[1][31][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample26", samplePWContainer[1][32][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample27", samplePWContainer[1][33][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample28", samplePWContainer[1][34][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample29", samplePWContainer[1][35][1], pipewire.RegistryGlobal{
			ID:          29,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "29"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-x11-bell"},
			},
		}, nil},

		{"sample30", samplePWContainer[1][36][1], pipewire.RegistryGlobal{
			ID:          30,
			Permissions: pipewire.PW_MODULE_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Module,
			Version:     pipewire.PW_VERSION_MODULE,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "30"},
				{Key: "module.name", Value: pipewire.PIPEWIRE_MODULE_PREFIX + "module-jackdbus-detect"},
			},
		}, nil},

		{"sample31", samplePWContainer[1][37][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample32", samplePWContainer[1][38][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample33", samplePWContainer[1][39][1], pipewire.RegistryGlobal{
			ID:          33,
			Permissions: pipewire.PW_METADATA_PERM_MASK,
			Type:        pipewire.PW_TYPE_INTERFACE_Metadata,
			Version:     pipewire.PW_VERSION_METADATA,
			Properties: &pipewire.SPADict{
				{Key: "object.serial", Value: "33"},
				{Key: "metadata.name", Value: "settings"},
			},
		}, nil},

		{"sample34", samplePWContainer[1][40][1], pipewire.RegistryGlobal{
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
		}, nil},

		{"sample35", samplePWContainer[1][42][1], pipewire.RegistryGlobal{
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
