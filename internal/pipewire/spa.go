package pipewire

import (
	"encoding/binary"
	"fmt"
)

// A SPAKind describes the kind of data being encoded right after it.
//
// These do not always follow the same rules, and encoding/decoding
// is very much context-dependent. Callers should therefore not
// attempt to use these values directly and rely on [Marshal] and
// [Unmarshal] and their variants instead.
type SPAKind Word

/* Basic types */
const (
	/* POD's can contain a number of basic SPA types: */

	SPA_TYPE_START SPAKind = 0x00000 + iota

	SPA_TYPE_None      // No value or a NULL pointer.
	SPA_TYPE_Bool      // A boolean value.
	SPA_TYPE_Id        // An enumerated value.
	SPA_TYPE_Int       // An integer value, 32-bit.
	SPA_TYPE_Long      // An integer value, 64-bit.
	SPA_TYPE_Float     // A floating point value, 32-bit.
	SPA_TYPE_Double    // A floating point value, 64-bit.
	SPA_TYPE_String    // A string.
	SPA_TYPE_Bytes     // A byte array.
	SPA_TYPE_Rectangle // A rectangle with width and height.
	SPA_TYPE_Fraction  // A fraction with numerator and denominator.
	SPA_TYPE_Bitmap    // An array of bits.

	/* POD's can be grouped together in these container types: */

	SPA_TYPE_Array    // An array of equal sized objects.
	SPA_TYPE_Struct   // A collection of types and objects.
	SPA_TYPE_Object   // An object with properties.
	SPA_TYPE_Sequence // A timed sequence of POD's.

	/* POD's can also contain some extra types: */

	SPA_TYPE_Pointer // A typed pointer in memory.
	SPA_TYPE_Fd      // A file descriptor.
	SPA_TYPE_Choice  // A choice of values.
	SPA_TYPE_Pod     // A generic type for the POD itself.

	_SPA_TYPE_LAST // not part of ABI
)

// append appends the representation of [SPAKind] to data and returns the appended slice.
func (kind SPAKind) append(data []byte) []byte {
	return binary.NativeEndian.AppendUint32(data, Word(kind))
}

// String returns the name of the [SPAKind] for basic types.
func (kind SPAKind) String() string {
	switch kind {
	case SPA_TYPE_None:
		return "None"
	case SPA_TYPE_Bool:
		return "Bool"
	case SPA_TYPE_Id:
		return "Id"
	case SPA_TYPE_Int:
		return "Int"
	case SPA_TYPE_Long:
		return "Long"
	case SPA_TYPE_Float:
		return "Float"
	case SPA_TYPE_Double:
		return "Double"
	case SPA_TYPE_String:
		return "String"
	case SPA_TYPE_Bytes:
		return "Bytes"
	case SPA_TYPE_Rectangle:
		return "Rectangle"
	case SPA_TYPE_Fraction:
		return "Fraction"
	case SPA_TYPE_Bitmap:
		return "Bitmap"
	case SPA_TYPE_Array:
		return "Array"
	case SPA_TYPE_Struct:
		return "Struct"
	case SPA_TYPE_Object:
		return "Object"
	case SPA_TYPE_Sequence:
		return "Sequence"
	case SPA_TYPE_Pointer:
		return "Pointer"
	case SPA_TYPE_Fd:
		return "Fd"
	case SPA_TYPE_Choice:
		return "Choice"
	case SPA_TYPE_Pod:
		return "Pod"

	default:
		return fmt.Sprintf("invalid type field %#x", Word(kind))
	}
}

/* Pointers */
const (
	SPA_TYPE_POINTER_START = 0x10000 + iota
	SPA_TYPE_POINTER_Buffer
	SPA_TYPE_POINTER_Meta
	SPA_TYPE_POINTER_Dict

	_SPA_TYPE_POINTER_LAST // not part of ABI
)

/* Events */
const (
	SPA_TYPE_EVENT_START = 0x20000 + iota
	SPA_TYPE_EVENT_Device
	SPA_TYPE_EVENT_Node

	_SPA_TYPE_EVENT_LAST // not part of ABI
)

