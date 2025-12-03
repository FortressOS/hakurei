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
	// An Id is an enumerated value representing SPA_TYPE_Id.
	Id = Word
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

	// A Fd is a signed integer value representing SPA_TYPE_Fd.
	Fd Long
)

const (
	// SizeAlign is the boundary which POD starts are always aligned to.
	SizeAlign = 8

	// SizeSPrefix is the fixed, unpadded size of the fixed-size prefix encoding POD wire size.
	SizeSPrefix = 4
	// SizeTPrefix is the fixed, unpadded size of the fixed-size prefix encoding POD value type.
	SizeTPrefix = 4
	// SizePrefix is the fixed, unpadded size of the fixed-size POD prefix.
	SizePrefix = SizeSPrefix + SizeTPrefix

	// SizeId is the fixed, unpadded size of a [SPA_TYPE_Id] value.
	SizeId Word = 4
	// SizeInt is the fixed, unpadded size of a [SPA_TYPE_Int] value.
	SizeInt Word = 4
	// SizeLong is the fixed, unpadded size of a [SPA_TYPE_Long] value.
	SizeLong Word = 8

	// SizeFd is the fixed, unpadded size of a [SPA_TYPE_Fd] value.
	SizeFd = SizeLong
)

// A KnownSize value has known POD encoded size before serialisation.
type KnownSize interface {
	// Size returns the POD encoded size of the receiver.
	Size() Word
}

// PaddingSize returns the padding size corresponding to a wire size.
func PaddingSize[W Word | int](wireSize W) W { return (SizeAlign - (wireSize)%SizeAlign) % SizeAlign }

// PaddedSize returns the padded size corresponding to a wire size.
func PaddedSize[W Word | int](wireSize W) W { return wireSize + PaddingSize(wireSize) }

// Size returns prefixed and padded size corresponding to a wire size.
func Size[W Word | int](wireSize W) W { return SizePrefix + PaddedSize(wireSize) }

// SizeString returns prefixed and padded size corresponding to a string.
func SizeString[W Word | int](s string) W { return Size(W(len(s)) + 1) }

// PODMarshaler is the interface implemented by an object that can
// marshal itself into PipeWire POD encoding.
type PODMarshaler interface {
	// MarshalPOD encodes the receiver into PipeWire POD encoding,
	// appends it to data, and returns the result.
	MarshalPOD(data []byte) ([]byte, error)
}

// An UnsupportedTypeError is returned by [Marshal] when attempting
// to encode an unsupported value type.
type UnsupportedTypeError struct{ Type reflect.Type }

func (e *UnsupportedTypeError) Error() string { return "unsupported type " + e.Type.String() }

// An UnsupportedSizeError is returned by [Marshal] when attempting
// to encode a value with its encoded size exceeding what could be
// represented by the format.
type UnsupportedSizeError int

func (e UnsupportedSizeError) Error() string { return "size " + strconv.Itoa(int(e)) + " out of range" }

// Marshal returns the PipeWire POD encoding of v.
func Marshal(v any) ([]byte, error) {
	var data []byte
	if s, ok := v.(KnownSize); ok {
		data = make([]byte, 0, s.Size())
	}
	return MarshalAppend(data, v)
}

// MarshalAppend appends the PipeWire POD encoding of v to data.
func MarshalAppend(data []byte, v any) ([]byte, error) {
	return marshalValueAppend(data, reflect.ValueOf(v))
}

// appendInner calls f and handles size prefix and padding around the appended data.
// f must only append to data.
func appendInner(data []byte, f func(data []byte) ([]byte, error)) ([]byte, error) {
	data = append(data, make([]byte, SizeSPrefix)...)

	rData, err := f(data)
	if err != nil {
		return data, err
	}

	size := len(rData) - len(data) + SizeSPrefix
	// compensated for size and type prefix
	wireSize := size - SizePrefix
	if wireSize > math.MaxUint32 {
		return data, UnsupportedSizeError(wireSize)
	}
	binary.NativeEndian.PutUint32(rData[len(data)-SizeSPrefix:len(data)], Word(wireSize))
	rData = append(rData, make([]byte, PaddingSize(size))...)

	return rData, nil
}

