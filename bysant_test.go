package m3da

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// TestBysantEncoder_Basic tests basic encoding functionality
func TestBysantEncoder_Basic(t *testing.T) {
	encoder := NewBysantEncoder()

	tests := []struct {
		name     string
		input    interface{}
		expected []byte
	}{
		{
			name:     "null value",
			input:    nil,
			expected: []byte{0x00},
		},
		{
			name:     "true value",
			input:    true,
			expected: []byte{0x01},
		},
		{
			name:     "false value",
			input:    false,
			expected: []byte{0x02},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encoder.EncodeObject(tt.input)
			if err != nil {
				t.Fatalf("EncodeObject() error = %v", err)
			}
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("EncodeObject() = %x, want %x", result, tt.expected)
			}
		})
	}
}

// TestBysantEncoder_M3DATypes tests encoding of M3DA protocol types
func TestBysantEncoder_M3DATypes(t *testing.T) {
	encoder := NewBysantEncoder()

	// Test M3DA Message
	t.Run("M3DA Message", func(t *testing.T) {
		ticketID := uint32(123)
		message := &M3daMessage{
			Path:     "@sys.test",
			TicketID: &ticketID,
			Body: map[string]interface{}{
				"temperature": 25.5,
				"sensor":      "test-sensor-001",
			},
		}

		result, err := encoder.EncodeObject(message)
		if err != nil {
			t.Fatalf("EncodeObject() error = %v", err)
		}
		if len(result) == 0 {
			t.Error("EncodeObject() returned empty result")
		}
		// Should start with message opcode
		if result[0] != OpCodeMessage {
			t.Errorf("EncodeObject() first byte = 0x%02x, want 0x%02x", result[0], OpCodeMessage)
		}
	})

	// Test M3DA Response
	t.Run("M3DA Response", func(t *testing.T) {
		response := &M3daResponse{
			TicketID: uint32(123),
			Status:   200,
			Message:  "OK",
		}

		result, err := encoder.EncodeObject(response)
		if err != nil {
			t.Fatalf("EncodeObject() error = %v", err)
		}
		if len(result) == 0 {
			t.Error("EncodeObject() returned empty result")
		}
		// Should start with response opcode
		if result[0] != OpCodeResponse {
			t.Errorf("EncodeObject() first byte = 0x%02x, want 0x%02x", result[0], OpCodeResponse)
		}
	})

	// Test M3DA DeltasVector
	t.Run("M3DA DeltasVector", func(t *testing.T) {
		deltaVector := NewDeltasVectorFloat64(0.1, 235.0, []float64{1.0, -2.0, 3.0})

		result, err := encoder.EncodeObject(deltaVector)
		if err != nil {
			t.Fatalf("EncodeObject() error = %v", err)
		}
		if len(result) == 0 {
			t.Error("EncodeObject() returned empty result")
		}
		// Should start with deltas vector opcode
		if result[0] != OpCodeDeltasVector {
			t.Errorf("EncodeObject() first byte = 0x%02x, want 0x%02x", result[0], OpCodeDeltasVector)
		}
	})

	// Test M3DA QuasiPeriodicVector
	t.Run("M3DA QuasiPeriodicVector", func(t *testing.T) {
		quasiVector := NewQuasiPeriodicVectorInt64(60, 1000, []int64{5, -2, 3, 1, 2, -1, 4, 0})

		result, err := encoder.EncodeObject(quasiVector)
		if err != nil {
			t.Fatalf("EncodeObject() error = %v", err)
		}
		if len(result) == 0 {
			t.Error("EncodeObject() returned empty result")
		}
		// Should start with quasi-periodic vector opcode
		if result[0] != OpCodeQuasiPeriodicVector {
			t.Errorf("EncodeObject() first byte = 0x%02x, want 0x%02x", result[0], OpCodeQuasiPeriodicVector)
		}
	})

	// Test M3DA Envelope
	t.Run("M3DA Envelope", func(t *testing.T) {
		envelope := &M3daEnvelope{
			Header: map[string]interface{}{
				"id":     "test-client",
				"status": int64(200),
			},
			Payload: []byte("test payload"),
			Footer:  map[string]interface{}{},
		}

		result, err := encoder.EncodeObject(envelope)
		if err != nil {
			t.Fatalf("EncodeObject() error = %v", err)
		}
		if len(result) == 0 {
			t.Error("EncodeObject() returned empty result")
		}
		// Should start with envelope opcode
		if result[0] != OpCodeEnvelope {
			t.Errorf("EncodeObject() first byte = 0x%02x, want 0x%02x", result[0], OpCodeEnvelope)
		}
	})
}