/* Commands */
const (
	SPA_TYPE_COMMAND_START = 0x30000 + iota
	SPA_TYPE_COMMAND_Device
	SPA_TYPE_COMMAND_Node

	_SPA_TYPE_COMMAND_LAST // not part of ABI
)

/* Objects */
const (
	SPA_TYPE_OBJECT_START = 0x40000 + iota
	SPA_TYPE_OBJECT_PropInfo
	SPA_TYPE_OBJECT_Props
	SPA_TYPE_OBJECT_Format
	SPA_TYPE_OBJECT_ParamBuffers
	SPA_TYPE_OBJECT_ParamMeta
	SPA_TYPE_OBJECT_ParamIO
	SPA_TYPE_OBJECT_ParamProfile
	SPA_TYPE_OBJECT_ParamPortConfig
	SPA_TYPE_OBJECT_ParamRoute
	SPA_TYPE_OBJECT_Profiler
	SPA_TYPE_OBJECT_ParamLatency
	SPA_TYPE_OBJECT_ParamProcessLatency
	SPA_TYPE_OBJECT_ParamTag
	_SPA_TYPE_OBJECT_LAST // not part of ABI
)

/* vendor extensions */
const (
	SPA_TYPE_VENDOR_PipeWire = 0x02000000

	SPA_TYPE_VENDOR_Other = 0x7f000000
)

/* spa/include/spa/utils/type.h */

const (
	SPA_TYPE_INFO_BASE = "Spa:"

	SPA_TYPE_INFO_Flags      = SPA_TYPE_INFO_BASE + "Flags"
	SPA_TYPE_INFO_FLAGS_BASE = SPA_TYPE_INFO_Flags + ":"

	SPA_TYPE_INFO_Enum      = SPA_TYPE_INFO_BASE + "Enum"
	SPA_TYPE_INFO_ENUM_BASE = SPA_TYPE_INFO_Enum + ":"

	SPA_TYPE_INFO_Pod      = SPA_TYPE_INFO_BASE + "Pod"
	SPA_TYPE_INFO_POD_BASE = SPA_TYPE_INFO_Pod + ":"

	SPA_TYPE_INFO_Struct      = SPA_TYPE_INFO_POD_BASE + "Struct"
	SPA_TYPE_INFO_STRUCT_BASE = SPA_TYPE_INFO_Struct + ":"

	SPA_TYPE_INFO_Object      = SPA_TYPE_INFO_POD_BASE + "Object"
	SPA_TYPE_INFO_OBJECT_BASE = SPA_TYPE_INFO_Object + ":"

	SPA_TYPE_INFO_Pointer      = SPA_TYPE_INFO_BASE + "Pointer"
	SPA_TYPE_INFO_POINTER_BASE = SPA_TYPE_INFO_Pointer + ":"

	SPA_TYPE_INFO_Interface      = SPA_TYPE_INFO_POINTER_BASE + "Interface"
	SPA_TYPE_INFO_INTERFACE_BASE = SPA_TYPE_INFO_Interface + ":"

	SPA_TYPE_INFO_Event      = SPA_TYPE_INFO_OBJECT_BASE + "Event"
	SPA_TYPE_INFO_EVENT_BASE = SPA_TYPE_INFO_Event + ":"

	SPA_TYPE_INFO_Command      = SPA_TYPE_INFO_OBJECT_BASE + "Command"
	SPA_TYPE_INFO_COMMAND_BASE = SPA_TYPE_INFO_Command + ":"
)

/* pipewire/device.h */

const (
	PW_TYPE_INTERFACE_Device = PW_TYPE_INFO_INTERFACE_BASE + "Device"
	PW_DEVICE_PERM_MASK      = PW_PERM_RWXM
	PW_VERSION_DEVICE        = 3
)