// marshalValueAppendRaw implements [MarshalAppend] on [reflect.Value].
func marshalValueAppend(data []byte, v reflect.Value) ([]byte, error) {
	if v.CanInterface() && (v.Kind() != reflect.Pointer || !v.IsNil()) {
		if m, ok := v.Interface().(PODMarshaler); ok {
			var err error
			data, err = m.MarshalPOD(data)
			return data, err
		}
	}

	return appendInner(data, func(data []byte) ([]byte, error) { return marshalValueAppendRaw(data, v) })
}

// marshalValueAppendRaw implements [MarshalAppend] on [reflect.Value] without the size prefix.
func marshalValueAppendRaw(data []byte, v reflect.Value) ([]byte, error) {
	if v.CanInterface() {
		switch c := v.Interface().(type) {
		case Fd:
			data = SPA_TYPE_Fd.append(data)
			data = binary.NativeEndian.AppendUint64(data, uint64(c))
			return data, nil
		}
	}

	switch v.Kind() {
	case reflect.Uint32:
		data = SPA_TYPE_Id.append(data)
		data = binary.NativeEndian.AppendUint32(data, Word(v.Uint()))
		return data, nil

	case reflect.Int32:
		data = SPA_TYPE_Int.append(data)
		data = binary.NativeEndian.AppendUint32(data, Word(v.Int()))
		return data, nil

	case reflect.Int64:
		data = SPA_TYPE_Long.append(data)
		data = binary.NativeEndian.AppendUint64(data, uint64(v.Int()))
		return data, nil

	case reflect.Struct:
		data = SPA_TYPE_Struct.append(data)
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
			data = SPA_TYPE_None.append(data)
			return data, nil
		}
		return marshalValueAppendRaw(data, v.Elem())

	case reflect.String:
		data = SPA_TYPE_String.append(data)
		data = append(data, []byte(v.String())...)
		data = append(data, 0)
		return data, nil

	default:
		return data, &UnsupportedTypeError{v.Type()}
	}
}

// PODUnmarshaler is the interface implemented by an object that can
// unmarshal a PipeWire POD encoding representation of itself.
type PODUnmarshaler interface {
	// UnmarshalPOD must be able to decode the form generated by MarshalPOD.
	// UnmarshalPOD must copy the data if it wishes to retain the data
	// after returning.
	UnmarshalPOD(data []byte) (Word, error)
}

// An InvalidUnmarshalError describes an invalid argument passed to [Unmarshal].
// (The argument to [Unmarshal] must be a non-nil pointer.)
type InvalidUnmarshalError struct{ Type reflect.Type }

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "attempting to unmarshal to nil"
	}

	if e.Type.Kind() != reflect.Pointer {
		return "attempting to unmarshal to non-pointer type " + e.Type.String()
	}
	return "attempting to unmarshal to nil " + e.Type.String()
}

// UnexpectedEOFError describes an unexpected EOF encountered in the middle of decoding POD data.
type UnexpectedEOFError uintptr

const (
	// ErrEOFPrefix is returned when unexpectedly encountering EOF
	// decoding the fixed-size POD prefix.
	ErrEOFPrefix UnexpectedEOFError = iota
	// ErrEOFData is returned when unexpectedly encountering EOF
	// establishing POD data bounds.
	ErrEOFData
	// ErrEOFDataString is returned when unexpectedly encountering EOF
	// establishing POD [String] bounds.
	ErrEOFDataString
)

func (UnexpectedEOFError) Unwrap() error { return io.ErrUnexpectedEOF }
func (e UnexpectedEOFError) Error() string {
	var suffix string
	switch e {
	case ErrEOFPrefix:
		suffix = "decoding fixed-size POD prefix"
	case ErrEOFData:
		suffix = "establishing POD data bounds"
	case ErrEOFDataString:
		suffix = "establishing POD String bounds"

	default:
		return "unexpected EOF"
	}

	return "unexpected EOF " + suffix
}

// Unmarshal parses the PipeWire POD encoded data and stores the result
// in the value pointed to by v. If v is nil or not a pointer,
// Unmarshal returns an [InvalidUnmarshalError].
func Unmarshal(data []byte, v any) error {
	if n, err := UnmarshalNext(data, v); err != nil {
		return err
	} else if len(data) > int(n) {
		return TrailingGarbageError(data[int(n):])
	}

	return nil
}

