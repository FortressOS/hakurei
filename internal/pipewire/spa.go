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

/* pipewire/keys.h */

/**
 * Key Names
 *
 * A collection of keys that are used to add extra information on objects.
 *
 * Keys that start with "pipewire." are in general set-once and then
 * read-only. They are usually used for security sensitive information that
 * needs to be fixed.
 *
 * Properties from other objects can also appear. This usually suggests some
 * sort of parent/child or owner/owned relationship.
 *
 */

const (
	PW_KEY_PROTOCOL      = "pipewire.protocol"      /* protocol used for connection */
	PW_KEY_ACCESS        = "pipewire.access"        /* how the client access is controlled */
	PW_KEY_CLIENT_ACCESS = "pipewire.client.access" /* how the client wants to be access controlled */

	/** Various keys related to the identity of a client process and its security.
	 * Must be obtained from trusted sources by the protocol and placed as
	 * read-only properties. */

	PW_KEY_SEC_PID   = "pipewire.sec.pid"   /* Client pid, set by protocol */
	PW_KEY_SEC_UID   = "pipewire.sec.uid"   /* Client uid, set by protocol*/
	PW_KEY_SEC_GID   = "pipewire.sec.gid"   /* client gid, set by protocol*/
	PW_KEY_SEC_LABEL = "pipewire.sec.label" /* client security label, set by protocol*/

	PW_KEY_SEC_SOCKET = "pipewire.sec.socket" /* client socket name, set by protocol */

	PW_KEY_SEC_ENGINE      = "pipewire.sec.engine"      /* client secure context engine, set by protocol. This can also be set by a client when making a new security context. */
	PW_KEY_SEC_APP_ID      = "pipewire.sec.app-id"      /* client secure application id */
	PW_KEY_SEC_INSTANCE_ID = "pipewire.sec.instance-id" /* client secure instance id */

	PW_KEY_LIBRARY_NAME_SYSTEM = "library.name.system" /* name of the system library to use */
	PW_KEY_LIBRARY_NAME_LOOP   = "library.name.loop"   /* name of the loop library to use */
	PW_KEY_LIBRARY_NAME_DBUS   = "library.name.dbus"   /* name of the dbus library to use */

	/** object properties */

	PW_KEY_OBJECT_PATH     = "object.path"     /* unique path to construct the object */
	PW_KEY_OBJECT_ID       = "object.id"       /* a global object id */
	PW_KEY_OBJECT_SERIAL   = "object.serial"   /* a 64 bit object serial number. This is a number incremented for each object that is created. The lower 32 bits are guaranteed to never be SPA_ID_INVALID. */
	PW_KEY_OBJECT_LINGER   = "object.linger"   /* the object lives on even after the client that created it has been destroyed */
	PW_KEY_OBJECT_REGISTER = "object.register" /* If the object should be registered. */
	PW_KEY_OBJECT_EXPORT   = "object.export"   /* If the object should be exported, since 0.3.72 */

	/* config */

	PW_KEY_CONFIG_PREFIX          = "config.prefix"          /* a config prefix directory */
	PW_KEY_CONFIG_NAME            = "config.name"            /* a config file name */
	PW_KEY_CONFIG_OVERRIDE_PREFIX = "config.override.prefix" /* a config override prefix directory */
	PW_KEY_CONFIG_OVERRIDE_NAME   = "config.override.name"   /* a config override file name */

	/* loop */

	PW_KEY_LOOP_NAME    = "loop.name"    /* the name of a loop */
	PW_KEY_LOOP_CLASS   = "loop.class"   /* the classes this loop handles, array of strings */
	PW_KEY_LOOP_RT_PRIO = "loop.rt-prio" /* realtime priority of the loop */
	PW_KEY_LOOP_CANCEL  = "loop.cancel"  /* if the loop can be canceled */

	/* context */

	PW_KEY_CONTEXT_PROFILE_MODULES = "context.profile.modules" /* a context profile for modules, deprecated */
	PW_KEY_USER_NAME               = "context.user-name"       /* The user name that runs pipewire */
	PW_KEY_HOST_NAME               = "context.host-name"       /* The host name of the machine */

	/* core */

	PW_KEY_CORE_NAME    = "core.name"    /* The name of the core. Default is `pipewire-<username>-<pid>`, overwritten by env(PIPEWIRE_CORE) */
	PW_KEY_CORE_VERSION = "core.version" /* The version of the core. */
	PW_KEY_CORE_DAEMON  = "core.daemon"  /* If the core is listening for connections. */

	PW_KEY_CORE_ID       = "core.id"       /* the core id */
	PW_KEY_CORE_MONITORS = "core.monitors" /* the apis monitored by core. */

	/* cpu */

	PW_KEY_CPU_MAX_ALIGN = "cpu.max-align" /* maximum alignment needed to support all CPU optimizations */
	PW_KEY_CPU_CORES     = "cpu.cores"     /* number of cores */

	/* priorities */

	PW_KEY_PRIORITY_SESSION = "priority.session" /* priority in session manager */
	PW_KEY_PRIORITY_DRIVER  = "priority.driver"  /* priority to be a driver */

	/* remote keys */

	PW_KEY_REMOTE_NAME      = "remote.name"      /* The name of the remote to connect to, default pipewire-0, overwritten by env(PIPEWIRE_REMOTE). May also be a SPA-JSON array of sockets, to be tried in order. The "internal" remote name and "generic" intention connects to the local PipeWire instance. */
	PW_KEY_REMOTE_INTENTION = "remote.intention" /* The intention of the remote connection, "generic", "screencast", "manager" */

	/** application keys */

	PW_KEY_APP_NAME      = "application.name"      /* application name. Ex: "Totem Music Player" */
	PW_KEY_APP_ID        = "application.id"        /* a textual id for identifying an application logically. Ex: "org.gnome.Totem" */
	PW_KEY_APP_VERSION   = "application.version"   /* application version. Ex: "1.2.0" */
	PW_KEY_APP_ICON      = "application.icon"      /* aa base64 blob with PNG image data */
	PW_KEY_APP_ICON_NAME = "application.icon-name" /* an XDG icon name for the application. Ex: "totem" */
	PW_KEY_APP_LANGUAGE  = "application.language"  /* application language if applicable, in standard POSIX format. Ex: "en_GB" */

	PW_KEY_APP_PROCESS_ID         = "application.process.id"         /* process id  (pid)*/
	PW_KEY_APP_PROCESS_BINARY     = "application.process.binary"     /* binary name */
	PW_KEY_APP_PROCESS_USER       = "application.process.user"       /* user name */
	PW_KEY_APP_PROCESS_HOST       = "application.process.host"       /* host name */
	PW_KEY_APP_PROCESS_MACHINE_ID = "application.process.machine-id" /* the D-Bus host id the application runs on */
	PW_KEY_APP_PROCESS_SESSION_ID = "application.process.session-id" /* login session of the application, on Unix the value of $XDG_SESSION_ID. */

	/** window system */

	PW_KEY_WINDOW_X11_DISPLAY = "window.x11.display" /* the X11 display string. Ex. ":0.0" */

	/** Client properties */

	PW_KEY_CLIENT_ID   = "client.id"   /* a client id */
	PW_KEY_CLIENT_NAME = "client.name" /* the client name */
	PW_KEY_CLIENT_API  = "client.api"  /* the client api used to access PipeWire */

	/** Node keys */

	PW_KEY_NODE_ID          = "node.id"          /* node id */
	PW_KEY_NODE_NAME        = "node.name"        /* node name */
	PW_KEY_NODE_NICK        = "node.nick"        /* short node name */
	PW_KEY_NODE_DESCRIPTION = "node.description" /* localized human readable node one-line description. Ex. "Foobar USB Headset" */
	PW_KEY_NODE_PLUGGED     = "node.plugged"     /* when the node was created. As a uint64 in nanoseconds. */

	PW_KEY_NODE_SESSION       = "node.session"       /* the session id this node is part of */
	PW_KEY_NODE_GROUP         = "node.group"         /* the group id this node is part of. Nodes in the same group are always scheduled with the same driver. Can be an array of group names. */
	PW_KEY_NODE_SYNC_GROUP    = "node.sync-group"    /* the sync group this node is part of. Nodes in the same sync group are always scheduled together with the same driver when the sync is active. Can be an array of sync names. */
	PW_KEY_NODE_SYNC          = "node.sync"          /* if the sync-group is active or not */
	PW_KEY_NODE_TRANSPORT     = "node.transport"     /* if the transport is active or not */
	PW_KEY_NODE_EXCLUSIVE     = "node.exclusive"     /* node wants exclusive access to resources */
	PW_KEY_NODE_AUTOCONNECT   = "node.autoconnect"   /* node wants to be automatically connected to a compatible node */
	PW_KEY_NODE_LATENCY       = "node.latency"       /* the requested latency of the node as a fraction. Ex: 128/48000 */
	PW_KEY_NODE_MAX_LATENCY   = "node.max-latency"   /* the maximum supported latency of the node as a fraction. Ex: 1024/48000 */
	PW_KEY_NODE_LOCK_QUANTUM  = "node.lock-quantum"  /* don't change quantum when this node is active */
	PW_KEY_NODE_FORCE_QUANTUM = "node.force-quantum" /* force a quantum while the node is active */
	PW_KEY_NODE_RATE          = "node.rate"          /* the requested rate of the graph as a fraction. Ex: 1/48000 */
	PW_KEY_NODE_LOCK_RATE     = "node.lock-rate"     /* don't change rate when this node is active */
	PW_KEY_NODE_FORCE_RATE    = "node.force-rate"    /* force a rate while the node is active. A value of 0 takes the denominator of node.rate */

	PW_KEY_NODE_DONT_RECONNECT          = "node.dont-reconnect"          /* don't reconnect this node. The node is initially linked to target.object or the default node. If the target is removed, the node is destroyed */
	PW_KEY_NODE_ALWAYS_PROCESS          = "node.always-process"          /* process even when unlinked */
	PW_KEY_NODE_WANT_DRIVER             = "node.want-driver"             /* the node wants to be grouped with a driver node in order to schedule the graph. */
	PW_KEY_NODE_PAUSE_ON_IDLE           = "node.pause-on-idle"           /* pause the node when idle */
	PW_KEY_NODE_SUSPEND_ON_IDLE         = "node.suspend-on-idle"         /* suspend the node when idle */
	PW_KEY_NODE_CACHE_PARAMS            = "node.cache-params"            /* cache the node params */
	PW_KEY_NODE_TRANSPORT_SYNC          = "node.transport.sync"          /* the node handles transport sync */
	PW_KEY_NODE_DRIVER                  = "node.driver"                  /* node can drive the graph. When the node is selected as the driver, it needs to start the graph periodically. */
	PW_KEY_NODE_SUPPORTS_LAZY           = "node.supports-lazy"           /* the node can be a lazy driver. It will listen to RequestProcess commands and take them into account when deciding to start the graph. A value of 0 disables support, a value of > 0 enables with increasing preference. */
	PW_KEY_NODE_SUPPORTS_REQUEST        = "node.supports-request"        /* The node supports emiting RequestProcess events when it wants the graph to be scheduled. A value of 0 disables support, a value of > 0 enables with increasing preference. */
	PW_KEY_NODE_DRIVER_ID               = "node.driver-id"               /* the node id of the node assigned as driver for this node */
	PW_KEY_NODE_ASYNC                   = "node.async"                   /* the node wants async scheduling */
	PW_KEY_NODE_LOOP_NAME               = "node.loop.name"               /* the loop name fnmatch pattern to run in */
	PW_KEY_NODE_LOOP_CLASS              = "node.loop.class"              /* the loop class fnmatch pattern to run in */
	PW_KEY_NODE_STREAM                  = "node.stream"                  /* node is a stream, the server side should add a converter */
	PW_KEY_NODE_VIRTUAL                 = "node.virtual"                 /* the node is some sort of virtual object */
	PW_KEY_NODE_PASSIVE                 = "node.passive"                 /* indicate that a node wants passive links on output/input/all ports when the value is "out"/"in"/"true" respectively */
	PW_KEY_NODE_LINK_GROUP              = "node.link-group"              /* the node is internally linked to nodes with the same link-group. Can be an array of group names. */
	PW_KEY_NODE_NETWORK                 = "node.network"                 /* the node is on a network */
	PW_KEY_NODE_TRIGGER                 = "node.trigger"                 /* the node is not scheduled automatically based on the dependencies in the graph but it will be triggered explicitly. */
	PW_KEY_NODE_CHANNELNAMES            = "node.channel-names"           /* names of node's channels (unrelated to positions) */
	PW_KEY_NODE_DEVICE_PORT_NAME_PREFIX = "node.device-port-name-prefix" /* override port name prefix for device ports, like capture and playback or disable the prefix completely if an empty string is provided */
	PW_KEY_NODE_PHYSICAL                = "node.physical"                /* ports from the node are physical */
	PW_KEY_NODE_TERMINAL                = "node.terminal"                /* ports from the node are terminal */

	PW_KEY_NODE_RELIABLE = "node.reliable" /* node uses reliable transport 1.6.0 */

	/** Port keys */

	PW_KEY_PORT_ID             = "port.id"             /* port id */
	PW_KEY_PORT_NAME           = "port.name"           /* port name */
	PW_KEY_PORT_DIRECTION      = "port.direction"      /* the port direction, one of "in" or "out" or "control" and "notify" for control ports */
	PW_KEY_PORT_ALIAS          = "port.alias"          /* port alias */
	PW_KEY_PORT_PHYSICAL       = "port.physical"       /* if this is a physical port */
	PW_KEY_PORT_TERMINAL       = "port.terminal"       /* if this port consumes the data */
	PW_KEY_PORT_CONTROL        = "port.control"        /* if this port is a control port */
	PW_KEY_PORT_MONITOR        = "port.monitor"        /* if this port is a monitor port */
	PW_KEY_PORT_CACHE_PARAMS   = "port.cache-params"   /* cache the node port params */
	PW_KEY_PORT_EXTRA          = "port.extra"          /* api specific extra port info, API name should be prefixed. "jack:flags:56" */
	PW_KEY_PORT_PASSIVE        = "port.passive"        /* the ports wants passive links, since 0.3.67 */
	PW_KEY_PORT_IGNORE_LATENCY = "port.ignore-latency" /* latency ignored by peers, since 0.3.71 */
	PW_KEY_PORT_GROUP          = "port.group"          /* the port group of the port 1.2.0 */
	PW_KEY_PORT_EXCLUSIVE      = "port.exclusive"      /* link port only once 1.6.0 */
	PW_KEY_PORT_RELIABLE       = "port.reliable"       /* port uses reliable transport 1.6.0 */

	/** link properties */

	PW_KEY_LINK_ID          = "link.id"          /* a link id */
	PW_KEY_LINK_INPUT_NODE  = "link.input.node"  /* input node id of a link */
	PW_KEY_LINK_INPUT_PORT  = "link.input.port"  /* input port id of a link */
	PW_KEY_LINK_OUTPUT_NODE = "link.output.node" /* output node id of a link */
	PW_KEY_LINK_OUTPUT_PORT = "link.output.port" /* output port id of a link */
	PW_KEY_LINK_PASSIVE     = "link.passive"     /* indicate that a link is passive and does not cause the graph to be runnable. */
	PW_KEY_LINK_FEEDBACK    = "link.feedback"    /* indicate that a link is a feedback link and the target will receive data in the next cycle */
	PW_KEY_LINK_ASYNC       = "link.async"       /* the link is using async io */

	/** device properties */

	PW_KEY_DEVICE_ID             = "device.id"             /* device id */
	PW_KEY_DEVICE_NAME           = "device.name"           /* device name */
	PW_KEY_DEVICE_PLUGGED        = "device.plugged"        /* when the device was created. As a uint64 in nanoseconds. */
	PW_KEY_DEVICE_NICK           = "device.nick"           /* a short device nickname */
	PW_KEY_DEVICE_STRING         = "device.string"         /* device string in the underlying layer's format. Ex. "surround51:0" */
	PW_KEY_DEVICE_API            = "device.api"            /* API this device is accessed with. Ex. "alsa", "v4l2" */
	PW_KEY_DEVICE_DESCRIPTION    = "device.description"    /* localized human readable device one-line description. Ex. "Foobar USB Headset" */
	PW_KEY_DEVICE_BUS_PATH       = "device.bus-path"       /* bus path to the device in the OS' format. Ex. "pci-0000:00:14.0-usb-0:3.2:1.0" */
	PW_KEY_DEVICE_SERIAL         = "device.serial"         /* Serial number if applicable */
	PW_KEY_DEVICE_VENDOR_ID      = "device.vendor.id"      /* vendor ID if applicable */
	PW_KEY_DEVICE_VENDOR_NAME    = "device.vendor.name"    /* vendor name if applicable */
	PW_KEY_DEVICE_PRODUCT_ID     = "device.product.id"     /* product ID if applicable */
	PW_KEY_DEVICE_PRODUCT_NAME   = "device.product.name"   /* product name if applicable */
	PW_KEY_DEVICE_CLASS          = "device.class"          /* device class */
	PW_KEY_DEVICE_FORM_FACTOR    = "device.form-factor"    /* form factor if applicable. One of "internal", "speaker", "handset", "tv", "webcam", "microphone", "headset", "headphone", "hands-free", "car", "hifi", "computer", "portable" */
	PW_KEY_DEVICE_BUS            = "device.bus"            /* bus of the device if applicable. One of "isa", "pci", "usb", "firewire", "bluetooth" */
	PW_KEY_DEVICE_SUBSYSTEM      = "device.subsystem"      /* device subsystem */
	PW_KEY_DEVICE_SYSFS_PATH     = "device.sysfs.path"     /* device sysfs path */
	PW_KEY_DEVICE_ICON           = "device.icon"           /* icon for the device. A base64 blob containing PNG image data */
	PW_KEY_DEVICE_ICON_NAME      = "device.icon-name"      /* an XDG icon name for the device. Ex. "sound-card-speakers-usb" */
	PW_KEY_DEVICE_INTENDED_ROLES = "device.intended-roles" /* intended use. A space separated list of roles (see PW_KEY_MEDIA_ROLE) this device is particularly well suited for, due to latency, quality or form factor. */
	PW_KEY_DEVICE_CACHE_PARAMS   = "device.cache-params"   /* cache the device spa params */

	/** module properties */

	PW_KEY_MODULE_ID          = "module.id"          /* the module id */
	PW_KEY_MODULE_NAME        = "module.name"        /* the name of the module */
	PW_KEY_MODULE_AUTHOR      = "module.author"      /* the author's name */
	PW_KEY_MODULE_DESCRIPTION = "module.description" /* a human readable one-line description of the module's purpose.*/
	PW_KEY_MODULE_USAGE       = "module.usage"       /* a human readable usage description of the module's arguments. */
	PW_KEY_MODULE_VERSION     = "module.version"     /* a version string for the module. */
	PW_KEY_MODULE_DEPRECATED  = "module.deprecated"  /* the module is deprecated with this message */

	/** Factory properties */

	PW_KEY_FACTORY_ID           = "factory.id"           /* the factory id */
	PW_KEY_FACTORY_NAME         = "factory.name"         /* the name of the factory */
	PW_KEY_FACTORY_USAGE        = "factory.usage"        /* the usage of the factory */
	PW_KEY_FACTORY_TYPE_NAME    = "factory.type.name"    /* the name of the type created by a factory */
	PW_KEY_FACTORY_TYPE_VERSION = "factory.type.version" /* the version of the type created by a factory */

	/** Stream properties */

	PW_KEY_STREAM_IS_LIVE     = "stream.is-live"     /* Indicates that the stream is live. */
	PW_KEY_STREAM_LATENCY_MIN = "stream.latency.min" /* The minimum latency of the stream. */
	PW_KEY_STREAM_LATENCY_MAX = "stream.latency.max" /* The maximum latency of the stream */
	PW_KEY_STREAM_MONITOR     = "stream.monitor"     /* Indicates that the stream is monitoring and might select a less accurate but faster conversion algorithm. Monitor streams are also ignored when calculating the latency of their peer ports (since 0.3.71).
	 */
	PW_KEY_STREAM_DONT_REMIX   = "stream.dont-remix"   /* don't remix channels */
	PW_KEY_STREAM_CAPTURE_SINK = "stream.capture.sink" /* Try to capture the sink output instead of source output */

	/** Media */

	PW_KEY_MEDIA_TYPE      = "media.type"      /* Media type, one of Audio, Video, Midi */
	PW_KEY_MEDIA_CATEGORY  = "media.category"  /* Media Category: Playback, Capture, Duplex, Monitor, Manager */
	PW_KEY_MEDIA_ROLE      = "media.role"      /* Role: Movie, Music, Camera, Screen, Communication, Game, Notification, DSP, Production, Accessibility, Test */
	PW_KEY_MEDIA_CLASS     = "media.class"     /* class Ex: "Video/Source" */
	PW_KEY_MEDIA_NAME      = "media.name"      /* media name. Ex: "Pink Floyd: Time" */
	PW_KEY_MEDIA_TITLE     = "media.title"     /* title. Ex: "Time" */
	PW_KEY_MEDIA_ARTIST    = "media.artist"    /* artist. Ex: "Pink Floyd" */
	PW_KEY_MEDIA_ALBUM     = "media.album"     /* album. Ex: "Dark Side of the Moon" */
	PW_KEY_MEDIA_COPYRIGHT = "media.copyright" /* copyright string */
	PW_KEY_MEDIA_SOFTWARE  = "media.software"  /* generator software */
	PW_KEY_MEDIA_LANGUAGE  = "media.language"  /* language in POSIX format. Ex: en_GB */
	PW_KEY_MEDIA_FILENAME  = "media.filename"  /* filename */
	PW_KEY_MEDIA_ICON      = "media.icon"      /* icon for the media, a base64 blob with PNG image data */
	PW_KEY_MEDIA_ICON_NAME = "media.icon-name" /* an XDG icon name for the media. Ex: "audio-x-mp3" */
	PW_KEY_MEDIA_COMMENT   = "media.comment"   /* extra comment */
	PW_KEY_MEDIA_DATE      = "media.date"      /* date of the media */
	PW_KEY_MEDIA_FORMAT    = "media.format"    /* format of the media */

	/** format related properties */

	PW_KEY_FORMAT_DSP = "format.dsp" /* a dsp format. Ex: "32 bit float mono audio" */

	/** audio related properties */

	PW_KEY_AUDIO_CHANNEL       = "audio.channel"       /* an audio channel. Ex: "FL" */
	PW_KEY_AUDIO_RATE          = "audio.rate"          /* an audio samplerate */
	PW_KEY_AUDIO_CHANNELS      = "audio.channels"      /* number of audio channels */
	PW_KEY_AUDIO_FORMAT        = "audio.format"        /* an audio format. Ex: "S16LE" */
	PW_KEY_AUDIO_ALLOWED_RATES = "audio.allowed-rates" /* a list of allowed samplerates ex. "[ 44100 48000 ]" */

	/** video related properties */

	PW_KEY_VIDEO_RATE   = "video.framerate" /* a video framerate */
	PW_KEY_VIDEO_FORMAT = "video.format"    /* a video format */
	PW_KEY_VIDEO_SIZE   = "video.size"      /* a video size as "<width>x<height" */

	PW_KEY_TARGET_OBJECT = "target.object" /* a target object to link to. This can be and object name or object.serial */
)
