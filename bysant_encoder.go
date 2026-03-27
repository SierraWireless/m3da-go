package m3da

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// BysantEncoder handles encoding of M3DA objects to binary format
type BysantEncoder struct {
	buf *bytes.Buffer
}

// NewBysantEncoder creates a new Bysant encoder
func NewBysantEncoder() *BysantEncoder {
	return &BysantEncoder{
		buf: &bytes.Buffer{},
	}
}

// Encode encodes M3DA body messages to binary format
func (e *BysantEncoder) Encode(messages ...M3daBodyMessage) ([]byte, error) {
	e.buf.Reset()

	for _, msg := range messages {
		if err := e.encodeObjectInGlobalContext(msg); err != nil {
			return nil, fmt.Errorf("failed to encode message: %w", err)
		}
	}

	return e.buf.Bytes(), nil
}

// EncodeObject encodes a single object (public method for testing and general use)
func (e *BysantEncoder) EncodeObject(obj interface{}) ([]byte, error) {
	e.buf.Reset()
	if err := e.encodeObjectInGlobalContext(obj); err != nil {
		return nil, err
	}
	return e.buf.Bytes(), nil
}

func (e *BysantEncoder) encodeObjectInContext(obj interface{}, ctx EncodingContext) ([]byte, error) {
	e.buf.Reset()
	var err error

	switch ctx {
	case ContextGlobal:
		err = e.encodeObjectInGlobalContext(obj)
	case ContextUintsAndStrs:
		err = e.encodeObjectInUISContext(obj)
	case ContextNumber:
		err = e.encodeObjectInNumberContext(obj)
	case ContextInt32:
		err = e.encodeObjectInInt32Context(obj)
	case ContextFloat:
		err = e.encodeObjectInFloatContext(obj)
	case ContextDouble:
		err = e.encodeObjectInDoubleContext(obj)
	case ContextListAndMaps:
		err = e.encodeObjectInListAndMapsContext(obj)
	default:
		err = fmt.Errorf("unsupported context: %d", ctx)
	}

	return e.buf.Bytes(), err
}

// Context 0: Global, allows the definition of nearly any object but is less compact encoding.
func (e *BysantEncoder) encodeObjectInGlobalContext(obj interface{}) error {
	switch v := obj.(type) {
	case nil:
		return e.encodeNull()
	case bool:
		return e.encodeBool(v)
	case string:
		return e.encodeStringInContext(v, ContextGlobal)
	case int:
		return e.encodeIntegerInContext(int64(v), ContextGlobal)
	case int32:
		return e.encodeIntegerInContext(int64(v), ContextGlobal)
	case int64:
		return e.encodeIntegerInContext(v, ContextGlobal)
	case float32:
		return e.encodeFloat32(v)
	case float64:
		return e.encodeFloat64(v)
	case []byte:
		return e.encodeBinary(v)
	case map[string]interface{}:
		return e.encodeMapInContext(v, ContextGlobal)
	case []interface{}:
		return e.encodeListInContext(v, ContextGlobal)
	case []int:
		// Convert to []interface{} and encode as list
		return e.encodeListInContext(convertSliceToInterface(v), ContextGlobal)
	case []int8:
		// Convert to []interface{} and encode as list
		return e.encodeListInContext(convertSliceToInterface(v), ContextGlobal)
	case []int16:
		// Convert to []interface{} and encode as list
		return e.encodeListInContext(convertSliceToInterface(v), ContextGlobal)
	case []int32:
		// Convert to []interface{} and encode as list
		return e.encodeListInContext(convertSliceToInterface(v), ContextGlobal)
	case []int64:
		// Convert to []interface{} and encode as list
		return e.encodeListInContext(convertSliceToInterface(v), ContextGlobal)
	case []float32:
		// Convert to []interface{} and encode as list
		return e.encodeListInContext(convertSliceToInterface(v), ContextGlobal)
	case []float64:
		// Convert to []interface{} and encode as list
		return e.encodeListInContext(convertSliceToInterface(v), ContextGlobal)
	case M3daEncodable:
		return v.EncodeTo(e)
	default:
		return fmt.Errorf("unsupported type: %T", obj)
	}
}