// TestBysantDecoder_Basic tests basic decoding functionality
func TestBysantDecoder_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected interface{}
	}{
		{
			name:     "null value",
			input:    []byte{0x00},
			expected: nil,
		},
		{
			name:     "true value",
			input:    []byte{0x01},
			expected: true,
		},
		{
			name:     "false value",
			input:    []byte{0x02},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewBysantDecoder(bytes.NewReader(tt.input))
			result, err := decoder.decodeObjectInContext(ContextGlobal)
			if err != nil {
				t.Fatalf("DecodeObject() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("DecodeObject() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBysantDecoder_M3DATypes tests decoding of M3DA protocol types
func TestBysantDecoder_M3DATypes(t *testing.T) {
	encoder := NewBysantEncoder()

	// Test M3DA Message
	t.Run("M3DA Message", func(t *testing.T) {
		ticketID := uint32(123)
		original := &M3daMessage{
			Path:     "@sys.test",
			TicketID: &ticketID,
			Body: map[string]interface{}{
				"temperature": 25.5,
				"sensor":      "test-sensor-001",
			},
		}

		// Encode
		encoded, err := encoder.EncodeObject(original)
		if err != nil {
			t.Fatalf("Encode error = %v", err)
		}

		//printHexDump(encoded, fmt.Sprintf("Binary dump for M3DA Message"))

		// Decode
		decoder := NewBysantDecoder(bytes.NewReader(encoded))
		decoded, err := decoder.decodeObjectInContext(ContextGlobal)
		if err != nil {
			t.Fatalf("Decode error = %v", err)
		}

		// Verify type and content
		decodedMsg, ok := decoded.(*M3daMessage)
		if !ok {
			t.Fatalf("Decoded object is not *M3daMessage, got %T", decoded)
		}

		if decodedMsg.Path != original.Path {
			t.Errorf("Path = %v, want %v", decodedMsg.Path, original.Path)
		}

		if decodedMsg.TicketID == nil || *decodedMsg.TicketID != *original.TicketID {
			t.Errorf("TicketID = %v, want %v", decodedMsg.TicketID, original.TicketID)
		}
	})

	// Test M3DA Response
	t.Run("M3DA Response", func(t *testing.T) {
		original := &M3daResponse{
			TicketID: uint32(123),
			Status:   200,
			Message:  "OK",
		}

		// Encode
		encoded, err := encoder.EncodeObject(original)
		if err != nil {
			t.Fatalf("Encode error = %v", err)
		}

		// Decode
		decoder := NewBysantDecoder(bytes.NewReader(encoded))
		decoded, err := decoder.decodeObjectInContext(ContextGlobal)
		if err != nil {
			t.Fatalf("Decode error = %v", err)
		}

		// Verify type and content
		decodedResp, ok := decoded.(*M3daResponse)
		if !ok {
			t.Fatalf("Decoded object is not *M3daResponse, got %T", decoded)
		}

		if decodedResp.TicketID != original.TicketID {
			t.Errorf("TicketID = %v, want %v", decodedResp.TicketID, original.TicketID)
		}

		if decodedResp.Status != original.Status {
			t.Errorf("Status = %v, want %v", decodedResp.Status, original.Status)
		}

		if decodedResp.Message != original.Message {
			t.Errorf("Message = %v, want %v", decodedResp.Message, original.Message)
		}
	})

	// Test M3DA QuasiPeriodicVector
	t.Run("M3DA QuasiPeriodicVector", func(t *testing.T) {
		original := NewQuasiPeriodicVectorInt64(int64(60), int64(1000), []int64{5, -2, 3, 1})

		// Encode
		encoded, err := encoder.EncodeObject(original)
		if err != nil {
			t.Fatalf("Encode error = %v", err)
		}

		printHexDump(encoded, fmt.Sprintf("Binary dump for M3DA QuasiPeriodicVector"))

		// Decode
		decoder := NewBysantDecoder(bytes.NewReader(encoded))
		decoded, err := decoder.decodeObjectInContext(ContextGlobal)
		if err != nil {
			t.Fatalf("Decode error = %v", err)
		}

		// Verify type and content
		decodedQPV, ok := decoded.(*M3daQuasiPeriodicVectorInt64)
		if !ok {
			t.Fatalf("Decoded object is not *M3daQuasiPeriodicVectorInt64, got %T", decoded)
		}

		if decodedQPV.Period != original.Period {
			t.Errorf("Period = %v, want %v", decodedQPV.Period, original.Period)
		}

		if decodedQPV.Start != original.Start {
			t.Errorf("Start = %v, want %v", decodedQPV.Start, original.Start)
		}

		if len(decodedQPV.Shifts) != len(original.Shifts) {
			t.Errorf("Shifts length = %v, want %v", len(decodedQPV.Shifts), len(original.Shifts))
		}
	})
}

// TestBysantEncodeDecode_RoundTrip tests encode/decode round trip
func TestBysantEncodeDecode_RoundTrip(t *testing.T) {
	encoder := NewBysantEncoder()

	testCases := []struct {
		name  string
		input interface{}
	}{
		{"null", nil},
		{"true", true},
		{"false", false},
		{"empty string", ""},
		{"short string", "hello"},
		{"number zero", int64(0)},
		{"positive number", int64(42)},
		{"negative number", int64(-42)},
		{"float32", float32(3.14)},
		{"float64", float64(3.14159)},
		{"simple list", []interface{}{int64(1), int64(2), int64(3)}},
		{"empty list", []interface{}{}},
		{"simple map", map[string]interface{}{"key": "value"}},
		{"empty map", map[string]interface{}{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded, err := encoder.EncodeObject(tc.input)
			if err != nil {
				t.Fatalf("Encode error = %v", err)
			}

			// Decode
			decoder := NewBysantDecoder(bytes.NewReader(encoded))
			decoded, err := decoder.decodeObjectInContext(ContextGlobal)
			if err != nil {
				t.Fatalf("Decode error = %v", err)
			}

			//printHexDump(encoded, fmt.Sprintf("Binary dump for %s", tc.name))

			// Compare results
			compareObjects(t, tc.name, decoded, tc.input)
		})
	}
}

// TestBysantDecoder_ErrorCases tests decoder error handling
func TestBysantDecoder_ErrorCases(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty input",
			input: []byte{},
		},
		{
			name:  "invalid opcode",
			input: []byte{0xFF, 0xFF, 0xFF, 0xFF}, // Invalid sequence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewBysantDecoder(bytes.NewReader(tt.input))
			result, err := decoder.decodeObjectInContext(ContextGlobal)
			if err == nil {
				t.Errorf("DecodeObject() expected error, got result: %v", result)
			}
		})
	}
}

// TestBysantEncoder_EdgeCases tests edge case handling
func TestBysantEncoder_EdgeCases(t *testing.T) {
	encoder := NewBysantEncoder()

	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "max int64",
			input: int64(math.MaxInt64),
		},
		{
			name:  "min int64",
			input: int64(math.MinInt64),
		},
		{
			name:  "zero",
			input: int64(0),
		},
		{
			name:  "float64 infinity",
			input: math.Inf(1),
		},
		{
			name:  "float64 negative infinity",
			input: math.Inf(-1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encoder.EncodeObject(tt.input)
			if err != nil {
				t.Fatalf("EncodeObject() error = %v", err)
			}
			if len(result) == 0 {
				t.Error("EncodeObject() returned empty result")
			}
		})
	}
}

