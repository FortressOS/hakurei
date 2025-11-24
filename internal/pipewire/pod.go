package pipewire

import (
	"encoding/binary"
	"io"
	"math"
	"reflect"
	"strconv"
)

type (
	// A Word is a 32-bit unsigned integer.
	//
	// Values internal to a message appear to always be aligned to 32-bit boundary.
	Word = uint32

	// A Bool is a boolean value representing SPA_TYPE_Bool.
	Bool = bool
	// An Int is a signed integer value representing SPA_TYPE_Int.
	Int = int32
	// A Long is a signed integer value representing SPA_TYPE_Long.
	Long = int64
	// A Float is a floating point value representing SPA_TYPE_Float.
	Float = float32
	// A Double is a floating point value representing SPA_TYPE_Double.
	Double = float64
	// A String is a string value representing SPA_TYPE_String.
	String = string
	// Bytes is a byte slice representing SPA_TYPE_Bytes.
	Bytes = []byte
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

// An UnsupportedTypeError is returned by [Marshal] when attempting
// to encode an unsupported value type.
type UnsupportedTypeError struct{ Type reflect.Type }

func (e *UnsupportedTypeError) Error() string { return "unsupported type: " + e.Type.String() }

// An UnsupportedSizeError is returned by [Marshal] when attempting
// to encode a value with its encoded size exceeding what could be
// represented by the format.
type UnsupportedSizeError int

func (e UnsupportedSizeError) Error() string { return "size out of range: " + strconv.Itoa(int(e)) }

// Marshal returns the PipeWire POD encoding of v.
func Marshal(v any) ([]byte, error) { return MarshalAppend(make([]byte, 0), v) }

// MarshalAppend appends the PipeWire POD encoding of v to data.
func MarshalAppend(data []byte, v any) ([]byte, error) {
	return marshalValueAppend(data, reflect.ValueOf(v))
}

// marshalValueAppendRaw implements [MarshalAppend] on [reflect.Value].
func marshalValueAppend(data []byte, v reflect.Value) ([]byte, error) {
	data = append(data, make([]byte, 4)...)

	rData, err := marshalValueAppendRaw(data, v)
	if err != nil {
		return data, err
	}

	size := len(rData) - len(data) + 4
	paddingSize := (8 - (size)%8) % 8
	// compensated for size and type prefix
	wireSize := size - 8
	if wireSize > math.MaxUint32 {
		return data, UnsupportedSizeError(wireSize)
	}
	binary.NativeEndian.PutUint32(rData[len(data)-4:len(data)], Word(wireSize))
	rData = append(rData, make([]byte, paddingSize)...)

	return rData, nil
}

// marshalValueAppendRaw implements [MarshalAppend] on [reflect.Value] without the size prefix.
func marshalValueAppendRaw(data []byte, v reflect.Value) ([]byte, error) {
	switch v.Kind() {

	case reflect.Int32:
		data = binary.NativeEndian.AppendUint32(data, SPA_TYPE_Int)
		data = binary.NativeEndian.AppendUint32(data, Word(v.Int()))
		return data, nil

	case reflect.Struct:
		data = binary.NativeEndian.AppendUint32(data, SPA_TYPE_Struct)
		var err error
		for i := 0; i < v.NumField(); i++ {
			data, err = marshalValueAppend(data, v.Field(i))
			if err != nil {
				return data, err
			}
		}
		return data, nil

	case reflect.Pointer:
		if v.IsNil() {
			data = binary.NativeEndian.AppendUint32(data, SPA_TYPE_None)
			return data, nil
		}
		return marshalValueAppendRaw(data, v.Elem())

	case reflect.String:
		data = binary.NativeEndian.AppendUint32(data, SPA_TYPE_String)
		data = append(data, []byte(v.String())...)
		data = append(data, 0)
		return data, nil

	default:
		return data, &UnsupportedTypeError{v.Type()}
	}
}

// An InvalidUnmarshalError describes an invalid argument passed to [Unmarshal].
// (The argument to [Unmarshal] must be a non-nil pointer.)
type InvalidUnmarshalError struct{ Type reflect.Type }

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "attempting to unmarshal to nil"
	}

	if e.Type.Kind() != reflect.Pointer {
		return "attempting to unmarshal to non-pointer type: " + e.Type.String()
	}
	return "attempting to unmarshal to nil " + e.Type.String()
}

// Unmarshal parses the JSON-encoded data and stores the result
// in the value pointed to by v. If v is nil or not a pointer,
// Unmarshal returns an [InvalidUnmarshalError].
func Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
	return unmarshalValue(data, rv.Elem(), new(Word))
}