const (
	PW_DEVICE_CHANGE_MASK_PROPS = 1 << iota
	PW_DEVICE_CHANGE_MASK_PARAMS

	PW_DEVICE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_DEVICE_EVENT_INFO = iota
	PW_DEVICE_EVENT_PARAM
	PW_DEVICE_EVENT_NUM

	PW_VERSION_DEVICE_EVENTS = 0
)

const (
	PW_DEVICE_METHOD_ADD_LISTENER = iota
	PW_DEVICE_METHOD_SUBSCRIBE_PARAMS
	PW_DEVICE_METHOD_ENUM_PARAMS
	PW_DEVICE_METHOD_SET_PARAM
	PW_DEVICE_METHOD_NUM

	PW_VERSION_DEVICE_METHODS = 0
)

/* pipewire/factory.h */

const (
	PW_TYPE_INTERFACE_Factory = PW_TYPE_INFO_INTERFACE_BASE + "Factory"
	PW_FACTORY_PERM_MASK      = PW_PERM_R | PW_PERM_M
	PW_VERSION_FACTORY        = 3
)

const (
	PW_FACTORY_CHANGE_MASK_PROPS = 1 << iota

	PW_FACTORY_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_FACTORY_EVENT_INFO = iota
	PW_FACTORY_EVENT_NUM

	PW_VERSION_FACTORY_EVENTS = 0
)

const (
	PW_FACTORY_METHOD_ADD_LISTENER = iota
	PW_FACTORY_METHOD_NUM

	PW_VERSION_FACTORY_METHODS = 0
)

/* pipewire/link.h */

const (
	PW_TYPE_INTERFACE_Link = PW_TYPE_INFO_INTERFACE_BASE + "Link"
	PW_LINK_PERM_MASK      = PW_PERM_R | PW_PERM_X
	PW_VERSION_LINK        = 3
)

const (
	PW_LINK_STATE_ERROR       = iota - 2 // the link is in error
	PW_LINK_STATE_UNLINKED               // the link is unlinked
	PW_LINK_STATE_INIT                   // the link is initialized
	PW_LINK_STATE_NEGOTIATING            // the link is negotiating formats
	PW_LINK_STATE_ALLOCATING             // the link is allocating buffers
	PW_LINK_STATE_PAUSED                 // the link is paused
	PW_LINK_STATE_ACTIVE                 // the link is active
)

const (
	PW_LINK_CHANGE_MASK_STATE = (1 << iota)
	PW_LINK_CHANGE_MASK_FORMAT
	PW_LINK_CHANGE_MASK_PROPS

	PW_LINK_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_LINK_EVENT_INFO = iota
	PW_LINK_EVENT_NUM

	PW_VERSION_LINK_EVENTS = 0
)

const (
	PW_LINK_METHOD_ADD_LISTENER = iota
	PW_LINK_METHOD_NUM

	PW_VERSION_LINK_METHODS = 0
)

/* pipewire/module.h */

const (
	PW_TYPE_INTERFACE_Module = PW_TYPE_INFO_INTERFACE_BASE + "Module"
	PW_MODULE_PERM_MASK      = PW_PERM_R | PW_PERM_M
	PW_VERSION_MODULE        = 3
)

const (
	PW_MODULE_CHANGE_MASK_PROPS = 1 << iota

	PW_MODULE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_MODULE_EVENT_INFO = iota
	PW_MODULE_EVENT_NUM

	PW_VERSION_MODULE_EVENTS = 0
)

const (
	PW_MODULE_METHOD_ADD_LISTENER = iota
	PW_MODULE_METHOD_NUM

	PW_VERSION_MODULE_METHODS = 0
)

/* pipewire/impl-module.h */

const (
	PIPEWIRE_SYMBOL_MODULE_INIT = "pipewire__module_init"
	PIPEWIRE_MODULE_PREFIX      = "libpipewire-"

	PW_VERSION_IMPL_MODULE_EVENTS = 0
)

/* pipewire/node.h */

const (
	PW_TYPE_INTERFACE_Node = PW_TYPE_INFO_INTERFACE_BASE + "Node"
	PW_NODE_PERM_MASK      = PW_PERM_RWXML
	PW_VERSION_NODE        = 3
)