// Context 1: Unsigned Integers and Strings (UIS), used to encode unsigned numbers and strings. Mainly used for map keys.
func (e *BysantEncoder) encodeObjectInUISContext(obj interface{}) error {
	switch v := obj.(type) {
	case uint:
		return e.encodeUnsignedIntegerInContext(uint32(v), ContextUintsAndStrs)
	case uint16:
		return e.encodeUnsignedIntegerInContext(uint32(v), ContextUintsAndStrs)
	case uint32:
		return e.encodeUnsignedIntegerInContext(uint32(v), ContextUintsAndStrs)
	case string:
		return e.encodeStringInContext(v, ContextUintsAndStrs)
	case []byte:
		return e.encodeBinaryInContext(v, ContextUintsAndStrs)

	default:
		return fmt.Errorf("unsupported type for uint and str context: %T", obj)
	}
}

// Context 2: Numbers, specialized to define numbers efficiently.
func (e *BysantEncoder) encodeObjectInNumberContext(obj interface{}) error {
	switch v := obj.(type) {
	case int:
		return e.encodeIntegerInContext(int64(v), ContextNumber)
	case int8:
		return e.encodeIntegerInContext(int64(v), ContextNumber)
	case int16:
		return e.encodeIntegerInContext(int64(v), ContextNumber)
	case int32:
		return e.encodeIntegerInContext(int64(v), ContextNumber)
	case int64:
		return e.encodeIntegerInContext(v, ContextNumber)
	case float32:
		return e.encodeFloat32(v)
	case float64:
		return e.encodeFloat64(v)
	default:
		return fmt.Errorf("unsupported type for number context: %T", obj)
	}
}

// Context 3: 4 bytes signed integer only (Int32)
func (e *BysantEncoder) encodeObjectInInt32Context(obj interface{}) error {
	const nullToken int32 = -1 << 31 // 0x80000000 = -2147483648

	switch v := obj.(type) {
	case nil:
		binary.Write(e.buf, binary.BigEndian, nullToken)
	case int32:
		binary.Write(e.buf, binary.BigEndian, v)
		if v == nullToken {
			e.buf.WriteByte(0x01)
		}
	default:
		return fmt.Errorf("unsupported type for int32 context: %T", obj)
	}
	return nil
}

// Context 4: 4 bytes floating numbers only (Float)
func (e *BysantEncoder) encodeObjectInFloatContext(obj interface{}) error {
	var nullToken float32 = math.Float32frombits(0xFFFFFFFF)

	switch v := obj.(type) {
	case nil:
		binary.Write(e.buf, binary.BigEndian, nullToken)
	case float32:
		binary.Write(e.buf, binary.BigEndian, v)
		if v == nullToken {
			e.buf.WriteByte(0x01)
		}
	default:
		return fmt.Errorf("unsupported type for float32 context: %T", obj)
	}
	return nil
}

// Context 5: 8 bytes floating numbers only (Double/Float64)
func (e *BysantEncoder) encodeObjectInDoubleContext(obj interface{}) error {
	var nullToken float64 = math.Float64frombits(0xFFFFFFFFFFFFFFF)

	switch v := obj.(type) {
	case nil:
		binary.Write(e.buf, binary.BigEndian, nullToken)
	case float64:
		if v == nullToken {
			e.buf.WriteByte(0x01)
		}
	default:
		return fmt.Errorf("unsupported type for float64 context: %T", obj)
	}
	return nil
}

// Context 6: Lists & Maps
func (e *BysantEncoder) encodeObjectInListAndMapsContext(obj interface{}) error {
	switch v := obj.(type) {
	case []interface{}:
		return e.encodeListInContext(v, ContextListAndMaps)
	case map[string]interface{}:
		return e.encodeMapInContext(v, ContextListAndMaps)
	default:
		return fmt.Errorf("unsupported type for list and map context: %T", obj)
	}
}

func (e *BysantEncoder) encodeNull() error {
	e.buf.WriteByte(0x00)
	return nil
}

