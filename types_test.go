package m3da

import (
	"math"
	"testing"
)

func TestStatusCodeString(t *testing.T) {
	tests := []struct {
		code     StatusCode
		expected string
	}{
		{StatusOK, "OK"},
		{StatusBadRequest, "BAD_REQUEST"},
		{StatusUnauthorized, "UNAUTHORIZED"},
		{StatusForbidden, "FORBIDDEN"},
		{StatusAuthenticationRequired, "AUTHENTICATION_REQUIRED"},
		{StatusEncryptionNeeded, "ENCRYPTION_NEEDED"},
		{StatusShortcutMapError, "SHORTCUT_MAP_ERROR"},
		{StatusUnexpectedError, "UNEXPECTED_ERROR"},
		{StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{StatusCode(999), "UNKNOWN(999)"},
	}

	for _, test := range tests {
		if got := test.code.String(); got != test.expected {
			t.Errorf("StatusCode(%d).String() = %s, want %s", int(test.code), got, test.expected)
		}
	}
}

func TestM3DAError(t *testing.T) {
	err := &M3DAError{
		StatusCode: StatusBadRequest,
		Message:    "Invalid payload",
	}

	expected := "M3DA error 400 (BAD_REQUEST): Invalid payload"
	if got := err.Error(); got != expected {
		t.Errorf("M3DAError.Error() = %s, want %s", got, expected)
	}

	// Test without message
	err2 := &M3DAError{
		StatusCode: StatusOK,
	}

	expected2 := "M3DA error 200 (OK)"
	if got := err2.Error(); got != expected2 {
		t.Errorf("M3DAError.Error() = %s, want %s", got, expected2)
	}
}

func TestM3daMessage(t *testing.T) {
	ticketID := uint32(12345)
	message := &M3daMessage{
		Path:     "@sys.test",
		TicketID: &ticketID,
		Body: map[string]interface{}{
			"key": "value",
		},
	}

	if message.GetOpCode() != OpCodeMessage {
		t.Errorf("M3daMessage.GetOpCode() = %d, want %d", message.GetOpCode(), OpCodeMessage)
	}

	if message.Path != "@sys.test" {
		t.Errorf("M3daMessage.Path = %s, want @sys.test", message.Path)
	}

	if message.TicketID == nil || *message.TicketID != 12345 {
		t.Errorf("M3daMessage.TicketID = %v, want 12345", message.TicketID)
	}
}

func TestM3daResponse(t *testing.T) {
	response := &M3daResponse{
		TicketID: 12345,
		Status:   200,
		Message:  "OK",
	}

	if response.GetOpCode() != OpCodeResponse {
		t.Errorf("M3daResponse.GetOpCode() = %d, want %d", response.GetOpCode(), OpCodeResponse)
	}

	if response.TicketID != 12345 {
		t.Errorf("M3daResponse.TicketID = %d, want 12345", response.TicketID)
	}

	if response.Status != 200 {
		t.Errorf("M3daResponse.Status = %d, want 200", response.Status)
	}
}

func TestM3daDeltasVector(t *testing.T) {
	// Test with float64 values (should return float64 results)
	dv := NewDeltasVectorFloat64(0.1, 235.0, []float64{1.0, -2.0, 3.0, 0.0})

	result := dv.AsFlatList()
	expected := []float64{23.5, 23.6, 23.4, 23.7, 23.7}

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i, val := range result {
		if i < len(expected) {
			if math.Abs(val-expected[i]) > 1e-10 {
				t.Errorf("At index %d: expected %f, got %f", i, expected[i], val)
			}
		}
	}

	// Test with int64 values (should return int64 results)
	dvInt := NewDeltasVectorInt64(2, 100, []int64{5, -3, 2, 0})

	resultInt := dvInt.AsFlatList()
	expectedInt := []int64{200, 210, 204, 208, 208}

	if len(resultInt) != len(expectedInt) {
		t.Errorf("Expected length %d, got %d", len(expectedInt), len(resultInt))
	}

	for i, val := range resultInt {
		if i < len(expectedInt) {
			if val != expectedInt[i] {
				t.Errorf("At index %d: expected %d, got %d", i, expectedInt[i], val)
			}
		}
	}

	// Test mixed types (should return float64 results due to float factor)
	dvMixed := NewDeltasVectorFloat64(0.5, 10.0, []float64{2.0, -1.0, 1.5})

	resultMixed := dvMixed.AsFlatList()
	expectedMixed := []float64{5.0, 6.0, 5.5, 6.25}

	if len(resultMixed) != len(expectedMixed) {
		t.Errorf("Expected length %d, got %d", len(expectedMixed), len(resultMixed))
	}

	for i, val := range resultMixed {
		if i < len(expectedMixed) {
			if math.Abs(val-expectedMixed[i]) > 1e-10 {
				t.Errorf("At index %d: expected %f, got %f", i, expectedMixed[i], val)
			}
		}
	}
}

func TestM3daQuasiPeriodicVector(t *testing.T) {
	qpv := NewQuasiPeriodicVectorFloat64(10.0, 100.0, []float64{2, 1, 1}) // 2 repeats, shift by 1, then 1 final repeat

	if qpv.GetOpCode() != OpCodeQuasiPeriodicVector {
		t.Errorf("M3daQuasiPeriodicVector.GetOpCode() = %d, want %d", qpv.GetOpCode(), OpCodeQuasiPeriodicVector)
	}

	flatList := qpv.AsFlatList()
	// Start: 100
	// 2 repeats: 110, 120
	// Shifted value: 130 + 1 = 131
	// 1 final repeat: 141
	expected := []float64{100, 110, 120, 131, 141}

	if len(flatList) != len(expected) {
		t.Errorf("AsFlatList() length = %d, want %d", len(flatList), len(expected))
	}

	for i, v := range expected {
		if i < len(flatList) && flatList[i] != v {
			t.Errorf("AsFlatList()[%d] = %f, want %f", i, flatList[i], v)
		}
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig("test-host", "test-client")

	if config.Host != "test-host" {
		t.Errorf("DefaultClientConfig.Host = %s, want test-host", config.Host)
	}

	if config.ClientID != "test-client" {
		t.Errorf("DefaultClientConfig.ClientID = %s, want test-client", config.ClientID)
	}

	if config.Port != DefaultPort {
		t.Errorf("DefaultClientConfig.Port = %d, want %d", config.Port, DefaultPort)
	}

	if config.ConnectTimeout == 0 {
		t.Error("DefaultClientConfig.ConnectTimeout should not be zero")
	}

	if config.ReadTimeout == 0 {
		t.Error("DefaultClientConfig.ReadTimeout should not be zero")
	}

	if config.WriteTimeout == 0 {
		t.Error("DefaultClientConfig.WriteTimeout should not be zero")
	}
}