const (
	PW_NODE_STATE_ERROR     = iota - 1 // error state
	PW_NODE_STATE_CREATING             // the node is being created
	PW_NODE_STATE_SUSPENDED            // the node is suspended, the device might be closed
	PW_NODE_STATE_IDLE                 // the node is running but there is no active port
	PW_NODE_STATE_RUNNING              // the node is running
)

const (
	PW_NODE_CHANGE_MASK_INPUT_PORTS = 1 << iota
	PW_NODE_CHANGE_MASK_OUTPUT_PORTS
	PW_NODE_CHANGE_MASK_STATE
	PW_NODE_CHANGE_MASK_PROPS
	PW_NODE_CHANGE_MASK_PARAMS

	PW_NODE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_NODE_EVENT_INFO = iota
	PW_NODE_EVENT_PARAM
	PW_NODE_EVENT_NUM

	PW_VERSION_NODE_EVENTS = 0
)

const (
	PW_NODE_METHOD_ADD_LISTENER = iota
	PW_NODE_METHOD_SUBSCRIBE_PARAMS
	PW_NODE_METHOD_ENUM_PARAMS
	PW_NODE_METHOD_SET_PARAM
	PW_NODE_METHOD_SEND_COMMAND
	PW_NODE_METHOD_NUM

	PW_VERSION_NODE_METHODS = 0
)

/* pipewire/permission.h */

const (
	PW_PERM_R = 0400 // object can be seen and events can be received
	PW_PERM_W = 0200 // methods can be called that modify the object
	PW_PERM_X = 0100 // methods can be called on the object. The W flag must be present in order to call methods that modify the object.
	PW_PERM_M = 0010 // metadata can be set on object, Since 0.3.9
	PW_PERM_L = 0020 // a link can be made between a node that doesn't have permission to see the other node, Since 0.3.77

	PW_PERM_RW    = PW_PERM_R | PW_PERM_W
	PW_PERM_RWX   = PW_PERM_RW | PW_PERM_X
	PW_PERM_RWXM  = PW_PERM_RWX | PW_PERM_M
	PW_PERM_RWXML = PW_PERM_RWXM | PW_PERM_L

	PW_PERM_ALL          = PW_PERM_RWXM
	PW_PERM_INVALID Word = 0xffffffff
)

/* pipewire/port.h */

const (
	PW_TYPE_INTERFACE_Port = PW_TYPE_INFO_INTERFACE_BASE + "Port"
	PW_PORT_PERM_MASK      = PW_PERM_R | PW_PERM_X | PW_PERM_M
	PW_VERSION_PORT        = 3
)

const (
	PW_PORT_CHANGE_MASK_PROPS = 1 << iota
	PW_PORT_CHANGE_MASK_PARAMS

	PW_PORT_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_PORT_EVENT_INFO = iota
	PW_PORT_EVENT_PARAM
	PW_PORT_EVENT_NUM

	PW_VERSION_PORT_EVENTS = 0
)

const (
	PW_PORT_METHOD_ADD_LISTENER = iota
	PW_PORT_METHOD_SUBSCRIBE_PARAMS
	PW_PORT_METHOD_ENUM_PARAMS
	PW_PORT_METHOD_NUM

	PW_VERSION_PORT_METHODS = 0
)

/* pipewire/extensions/client-node.h */

const (
	PW_TYPE_INTERFACE_ClientNode = PW_TYPE_INFO_INTERFACE_BASE + "ClientNode"
	PW_VERSION_CLIENT_NODE       = 6

	PW_EXTENSION_MODULE_CLIENT_NODE = PIPEWIRE_MODULE_PREFIX + "module-client-node"
)

