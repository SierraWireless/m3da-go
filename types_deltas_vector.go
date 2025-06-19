package m3da

import "fmt"

// Register the deltas vector decoder
func init() {
	registerCustomDecoder(OpCodeDeltasVector, decodeDeltasVector)
}

// M3daDeltasVector represents a type-safe deltas vector for data compression
type M3daDeltasVector[T Numeric] struct {
	Factor T
	Start  T
	Deltas []T
}

// GetOpCode returns the operation code for deltas vectors
func (d *M3daDeltasVector[T]) GetOpCode() byte {
	return OpCodeDeltasVector
}

// AsFlatList reconstructs the original values from the deltas vector
func (d *M3daDeltasVector[T]) AsFlatList() []T {
	result := make([]T, len(d.Deltas)+1)
	result[0] = d.Factor * d.Start

	for i, delta := range d.Deltas {
		result[i+1] = result[i] + delta*d.Factor
	}

	return result
}

// IsFloatingPoint returns true if the vector contains floating-point values
func (d *M3daDeltasVector[T]) IsFloatingPoint() bool {
	var zero T
	switch any(zero).(type) {
	case float32, float64:
		return true
	default:
		return false
	}
}

// EncodeTo encodes the deltas vector using the provided encoder
func (d *M3daDeltasVector[T]) EncodeTo(encoder *BysantEncoder) error {
	// Write opcode directly to encoder's buffer (like encodeDeltasVector does)
	encoder.buf.WriteByte(d.GetOpCode())

	// Encode factor in NUMBERS context (matches Java BysantContext.NUMBERS)
	if err := encoder.encodeNumberInNumberContext(d.Factor); err != nil {
		return err
	}

	// Encode start in NUMBERS context (matches Java BysantContext.NUMBERS)
	if err := encoder.encodeNumberInNumberContext(d.Start); err != nil {
		return err
	}

	// Convert deltas to []interface{} and encode as list
	deltasList := make([]interface{}, len(d.Deltas))
	for i, delta := range d.Deltas {
		deltasList[i] = delta
	}

	// Encode deltas as list in LIST_AND_MAPS context (matches original implementation)
	return encoder.encodeListInContext(deltasList, ContextListAndMaps)
}

// Type aliases for common numeric types
type (
	M3daDeltasVectorInt32   = M3daDeltasVector[int32]
	M3daDeltasVectorInt64   = M3daDeltasVector[int64]
	M3daDeltasVectorFloat32 = M3daDeltasVector[float32]
	M3daDeltasVectorFloat64 = M3daDeltasVector[float64]
)

// Factory functions for creating specific vector types
func NewDeltasVectorInt32(factor, start int32, deltas []int32) *M3daDeltasVectorInt32 {
	return &M3daDeltasVectorInt32{
		Factor: factor,
		Start:  start,
		Deltas: deltas,
	}
}

func NewDeltasVectorInt64(factor, start int64, deltas []int64) *M3daDeltasVectorInt64 {
	return &M3daDeltasVectorInt64{
		Factor: factor,
		Start:  start,
		Deltas: deltas,
	}
}

func NewDeltasVectorFloat32(factor, start float32, deltas []float32) *M3daDeltasVectorFloat32 {
	return &M3daDeltasVectorFloat32{
		Factor: factor,
		Start:  start,
		Deltas: deltas,
	}
}

func NewDeltasVectorFloat64(factor, start float64, deltas []float64) *M3daDeltasVectorFloat64 {
	return &M3daDeltasVectorFloat64{
		Factor: factor,
		Start:  start,
		Deltas: deltas,
	}
}

// DecodeDeltasVector creates a deltas vector from decoder data
// It determines the appropriate type based on the decoded values
func decodeDeltasVector(decoder *BysantDecoder) (M3daEncodable, error) {
	// Decode factor in NUMBERS context
	factorObj, err := decoder.decodeObjectInContext(ContextNumber)
	if err != nil {
		return nil, err
	}

	// Decode start in NUMBERS context
	startObj, err := decoder.decodeObjectInContext(ContextNumber)
	if err != nil {
		return nil, err
	}

	// Decode deltas list in LIST_AND_MAPS context
	deltasObj, err := decoder.decodeObjectInContext(ContextListAndMaps)
	if err != nil {
		return nil, err
	}

	deltasList, ok := deltasObj.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid deltas type")
	}

	// Determine the type based on the factor type and create appropriate vector
	switch factor := factorObj.(type) {
	case int32:
		start, ok := startObj.(int32)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected %T, got %T", factorObj, startObj)
		}

		deltas := make([]int32, len(deltasList))
		for i, delta := range deltasList {
			d, ok := delta.(int32)
			if !ok {
				return nil, fmt.Errorf("inconsistent delta type at index %d: expected %T, got %T", i, factorObj, delta)
			}
			deltas[i] = d
		}

		return NewDeltasVectorInt32(factor, start, deltas), nil

	case int64:
		start, ok := startObj.(int64)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected %T, got %T", factorObj, startObj)
		}

		deltas := make([]int64, len(deltasList))
		for i, delta := range deltasList {
			d, ok := delta.(int64)
			if !ok {
				return nil, fmt.Errorf("inconsistent delta type at index %d: expected %T, got %T", i, factorObj, delta)
			}
			deltas[i] = d
		}

		return NewDeltasVectorInt64(factor, start, deltas), nil

	case float32:
		start, ok := startObj.(float32)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected %T, got %T", factorObj, startObj)
		}

		deltas := make([]float32, len(deltasList))
		for i, delta := range deltasList {
			d, ok := delta.(float32)
			if !ok {
				return nil, fmt.Errorf("inconsistent delta type at index %d: expected %T, got %T", i, factorObj, delta)
			}
			deltas[i] = d
		}

		return NewDeltasVectorFloat32(factor, start, deltas), nil

	case float64:
		start, ok := startObj.(float64)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected %T, got %T", factorObj, startObj)
		}

		deltas := make([]float64, len(deltasList))
		for i, delta := range deltasList {
			d, ok := delta.(float64)
			if !ok {
				return nil, fmt.Errorf("inconsistent delta type at index %d: expected %T, got %T", i, factorObj, delta)
			}
			deltas[i] = d
		}

		return NewDeltasVectorFloat64(factor, start, deltas), nil

	default:
		return nil, fmt.Errorf("unsupported factor type: %T", factorObj)
	}
}
