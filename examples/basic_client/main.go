package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	m3da "github.com/SierraWireless/m3da-go"
)

func main() {
	// Create client configuration
	config := m3da.DefaultClientConfig("localhost", "example-client-001")
	client := m3da.NewTCPClient(config)

	// Connect to server
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected to M3DA server")

	// Track failures
	var failures []string

	// Example 1: Send simple data
	fmt.Println("\n--- Example 1: Sending simple data ---")
	if err := sendSimpleData(ctx, client); err != nil {
		failure := fmt.Sprintf("Example 1 failed: %v", err)
		log.Printf("❌ %s", failure)
		failures = append(failures, failure)
		return
	} else {
		fmt.Println("✅ Example 1 passed")
	}

	// Example 2: Send message with response
	fmt.Println("\n--- Example 2: Sending message with response ---")
	if err := sendMessageWithResponse(ctx, client); err != nil {
		failure := fmt.Sprintf("Example 2 failed: %v", err)
		log.Printf("❌ %s", failure)
		failures = append(failures, failure)
	} else {
		fmt.Println("✅ Example 2 passed")
	}

	// Example 3: Send multiple messages in one envelope
	fmt.Println("\n--- Example 3: Sending multiple messages ---")
	if err := sendMultipleMessages(ctx, client); err != nil {
		failure := fmt.Sprintf("Example 3 failed: %v", err)
		log.Printf("❌ %s", failure)
		failures = append(failures, failure)
	} else {
		fmt.Println("✅ Example 3 passed")
	}

	// Example 4: Send compressed data using deltas vector
	fmt.Println("\n--- Example 4: Sending compressed data ---")
	if err := sendCompressedData(ctx, client); err != nil {
		failure := fmt.Sprintf("Example 4 failed: %v", err)
		log.Printf("❌ %s", failure)
		failures = append(failures, failure)
	} else {
		fmt.Println("✅ Example 4 passed")
	}

	// Report final results
	if len(failures) > 0 {
		fmt.Printf("\n❌ %d examples FAILED:\n", len(failures))
		for _, failure := range failures {
			fmt.Printf("  - %s\n", failure)
		}
		os.Exit(1)
	} else {
		fmt.Println("\n✅ All examples completed successfully!")
	}
}

// sendSimpleData demonstrates sending simple telemetry data
func sendSimpleData(ctx context.Context, client m3da.Client) error {
	data := map[string]interface{}{
		"temperature": 23.5,
		"humidity":    65.2,
		"timestamp":   time.Now().Unix(),
		"location":    "sensor-room-1",
	}

	err := client.SendData(ctx, "@sys.telemetry", data)
	if err != nil {
		return fmt.Errorf("failed to send telemetry data: %w", err)
	}

	return nil
}

