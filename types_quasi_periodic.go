package m3da

import "fmt"

// Register the quasi-periodic vector decoder
func init() {
	registerCustomDecoder(OpCodeQuasiPeriodicVector, decodeQuasiPeriodicVector)
}

// M3daQuasiPeriodicVector represents a type-safe quasi-periodic vector for data compression
type M3daQuasiPeriodicVector[T Numeric] struct {
	Period T
	Start  T
	Shifts []T
}

// GetOpCode returns the operation code for quasi-periodic vectors
func (q *M3daQuasiPeriodicVector[T]) GetOpCode() byte {
	return OpCodeQuasiPeriodicVector
}

// AsFlatList reconstructs the original values from the quasi-periodic vector
func (q *M3daQuasiPeriodicVector[T]) AsFlatList() []T {
	result := []T{q.Start}
	last := q.Start

	for i := 0; i < len(q.Shifts)-1; i += 2 {
		// Get repeat count (convert to int)
		nbRepeat := int(q.Shifts[i])
		shift := q.Shifts[i+1]

		// Add periodic values
		for j := 0; j < nbRepeat; j++ {
			newValue := last + q.Period
			result = append(result, newValue)
			last = newValue
		}

		// Add shifted value
		newValue := last + q.Period + shift
		result = append(result, newValue)
		last = newValue
	}

	// Handle last repeat count
	if len(q.Shifts) > 0 {
		lastRepeat := int(q.Shifts[len(q.Shifts)-1])
		for i := 0; i < lastRepeat; i++ {
			newValue := last + q.Period
			result = append(result, newValue)
			last = newValue
		}
	}

	return result
}

// IsFloatingPoint returns true if the vector contains floating-point values
func (q *M3daQuasiPeriodicVector[T]) IsFloatingPoint() bool {
	var zero T
	switch any(zero).(type) {
	case float32, float64:
		return true
	default:
		return false
	}
}

// EncodeTo encodes the quasi-periodic vector using the provided encoder
func (q *M3daQuasiPeriodicVector[T]) EncodeTo(encoder *BysantEncoder) error {
	// Write opcode directly to encoder's buffer
	encoder.buf.WriteByte(q.GetOpCode())

	// Encode period in NUMBERS context (matches Java BysantContext.NUMBERS)
	if err := encoder.encodeObjectInNumberContext(q.Period); err != nil {
		return err
	}

	// Encode start in NUMBERS context (matches Java BysantContext.NUMBERS)
	if err := encoder.encodeObjectInNumberContext(q.Start); err != nil {
		return err
	}

	// Convert shifts to []interface{} and encode as list
	shiftsList := make([]interface{}, len(q.Shifts))
	for i, shift := range q.Shifts {
		shiftsList[i] = shift
	}

	// Encode shifts as list in LIST_AND_MAPS context (matches original implementation)
	return encoder.encodeListInContext(shiftsList, ContextListAndMaps)
}

// Type aliases for common numeric types
type (
	M3daQuasiPeriodicVectorInt32   = M3daQuasiPeriodicVector[int32]
	M3daQuasiPeriodicVectorInt64   = M3daQuasiPeriodicVector[int64]
	M3daQuasiPeriodicVectorFloat32 = M3daQuasiPeriodicVector[float32]
	M3daQuasiPeriodicVectorFloat64 = M3daQuasiPeriodicVector[float64]
)

// Factory functions for creating specific vector types
func NewQuasiPeriodicVectorInt32(period, start int32, shifts []int32) *M3daQuasiPeriodicVectorInt32 {
	return &M3daQuasiPeriodicVectorInt32{
		Period: period,
		Start:  start,
		Shifts: shifts,
	}
}

func NewQuasiPeriodicVectorInt64(period, start int64, shifts []int64) *M3daQuasiPeriodicVectorInt64 {
	return &M3daQuasiPeriodicVectorInt64{
		Period: period,
		Start:  start,
		Shifts: shifts,
	}
}

func NewQuasiPeriodicVectorFloat32(period, start float32, shifts []float32) *M3daQuasiPeriodicVectorFloat32 {
	return &M3daQuasiPeriodicVectorFloat32{
		Period: period,
		Start:  start,
		Shifts: shifts,
	}
}

func NewQuasiPeriodicVectorFloat64(period, start float64, shifts []float64) *M3daQuasiPeriodicVectorFloat64 {
	return &M3daQuasiPeriodicVectorFloat64{
		Period: period,
		Start:  start,
		Shifts: shifts,
	}
}

// DecodeQuasiPeriodicVector creates a quasi-periodic vector from decoder data
// It determines the appropriate type based on the decoded values
func decodeQuasiPeriodicVector(decoder *BysantDecoder) (M3daEncodable, error) {
	// Decode period in NUMBERS context
	periodObj, err := decoder.decodeObjectInContext(ContextNumber)
	if err != nil {
		return nil, err
	}

	// Decode start in NUMBERS context
	startObj, err := decoder.decodeObjectInContext(ContextNumber)
	if err != nil {
		return nil, err
	}

	// Decode shifts list in LIST_AND_MAPS context
	shiftsObj, err := decoder.decodeObjectInContext(ContextListAndMaps)
	if err != nil {
		return nil, err
	}

	shiftsList, ok := shiftsObj.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid shifts type")
	}

	// Determine the type based on the period type and create appropriate vector
	switch period := periodObj.(type) {
	case int32:
		start, ok := startObj.(int32)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected %T, got %T", periodObj, startObj)
		}

		shifts := make([]int32, len(shiftsList))
		for i, shift := range shiftsList {
			s, ok := shift.(int32)
			if !ok {
				return nil, fmt.Errorf("inconsistent shift type at index %d: expected %T, got %T", i, periodObj, shift)
			}
			shifts[i] = s
		}

		return NewQuasiPeriodicVectorInt32(period, start, shifts), nil
	case int64:
		start, ok := startObj.(int64)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected %T, got %T", periodObj, startObj)
		}

		shifts := make([]int64, len(shiftsList))
		for i, shift := range shiftsList {
			s, ok := shift.(int64)
			if !ok {
				return nil, fmt.Errorf("inconsistent shift type at index %d: expected %T, got %T", i, periodObj, shift)
			}
			shifts[i] = s
		}

		return NewQuasiPeriodicVectorInt64(period, start, shifts), nil

	case float32:
		start, ok := startObj.(float32)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected float32, got %T", startObj)
		}

		shifts := make([]float32, len(shiftsList))
		for i, shift := range shiftsList {
			s, ok := shift.(float32)
			if !ok {
				return nil, fmt.Errorf("inconsistent shift type at index %d: expected float32, got %T", i, shift)
			}
			shifts[i] = s
		}

		return NewQuasiPeriodicVectorFloat32(period, start, shifts), nil

	case float64:
		start, ok := startObj.(float64)
		if !ok {
			return nil, fmt.Errorf("inconsistent start type: expected float64, got %T", startObj)
		}

		shifts := make([]float64, len(shiftsList))
		for i, shift := range shiftsList {
			s, ok := shift.(float64)
			if !ok {
				return nil, fmt.Errorf("inconsistent shift type at index %d: expected float64, got %T", i, shift)
			}
			shifts[i] = s
		}

		return NewQuasiPeriodicVectorFloat64(period, start, shifts), nil

	default:
		return nil, fmt.Errorf("unsupported period type: %T", periodObj)
	}
}
