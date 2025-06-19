package main

import (
	"context"
	"fmt"
	"log"
	"time"

	m3da "github.com/SierraWireless/m3da-go"
)

func main() {
	fmt.Println("🔐 M3DA Secure Client Example")
	fmt.Println("=============================")

	// Test different security configurations
	securityConfigs := []struct {
		name   string
		commId string
		config *m3da.SecurityConfig
	}{
		{
			name: "HMAC-SHA1 Authentication Only",
			//commId: "SECURE-SHA1-CLIENT",
			commId: "secure-sha1-client",
			config: &m3da.SecurityConfig{
				Authentication: m3da.HMACTypeSHA1,
				Encryption:     m3da.CipherNone,
				Password:       "my-secret-key-123",
				ServerID:       "AIRVANTAGE",
			},
		},
		{
			name:   "AES-CTR-256 + HMAC-SHA1",
			commId: "secure-aes-sha1-client",
			config: &m3da.SecurityConfig{
				Authentication: m3da.HMACTypeSHA1,
				Encryption:     m3da.CipherAESCTR256,
				Password:       "my-secret-key-123",
				ServerID:       "AIRVANTAGE",
			},
		},
		{
			name:   "AES-CBC-128 + HMAC-MD5",
			commId: "secure-aes-md5-client",
			config: &m3da.SecurityConfig{
				Authentication: m3da.HMACTypeMD5,
				Encryption:     m3da.CipherAESCBC128,
				Password:       "my-secret-key-123",
				ServerID:       "AIRVANTAGE",
			},
		},
	}

	ctx := context.Background()
	var success bool = true

	for i, secConfig := range securityConfigs {
		fmt.Printf("\n--- Test %d: %s ---\n", i+1, secConfig.name)

		// Create client configuration with security
		//config := m3da.DefaultClientConfig("qa.airvantage.io", secConfig.commId)
		config := m3da.DefaultClientConfig("localhost", secConfig.commId)
		config.SecurityConfig = secConfig.config

		// Create and test client
		if err := testSecureClient(ctx, config, secConfig.name); err != nil {
			log.Printf("❌ %s failed: %v", secConfig.name, err)
			success = false
		} else {
			fmt.Printf("✅ %s completed successfully\n", secConfig.name)
		}
	}

	if success {
		fmt.Println("\n🎉 All security tests completed!")
	} else {
		fmt.Println("\n❌ At least one security tests failed!")
	}
}

func testSecureClient(ctx context.Context, config *m3da.ClientConfig, testName string) error {
	// Create secure client
	client := m3da.NewTCPClient(config)

	// Connect to server
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	fmt.Printf("  🔗 Connected with %s\n", testName)

	// Test 1: Send simple encrypted telemetry data
	if err := sendSecureTelemetry(ctx, client); err != nil {
		return fmt.Errorf("secure telemetry failed: %w", err)
	}

	// Test 2: Send encrypted compressed data
	if err := sendSecureCompressedData(ctx, client); err != nil {
		return fmt.Errorf("secure compressed data failed: %w", err)
	}

	// Test 3: Send encrypted command with response
	if err := sendSecureCommand(ctx, client); err != nil {
		return fmt.Errorf("secure command failed: %w", err)
	}

	return nil
}

func sendSecureTelemetry(ctx context.Context, client m3da.Client) error {
	fmt.Println("    📊 Sending encrypted telemetry data...")

	data := map[string]interface{}{
		"temperature":    23.5,
		"humidity":       65.2,
		"pressure":       1013.25,
		"timestamp":      time.Now().Unix(),
		"location":       "secure-sensor-room",
		"security_level": "encrypted",
	}

	err := client.SendData(ctx, "@sys.secure.telemetry", data)
	if err != nil {
		return fmt.Errorf("failed to send secure telemetry: %w", err)
	}

	fmt.Println("    ✅ Encrypted telemetry sent successfully")
	return nil
}

func sendSecureCompressedData(ctx context.Context, client m3da.Client) error {
	fmt.Println("    📈 Sending encrypted compressed data...")

	// Create encrypted temperature deltas
	temperaturesDeltas := m3da.NewDeltasVectorFloat64(0.1, 235.0, []float64{1.0, -2.0, 3.0, 0.0, -1.0, 2.0, 1.5, -0.5})

	// Create encrypted quasi-periodic timestamps
	timestampsQP := m3da.NewQuasiPeriodicVectorInt64(int64(60), time.Now().Unix(), []int64{5, -2, 3, 1, 2, -1, 4, 0})

	message := &m3da.M3daMessage{
		Path: "@sys.secure.compressed",
		Body: map[string]interface{}{
			"sensor_id":   "secure-temp-001",
			"unit":        "celsius",
			"temperature": temperaturesDeltas,
			"timestamps":  timestampsQP,
			"encrypted":   true,
			"compression": "m3da_deltas_vector",
		},
	}

	responses, err := client.SendEnvelope(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send secure compressed data: %w", err)
	}

	fmt.Printf("    ✅ Encrypted compressed data sent, received %d responses\n", len(responses))
	return nil
}

func sendSecureCommand(ctx context.Context, client m3da.Client) error {
	fmt.Println("    🎯 Sending encrypted command...")

	requestData := map[string]interface{}{
		"command":        "get_secure_status",
		"target":         "secure-device-001",
		"security_token": "encrypted-token-12345",
		"timestamp":      time.Now().Unix(),
	}

	response, err := client.SendMessage(ctx, "@sys.secure.commands", requestData)
	if err != nil {
		// Note: Demo server might not support secure commands, so we log but don't fail
		fmt.Printf("    ⚠️  Secure command sent but no response (demo server behavior): %v\n", err)
		return nil
	}

	fmt.Printf("    ✅ Encrypted command sent, response status: %d\n", response.Status)
	return nil
}

// Additional utility functions for security demonstration

func demonstrateSecurityFeatures() {
	fmt.Println("\n🔐 M3DA Security Features:")
	fmt.Println("  • HMAC Authentication (MD5/SHA1)")
	fmt.Println("  • AES Encryption (CBC/CTR modes)")
	fmt.Println("  • Key sizes: 128-bit and 256-bit")
	fmt.Println("  • PBKDF2 key derivation")
	fmt.Println("  • Cryptographic nonce generation")
	fmt.Println("  • Envelope integrity protection")
	fmt.Println("  • Payload encryption")
}

func printSecurityInfo(config *m3da.SecurityConfig) {
	fmt.Printf("  🔒 Authentication: %s\n", config.Authentication)
	fmt.Printf("  🔐 Encryption: %s\n", config.Encryption)
	fmt.Printf("  🔑 Password: %s\n", maskPassword(config.Password))
}

func maskPassword(password string) string {
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}