func (e *BysantEncoder) encodeBool(b bool) error {
	if b {
		e.buf.WriteByte(0x01)
	} else {
		e.buf.WriteByte(0x02)
	}
	return nil
}

func (e *BysantEncoder) encodeString(s string) error {
	// Use GLOBAL context by default (can be overridden by encodeStringInContext)
	return e.encodeStringInContext(s, ContextGlobal)
}

func (e *BysantEncoder) encodeStringInContext(s string, ctx EncodingContext) error {
	data := []byte(s)
	length := len(data)

	var smallLimit int
	var smallOpcode byte
	var mediumLimit int
	var mediumOpcode byte

	switch ctx {
	case ContextGlobal:
		// GLOBAL context (default)
		smallLimit = 32
		smallOpcode = 0x03
		mediumLimit = 1056
		mediumOpcode = 0x24
	case ContextUintsAndStrs:
		// UINTS_AND_STRS context (for map keys, message paths, etc.)
		smallLimit = 47
		smallOpcode = 0x01
		mediumLimit = 2095
		mediumOpcode = 0x31
	default:
		return fmt.Errorf("unknown context for string encoding: %d", ctx)
	}

	switch {
	case length <= smallLimit:
		// Tiny/small string
		e.buf.WriteByte(smallOpcode + byte(length))
	case length <= mediumLimit:
		// Medium string
		sentLen := length - (smallLimit + 1)
		e.buf.WriteByte(mediumOpcode + byte(sentLen>>8))
		e.buf.WriteByte(byte(sentLen & 0xFF))
	default:
		// Large/chunked strings not implemented for simplicity
		return fmt.Errorf("string too large: %d bytes", length)
	}

	e.buf.Write(data)
	return nil
}

func (e *BysantEncoder) encodeBinary(data []byte) error {
	// Encode as string for simplicity (M3DA treats binary data as strings)
	return e.encodeStringInContext(string(data), ContextGlobal)
}

func (e *BysantEncoder) encodeBinaryInContext(data []byte, ctx EncodingContext) error {
	// Encode as string with specific context (M3DA treats binary data as strings)
	return e.encodeStringInContext(string(data), ctx)
}

func (e *BysantEncoder) encodeIntegerInContext(n int64, ctx EncodingContext) error {
	switch ctx {
	case ContextGlobal:
		return e.encodeSignedIntegerInGlobalContext(n)
	case ContextNumber:
		return e.encodeSignedIntegerInNumbersContext(n)
	default:
		return fmt.Errorf("unsupported context for number encoding: %d", ctx)
	}
}

// encodeSignedIntegerInGlobalContext encodes signed integers in GLOBAL context
func (e *BysantEncoder) encodeSignedIntegerInGlobalContext(n int64) error {
	// Follow Java M3DA compact number encoding exactly
	// Tiny numbers: -31 to 64 (encoded as single byte 0x80 + offset)
	if n >= -31 && n <= 64 {
		e.buf.WriteByte(byte(0x80 + (n + 31)))
		return nil
	}

	// Small numbers: -2079 to 2112 (2-byte encoding)
	if n >= -2079 && n <= 2112 {
		if n < 0 {

			// Small negative
			offset := (-n) - 32
			e.buf.WriteByte(byte(0xE8 + (offset >> 8)))
			e.buf.WriteByte(byte(offset & 0xFF))
		} else {
			// Small positive
			offset := n - 65
			e.buf.WriteByte(byte(0xE0 + (offset >> 8)))
			e.buf.WriteByte(byte(offset & 0xFF))
		}
		return nil
	}

	// Large integer (26 bits) with MSB as sign bit.
	// Possible range is from -33818655 (-(1<<25)-264223) to 33818688 ((1<<25)+264256).
	// FIXME To Implement

	// For larger numbers, use int64 encoding
	// Note there is a 0xFC for Int32
	e.buf.WriteByte(OpCodeInt64)
	return binary.Write(e.buf, binary.BigEndian, n)
}

