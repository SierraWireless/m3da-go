package m3da

import (
	"encoding/binary"
	"fmt"
	"io"
)

// customTypeDecoders maps opcodes to decoder functions for extensible types
var customTypeDecoders = make(map[byte]func(*BysantDecoder) (M3daEncodable, error))

// RegisterCustomDecoder allows types to register their own decoders
func registerCustomDecoder(opcode byte, decoder func(*BysantDecoder) (M3daEncodable, error)) {
	customTypeDecoders[opcode] = decoder
}

// BysantDecoder handles decoding of binary data to M3DA objects
type BysantDecoder struct {
	reader io.Reader
}

// NewBysantDecoder creates a new Bysant decoder
func NewBysantDecoder(reader io.Reader) *BysantDecoder {
	return &BysantDecoder{reader: reader}
}

// Decode decodes binary data to M3DA body messages
func (d *BysantDecoder) Decode() ([]M3daBodyMessage, error) {
	var messages []M3daBodyMessage

	for {
		obj, err := d.decodeObjectInContext(ContextGlobal)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if msg, ok := obj.(M3daBodyMessage); ok {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// Decode One Message
func (d *BysantDecoder) decodeMessage() (M3daBodyMessage, error) {
	obj, err := d.decodeObjectInContext(ContextGlobal)
	if err != nil {
		return nil, err
	}

	if msg, ok := obj.(M3daBodyMessage); ok {
		return msg, nil
	} else {
		return nil, fmt.Errorf("decoded message is not a main body type")
	}
}

func (d *BysantDecoder) decodeObjectInContext(ctx EncodingContext) (interface{}, error) {
	opcode, err := d.readByte()
	if err != nil {
		return nil, err
	}

	switch ctx {
	case ContextGlobal:
		return d.decodeGlobalObject(opcode)
	case ContextUintsAndStrs:
		return d.decodeUintsAndStrsObject(opcode)
	case ContextNumber:
		return d.decodeNumbersObject(opcode)
	case ContextListAndMaps:
		return d.decodeListAndMapsObject(opcode)
	default:
		return nil, fmt.Errorf("unsupported context: %d", ctx)
	}
}

func (d *BysantDecoder) decodeGlobalObject(opcode byte) (interface{}, error) {
	switch {
	case opcode == 0x00: // NULL
		return nil, nil
	case opcode == 0x01: // Boolean TRUE
		return true, nil
	case opcode == 0x02: // Boolean FALSE
		return false, nil
	case opcode >= 0x03 && opcode <= 0x23:
		// Small strings (0-31 bytes)
		return d.decodeStringLength(int(opcode - 0x03))
	case opcode >= 0x24 && opcode <= 0x27:
		// Medium strings (33-1056 bytes)
		// Length is 33 + (OPCODE - 0x24) * 256 + BYTEa
		return d.decodeMediumString(opcode-0x24, 33)
	case opcode == 0x28: // Large strings 1057 to 66592 (1056+65535) bytes.
		// String length is 1057 + BYTEaBYTEb
		return d.decodeLargeString(1057)
	case opcode == 0x29: // Chunked strings
		return nil, fmt.Errorf("un-implemented chunked strings format in GLOBAL context, opcode: 0x%02X", opcode)
	case opcode == 0x2a: // Empty list in GLOBAL context
		return []interface{}{}, nil
	case opcode >= 0x2b && opcode <= 0x33:
		// Small untyped lists (1-9 elements) in GLOBAL context
		size := int(opcode - 0x2b + 1)
		return d.decodeListInContext(size, ContextGlobal)
	case opcode == 0x34: // Large untyped list in GLOBAL context
		sizeObj, err := d.decodeObjectInContext(ContextUintsAndStrs)
		if err != nil {
			return nil, err
		}
		sizeVal, ok := sizeObj.(int64)
		if !ok {
			return nil, fmt.Errorf("invalid list size type")
		}
		size := int(sizeVal) + 9 + 1 // offset + BS_GSC_MAX + 1
		return d.decodeListInContext(size, ContextGlobal)
	case opcode == 0x41: // Empty map
		return make(map[string]interface{}), nil
	case opcode >= 0x42 && opcode <= 0x4A:
		// Small maps (1-9 pairs) in GLOBAL context
		size := int(opcode - 0x42 + 1)
		return d.decodeMapInContext(size, ContextGlobal)
	case opcode == 0x4B:
		// map of 10 or more untyped pairs
		sizeObj, err := d.decodeObjectInContext(ContextUintsAndStrs)
		if err != nil {
			return nil, err
		}
		sizeVal, ok := sizeObj.(uint)
		if !ok {
			return nil, fmt.Errorf("invalid map size type")
		}
		return d.decodeMapInContext(int(sizeVal), ContextGlobal)
	case opcode >= 0x80 && opcode <= 0xDF:
		// Tiny integers (-31 to +64)
		return int64(opcode) - (0x80 + 31), nil
	case opcode >= 0xE0 && opcode <= 0xE7:
		// Small positive integers - 2 bytes
		return d.decodeSmallPositiveInt(opcode-0xE0, 65)
	case opcode >= 0xE8 && opcode <= 0xEF:
		// Small negative integers - 2 bytes
		return d.decodeSmallNegativeInt(opcode-0xE8, 32)
	case opcode >= 0xF0 && opcode <= 0xF3:
		// Medium positive integers - 3 bytes
		return d.decodeMediumPositiveInt(opcode-0xF0, 2113)
	case opcode >= 0xF4 && opcode <= 0xF7:
		// Medium negative integers - 3 bytes
		return d.decodeMediumNegativeInt(opcode-0xF4, 2080)
	case opcode >= 0xF8 && opcode <= 0xF9:
		// Large positive integers - 4 bytes
		return d.decodeLargePositiveInt(opcode-0xF8, 264257)
	case opcode >= 0xFA && opcode <= 0xFB:
		// Large negative integers - 4 bytes
		return d.decodeLargeNegativeInt(opcode-0xFA, 264224)

	case opcode == 0xFD: // 64-bit integer
		return d.decodeInt64()
	case opcode == 0xFE: // 32-bit float
		return d.decodeFloat32()
	case opcode == 0xFF: // 64-bit double
		return d.decodeFloat64()
	case customTypeDecoders[opcode] != nil:
		return customTypeDecoders[opcode](d)
	default:
		return nil, fmt.Errorf("unknown GLOBAL opcode: 0x%02X", opcode)
	}
}

func (d *BysantDecoder) decodeUintsAndStrsObject(opcode byte) (interface{}, error) {
	switch {
	case opcode == 0x00: // NULL
		return nil, nil
	case opcode >= 0x01 && opcode <= 0x30:
		// Small strings (0-47 bytes) - Note: opcode 0x01 = length 0
		return d.decodeStringLength(int(opcode - 0x01))
	case opcode >= 0x31 && opcode <= 0x38:
		// Medium strings (48-2095 bytes) in UINTS_AND_STRS context
		// Length is 48 + (OPCODE - 0x31) * 256 + BYTEa
		return d.decodeMediumString(opcode-0x31, 48)
	case opcode == 0x39: // Large strings 2096 to 67631 (2096+65535) bytes.
		// String length is 2095 + BYTEaBYTEb
		return d.decodeLargeString(2095)
	case opcode == 0x3A: // Chunked strings
		return nil, fmt.Errorf("un-implemented chunked strings format in UINTS_AND_STRS opcode: 0x%02X", opcode)
	case opcode >= 0x3B && opcode <= 0xC6:
		// Tiny unsigned integers (0-139)
		return uint32(opcode - 0x3B), nil
	case opcode >= 0xC7 && opcode <= 0xE6:
		// Small unsigned integers
		return d.decodeUnsignedShort(opcode-0xC7, 140)
	case opcode >= 0xE7 && opcode <= 0xF6:
		// Medium unsigned integers
		return d.decodeUnsignedMedium(opcode-0xE7, 8332)
	case opcode >= 0xF7 && opcode <= 0xFE:
		// Large unsigned integers
		return d.decodeUnsignedLong(opcode-0xF7, 1056908)
	case opcode == 0xFF: // Very large unsigned integer (> 135274635)
		var n uint32
		err := binary.Read(d.reader, binary.BigEndian, &n)
		return int64(n), err
	default:
		return nil, fmt.Errorf("unknown UINTS_AND_STRS opcode: 0x%02X", opcode)
	}
}

func (d *BysantDecoder) decodeNumbersObject(opcode byte) (interface{}, error) {
	switch {
	case opcode >= 0x01 && opcode <= 0xC3:
		// Tiny numbers: -97 to +97 (encoded as 0x01 + (n + 97))
		// So to decode: n = opcode - 0x01 - 97
		return int64(opcode) - 0x01 - 97, nil
	case opcode >= 0xC4 && opcode <= 0xD3:
		// Small positive numbers: 98 to 4193
		return d.decodeSmallPositiveInt(opcode-0xC4, 98)
	case opcode >= 0xD4 && opcode <= 0xE3:
		// Small negative numbers: -98 to -4193
		return d.decodeSmallNegativeInt(opcode-0xD4, 98)

	// Medium integer (20 bits), MSB as sign bit
	// Possible range is from -528481 (-(1<<19)-4193) to 528481 ((1<<19)+4193).
	case opcode >= 0xE4 && opcode <= 0xEB:
		// Medium positive numbers: 4194 to 528481
		return d.decodeMediumPositiveInt(opcode-0xE4, 4194)
	case opcode >= 0xEC && opcode <= 0xF3:
		// Medium negative numbers: -528481 to -4193
		return d.decodeMediumNegativeInt(opcode-0xEC, 4194)

	// Large integer (27 bits), MSB as sign bit
	// Possible range is from -67637345 (-(1<<26)-528481) to 67637345 ((1<<26)+528481).
	case opcode >= 0xF4 && opcode <= 0xF7:
		// Large positive numbers: 528482 to 67637345
		return d.decodeLargePositiveInt(opcode-0xF4, 528481)
	case opcode >= 0xF8 && opcode <= 0xFB:
		// Large negative numbers: -67637345 to -528482
		return d.decodeLargeNegativeInt(opcode-0xFB, 528481)

	case opcode == 0xFC: // 32-bit integer in NUMBERS context
		return d.decodeInt32()
	case opcode == 0xFD: // 64-bit integer in NUMBERS context
		return d.decodeInt64()
	case opcode == 0xFE: // 32-bit floating point in NUMBERS context
		return d.decodeFloat32()
	case opcode == 0xFF: // 64-bit floating point in NUMBERS context
		return d.decodeFloat64()
	default:
		return nil, fmt.Errorf("unknown NUMBERS context opcode: 0x%02X", opcode)
	}
}

func (d *BysantDecoder) decodeListAndMapsObject(opcode byte) (interface{}, error) {
	switch {
	case opcode == 0x01: // Empty list in LIST_AND_MAPS context
		return []interface{}{}, nil
	case opcode >= 0x02 && opcode <= 0x3D:
		// Small untyped lists (1-60 elements) in LIST_AND_MAPS context
		size := int(opcode - 0x02 + 1)
		return d.decodeListInContext(size, ContextGlobal)
	case opcode == 0x3f: // Large untyped list in LIST_AND_MAPS context
		sizeObj, err := d.decodeObjectInContext(ContextUintsAndStrs)
		if err != nil {
			return nil, err
		}
		sizeVal, ok := sizeObj.(int64)
		if !ok {
			return nil, fmt.Errorf("invalid list size type")
		}
		size := int(sizeVal) + 60 + 1 // offset + BS_LMSC_MAX + 1
		return d.decodeListInContext(size, ContextGlobal)
	case opcode == 0x83: // Empty map
		return make(map[string]interface{}), nil
	case opcode >= 0x84 && opcode <= 0xBF:
		// Small maps (1-60 pairs)
		size := int(opcode - 0x84 + 1)
		return d.decodeMapInContext(size, ContextGlobal)
	case opcode == 0xC0:
		// map of 61 or more untyped pairs
		sizeObj, err := d.decodeObjectInContext(ContextUintsAndStrs)
		if err != nil {
			return nil, err
		}
		sizeVal, ok := sizeObj.(uint)
		if !ok {
			return nil, fmt.Errorf("invalid map size type")
		}
		return d.decodeMapInContext(int(sizeVal), ContextGlobal)
	//case opcode == 0xC1: // null terminated map pairs
	default:
		return nil, fmt.Errorf("unknown LIST_AND_MAPS opcode: 0x%02X", opcode)
	}
}

func (d *BysantDecoder) decodeStringLength(length int) (string, error) {
	data := make([]byte, length)
	_, err := io.ReadFull(d.reader, data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (d *BysantDecoder) decodeMediumString(highByte byte, offset int) (string, error) {
	// Medium strings in UINTS_AND_STRS context: 0x31 + high_byte, low_byte, data
	// sentLen = length - (UISStringSmallLimit + 1) = length - 48
	// opcode = UISStringMediumOpcode + (sentLen >> 8) = 0x31 + (sentLen >> 8)

	lowByte, err := d.readByte()
	if err != nil {
		return "", err
	}

	length := int(highByte)<<8 + int(lowByte) + offset

	return d.decodeStringLength(length)
}

func (d *BysantDecoder) decodeLargeString(offset int) (string, error) {
	// Large strings in UINTS_AND_STRS context: 0x3a, high_byte, low_byte, data
	// sentLen = length - (UISStringSmallLimit + 1) = length - 48
	// opcode = UISStringMediumOpcode + (sentLen >> 8) = 0x31 + (sentLen >> 8)
	var highByte byte
	var err error

	if highByte, err = d.readByte(); err != nil {
		return "", err
	}

	return d.decodeMediumString(highByte, offset)
}

func (d *BysantDecoder) decodeInt32() (int32, error) {
	var n int32
	err := binary.Read(d.reader, binary.BigEndian, &n)
	return n, err
}

func (d *BysantDecoder) decodeInt64() (int64, error) {
	var n int64
	err := binary.Read(d.reader, binary.BigEndian, &n)
	return n, err
}

func (d *BysantDecoder) decodeFloat32() (float32, error) {
	var f float32
	err := binary.Read(d.reader, binary.BigEndian, &f)
	return f, err
}

func (d *BysantDecoder) decodeFloat64() (float64, error) {
	var f float64
	err := binary.Read(d.reader, binary.BigEndian, &f)
	return f, err
}

func (d *BysantDecoder) readByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(d.reader, buf)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

func (d *BysantDecoder) decodeUnsignedShort(highByte byte, offset uint32) (uint32, error) {
	// Unsigned short integer: 2 bytes
	lowByte, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return uint32(highByte)<<8 + uint32(lowByte) + offset, nil
}

func (d *BysantDecoder) decodeUnsignedMedium(highByte byte, offset uint32) (uint32, error) {
	// Unsigned medium integer: 3 bytes
	b1, err := d.readByte()
	if err != nil {
		return 0, err
	}
	b2, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return uint32(highByte)<<16 + uint32(b1)<<8 + uint32(b2) + offset, nil
}

func (d *BysantDecoder) decodeUnsignedLong(highByte byte, offset uint32) (uint32, error) {
	// Unsigned long integer: 4 bytes
	b1, err := d.readByte()
	if err != nil {
		return 0, err
	}
	b2, err := d.readByte()
	if err != nil {
		return 0, err
	}
	b3, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return uint32(highByte)<<24 + uint32(b1)<<16 + uint32(b2)<<8 + uint32(b3) + offset, nil
}

func (d *BysantDecoder) decodeMapInContext(size int, valueContext EncodingContext) (map[string]interface{}, error) {
	m := make(map[string]interface{})

	for i := 0; i < size; i++ {
		// Decode key (always in UINTS_AND_STRS context)
		keyObj, err := d.decodeObjectInContext(ContextUintsAndStrs)
		if err != nil {
			return nil, fmt.Errorf("failed to decode map key: %w", err)
		}

		key, ok := keyObj.(string)
		if !ok {
			return nil, fmt.Errorf("map key is not a string")
		}

		// Decode value in specified context
		value, err := d.decodeObjectInContext(valueContext)
		if err != nil {
			return nil, fmt.Errorf("failed to decode map value: %w", err)
		}

		m[key] = value
	}

	return m, nil
}

func (d *BysantDecoder) decodeSmallPositiveInt(highByte byte, offset int64) (int64, error) {
	// Small positive integers - 2 bytes
	b, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return int64(highByte)<<8 + int64(b) + offset, nil
}

func (d *BysantDecoder) decodeSmallNegativeInt(highByte byte, offset int64) (int64, error) {
	// Small negative integers - 2 bytes
	v, err := d.decodeSmallPositiveInt(highByte, offset)
	if err != nil {
		return 0, err
	}
	return -1 * v, nil
}

func (d *BysantDecoder) decodeMediumPositiveInt(highByte byte, offset int64) (int64, error) {
	b1, err := d.readByte()
	if err != nil {
		return 0, err
	}
	b2, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return int64(highByte)<<16 + int64(b1)<<8 + int64(b2) + offset, nil
}

func (d *BysantDecoder) decodeMediumNegativeInt(highByte byte, offset int64) (int64, error) {
	v, err := d.decodeMediumPositiveInt(highByte, offset)
	if err != nil {
		return 0, err
	}
	return -1 * v, nil
}

func (d *BysantDecoder) decodeLargePositiveInt(highByte byte, offset int64) (int64, error) {
	b1, err := d.readByte()
	if err != nil {
		return 0, err
	}
	b2, err := d.readByte()
	if err != nil {
		return 0, err
	}
	b3, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return int64(highByte)<<24 + int64(b1)<<16 + int64(b2)<<8 + int64(b3) + offset, nil
}

func (d *BysantDecoder) decodeLargeNegativeInt(highByte byte, offset int64) (int64, error) {
	v, err := d.decodeLargePositiveInt(highByte, offset)
	if err != nil {
		return 0, err
	}
	return -1 * v, nil
}

func (d *BysantDecoder) decodeListInContext(size int, elementContext EncodingContext) ([]interface{}, error) {
	list := make([]interface{}, size)

	for i := 0; i < size; i++ {
		obj, err := d.decodeObjectInContext(elementContext)
		if err != nil {
			return nil, fmt.Errorf("failed to decode list element %d: %w", i, err)
		}
		list[i] = obj
	}

	return list, nil
}