// UnmarshalSetError describes a value that cannot be set during [Unmarshal].
// This is likely an unexported struct field.
type UnmarshalSetError struct{ Type reflect.Type }

func (u *UnmarshalSetError) Error() string { return "cannot set: " + u.Type.String() }

// A TrailingGarbageError describes extra bytes after decoding
// has completed during [Unmarshal].
type TrailingGarbageError struct{ Data []byte }

func (e *TrailingGarbageError) Error() string {
	if len(e.Data) < 8 {
		return "got " + strconv.Itoa(len(e.Data)) + " bytes of trailing garbage"
	}
	return "data has extra values starting with type " + strconv.Itoa(int(binary.NativeEndian.Uint32(e.Data[4:])))
}

// A StringTerminationError describes an incorrectly terminated string
// encountered during [Unmarshal].
type StringTerminationError struct{ Value byte }

func (e StringTerminationError) Error() string {
	return "got byte " + strconv.Itoa(int(e.Value)) + " instead of NUL"
}

// unmarshalValue implements [Unmarshal] on [reflect.Value].
func unmarshalValue(data []byte, v reflect.Value, sizeP *Word) error {
	switch v.Kind() {

	case reflect.Int32:
		*sizeP = 4
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Int, sizeP); err != nil {
			return err
		}
		if !v.CanSet() {
			return &UnmarshalSetError{v.Type()}
		}
		v.SetInt(int64(binary.NativeEndian.Uint32(data)))
		return nil

	case reflect.Struct:
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Struct, sizeP); err != nil {
			return err
		}

		var fieldWireSize Word
		for i := 0; i < v.NumField(); i++ {
			if err := unmarshalValue(data, v.Field(i), &fieldWireSize); err != nil {
				return err
			}
			paddingSize := (8 - (fieldWireSize)%8) % 8
			// already bounds checked by the successful unmarshalValue call
			data = data[8+fieldWireSize+paddingSize:]
		}

		if len(data) != 0 {
			return &TrailingGarbageError{data}
		}
		return nil

	case reflect.Pointer:
		if !v.CanSet() {
			return &UnmarshalSetError{v.Type()}
		}

		if len(data) < 8 {
			return io.ErrUnexpectedEOF
		}
		switch binary.NativeEndian.Uint32(data[4:]) {
		case SPA_TYPE_None:
			v.SetZero()
			return nil

		default:
			v.Set(reflect.New(v.Type().Elem()))
			return unmarshalValue(data, v.Elem(), sizeP)
		}

	case reflect.String:
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_String, sizeP); err != nil {
			return err
		}

		// string size, one extra NUL byte
		size := int(*sizeP)
		if len(data) < size {
			return io.ErrUnexpectedEOF
		}

		// the serialised strings still include NUL termination
		if data[size-1] != 0 {
			return StringTerminationError{data[size-1]}
		}

		v.SetString(string(data[:size-1]))
		return nil

	default:
		return &UnsupportedTypeError{v.Type()}
	}
}

// An InconsistentSizeError describes an inconsistent size prefix encountered
// in data passed to [Unmarshal].
type InconsistentSizeError struct{ Prefix, Expect Word }

func (e *InconsistentSizeError) Error() string {
	return "unexpected size prefix: " + strconv.Itoa(int(e.Prefix)) + ", want " + strconv.Itoa(int(e.Expect))
}

// An UnexpectedTypeError describes an unexpected type encountered
// in data passed to [Unmarshal].
type UnexpectedTypeError struct{ Type, Expect Word }

func (u *UnexpectedTypeError) Error() string {
	return "unexpected type: " + strconv.Itoa(int(u.Type)) + ", want " + strconv.Itoa(int(u.Expect))
}

// unmarshalCheckTypeBounds performs bounds checks on data and validates the type and size prefixes.
// An expected size of zero skips further bounds checks.
func unmarshalCheckTypeBounds(data *[]byte, t Word, sizeP *Word) error {
	if len(*data) < 8 {
		return io.ErrUnexpectedEOF
	}

	wantSize := *sizeP
	gotSize := binary.NativeEndian.Uint32(*data)
	*sizeP = gotSize

	if wantSize != 0 && gotSize != wantSize {
		return &InconsistentSizeError{gotSize, wantSize}
	}
	if len(*data)-8 < int(wantSize) {
		return io.ErrUnexpectedEOF
	}

	gotType := binary.NativeEndian.Uint32((*data)[4:])
	if gotType != t {
		return &UnexpectedTypeError{gotType, t}
	}

	*data = (*data)[8:]
	return nil
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