// encodeSignedIntegerInNumbersContext encodes signed integers in NUMBERS context
func (e *BysantEncoder) encodeSignedIntegerInNumbersContext(n int64) error {
	// NUMBERS context encoding (matches Java NumbersCtxEncoding)
	// Tiny numbers: -97 to +97 (encoded as single byte 0x01 + offset)
	if n >= -97 && n <= 97 {
		e.buf.WriteByte(byte(0x01 + (n + 97)))
		return nil
	}

	// Small numbers: -4193 to +4193 (2-byte encoding)
	if n >= -4193 && n <= 4193 {
		if n < 0 {
			// Small negative: 0xD4 + offset
			offset := (-n) - 98
			e.buf.WriteByte(byte(0xD4 + (offset >> 8)))
			e.buf.WriteByte(byte(offset & 0xFF))
		} else {
			// Small positive: 0xC4 + offset
			offset := n - 98
			e.buf.WriteByte(byte(0xC4 + (offset >> 8)))
			e.buf.WriteByte(byte(offset & 0xFF))
		}
		return nil
	}

	// Large integer (27 bits) with MSB as sign bit.
	// Possible range is from -67637345 (-(1<<26)-528481) to 67637345 ((1<<26)+528481).
	// FIXME To Implement

	// For larger numbers, use int64 encoding
	// Note there is a 0xFC for Int32
	e.buf.WriteByte(OpCodeInt64) // same in both contexts
	return binary.Write(e.buf, binary.BigEndian, n)
}

// encodeUnsignedIntegerInContext encodes unsigned integers in UINTS_AND_STRS context
func (e *BysantEncoder) encodeUnsignedIntegerInContext(n uint32, ctx EncodingContext) error {

	switch ctx {
	case ContextUintsAndStrs:
	default:
		return fmt.Errorf("unknown context for unsigned integer encoding: %d", ctx)
	}

	// From C implementation: writeUnsignedInteger
	switch {
	case n <= 139: // BS_UTI_MAX
		e.buf.WriteByte(byte(n + 0x3b))
		return nil
	case n <= 8331: // BS_USI_MAX
		offset := n - 140
		e.buf.WriteByte(byte(0xc7 + (offset >> 8)))
		e.buf.WriteByte(byte(offset & 0xff))
		return nil
	case n <= 1056907: // BS_UMI_MAX
		offset := n - 8332
		e.buf.WriteByte(byte(0xe7 + (offset >> 16)))
		e.buf.WriteByte(byte((offset >> 8) & 0xff))
		e.buf.WriteByte(byte(offset & 0xff))
		return nil
	case n <= 135274635: // BS_ULI_MAX
		offset := n - 1056908
		e.buf.WriteByte(byte(0xf7 + (offset >> 24)))
		e.buf.WriteByte(byte((offset >> 16) & 0xff))
		e.buf.WriteByte(byte((offset >> 8) & 0xff))
		e.buf.WriteByte(byte(offset & 0xff))
		return nil
	default:
		e.buf.WriteByte(0xff)
		return binary.Write(e.buf, binary.BigEndian, n)
	}
}

func (e *BysantEncoder) encodeFloat32(f float32) error {
	e.buf.WriteByte(OpCodeFloat32)
	return binary.Write(e.buf, binary.BigEndian, f)
}

func (e *BysantEncoder) encodeFloat64(f float64) error {
	e.buf.WriteByte(OpCodeFloat64)
	return binary.Write(e.buf, binary.BigEndian, f)
}

