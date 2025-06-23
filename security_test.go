package m3da

import (
	"bytes"
	"testing"
)

func TestSecurityManager_KeyDerivation(t *testing.T) {
	tests := []struct {
		name       string
		config     *SecurityConfig
		expectKeys bool
	}{
		{
			name: "AES-CTR-256 + HMAC-SHA1",
			config: &SecurityConfig{
				Authentication: HMACTypeSHA1,
				Encryption:     CipherAESCTR256,
				Password:       "test-password-123",
			},
			expectKeys: true,
		},
		{
			name: "AES-CBC-128 + HMAC-MD5",
			config: &SecurityConfig{
				Authentication: HMACTypeMD5,
				Encryption:     CipherAESCBC128,
				Password:       "test-password-123",
			},
			expectKeys: true,
		},
		{
			name: "HMAC-SHA1 Only",
			config: &SecurityConfig{
				Authentication: HMACTypeSHA1,
				Encryption:     CipherNone,
				Password:       "test-password-123",
			},
			expectKeys: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, err := newSecurityManager(tt.config)
			if err != nil {
				t.Fatalf("NewSecurityManager() error = %v", err)
			}

			if tt.expectKeys {
				if tt.config.Encryption != CipherNone {
					// Generate a test nonce to enable key derivation
					nonce, err := sm.generateNonce()
					if err != nil {
						t.Fatalf("GenerateNonce() error = %v", err)
					}

					if len(nonce) == 0 {
						t.Error("Generated nonce is empty")
					}

					t.Logf("Generated nonce: %x", nonce)
				}
			}
		})
	}
}

func TestSecurityManager_EncryptDecrypt(t *testing.T) {
	testPayload := []byte("This is a test payload for encryption/decryption testing with M3DA protocol")

	tests := []struct {
		name   string
		config *SecurityConfig
	}{
		{
			name: "AES-CTR-256",
			config: &SecurityConfig{
				Authentication: HMACTypeSHA1,
				Encryption:     CipherAESCTR256,
				Password:       "test-password-123",
			},
		},
		{
			name: "AES-CTR-128",
			config: &SecurityConfig{
				Authentication: HMACTypeSHA1,
				Encryption:     CipherAESCTR128,
				Password:       "test-password-123",
			},
		},
		{
			name: "AES-CBC-256",
			config: &SecurityConfig{
				Authentication: HMACTypeSHA1,
				Encryption:     CipherAESCBC256,
				Password:       "test-password-123",
			},
		},
		{
			name: "AES-CBC-128",
			config: &SecurityConfig{
				Authentication: HMACTypeSHA1,
				Encryption:     CipherAESCBC128,
				Password:       "test-password-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, err := newSecurityManager(tt.config)
			if err != nil {
				t.Fatalf("NewSecurityManager() error = %v", err)
			}

			// Generate a nonce for encryption
			nonce, err := sm.generateNonce()
			if err != nil {
				t.Fatalf("GenerateNonce() error = %v", err)
			}

			// Test encryption
			encrypted, err := sm.encryptPayloadWithNonce(testPayload, nonce, "test-client")
			if err != nil {
				t.Fatalf("EncryptPayloadWithNonce() error = %v", err)
			}

			// Encrypted data should be different from original
			if bytes.Equal(encrypted, testPayload) {
				t.Error("Encrypted payload is identical to original")
			}

			// Test decryption
			decrypted, err := sm.decryptPayloadWithNonce(encrypted, nonce, "test-client")
			if err != nil {
				t.Fatalf("DecryptPayloadWithNonce() error = %v", err)
			}

			// Decrypted data should match original
			if !bytes.Equal(decrypted, testPayload) {
				t.Errorf("Decrypted payload doesn't match original.\nOriginal:  %s\nDecrypted: %s", testPayload, decrypted)
			}
		})
	}
}