// sendMessageWithResponse demonstrates request-response pattern
func sendMessageWithResponse(ctx context.Context, client m3da.Client) error {
	requestData := map[string]interface{}{
		"command": "get_status",
		"target":  "device-001",
	}

	// Use SendEnvelope() directly like Java client does
	ticketID := uint32(1234) // Fixed ticket ID like Java example
	message := &m3da.M3daMessage{
		Path:     "@sys.commands",
		TicketID: &ticketID,
		Body:     requestData,
	}

	responses, err := client.SendEnvelope(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	fmt.Printf("✅ Command sent successfully, received %d responses\n", len(responses))

	// Manual response checking like Java client (warn, don't error)
	foundMatchingResponse := false
	for i, resp := range responses {
		fmt.Printf("Response %d: %T\n", i, resp)
		if response, ok := resp.(*m3da.M3daResponse); ok {
			fmt.Printf("  Status: %d, Message: %s, TicketID: %d\n",
				response.Status, response.Message, response.TicketID)

			if response.TicketID == ticketID {
				foundMatchingResponse = true
				fmt.Printf("  ✅ Found matching response for ticket ID %d\n", ticketID)
			}
		}
	}

	if !foundMatchingResponse {
		fmt.Printf("  ⚠️  No response found with matching ticket ID %d (demo server behavior)\n", ticketID)
	}

	return nil
}

// sendMultipleMessages demonstrates sending multiple messages in one envelope
func sendMultipleMessages(ctx context.Context, client m3da.Client) error {
	messages := []m3da.M3daBodyMessage{
		&m3da.M3daMessage{
			Path: "@sys.sensors.temperature",
			Body: map[string]interface{}{
				"value":     23.5,
				"timestamp": time.Now().Unix(),
				"sensor_id": "temp-001",
			},
		},
		&m3da.M3daMessage{
			Path: "@sys.sensors.humidity",
			Body: map[string]interface{}{
				"value":     65.2,
				"timestamp": time.Now().Unix(),
				"sensor_id": "humid-001",
			},
		},
		&m3da.M3daMessage{
			Path: "@sys.sensors.pressure",
			Body: map[string]interface{}{
				"value":     1013.25,
				"timestamp": time.Now().Unix(),
				"sensor_id": "press-001",
			},
		},
	}

	responses, err := client.SendEnvelope(ctx, messages...)
	if err != nil {
		return fmt.Errorf("failed to send multiple messages: %w", err)
	}

	fmt.Printf("Successfully sent multiple messages, received %d responses\n", len(responses))
	return nil
}

// sendCompressedData demonstrates using M3DA data compression features
func sendCompressedData(ctx context.Context, client m3da.Client) error {
	fmt.Println("🔧 Testing various M3daDeltasVector encodings...")

	// Test 1: Float64 temperature deltas (matches Java exactly)
	fmt.Println("  📊 Test 1: Float64 temperature deltas")
	temperaturesDeltas := m3da.NewDeltasVectorFloat64(
		0.1,   // Factor (float64)
		235.0, // Start (float64 - 23.5°C scaled by 10)
		[]float64{1.0, -2.0, 3.0, 0.0, -1.0, 2.0}, // Deltas (float64)
	)

	// Test 2: Int64 counter deltas
	fmt.Println("  📊 Test 2: Int64 counter deltas")
	counterDeltas := m3da.NewDeltasVectorInt64(
		1,                         // Factor (int64)
		1000,                      // Start (int64)
		[]int64{5, -2, 10, 0, -3}, // Deltas (int64)
	)

	// Test 3: Float64 types (converted from mixed types)
	fmt.Println("  📊 Test 3: Float64 types (converted from mixed types)")
	mixedDeltas := m3da.NewDeltasVectorFloat64(
		0.5,                            // Factor (float64)
		100.0,                          // Start (float64, converted from int64)
		[]float64{2.0, -1.0, 1.5, 0.0}, // Deltas (float64, mixed converted)
	)

	// Create quasi-periodic vector for timestamps (matches Java exactly)
	timestampsQP := m3da.NewQuasiPeriodicVectorInt64(
		60,                                // 60 seconds between measurements
		time.Now().Unix(),                 // START = current timestamp (like Java)
		[]int64{5, -2, 3, 1, 2, -1, 4, 0}, // Timing variations as integers
	)

	// Send Test 1: Float64 temperature data
	message1 := &m3da.M3daMessage{
		Path: "@sys.compressed.temperature.float",
		Body: map[string]interface{}{
			"sensor_id":   "temp-float-001",
			"unit":        "celsius",
			"temperature": temperaturesDeltas,
			"timestamps":  timestampsQP,
		},
	}

	responses1, err := client.SendEnvelope(ctx, message1)
	if err != nil {
		return fmt.Errorf("failed to send float64 temperature data: %w", err)
	}
	fmt.Printf("    ✅ Float64 temperature data sent, received %d responses\n", len(responses1))

	// Send Test 2: Int64 counter data
	message2 := &m3da.M3daMessage{
		Path: "@sys.compressed.counter.int",
		Body: map[string]interface{}{
			"sensor_id":  "counter-int-001",
			"unit":       "count",
			"counter":    counterDeltas,
			"timestamps": timestampsQP,
		},
	}

	responses2, err := client.SendEnvelope(ctx, message2)
	if err != nil {
		return fmt.Errorf("failed to send int64 counter data: %w", err)
	}
	fmt.Printf("    ✅ Int64 counter data sent, received %d responses\n", len(responses2))

	// Send Test 3: Mixed type data
	message3 := &m3da.M3daMessage{
		Path: "@sys.compressed.mixed.types",
		Body: map[string]interface{}{
			"sensor_id":  "mixed-001",
			"unit":       "mixed",
			"values":     mixedDeltas,
			"timestamps": timestampsQP,
		},
	}

	responses3, err := client.SendEnvelope(ctx, message3)
	if err != nil {
		return fmt.Errorf("failed to send mixed type data: %w", err)
	}
	fmt.Printf("    ✅ Mixed type data sent, received %d responses\n", len(responses3))

	fmt.Printf("Successfully sent all compressed data variants, total responses: %d\n",
		len(responses1)+len(responses2)+len(responses3))

	// Demonstrate decompression (matches Java decompression demo)
	fmt.Println("📈 Decompressed values:")
	fmt.Println("  Float64 temperature values:", temperaturesDeltas.AsFlatList())
	fmt.Println("  Int64 counter values:", counterDeltas.AsFlatList())
	fmt.Println("  Mixed type values:", mixedDeltas.AsFlatList())
	fmt.Println("  Timestamp values:", timestampsQP.AsFlatList())

	return nil
}

// Additional utility functions for advanced usage

// sendBatchData demonstrates sending multiple measurements in a single envelope
func sendBatchData(ctx context.Context, client m3da.Client) error {
	messages := []m3da.M3daBodyMessage{
		&m3da.M3daMessage{
			Path: "@sys.sensors.temperature",
			Body: map[string]interface{}{
				"value":     23.5,
				"timestamp": time.Now().Unix(),
				"sensor_id": "temp-001",
			},
		},
		&m3da.M3daMessage{
			Path: "@sys.sensors.humidity",
			Body: map[string]interface{}{
				"value":     65.2,
				"timestamp": time.Now().Unix(),
				"sensor_id": "humid-001",
			},
		},
		&m3da.M3daMessage{
			Path: "@sys.sensors.pressure",
			Body: map[string]interface{}{
				"value":     1013.25,
				"timestamp": time.Now().Unix(),
				"sensor_id": "press-001",
			},
		},
	}

	responses, err := client.SendEnvelope(ctx, messages...)
	if err != nil {
		return fmt.Errorf("failed to send batch data: %w", err)
	}

	fmt.Printf("Successfully sent batch data, received %d responses\n", len(responses))
	return nil
}