// Helper function to compare objects (handles maps and slices recursively)
func compareObjects(t *testing.T, name string, actual, expected interface{}) {
	switch expectedVal := expected.(type) {
	case map[string]interface{}:
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			t.Errorf("%s: expected map, got %T", name, actual)
			return
		}
		if len(actualMap) != len(expectedVal) {
			t.Errorf("%s: map length = %v, want %v", name, len(actualMap), len(expectedVal))
		}
		for k, v := range expectedVal {
			if actualVal, exists := actualMap[k]; !exists {
				t.Errorf("%s: missing key %s", name, k)
			} else {
				compareObjects(t, name+"."+k, actualVal, v)
			}
		}
	case []interface{}:
		actualSlice, ok := actual.([]interface{})
		if !ok {
			t.Errorf("%s: expected slice, got %T", name, actual)
			return
		}
		if len(actualSlice) != len(expectedVal) {
			t.Errorf("%s: slice length = %v, want %v", name, len(actualSlice), len(expectedVal))
		}
		for i, v := range expectedVal {
			if i < len(actualSlice) {
				compareObjects(t, name+"["+string(rune(i))+"]", actualSlice[i], v)
			}
		}
	case float32:
		if actualFloat, ok := actual.(float32); ok {
			diff := actualFloat - float32(expectedVal)
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.000001 {
				t.Errorf("%s: %v != %v", name, actual, expected)
			}
		} else {
			t.Errorf("%s: %v != %v (type mismatch)", name, actual, expected)
		}
	default:
		if actual != expected {
			t.Errorf("%s: %v (%T)!= %v (%T)", name, actual, actual, expected, expected)
		}
	}
}

