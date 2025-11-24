// Package pipewire provides a partial implementation of the PipeWire protocol native.
//
// This implementation is created based on black box analysis and very limited static
// analysis. The PipeWire documentation is vague and mostly nonexistent, and source code
// readability is not great due to frequent macro abuse, confusing and inconsistent naming
// schemes, almost complete absence of comments and the multiple layers of abstractions
// even internal to the library. The convoluted build system and frequent (mis)use of
// dlopen(3) further complicates static analysis efforts.
//
// Because of this, extreme care must be taken when reusing any code found in this package.
// While it is extensively tested to be correct for its role within Hakurei, remember that
// work is only done against PipeWire behaviour specific to this use case, and it is nearly
// impossible to guarantee that this interpretation of its behaviour is intended, or correct
// for any other uses of the protocol.
package pipewire

/* pipewire/client.h */

const (
	PW_TYPE_INTERFACE_Client = PW_TYPE_INFO_INTERFACE_BASE + "Client"
	PW_CLIENT_PERM_MASK      = PW_PERM_RWXM
	PW_VERSION_CLIENT        = 3

	PW_ID_CLIENT = 1
)

const (
	PW_CLIENT_CHANGE_MASK_PROPS = 1 << iota

	PW_CLIENT_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_CLIENT_EVENT_INFO = iota
	PW_CLIENT_EVENT_PERMISSIONS
	PW_CLIENT_EVENT_NUM

	PW_VERSION_CLIENT_EVENTS = 0
)

const (
	PW_CLIENT_METHOD_ADD_LISTENER = iota
	PW_CLIENT_METHOD_ERROR
	PW_CLIENT_METHOD_UPDATE_PROPERTIES
	PW_CLIENT_METHOD_GET_PERMISSIONS
	PW_CLIENT_METHOD_UPDATE_PERMISSIONS
	PW_CLIENT_METHOD_NUM

	PW_VERSION_CLIENT_METHODS = 0
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

/* pipewire/core.h */

const (
	PW_TYPE_INTERFACE_Core     = PW_TYPE_INFO_INTERFACE_BASE + "Core"
	PW_TYPE_INTERFACE_Registry = PW_TYPE_INFO_INTERFACE_BASE + "Registry"
	PW_CORE_PERM_MASK          = PW_PERM_R | PW_PERM_X | PW_PERM_M
	PW_VERSION_CORE            = 4
	PW_VERSION_REGISTRY        = 3

	PW_DEFAULT_REMOTE = "pipewire-0"
	PW_ID_CORE        = 0
	PW_ID_ANY         = uint32(0xffffffff)
)

const (
	PW_CORE_CHANGE_MASK_PROPS = 1 << iota

	PW_CORE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_CORE_EVENT_INFO = iota
	PW_CORE_EVENT_DONE
	PW_CORE_EVENT_PING
	PW_CORE_EVENT_ERROR
	PW_CORE_EVENT_REMOVE_ID
	PW_CORE_EVENT_BOUND_ID
	PW_CORE_EVENT_ADD_MEM
	PW_CORE_EVENT_REMOVE_MEM
	PW_CORE_EVENT_BOUND_PROPS
	PW_CORE_EVENT_NUM

	PW_VERSION_CORE_EVENTS = 1
)

const (
	PW_CORE_METHOD_ADD_LISTENER = iota
	PW_CORE_METHOD_HELLO
	PW_CORE_METHOD_SYNC
	PW_CORE_METHOD_PONG
	PW_CORE_METHOD_ERROR
	PW_CORE_METHOD_GET_REGISTRY
	PW_CORE_METHOD_CREATE_OBJECT
	PW_CORE_METHOD_DESTROY
	PW_CORE_METHOD_NUM

	PW_VERSION_CORE_METHODS = 0
)

const (
	PW_REGISTRY_EVENT_GLOBAL = iota
	PW_REGISTRY_EVENT_GLOBAL_REMOVE
	PW_REGISTRY_EVENT_NUM

	PW_VERSION_REGISTRY_EVENTS = 0
)

const (
	PW_REGISTRY_METHOD_ADD_LISTENER = iota
	PW_REGISTRY_METHOD_BIND
	PW_REGISTRY_METHOD_DESTROY
	PW_REGISTRY_METHOD_NUM

	PW_VERSION_REGISTRY_METHODS = 0
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
	W_PROFILER_PERM_MASK       = PW_PERM_R
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

/* pipewire/extensions/security-context.h */

const (
	PW_TYPE_INTERFACE_SecurityContext = PW_TYPE_INFO_INTERFACE_BASE + "SecurityContext"
	PW_SECURITY_CONTEXT_PERM_MASK     = PW_PERM_RWX
	PW_VERSION_SECURITY_CONTEXT       = 3
)

const (
	PW_SECURITY_CONTEXT_EVENT_NUM = iota

	PW_VERSION_SECURITY_CONTEXT_EVENTS = 0
)

const (
	PW_SECURITY_CONTEXT_METHOD_ADD_LISTENER = iota
	PW_SECURITY_CONTEXT_METHOD_CREATE
	PW_SECURITY_CONTEXT_METHOD_NUM

	PW_VERSION_SECURITY_CONTEXT_METHODS = 0
)

/* pipewire/type.h */

const (
	PW_TYPE_INFO_BASE = "PipeWire:"

	PW_TYPE_INFO_Object      = PW_TYPE_INFO_BASE + "Object"
	PW_TYPE_INFO_OBJECT_BASE = PW_TYPE_INFO_Object + ":"

	PW_TYPE_INFO_Interface      = PW_TYPE_INFO_BASE + "Interface"
	PW_TYPE_INFO_INTERFACE_BASE = PW_TYPE_INFO_Interface + ":"
)