func TestSecurityManager_HMAC(t *testing.T) {
	tests := []struct {
		name   string
		config *SecurityConfig
	}{
		{
			name: "HMAC-SHA1",
			config: &SecurityConfig{
				Authentication: HMACTypeSHA1,
				Encryption:     CipherNone,
				Password:       "test-password-123",
			},
		},
		{
			name: "HMAC-MD5",
			config: &SecurityConfig{
				Authentication: HMACTypeMD5,
				Encryption:     CipherNone,
				Password:       "test-password-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, err := newSecurityManager(tt.config)
			if err != nil {
				t.Fatalf("NewSecurityManager() error = %v", err)
			}

			// Create test envelope
			envelope := &M3daEnvelope{
				Header: map[string]interface{}{
					HeaderKeyID:    "test-client",
					HeaderKeyNonce: int64(12345),
				},
				Payload: []byte("test payload for HMAC verification"),
				Footer:  map[string]interface{}{},
			}

			// Calculate HMAC (requires salt parameter)
			salt := []byte("test-salt")
			mac, err := sm.calculateHMAC(envelope, salt)
			if err != nil {
				t.Fatalf("CalculateHMAC() error = %v", err)
			}

			if len(mac) == 0 {
				t.Error("HMAC is empty")
			}

			// Calculate HMAC again for verification (no VerifyHMAC method available)
			mac2, err := sm.calculateHMAC(envelope, salt)
			if err != nil {
				t.Fatalf("CalculateHMAC() verification error = %v", err)
			}

			if !bytes.Equal(mac, mac2) {
				t.Error("HMAC calculation is not consistent")
			}

			// Test with tampered envelope
			originalPayload := envelope.Payload
			envelope.Payload = []byte("tampered payload")
			macTampered, err := sm.calculateHMAC(envelope, salt)
			if err != nil {
				t.Fatalf("CalculateHMAC() tampered error = %v", err)
			}

			if bytes.Equal(mac, macTampered) {
				t.Error("HMAC should be different for tampered envelope")
			}

			// Restore original payload
			envelope.Payload = originalPayload
		})
	}
}

func TestSecurityManager_FullEnvelopeSecurity(t *testing.T) {
	config := &SecurityConfig{
		Authentication: HMACTypeSHA1,
		Encryption:     CipherAESCTR256,
		Password:       "test-password-123",
	}

	sm, err := newSecurityManager(config)
	if err != nil {
		t.Fatalf("NewSecurityManager() error = %v", err)
	}

	// Create test envelope
	originalPayload := []byte("This is a test payload for full envelope security testing")
	envelope := &M3daEnvelope{
		Header: map[string]interface{}{
			HeaderKeyID: "test-client",
		},
		Payload: originalPayload,
		Footer:  map[string]interface{}{},
	}

	// Apply security
	securedEnvelope, err := sm.applySecurityToEnvelope(envelope)
	if err != nil {
		t.Fatalf("ApplySecurityToEnvelope() error = %v", err)
	}

	// Use the secured envelope for further tests
	envelope = securedEnvelope

	// Check that security was applied
	if _, ok := envelope.Header[HeaderKeyNonce]; !ok {
		t.Error("Nonce not added to envelope header")
	}

	if _, ok := envelope.Footer[HeaderKeyMAC]; !ok {
		t.Error("HMAC not added to envelope footer")
	}

	// Payload should be encrypted (different from original)
	if bytes.Equal(envelope.Payload, originalPayload) {
		t.Error("Payload was not encrypted")
	}

	// Verify security
	err = sm.verifyEnvelopeSecurity(envelope)
	if err != nil {
		t.Fatalf("VerifyEnvelopeSecurity() error = %v", err)
	}

	// After verification, payload should be decrypted back to original
	if !bytes.Equal(envelope.Payload, originalPayload) {
		t.Error("Payload was not properly decrypted after verification")
	}
}

func TestSecurityManager_NonceGeneration(t *testing.T) {
	config := &SecurityConfig{
		Authentication: HMACTypeSHA1,
		Encryption:     CipherNone,
		Password:       "test-password-123",
	}

	sm, err := newSecurityManager(config)
	if err != nil {
		t.Fatalf("NewSecurityManager() error = %v", err)
	}

	// Generate multiple nonces
	nonces := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		nonce, err := sm.generateNonce()
		if err != nil {
			t.Fatalf("GenerateNonce() error = %v", err)
		}

		if len(nonce) != 16 {
			t.Errorf("Expected 16-byte nonce, got %d bytes", len(nonce))
		}

		nonces[i] = nonce
	}

	// Check that nonces are unique
	for i := 0; i < len(nonces); i++ {
		for j := i + 1; j < len(nonces); j++ {
			if bytes.Equal(nonces[i], nonces[j]) {
				t.Errorf("Nonces %d and %d are identical: %x", i, j, nonces[i])
			}
		}
	}
}

func TestSecurityManager_ErrorCases(t *testing.T) {
	t.Run("No password", func(t *testing.T) {
		config := &SecurityConfig{
			Authentication: HMACTypeSHA1,
			Encryption:     CipherNone,
			Password:       "", // Empty password
		}

		_, err := newSecurityManager(config)
		if err == nil {
			t.Error("Expected error for empty password")
		}
	})

	t.Run("Unsupported cipher", func(t *testing.T) {
		config := &SecurityConfig{
			Authentication: HMACTypeSHA1,
			Encryption:     "UNSUPPORTED_CIPHER",
			Password:       "test-password",
		}

		_, err := newSecurityManager(config)
		if err == nil {
			t.Error("Expected error for unsupported cipher")
		}
	})

	t.Run("Nil config", func(t *testing.T) {
		_, err := newSecurityManager(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})
}