// Maps can be in global context or in List And Map context
func (e *BysantEncoder) encodeMapInContext(m map[string]interface{}, ctx EncodingContext) error {

	// Setup per-context specificities
	var emptyOpcode byte
	var smallLimit int
	var smallUntypedOpcode byte
	// FIXME: not implemented
	// var smallTypedOpcode byte
	var knownLenUntypedOpcode byte
	// FIXME: not implemented
	// var knownLenTypedOpcode byte
	// null terminated maps
	// FIXME: not implemented
	// var UnknownLenUntypedOpcode byte
	// var UnknownLenTypedOpcode byte

	switch ctx {
	case ContextGlobal:
		// GLOBAL context (default)
		emptyOpcode = 0x41
		smallLimit = 9
		smallUntypedOpcode = 0x42
		// smallTypedOpcode = 0x4D
		knownLenUntypedOpcode = 0x4B
		// knownLenTypedOpcode = 0x56
		// UnknownLenUntypedOpcode = 0x4C
		// UnknownLenTypedOpcode = 0x57
	case ContextListAndMaps:
		emptyOpcode = 0x83
		smallLimit = 60
		smallUntypedOpcode = 0x84
		// smallTypedOpcode = 0xC2
		knownLenUntypedOpcode = 0xC0
		// knownLenTypedOpcode = 0xFE
		// UnknownLenUntypedOpcode = 0xC1
		// UnknownLenTypedOpcode = 0xFF
	default:
		return fmt.Errorf("unknown context for map encoding: %d", ctx)
	}

	size := len(m)
	switch {
	case size == 0:
		e.buf.WriteByte(emptyOpcode)
		return nil
	case size <= smallLimit:
		e.buf.WriteByte(smallUntypedOpcode + byte(len(m)) - 1)
	default:
		e.buf.WriteByte(knownLenUntypedOpcode)
		if err := e.encodeUnsignedIntegerInContext(uint32(len(m)-smallLimit-1), ContextUintsAndStrs); err != nil {
			return err
		}
	}

	// Encode key-value pairs
	for key, value := range m {
		// Map keys ALWAYS use UINTS_AND_STRS context
		if err := e.encodeStringInContext(key, ContextUintsAndStrs); err != nil {
			return err
		}
		// FIXME only support untyped pair (value in global context) currently
		if err := e.encodeObjectInGlobalContext(value); err != nil {
			return err
		}
	}

	return nil
}

func (e *BysantEncoder) encodeListInContext(list []interface{}, ctx EncodingContext) error {

	// Setup per-context specificities
	var emptyOpcode byte
	var smallLimit int
	var smallUntypedOpcode byte
	// FIXME: not implemented
	// var smallTypedOpcode byte
	var knownLenUntypedOpcode byte
	// FIXME: not implemented
	// var knownLenTypedOpcode byte
	// null terminated maps
	// FIXME: not implemented
	// var UnknownLenUntypedOpcode byte
	// var UnknownLenTypedOpcode byte

	switch ctx {
	case ContextGlobal:
		// GLOBAL context (default)
		emptyOpcode = 0x2A
		smallLimit = 9
		smallUntypedOpcode = 0x2B
		// smallTypedOpcode = 0x36
		knownLenUntypedOpcode = 0x34
		// knownLenTypedOpcode = 0x3F
		// UnknownLenUntypedOpcode = 0x35
		// UnknownLenTypedOpcode = 0x40
	case ContextListAndMaps:
		emptyOpcode = 0x01
		smallLimit = 60
		smallUntypedOpcode = 0x02
		// smallTypedOpcode = 0x40
		knownLenUntypedOpcode = 0x3E
		// knownLenTypedOpcode = 0x7C
		// UnknownLenUntypedOpcode = 0x3F
		// UnknownLenTypedOpcode = 0x7D
	default:
		return fmt.Errorf("unknown context for list encoding: %d", ctx)
	}

	size := len(list)
	switch {
	case size == 0:
		// Empty list opcodes by context
		e.buf.WriteByte(emptyOpcode)
		return nil
	case size <= smallLimit:
		// Small untyped list: opcode include size
		e.buf.WriteByte(byte(int(smallUntypedOpcode) + size - 1))
	default:
		// Large untyped list: opcodefollowed by size
		e.buf.WriteByte(knownLenUntypedOpcode)
		if err := e.encodeUnsignedIntegerInContext(uint32(size-smallLimit-1), ContextUintsAndStrs); err != nil {
			return err
		}
	}

	// Encode list elements (all in GLOBAL context for untyped mode)
	for _, item := range list {
		if err := e.encodeObjectInGlobalContext(item); err != nil {
			return err
		}
	}

	return nil
}
