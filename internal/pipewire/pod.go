package pipewire

type (
	// A Word is a 32-bit unsigned integer.
	//
	// Values internal to a message appear to always be aligned to 32-bit boundary.
	Word = uint32

	// An Int is a signed integer the size of a PipeWire Word.
	Int = int32
	// An Uint is an unsigned integer the size of a PipeWire Word.
	Uint = Word
)

/* Basic types */
const (
	/* POD's can contain a number of basic SPA types: */

	SPA_TYPE_START     = 0x00000 + iota
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