// UnmarshalNext implements [Unmarshal] but returns the size of message decoded
// and skips the final trailing garbage check.
func UnmarshalNext(data []byte, v any) (size Word, err error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return 0, &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
	err = unmarshalValue(data, rv.Elem(), &size)
	// prefix and padding size
	size = Size(size)
	return
}

// UnmarshalSetError describes a value that cannot be set during [Unmarshal].
// This is likely an unexported struct field.
type UnmarshalSetError struct{ Type reflect.Type }

func (u *UnmarshalSetError) Error() string { return "cannot set " + u.Type.String() }

// A TrailingGarbageError describes extra bytes after decoding
// has completed during [Unmarshal].
type TrailingGarbageError []byte

func (e TrailingGarbageError) Error() string {
	if len(e) < SizePrefix {
		return "got " + strconv.Itoa(len(e)) + " bytes of trailing garbage"
	}
	return "data has extra values starting with " + SPAKind(binary.NativeEndian.Uint32(e[SizeSPrefix:])).String()
}

// A StringTerminationError describes an incorrectly terminated string
// encountered during [Unmarshal].
type StringTerminationError byte

func (e StringTerminationError) Error() string {
	return "got byte " + strconv.Itoa(int(e)) + " instead of NUL"
}

// unmarshalValue implements [Unmarshal] on [reflect.Value] without compensating for prefix and padding size.
func unmarshalValue(data []byte, v reflect.Value, wireSizeP *Word) error {
	if !v.CanSet() {
		return &UnmarshalSetError{v.Type()}
	}

	if v.CanInterface() {
		if v.Kind() == reflect.Pointer {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if u, ok := v.Interface().(PODUnmarshaler); ok {
			var err error
			*wireSizeP, err = u.UnmarshalPOD(data)
			return err
		}

		switch v.Interface().(type) {
		case Fd:
			*wireSizeP = SizeFd
			if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Fd, wireSizeP); err != nil {
				return err
			}
			v.SetInt(int64(binary.NativeEndian.Uint64(data)))
			return nil
		}
	}

	switch v.Kind() {
	case reflect.Uint32:
		*wireSizeP = SizeId
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Id, wireSizeP); err != nil {
			return err
		}
		v.SetUint(uint64(binary.NativeEndian.Uint32(data)))
		return nil

	case reflect.Int32:
		*wireSizeP = SizeInt
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Int, wireSizeP); err != nil {
			return err
		}
		v.SetInt(int64(binary.NativeEndian.Uint32(data)))
		return nil

	case reflect.Int64:
		*wireSizeP = SizeLong
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Long, wireSizeP); err != nil {
			return err
		}
		v.SetInt(int64(binary.NativeEndian.Uint64(data)))
		return nil

	case reflect.Struct:
		*wireSizeP = 0
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Struct, wireSizeP); err != nil {
			return err
		}

		var fieldWireSize Word
		for i := 0; i < v.NumField(); i++ {
			if err := unmarshalValue(data, v.Field(i), &fieldWireSize); err != nil {
				return err
			}
			// bounds check completed in successful call to unmarshalValue
			data = data[Size(fieldWireSize):]
		}

		if len(data) != 0 {
			return TrailingGarbageError(data)
		}
		return nil

	case reflect.Pointer:
		if len(data) < SizePrefix {
			return ErrEOFPrefix
		}
		switch SPAKind(binary.NativeEndian.Uint32(data[SizeSPrefix:])) {
		case SPA_TYPE_None:
			v.SetZero()
			return nil

		default:
			v.Set(reflect.New(v.Type().Elem()))
			return unmarshalValue(data, v.Elem(), wireSizeP)
		}

	case reflect.String:
		*wireSizeP = 0
		if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_String, wireSizeP); err != nil {
			return err
		}

		// string size, one extra NUL byte
		size := int(*wireSizeP)
		if len(data) < size {
			return ErrEOFDataString
		}

		// the serialised strings still include NUL termination
		if data[size-1] != 0 {
			return StringTerminationError(data[size-1])
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

func (e InconsistentSizeError) Error() string {
	return "prefix claims size " + strconv.Itoa(int(e.Prefix)) +
		" for a " + strconv.Itoa(int(e.Expect)) + "-byte long segment"
}

// An UnexpectedTypeError describes an unexpected type encountered
// in data passed to [Unmarshal].
type UnexpectedTypeError struct{ Type, Expect SPAKind }

func (e UnexpectedTypeError) Error() string {
	return "received " + e.Type.String() + " for a value of type " + e.Expect.String()
}

// unmarshalCheckTypeBounds performs bounds checks on data and validates the type and size prefixes.
// An expected size of zero skips further bounds checks.
func unmarshalCheckTypeBounds(data *[]byte, t SPAKind, sizeP *Word) error {
	if len(*data) < SizePrefix {
		return ErrEOFPrefix
	}

	wantSize := *sizeP
	gotSize := binary.NativeEndian.Uint32(*data)
	*sizeP = gotSize

	if wantSize != 0 && gotSize != wantSize {
		return InconsistentSizeError{gotSize, wantSize}
	}
	if len(*data)-SizePrefix < int(gotSize) {
		return ErrEOFData
	}

	gotType := SPAKind(binary.NativeEndian.Uint32((*data)[SizeSPrefix:]))
	if gotType != t {
		return UnexpectedTypeError{gotType, t}
	}

	*data = (*data)[SizePrefix : gotSize+SizePrefix]
	return nil
}

// The Footer contains additional messages, not directed to
// the destination object defined by the Id field.
type Footer[P KnownSize] struct {
	// The footer opcode.
	Opcode Id `json:"opcode"`
	// The footer payload struct.
	Payload P `json:"payload"`
}

// Size satisfies [KnownSize] with a usually compile-time known value.
func (f *Footer[P]) Size() Word {
	return SizePrefix +
		Size(SizeId) +
		f.Payload.Size()
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (f *Footer[T]) MarshalBinary() ([]byte, error) { return Marshal(f) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (f *Footer[T]) UnmarshalBinary(data []byte) error { return Unmarshal(data, f) }

// SPADictItem represents spa_dict_item.
type SPADictItem struct {
	// Dot-separated string.
	Key string `json:"key"`
	// Arbitrary string.
	//
	// Integer values are represented in base 10,
	// boolean values are represented as "true" or "false".
	Value string `json:"value"`
}

// SPADict represents spa_dict.
type SPADict []SPADictItem

// Size satisfies [KnownSize] with a value computed at runtime.
func (d *SPADict) Size() Word {
	if d == nil {
		return 0
	}

	// struct prefix, NItems value
	size := SizePrefix + int(Size(SizeInt))
	for i := range *d {
		size += SizeString[int]((*d)[i].Key)
		size += SizeString[int]((*d)[i].Value)
	}
	return Word(size)
}

// MarshalPOD satisfies [PODMarshaler] as [SPADict] violates the POD type system.
func (d *SPADict) MarshalPOD(data []byte) ([]byte, error) {
	return appendInner(data, func(dataPrefix []byte) (data []byte, err error) {
		data = SPA_TYPE_Struct.append(dataPrefix)
		if data, err = MarshalAppend(data, Int(len(*d))); err != nil {
			return
		}
		for i := range *d {
			if data, err = MarshalAppend(data, (*d)[i].Key); err != nil {
				return
			}
			if data, err = MarshalAppend(data, (*d)[i].Value); err != nil {
				return
			}
		}
		return
	})
}

// UnmarshalPOD satisfies [PODUnmarshaler] as [SPADict] violates the POD type system.
func (d *SPADict) UnmarshalPOD(data []byte) (Word, error) {
	var wireSize Word
	if err := unmarshalCheckTypeBounds(&data, SPA_TYPE_Struct, &wireSize); err != nil {
		return wireSize, err
	}
	// bounds check completed in successful call to unmarshalCheckTypeBounds
	data = data[:wireSize]

	var count Int
	if size, err := UnmarshalNext(data, &count); err != nil {
		return wireSize, err
	} else {
		// bounds check completed in successful call to Unmarshal
		data = data[size:]
	}

	*d = make([]SPADictItem, count)
	for i := range *d {
		if size, err := UnmarshalNext(data, &(*d)[i].Key); err != nil {
			return wireSize, err
		} else {
			// bounds check completed in successful call to Unmarshal
			data = data[size:]
		}
		if size, err := UnmarshalNext(data, &(*d)[i].Value); err != nil {
			return wireSize, err
		} else {
			// bounds check completed in successful call to Unmarshal
			data = data[size:]
		}
	}

	if len(data) != 0 {
		return wireSize, TrailingGarbageError(data)
	}
	return wireSize, nil
}