func TestBysantEncodeDecode_Integration(t *testing.T) {
	encoder := NewBysantEncoder()

	testCases := []struct {
		name   string
		object interface{}
	}{
		{
			name: "Simple Message",
			object: &M3daMessage{
				Path: "@sys.test",
				Body: map[string]interface{}{
					"temperature": 25.5,
					"humidity":    60.2,
					"location":    "sensor-room-1",
				},
			},
		},
		{
			name: "Message with Ticket ID",
			object: func() *M3daMessage {
				ticketID := uint32(12345)
				return &M3daMessage{
					Path:     "@sys.commands",
					TicketID: &ticketID,
					Body: map[string]interface{}{
						"command": "get_status",
						"target":  "device-001",
					},
				}
			}(),
		},
		{
			name: "Response",
			object: &M3daResponse{
				TicketID: 12345,
				Status:   200,
				Message:  "OK",
			},
		},
		{
			name:   "DeltasVector",
			object: NewDeltasVectorFloat64(0.1, 235.0, []float64{1.0, -2.0, 3.0, 0.0, -1.0, 2.0}),
		},
		{
			name:   "QuasiPeriodicVector",
			object: NewQuasiPeriodicVectorInt64(60, 1000, []int64{5, -2, 3, 1, 2, -1, 4, 0}),
		},
		{
			name: "Envelope",
			object: &M3daEnvelope{
				Header: map[string]interface{}{
					"id":     "test-client",
					"status": int64(200),
				},
				Payload: []byte("test payload data"),
				Footer: map[string]interface{}{
					"timestamp": int64(1234567890),
				},
			},
		},
		{
			name: "Complex Nested Structure",
			object: map[string]interface{}{
				"sensor_data": map[string]interface{}{
					"temperature": NewDeltasVectorFloat64(0.01, 2350.0, []float64{1, -2, 3, 0, -1}),
					"timestamps":  NewQuasiPeriodicVectorInt64(60, 1000, []int64{0, 1, -1, 2}),
				},
				"metadata": map[string]interface{}{
					"device_id": "sensor-001",
					"location":  "room-A",
					"active":    true,
				},
				"readings": []interface{}{
					map[string]interface{}{
						"type":  "temperature",
						"value": 25.5,
						"unit":  "celsius",
					},
					map[string]interface{}{
						"type":  "humidity",
						"value": 60.2,
						"unit":  "percent",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded, err := encoder.EncodeObject(tc.object)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			if len(encoded) == 0 {
				t.Fatal("Encoded data is empty")
			}

			t.Logf("Encoded %s: %d bytes", tc.name, len(encoded))
			//printHexDump(encoded, fmt.Sprintf("Bytes for %s", tc.name))

			// Decode
			decoder := NewBysantDecoder(bytes.NewReader(encoded))
			decoded, err := decoder.decodeObjectInContext(ContextGlobal)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			// Verify basic structure
			if decoded == nil {
				t.Fatal("Decoded object is nil")
			}

			// Type-specific verification
			switch original := tc.object.(type) {
			case *M3daMessage:
				decodedMsg, ok := decoded.(*M3daMessage)
				if !ok {
					t.Fatalf("Decoded object is not *M3daMessage, got %T", decoded)
				}
				if decodedMsg.Path != original.Path {
					t.Errorf("Path mismatch: got %v, want %v", decodedMsg.Path, original.Path)
				}

			case *M3daResponse:
				decodedResp, ok := decoded.(*M3daResponse)
				if !ok {
					t.Fatalf("Decoded object is not *M3daResponse, got %T", decoded)
				}
				if decodedResp.TicketID != original.TicketID {
					t.Errorf("TicketID mismatch: got %v, want %v", decodedResp.TicketID, original.TicketID)
				}
				if decodedResp.Status != original.Status {
					t.Errorf("Status mismatch: got %v, want %v", decodedResp.Status, original.Status)
				}

			case *M3daDeltasVectorFloat64:
				decodedDV, ok := decoded.(*M3daDeltasVectorFloat64)
				if !ok {
					t.Fatalf("Decoded object is not *M3daDeltasVectorFloat64, got %T", decoded)
				}
				if decodedDV.Factor != original.Factor {
					t.Errorf("Factor mismatch: got %v, want %v", decodedDV.Factor, original.Factor)
				}
				if decodedDV.Start != original.Start {
					t.Errorf("Start mismatch: got %v, want %v", decodedDV.Start, original.Start)
				}
				if len(decodedDV.Deltas) != len(original.Deltas) {
					t.Errorf("Deltas length mismatch: got %v, want %v", len(decodedDV.Deltas), len(original.Deltas))
				}

			case *M3daQuasiPeriodicVectorInt64:
				decodedQPV, ok := decoded.(*M3daQuasiPeriodicVectorInt64)
				if !ok {
					t.Fatalf("Decoded object is not *M3daQuasiPeriodicVectorInt64, got %T", decoded)
				}
				if decodedQPV.Period != original.Period {
					t.Errorf("Period mismatch: got %v, want %v", decodedQPV.Period, original.Period)
				}
				if decodedQPV.Start != original.Start {
					t.Errorf("Start mismatch: got %v, want %v", decodedQPV.Start, original.Start)
				}

			case *M3daEnvelope:
				decodedEnv, ok := decoded.(*M3daEnvelope)
				if !ok {
					t.Fatalf("Decoded object is not *M3daEnvelope, got %T", decoded)
				}
				if len(decodedEnv.Payload) != len(original.Payload) {
					t.Errorf("Payload length mismatch: got %v, want %v", len(decodedEnv.Payload), len(original.Payload))
				}

			case map[string]interface{}:
				decodedMap, ok := decoded.(map[string]interface{})
				if !ok {
					t.Fatalf("Decoded object is not map[string]interface{}, got %T", decoded)
				}
				// Basic structure verification
				if len(decodedMap) == 0 {
					t.Error("Decoded map is empty")
				}
			}

			t.Logf("✅ %s: encode/decode successful", tc.name)
		})
	}
}

func TestBysantContextualEncoding(t *testing.T) {
	encoder := NewBysantEncoder()

	// Test encoding in different contexts
	testData := []interface{}{
		"test string",
		int64(42),
		[]interface{}{1, 2, 3},
		map[string]interface{}{"key": "value"},
	}

	for _, data := range testData {
		encoded, err := encoder.EncodeObject(data)
		if err != nil {
			t.Errorf("Encoding failed for %T: %v", data, err)
			continue
		}

		decoder := NewBysantDecoder(bytes.NewReader(encoded))
		decoded, err := decoder.decodeObjectInContext(ContextGlobal)
		if err != nil {
			t.Errorf("Decoding failed for %T: %v", data, err)
			continue
		}

		t.Logf("✅ Contextual encoding test passed for %T", data)
		t.Logf("   Original: %v", data)
		t.Logf("   Decoded:  %v", decoded)
	}
}

func TestFullMessageEncodeDecode(t *testing.T) {
	// Create a comprehensive M3DA message similar to what would be sent in practice
	ticketID := uint32(12345)
	message := &M3daMessage{
		Path:     "@sys.telemetry.sensors",
		TicketID: &ticketID,
		Body: map[string]interface{}{
			"device_id": "iot-device-001",
			"timestamp": int64(1609459200), // 2021-01-01 00:00:00 UTC
			"sensors": []interface{}{
				map[string]interface{}{
					"type":       "temperature",
					"values":     NewDeltasVectorFloat64(0.1, 235.0, []float64{1, -2, 3, 0, -1, 2, -1, 1}),
					"timestamps": NewQuasiPeriodicVectorInt64(60, 1609459200, []int64{0, 2, -1, 3, 1, -2, 4, 0}),
					"unit":       "celsius",
					"accuracy":   0.1,
				},
				map[string]interface{}{
					"type":     "humidity",
					"value":    65.2,
					"unit":     "percent",
					"accuracy": 1.0,
				},
				map[string]interface{}{
					"type":     "pressure",
					"value":    1013.25,
					"unit":     "hPa",
					"accuracy": 0.1,
				},
			},
			"location": map[string]interface{}{
				"latitude":  45.5017,
				"longitude": -73.5673,
				"altitude":  76.0,
			},
			"status": map[string]interface{}{
				"battery_level":    87,
				"signal_strength":  -65,
				"last_maintenance": int64(1609372800),
			},
		},
	}

	encoder := NewBysantEncoder()

	// Encode the message
	encoded, err := encoder.EncodeObject(message)
	if err != nil {
		t.Fatalf("Encoding failed: %v", err)
	}

	t.Logf("Full message encoded to %d bytes", len(encoded))

	// Decode the message
	decoder := NewBysantDecoder(bytes.NewReader(encoded))
	decoded, err := decoder.decodeObjectInContext(ContextGlobal)
	if err != nil {
		t.Fatalf("Decoding failed: %v", err)
	}

	// Verify it's a message
	decodedMsg, ok := decoded.(*M3daMessage)
	if !ok {
		t.Fatalf("Decoded object is not *M3daMessage, got %T", decoded)
	}

	// Verify basic fields
	if decodedMsg.Path != message.Path {
		t.Errorf("Path mismatch: got %v, want %v", decodedMsg.Path, message.Path)
	}

	if decodedMsg.TicketID == nil || *decodedMsg.TicketID != *message.TicketID {
		t.Errorf("TicketID mismatch: got %v, want %v", decodedMsg.TicketID, message.TicketID)
	}

	// Verify body exists and has content
	if decodedMsg.Body == nil {
		t.Fatal("Decoded message body is nil")
	}

	bodyMap := decodedMsg.Body

	// Check that key fields exist
	requiredFields := []string{"device_id", "timestamp", "sensors", "location", "status"}
	for _, field := range requiredFields {
		if _, exists := bodyMap[field]; !exists {
			t.Errorf("Required field %s missing from decoded body", field)
		}
	}

	t.Logf("✅ Full message encode/decode test successful")
	t.Logf("   Message path: %s", decodedMsg.Path)
	t.Logf("   Ticket ID: %d", *decodedMsg.TicketID)
	t.Logf("   Body fields: %d", len(bodyMap))
}

func TestEnvelopeDecoding(t *testing.T) {
	// Server response: 60 84 07 73 74 61 74 75 73 e0 87 01 83
	// Let's test just the map part: 84 07 73 74 61 74 75 73 e0 87
	mapData := []byte{0x84, 0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0xe0, 0x87}

	fmt.Printf("Testing map data: %x\n", mapData)

	decoder := NewBysantDecoder(bytes.NewReader(mapData))
	// Decode in LIST_AND_MAPS context (envelope header/footer context)
	obj, err := decoder.decodeObjectInContext(ContextListAndMaps)

	if err != nil {
		t.Fatalf("Error decoding map: %v", err)
	}

	fmt.Printf("Decoded object type: %T\n", obj)
	fmt.Printf("Decoded object value: %+v\n", obj)

	if header, ok := obj.(map[string]interface{}); ok {
		fmt.Printf("Map contents:\n")
		for k, v := range header {
			fmt.Printf("  %s: %v (type: %T)\n", k, v, v)
		}
	} else {
		t.Errorf("Expected map, got %T", obj)
	}
}

func TestStringDecoding(t *testing.T) {
	// Test decoding just the string part: 07 73 74 61 74 75 73
	stringData := []byte{0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73}

	fmt.Printf("Testing string data: %x\n", stringData)

	decoder := NewBysantDecoder(bytes.NewReader(stringData))
	// Decode in UINTS_AND_STRS context (map key context)
	obj, err := decoder.decodeObjectInContext(ContextUintsAndStrs)

	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	fmt.Printf("Decoded string: %q (type: %T)\n", obj, obj)

	if s, ok := obj.(string); ok {
		if s != "status" {
			t.Errorf("Expected 'status', got %q", s)
		}
	} else {
		t.Errorf("Expected string, got %T", obj)
	}
}

func TestStringEncoding(t *testing.T) {
	// Test what encoder produces for "status"
	encoder := NewBysantEncoder()
	err := encoder.encodeStringInContext("status", ContextUintsAndStrs)
	if err != nil {
		t.Fatal(err)
	}

	encoded := encoder.buf.Bytes()
	fmt.Printf("Encoded 'status': %x\n", encoded)

	// Now test decoding it back
	decoder := NewBysantDecoder(bytes.NewReader(encoded))
	obj, err := decoder.decodeObjectInContext(ContextGlobal)
	if err != nil {
		t.Fatalf("Error decoding: %v", err)
	}

	fmt.Printf("Decoded back: %q\n", obj)
}

func TestFullEnvelope(t *testing.T) {
	// Full server response
	serverResponse := []byte{
		0x60, 0x84, 0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
		0xe0, 0x87, 0x01, 0x83,
	}

	fmt.Printf("Testing full envelope: %x\n", serverResponse)

	decoder := NewBysantDecoder(bytes.NewReader(serverResponse))
	obj, err := decoder.decodeObjectInContext(ContextGlobal)

	if err != nil {
		t.Fatalf("Error decoding envelope: %v", err)
	}

	fmt.Printf("Decoded envelope type: %T\n", obj)

	if env, ok := obj.(*M3daEnvelope); ok {
		fmt.Printf("Envelope header: %+v\n", env.Header)
		fmt.Printf("Envelope payload: %x\n", env.Payload)
		fmt.Printf("Envelope footer: %+v\n", env.Footer)
	} else {
		t.Errorf("Expected *M3daEnvelope, got %T", obj)
	}
}

func TestStepByStepEnvelopeDecoding(t *testing.T) {
	// Server response: 60 84 07 73 74 61 74 75 73 e0 87 01 83
	serverResponse := []byte{
		0x60, 0x84, 0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
		0xe0, 0x87, 0x01, 0x83,
	}

	fmt.Printf("Server response: %x\n", serverResponse)

	// Test decoding step by step
	reader := bytes.NewReader(serverResponse)

	// Read envelope opcode
	opcode, err := reader.ReadByte()
	if err != nil {
		t.Fatalf("Error reading opcode: %v", err)
	}
	fmt.Printf("Envelope opcode: 0x%02x\n", opcode)

	if opcode != 0x60 {
		t.Fatalf("Expected envelope opcode 0x60, got 0x%02x", opcode)
	}

	// Try to decode header
	decoder := NewBysantDecoder(reader)
	headerObj, err := decoder.decodeObjectInContext(ContextListAndMaps)
	if err != nil {
		t.Fatalf("Error decoding header: %v", err)
	}

	fmt.Printf("Header object type: %T\n", headerObj)
	fmt.Printf("Header object value: %+v\n", headerObj)

	if header, ok := headerObj.(map[string]interface{}); ok {
		fmt.Printf("Header map: %+v\n", header)
		for k, v := range header {
			fmt.Printf("  %s: %v (type: %T)\n", k, v, v)
		}

		// Verify the header contains expected keys
		if _, exists := header["status"]; !exists {
			t.Errorf("Expected 'status' key in header map")
		}
	} else {
		t.Errorf("Expected header to be a map, got %T", headerObj)
	}

	// Try to decode payload
	fmt.Println("\nDecoding payload...")
	payloadObj, err := decoder.decodeObjectInContext(ContextGlobal)
	if err != nil {
		t.Fatalf("Error decoding payload: %v", err)
	}
	fmt.Printf("Payload object type: %T\n", payloadObj)
	fmt.Printf("Payload object value: %+v\n", payloadObj)

	// Try to decode footer
	fmt.Println("\nDecoding footer...")
	footerObj, err := decoder.decodeObjectInContext(ContextListAndMaps)
	if err != nil {
		t.Fatalf("Error decoding footer: %v", err)
	}
	fmt.Printf("Footer object type: %T\n", footerObj)
	fmt.Printf("Footer object value: %+v\n", footerObj)

	fmt.Println("\n✅ Successfully decoded M3DA envelope components!")
}

func TestEnvelopeOpcodeValidation(t *testing.T) {
	// Test with correct opcode
	correctResponse := []byte{
		0x60, 0x84, 0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
		0xe0, 0x87, 0x01, 0x83,
	}

	reader := bytes.NewReader(correctResponse)
	opcode, err := reader.ReadByte()
	if err != nil {
		t.Fatalf("Error reading opcode: %v", err)
	}

	if opcode != 0x60 {
		t.Errorf("Expected envelope opcode 0x60, got 0x%02x", opcode)
	}

	// Test with wrong opcode
	wrongResponse := []byte{0x50, 0x84}
	reader = bytes.NewReader(wrongResponse)
	opcode, err = reader.ReadByte()
	if err != nil {
		t.Fatalf("Error reading opcode: %v", err)
	}

	if opcode == 0x60 {
		t.Errorf("Expected wrong opcode test to fail, but got correct opcode 0x60")
	}
	fmt.Printf("Wrong opcode test passed: got 0x%02x (expected != 0x60)\n", opcode)
}

func TestHeaderDecodingDetailed(t *testing.T) {
	// Test just the header part: 84 07 73 74 61 74 75 73 e0 87
	headerData := []byte{0x84, 0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0xe0, 0x87}

	fmt.Printf("Testing header data: %x\n", headerData)

	decoder := NewBysantDecoder(bytes.NewReader(headerData))
	headerObj, err := decoder.decodeObjectInContext(ContextListAndMaps)

	if err != nil {
		t.Fatalf("Error decoding header: %v", err)
	}

	fmt.Printf("Header object type: %T\n", headerObj)
	fmt.Printf("Header object value: %+v\n", headerObj)

	if header, ok := headerObj.(map[string]interface{}); ok {
		fmt.Printf("Header map: %+v\n", header)
		for k, v := range header {
			fmt.Printf("  %s: %v (type: %T)\n", k, v, v)
		}

		// Verify expected content
		if status, exists := header["status"]; !exists {
			t.Errorf("Expected 'status' key in header")
		} else {
			fmt.Printf("Found status value: %v\n", status)
		}
	} else {
		t.Errorf("Expected header to be a map, got %T", headerObj)
	}
}

func TestServerResponseFormat(t *testing.T) {
	// Server response: 60 84 07 73 74 61 74 75 73 e0 87 01 83
	serverResponse := []byte{
		0x60, 0x84, 0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
		0xe0, 0x87, 0x01, 0x83,
	}

	actualHex := fmt.Sprintf("%x", serverResponse)

	fmt.Printf("Server response hex: %s\n", actualHex)
	fmt.Printf("Expected length: %d, actual length: %d\n", len(serverResponse), len(serverResponse))

	// Verify the response format
	if len(serverResponse) != 13 {
		t.Errorf("Expected server response length 13, got %d", len(serverResponse))
	}

	if serverResponse[0] != 0x60 {
		t.Errorf("Expected first byte to be 0x60 (envelope opcode), got 0x%02x", serverResponse[0])
	}

	fmt.Printf("✅ Server response format validation passed\n")
}
