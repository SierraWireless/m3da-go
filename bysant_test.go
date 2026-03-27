package m3da

import (
	"bytes"
	"math"
	"testing"
)

// TestBysantEncodeDecode_Comprehensive tests full encode->validate bytes->decode cycle
func TestBysantRoundTrip_Comprehensive(t *testing.T) {
	encoder := NewBysantEncoder()

	testCases := []struct {
		name          string
		input         interface{}
		expectedBytes []byte
		context       EncodingContext
	}{
		{
			name:          "null value",
			input:         nil,
			expectedBytes: []byte{0x00},
			context:       ContextGlobal,
		},
		{
			name:          "true value",
			input:         true,
			expectedBytes: []byte{0x01},
			context:       ContextGlobal,
		},
		{
			name:          "false value",
			input:         false,
			expectedBytes: []byte{0x02},
			context:       ContextGlobal,
		},
		{
			name:          "empty string",
			input:         "",
			expectedBytes: []byte{0x03}, // Just the string opcode for empty string
			context:       ContextGlobal,
		},
		{
			name:          "short string",
			input:         "hello",
			expectedBytes: []byte{0x08, 0x68, 0x65, 0x6c, 0x6c, 0x6f}, // context-sensitive encoding
			context:       ContextGlobal,
		},
		{
			name:          "number zero",
			input:         int64(0),
			expectedBytes: []byte{0x9f}, // compressed integer encoding
			context:       ContextGlobal,
		},
		{
			name:          "positive small number",
			input:         int64(42),
			expectedBytes: []byte{0xc9}, // compressed integer encoding
			context:       ContextGlobal,
		},
		{
			name:          "negative small number",
			input:         int64(-1),
			expectedBytes: []byte{0x9e}, // compressed integer encoding
			context:       ContextGlobal,
		},
		// Note: For complex types like float64, lists, maps, we'll test structure rather than exact bytes
		// since the encoding might be more complex and implementation-dependent
		{
			name:          "float32",
			input:         float32(3.14),
			expectedBytes: []byte{0xfe, 0x40, 0x48, 0xf5, 0xc3},
			context:       ContextGlobal,
		},
		{
			name:          "float64",
			input:         float64(3.14159),
			expectedBytes: []byte{0xff, 0x40, 0x09, 0x21, 0xf9, 0xf0, 0x1b, 0x86, 0x6e},
			context:       ContextGlobal,
		},
		{
			name:          "simple list global context",
			input:         []interface{}{int64(1), int64(2), int64(3)},
			expectedBytes: []byte{0x2d, 0xa0, 0xa1, 0xa2},
			context:       ContextGlobal,
		},
		{
			name:          "empty list global context",
			input:         []interface{}{},
			expectedBytes: []byte{0x2a},
			context:       ContextGlobal,
		},
		{
			name:          "simple map global context",
			input:         map[string]interface{}{"key": "value"},
			expectedBytes: []byte{0x42, 0x04, 0x6b, 0x65, 0x79, 0x08, 0x76, 0x61, 0x6c, 0x75, 0x65},
			context:       ContextGlobal,
		},
		{
			name:          "empty map global context",
			input:         map[string]interface{}{},
			expectedBytes: []byte{0x41},
			context:       ContextGlobal,
		},
		{
			name:          "simple list in List and map context",
			input:         []interface{}{int64(1), int64(2), int64(3)},
			expectedBytes: []byte{0x04, 0xa0, 0xa1, 0xa2},
			context:       ContextListAndMaps,
		},
		{
			name:          "empty list in List and map context",
			input:         []interface{}{},
			expectedBytes: []byte{0x01},
			context:       ContextListAndMaps,
		},
		{
			name:          "simple map in List and map context",
			input:         map[string]interface{}{"key": "value"},
			expectedBytes: []byte{0x84, 0x04, 0x6b, 0x65, 0x79, 0x08, 0x76, 0x61, 0x6c, 0x75, 0x65},
			context:       ContextListAndMaps,
		},
		{
			name:          "empty map in List and map context",
			input:         map[string]interface{}{},
			expectedBytes: []byte{0x83},
			context:       ContextListAndMaps,
		},
		{
			name:          "combined data",
			input:         []interface{}{"test string", int64(42), []interface{}{int64(1), int64(2), int64(3)}, map[string]interface{}{"key": "value"}},
			expectedBytes: []byte{0x2e, 0x0e, 0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0xc9, 0x2d, 0xa0, 0xa1, 0xa2, 0x42, 0x04, 0x6b, 0x65, 0x79, 0x08, 0x76, 0x61, 0x6c, 0x75, 0x65},
			context:       ContextGlobal,
		},
		{
			name:          "max int64",
			input:         int64(math.MaxInt64),
			expectedBytes: []byte{0xfd, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			context:       ContextGlobal,
		},
		{
			name:          "min int64",
			input:         int64(math.MinInt64),
			expectedBytes: []byte{0xfd, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			context:       ContextGlobal,
		},
		{
			name:          "zero",
			input:         int64(0),
			expectedBytes: []byte{0x9f},
			context:       ContextGlobal,
		},
		{
			name:          "float64 infinity",
			input:         math.Inf(1),
			expectedBytes: []byte{0xff, 0x7f, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			context:       ContextGlobal,
		},
		{
			name:          "float64 negative infinity",
			input:         math.Inf(-1),
			expectedBytes: []byte{0xff, 0xff, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			context:       ContextGlobal,
		},
		{
			name:          "status string in uints context",
			input:         "status",
			expectedBytes: []byte{0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73},
			context:       ContextUintsAndStrs,
		},
		{
			name:          "path string in global context",
			input:         "@sys.test",
			expectedBytes: []byte{0x0c, 0x40, 0x73, 0x79, 0x73, 0x2e, 0x74, 0x65, 0x73, 0x74},
			context:       ContextGlobal,
		},
		{
			name: "Simple Message",
			input: &M3daMessage{
				Path: "@sys.test",
				Body: map[string]interface{}{
					"temperature": 25.5,
					"humidity":    60.2,
					"location":    "sensor-room-1",
				},
			},
			expectedBytes: nil, // map order is not garanted
			context:       ContextGlobal,
		},
		{
			name: "Message with Ticket ID",
			input: func() *M3daMessage {
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
			expectedBytes: nil, // map order is not garanted
			context:       ContextGlobal,
		},
		{
			name: "Response",
			input: &M3daResponse{
				TicketID: 12345,
				Status:   0,
				Message:  "",
			},
			expectedBytes: []byte{0x62, 0xe7, 0x0f, 0xad, 0x62, 0x01},
			context:       ContextGlobal,
		},
		{
			name:          "DeltasVector",
			input:         NewDeltasVectorFloat64(0.1, 235.0, []float64{1.0, -2.0, 3.0, 0.0, -1.0, 2.0}),
			expectedBytes: []byte{0x63, 0xff, 0x3f, 0xb9, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9a, 0xff, 0x40, 0x6d, 0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0xff, 0x3f, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x40, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xbf, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			context:       ContextGlobal,
		},
		{
			name:          "QuasiPeriodicVector",
			input:         NewQuasiPeriodicVectorInt64(60, 1000, []int64{5, -2, 3, 1, 2, -1, 4, 0}),
			expectedBytes: []byte{0x64, 0x9e, 0xc7, 0x86, 0x09, 0xa4, 0x9d, 0xa2, 0xa0, 0xa1, 0x9e, 0xa3, 0x9f},
			context:       ContextGlobal,
		},
		{
			name: "Envelope",
			input: &M3daEnvelope{
				Header: map[string]interface{}{
					"id":     "test-client",
					"status": int64(200),
				},
				Payload: []byte("test payload data"),
				Footer: map[string]interface{}{
					"timestamp": int64(1234567890),
				},
			},
			expectedBytes: nil, // map order is not garanted
			context:       ContextGlobal,
		},
		{
			name: "Complex Nested Structure",
			input: map[string]interface{}{
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
			expectedBytes: nil, // map order is not garanted
			context:       ContextGlobal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Encode
			encoded, err := encoder.encodeObjectInContext(tc.input, tc.context)
			if err != nil {
				t.Fatalf("❌ Encode error = %v", err)
			}

			// Step 2: Validate encoded bytes (if expected bytes provided)
			if tc.expectedBytes != nil {
				if !bytes.Equal(encoded, tc.expectedBytes) {
					t.Errorf("❌ Encoded bytes mismatch:\n  got:      %x\n  expected: %x", encoded, tc.expectedBytes)
					printHexDump(encoded, tc.name)
				} else {
					t.Logf("✅ Encoded bytes validated: %x", encoded)
				}
			} else {
				// For complex types, at least validate that we got some output and log it
				if len(encoded) == 0 {
					t.Error("❌ Encoded data is empty")
				}
				t.Logf("ℹ Encoded %s: %x (%d bytes)", tc.name, encoded, len(encoded))
			}

			// Step 3: Decode
			decoder := NewBysantDecoder(bytes.NewReader(encoded))
			decoded, err := decoder.decodeObjectInContext(tc.context)
			if err != nil {
				t.Fatalf("❌ Decode error = %v", err)
			}

			// Step 4: Validate round-trip consistency
			// Type-specific verification
			switch original := tc.input.(type) {
			case *M3daMessage:
				decodedMsg, ok := decoded.(*M3daMessage)
				if !ok {
					t.Fatalf("❌ Decoded object is not *M3daMessage, got %T", decoded)
				}
				if decodedMsg.Path != original.Path {
					t.Errorf("❌ Path mismatch: got %v, want %v", decodedMsg.Path, original.Path)
				}
				// Verify basic fields
				if decodedMsg.Path != original.Path {
					t.Errorf("❌ Path mismatch: got %v, want %v", decodedMsg.Path, original.Path)
				}

				if original.TicketID != nil && (decodedMsg.TicketID == nil || *decodedMsg.TicketID != *original.TicketID) {
					t.Errorf("❌ TicketID mismatch: got %v, want %v", decodedMsg.TicketID, original.TicketID)
				}

				// Verify body exists and has content
				if decodedMsg.Body == nil {
					t.Fatal("❌ Decoded message body is nil")
				} else if len(decodedMsg.Body) != len(original.Body) {
					t.Errorf("❌ Body field count mismatch: got %d, want %d", len(decodedMsg.Body), len(original.Body))
				}

			case *M3daResponse:
				decodedResp, ok := decoded.(*M3daResponse)
				if !ok {
					t.Fatalf("❌ Decoded object is not *M3daResponse, got %T", decoded)
				}
				if decodedResp.TicketID != original.TicketID {
					t.Errorf("❌ TicketID mismatch: got %v, want %v", decodedResp.TicketID, original.TicketID)
				}
				if decodedResp.Status != original.Status {
					t.Errorf("❌ Status mismatch: got %v, want %v", decodedResp.Status, original.Status)
				}

				if decodedResp.Message != original.Message {
					t.Errorf("❌ Message = %v, want %v", decodedResp.Message, original.Message)
				}

			case *M3daDeltasVectorFloat64:
				decodedDV, ok := decoded.(*M3daDeltasVectorFloat64)
				if !ok {
					t.Fatalf("❌ Decoded object is not *M3daDeltasVectorFloat64, got %T", decoded)
				}
				if decodedDV.Factor != original.Factor {
					t.Errorf("❌ Factor mismatch: got %v, want %v", decodedDV.Factor, original.Factor)
				}
				if decodedDV.Start != original.Start {
					t.Errorf("❌ Start mismatch: got %v, want %v", decodedDV.Start, original.Start)
				}
				if len(decodedDV.Deltas) != len(original.Deltas) {
					t.Errorf("❌ Deltas length mismatch: got %v, want %v", len(decodedDV.Deltas), len(original.Deltas))
				}

			case *M3daQuasiPeriodicVectorInt64:
				decodedQPV, ok := decoded.(*M3daQuasiPeriodicVectorInt64)
				if !ok {
					t.Fatalf("❌ Decoded object is not *M3daQuasiPeriodicVectorInt64, got %T", decoded)
				}
				if decodedQPV.Period != original.Period {
					t.Errorf("❌ Period mismatch: got %v, want %v", decodedQPV.Period, original.Period)
				}
				if decodedQPV.Start != original.Start {
					t.Errorf("❌ Start mismatch: got %v, want %v", decodedQPV.Start, original.Start)
				}
				if len(decodedQPV.Shifts) != len(original.Shifts) {
					t.Errorf("❌ Shifts length = %v, want %v", len(decodedQPV.Shifts), len(original.Shifts))
				}

			case *M3daEnvelope:
				decodedEnv, ok := decoded.(*M3daEnvelope)
				if !ok {
					t.Fatalf("❌ Decoded object is not *M3daEnvelope, got %T", decoded)
				}
				if len(decodedEnv.Payload) != len(original.Payload) {
					t.Errorf("❌ Payload length mismatch: got %v, want %v", len(decodedEnv.Payload), len(original.Payload))
				}

			case map[string]interface{}:
				decodedMap, ok := decoded.(map[string]interface{})
				if !ok {
					t.Fatalf("❌ Decoded object is not map[string]interface{}, got %T", decoded)
				}
				// Basic structure verification
				if len(decodedMap) != len(original) {
					t.Error("❌ Decoded map is not same size")
				}
			default:
				if compareObjects(t, tc.name, decoded, tc.input) {
					t.Logf("✅ Round-trip validated for %s", tc.name)
				}
			}
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
				t.Errorf("❌ DecodeObject() expected error, got result: %v", result)
			}
		})
	}
}

// TestFullMessageEncodeDecode tests a comprehensive M3DA message with real-world complexity
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

// TestStringEncodeDecode tests string encoding/decoding with different contexts

// TestEnvelopeComprehensive tests comprehensive envelope functionality
func TestEnvelopeComprehensive(t *testing.T) {
	// Test with known server response data
	serverResponse := []byte{
		0x60, 0x84, 0x07, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
		0xe0, 0x87, 0x01, 0x83,
	}

	t.Logf("Testing decoding server response: %x", serverResponse)

	// Test full envelope decoding
	decoder := NewBysantDecoder(bytes.NewReader(serverResponse))
	obj, err := decoder.decodeObjectInContext(ContextGlobal)
	if err != nil {
		t.Fatalf("Error decoding envelope: %v", err)
	}

	env, ok := obj.(*M3daEnvelope)
	if !ok {
		t.Fatalf("Expected *M3daEnvelope, got %T", obj)
	}

	// Verify envelope structure
	if env.Header == nil {
		t.Error("Envelope header is nil")
	} else {
		t.Logf("Header: %+v", env.Header)
		if _, exists := env.Header["status"]; !exists {
			t.Error("Expected 'status' key in header")
		}
	}

	if env.Payload == nil {
		t.Error("Envelope payload is nil")
	} else {
		t.Logf("Payload: %x", env.Payload)
	}

	if env.Footer == nil {
		t.Error("Envelope footer is nil")
	} else {
		t.Logf("Footer: %+v", env.Footer)
	}

	// Test envelope encoding round-trip
	testEnvelope := &M3daEnvelope{
		Header: map[string]interface{}{
			"id":     "test-client",
			"status": int64(200),
		},
		Payload: []byte("test payload"),
		Footer: map[string]interface{}{
			"timestamp": int64(1234567890),
		},
	}

	encoder := NewBysantEncoder()
	encoded, err := encoder.EncodeObject(testEnvelope)
	if err != nil {
		t.Fatalf("Encoding test envelope failed: %v", err)
	}

	// Verify opcode
	if encoded[0] != OpCodeEnvelope {
		t.Errorf("Expected envelope opcode 0x%02x, got 0x%02x", OpCodeEnvelope, encoded[0])
	}

	// Decode back
	decoder = NewBysantDecoder(bytes.NewReader(encoded))
	decodedEnv, err := decoder.decodeObjectInContext(ContextGlobal)
	if err != nil {
		t.Fatalf("Decoding test envelope failed: %v", err)
	}

	decodedEnvelope, ok := decodedEnv.(*M3daEnvelope)
	if !ok {
		t.Fatalf("Expected *M3daEnvelope, got %T", decodedEnv)
	}

	if len(decodedEnvelope.Payload) != len(testEnvelope.Payload) {
		t.Errorf("Payload length mismatch: got %d, want %d",
			len(decodedEnvelope.Payload), len(testEnvelope.Payload))
	}

	t.Log("✅ Comprehensive envelope test successful")
}

// Helper function to compare objects (handles maps and slices recursively)
func compareObjects(t *testing.T, name string, actual, expected interface{}) bool {
	var result bool = true
	switch expectedVal := expected.(type) {
	case map[string]interface{}:
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			t.Errorf("%s: expected map, got %T", name, actual)
			return false
		}
		if len(actualMap) != len(expectedVal) {
			t.Errorf("%s: map length = %v, want %v", name, len(actualMap), len(expectedVal))
			result = false
		}
		for k, v := range expectedVal {
			if actualVal, exists := actualMap[k]; !exists {
				t.Errorf("%s: missing key %s", name, k)
				result = false
			} else if !compareObjects(t, name+"."+k, actualVal, v) {
				result = false
			}
		}
	case []interface{}:
		actualSlice, ok := actual.([]interface{})
		if !ok {
			t.Errorf("%s: expected slice, got %T", name, actual)
			return false
		}
		if len(actualSlice) != len(expectedVal) {
			t.Errorf("%s: slice length = %v, want %v", name, len(actualSlice), len(expectedVal))
			result = false
		}
		for i, v := range expectedVal {
			if i < len(actualSlice) {
				if !compareObjects(t, name+"["+string(rune(i))+"]", actualSlice[i], v) {
					result = false
				}
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
				result = false
			}
		} else {
			t.Errorf("%s: %v != %v (type mismatch)", name, actual, expected)
			result = false
		}
	default:
		if actual != expected {
			t.Errorf("%s: %v (%T)!= %v (%T)", name, actual, actual, expected, expected)
			result = false
		}
	}
	return result
}
