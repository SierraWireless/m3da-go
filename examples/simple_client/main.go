package main

import (
	"context"
	"fmt"
	"log"
	"os"

	m3da "github.com/SierraWireless/m3da-go"
)

func main() {
	// Create client with Go name (matching Java test pattern)
	config := m3da.DefaultClientConfig("localhost", "golang-test-client")
	client := m3da.NewTCPClient(config)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected to M3DA server")

	// Send EXACTLY the same message as Java TestClient
	// Java: path="@sys.test", body={temperature: 25.5, timestamp: 1749801395, sensor: "test-sensor-001"}
	messageBody := map[string]interface{}{
		"temperature": 25.5,
		"timestamp":   int64(1749801395), // Same value as Java
		"sensor":      "test-sensor-001",
	}

	ticketID := uint32(123)
	message := &m3da.M3daMessage{
		Path:     "@sys.test", // Same path as Java
		TicketID: &ticketID,   // Same ID as Java, but correct field name
		Body:     messageBody,
	}

	fmt.Println("Sending single test message (matching Java TestClient)...")
	responses, err := client.SendEnvelope(ctx, message)
	if err != nil {
		log.Printf("❌ Failed: %v", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Success! Received %d responses\n", len(responses))
}