const (
	PW_CLIENT_NODE_EVENT_TRANSPORT = iota
	PW_CLIENT_NODE_EVENT_SET_PARAM
	PW_CLIENT_NODE_EVENT_SET_IO
	PW_CLIENT_NODE_EVENT_EVENT
	PW_CLIENT_NODE_EVENT_COMMAND
	PW_CLIENT_NODE_EVENT_ADD_PORT
	PW_CLIENT_NODE_EVENT_REMOVE_PORT
	PW_CLIENT_NODE_EVENT_PORT_SET_PARAM
	PW_CLIENT_NODE_EVENT_PORT_USE_BUFFERS
	PW_CLIENT_NODE_EVENT_PORT_SET_IO
	PW_CLIENT_NODE_EVENT_SET_ACTIVATION
	PW_CLIENT_NODE_EVENT_PORT_SET_MIX_INFO
	PW_CLIENT_NODE_EVENT_NUM

	PW_VERSION_CLIENT_NODE_EVENTS = 1
)

const (
	PW_CLIENT_NODE_METHOD_ADD_LISTENER = iota
	PW_CLIENT_NODE_METHOD_GET_NODE
	PW_CLIENT_NODE_METHOD_UPDATE
	PW_CLIENT_NODE_METHOD_PORT_UPDATE
	PW_CLIENT_NODE_METHOD_SET_ACTIVE
	PW_CLIENT_NODE_METHOD_EVENT
	PW_CLIENT_NODE_METHOD_PORT_BUFFERS
	PW_CLIENT_NODE_METHOD_NUM

	PW_VERSION_CLIENT_NODE_METHODS = 0
)

const (
	PW_CLIENT_NODE_UPDATE_PARAMS = 1 << iota
	PW_CLIENT_NODE_UPDATE_INFO
)

const (
	PW_CLIENT_NODE_PORT_UPDATE_PARAMS = 1 << iota
	PW_CLIENT_NODE_PORT_UPDATE_INFO
)

/* pipewire/extensions/metadata.h */

const (
	PW_TYPE_INTERFACE_Metadata = PW_TYPE_INFO_INTERFACE_BASE + "Metadata"
	PW_METADATA_PERM_MASK      = PW_PERM_RWX
	PW_VERSION_METADATA        = 3

	PW_EXTENSION_MODULE_METADATA = PIPEWIRE_MODULE_PREFIX + "module-metadata"
)

const (
	PW_METADATA_EVENT_PROPERTY = iota
	PW_METADATA_EVENT_NUM

	PW_VERSION_METADATA_EVENTS = 0
)

const (
	PW_METADATA_METHOD_ADD_LISTENER = iota
	PW_METADATA_METHOD_SET_PROPERTY
	PW_METADATA_METHOD_CLEAR
	PW_METADATA_METHOD_NUM

	PW_VERSION_METADATA_METHODS = 0
)

const (
	PW_KEY_METADATA_NAME   = "metadata.name"
	PW_KEY_METADATA_VALUES = "metadata.values"
)

/* pipewire/extensions/profiler.h */

const (
	PW_TYPE_INTERFACE_Profiler = PW_TYPE_INFO_INTERFACE_BASE + "Profiler"
	PW_VERSION_PROFILER        = 3
	PW_PROFILER_PERM_MASK      = PW_PERM_R

	PW_EXTENSION_MODULE_PROFILER = PIPEWIRE_MODULE_PREFIX + "module-profiler"
)

const (
	PW_PROFILER_EVENT_PROFILE = iota
	PW_PROFILER_EVENT_NUM

	PW_VERSION_PROFILER_EVENTS = 0
)

const (
	PW_PROFILER_METHOD_ADD_LISTENER = iota
	PW_PROFILER_METHOD_NUM

	PW_VERSION_PROFILER_METHODS = 0
)

const (
	PW_KEY_PROFILER_NAME = "profiler.name"
)

/* pipewire/type.h */

const (
	PW_TYPE_INFO_BASE = "PipeWire:"

	PW_TYPE_INFO_Object      = PW_TYPE_INFO_BASE + "Object"
	PW_TYPE_INFO_OBJECT_BASE = PW_TYPE_INFO_Object + ":"

	PW_TYPE_INFO_Interface      = PW_TYPE_INFO_BASE + "Interface"
	PW_TYPE_INFO_INTERFACE_BASE = PW_TYPE_INFO_Interface + ":"
)
