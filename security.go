package m3da

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"hash"
)

// securityManager handles M3DA security operations according to M3DA Security Extension
type securityManager struct {
	config    *SecurityConfig
	nonce     []byte // Current nonce from the last message in the stream (anti-replay)
	nextNonce []byte // nonce for next response (anti-replay, should not be needed but demo server seem to not rotate the nonce correctly)
}

// newSecurityManager creates a new security manager
func newSecurityManager(config *SecurityConfig) (*securityManager, error) {
	if config == nil {
		return nil, fmt.Errorf("security config is required")
	}

	sm := &securityManager{
		config: config,
	}
	debugf("Fallback server ID set to %s", sm.config.ServerID)

	if sm.config.Password == "" {
		return nil, fmt.Errorf("password is required for security")
	}

	switch sm.config.Encryption {
	case CipherAESCBC128, CipherAESCTR128:
	case CipherAESCBC256, CipherAESCTR256:
	case CipherNone:
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", sm.config.Encryption)
	}

	return sm, nil
}

// generateNonce generates a cryptographic nonce with correct length for hash function
func (sm *securityManager) generateNonce() ([]byte, error) {
	// M3DA Security Extension: nonce length depends on hash function
	var nonceLength int = 16

	/* Test server and AirVantage implementation doesn't follow spec
	switch sm.config.Authentication {
	case HMACTypeMD5:
		nonceLength = md5.Size // MD5 output length
	case HMACTypeSHA1:
		nonceLength = sha1.New().Size() // SHA1 output length
	default:
		nonceLength = sha1.New().Size() // Default fallback
	}
	*/

	nonce := make([]byte, nonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	debugf("Generated %d-byte nonce for %s: %x", nonceLength, sm.config.Authentication, nonce)
	return nonce, nil
}

// ExtractAndUpdateNonces extracts nonce from envelope and updates current nonce for anti-replay
func (sm *securityManager) extractAndUpdateNonces(envelope *M3daEnvelope) error {
	// Extract the nonce from the received envelope for anti-replay mechanism
	if nonceValue, exists := envelope.Header[HeaderKeyNonce]; exists {
		var receivedNonce []byte

		if nonceBytes, ok := nonceValue.([]byte); ok {
			receivedNonce = nonceBytes
		} else if nonceStr, ok := nonceValue.(string); ok {
			receivedNonce = []byte(nonceStr)
		} else {
			return fmt.Errorf("invalid nonce type in envelope: %T", nonceValue)
		}

		// Update current nonce for anti-replay: the nonce from this message
		// will be used to authenticate our next outgoing message
		sm.nonce = receivedNonce

		debugf("Extracted next nonce from envelope: %x", sm.nonce)
		return nil
	}

	return fmt.Errorf("no nonce found in envelope header")
}

// deriveCipherKey derives cipher key from base key and current nonce
// M3DA Security Extension: CK = <HMAC_xxx(K, nonce)>_ckl
func (sm *securityManager) deriveCipherKey(keyLength int, clientID string) ([]byte, error) {
	if sm.config.Encryption == CipherNone {
		return nil, nil
	}

	if len(sm.nonce) == 0 {
		return nil, fmt.Errorf("current nonce is required for cipher key derivation")
	}

	/* Theorical Key generation:
	// Derive Password key K = H_MD5(password)
	Kmd5 := md5.Sum([]byte(sm.config.Password))

	// Derive base cipher key Kc = H_MD5( clientID o K)
	credential := append([]byte(clientID), Kmd5[:]...)
	Kc := md5.Sum([]byte(credential))
	*/
	// Neither Test server not Airvantage do concat the clientID
	Kc := md5.Sum([]byte(sm.config.Password))

	h := hmac.New(md5.New, Kc[:])
	h.Write(sm.nonce)
	cipherKey := h.Sum(nil)

	// If we need more key material, concatenate additional HMAC
	if len(cipherKey) < keyLength {
		// CK = <HMAC_xxx(K, nonce) o HMAC_xxx(K, nonce o nonce)>_ckl
		h.Reset()
		h.Write(sm.nonce)
		h.Write(sm.nonce) // nonce o nonce
		additionalKey := h.Sum(nil)
		cipherKey = append(cipherKey, additionalKey...)
	}

	// Truncate to required length
	if len(cipherKey) > keyLength {
		cipherKey = cipherKey[:keyLength]
	}

	debugf("Derived cipher key (len=%d): %x", keyLength, cipherKey)
	return cipherKey, nil
}

// deriveIV derives IV from current nonce
// M3DA Security Extension: IV = H_MD5(nonce)
func (sm *securityManager) deriveIV() ([]byte, error) {
	if len(sm.nonce) == 0 {
		return nil, fmt.Errorf("current nonce is required for IV derivation")
	}

	// Always use MD5 for IV derivation as per spec
	hash := md5.Sum(sm.nonce)
	iv := hash[:aes.BlockSize] // Take first 16 bytes for AES block size

	debugf("Derived IV from nonce: %x", iv)
	return iv, nil
}

// EncryptPayloadWithNonce encrypts payload using M3DA Security Extension methods
func (sm *securityManager) encryptPayloadWithNonce(payload []byte, nonce []byte, clientID string) ([]byte, error) {
	if sm.config.Encryption == CipherNone {
		return payload, nil
	}

	// Set current nonce for key/IV derivation
	sm.nonce = nonce

	// Derive cipher key based on encryption algorithm
	var keyLength int
	switch sm.config.Encryption {
	case CipherAESCBC128, CipherAESCTR128:
		keyLength = 16 // 128-bit
	case CipherAESCBC256, CipherAESCTR256:
		keyLength = 32 // 256-bit
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", sm.config.Encryption)
	}

	cipherKey, err := sm.deriveCipherKey(keyLength, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive cipher key: %w", err)
	}

	iv, err := sm.deriveIV()
	if err != nil {
		return nil, fmt.Errorf("failed to derive IV: %w", err)
	}

	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	switch sm.config.Encryption {
	case CipherAESCBC128, CipherAESCBC256:
		return sm.encryptCBC(block, payload, iv)
	case CipherAESCTR128, CipherAESCTR256:
		return sm.encryptCTR(block, payload, iv)
	default:
		return nil, fmt.Errorf("unsupported encryption mode: %s", sm.config.Encryption)
	}
}

// encryptCBC encrypts using AES-CBC mode with PKCS5 padding
func (sm *securityManager) encryptCBC(block cipher.Block, payload []byte, iv []byte) ([]byte, error) {
	// Add PKCS5 padding
	padding := aes.BlockSize - len(payload)%aes.BlockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	payload = append(payload, padtext...)

	// Encrypt
	mode := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(payload))
	mode.CryptBlocks(encrypted, payload)

	return encrypted, nil
}

// encryptCTR encrypts using AES-CTR mode
func (sm *securityManager) encryptCTR(block cipher.Block, payload []byte, iv []byte) ([]byte, error) {
	stream := cipher.NewCTR(block, iv)
	encrypted := make([]byte, len(payload))
	stream.XORKeyStream(encrypted, payload)
	return encrypted, nil
}

// CalculateHMAC calculates HMAC according to M3DA Security Extension
func (sm *securityManager) calculateHMAC(envelope *M3daEnvelope, salt []byte) ([]byte, error) {
	if sm.config.Authentication == HMACTypeNone {
		return nil, nil
	}

	// Try to get ID from envelope header first
	var clientID string
	var ok bool
	if clientID, ok = envelope.Header[HeaderKeyID].(string); !ok {
		// If no ID in envelope and we have a fallback ID, use it
		if len(sm.config.ServerID) > 0 {
			clientID = sm.config.ServerID
		} else {
			return nil, fmt.Errorf("no entity ID found in envelope header and no fallback ID provided")
		}
	}

	// M3DA Security Extension: derive base key = HMD5(username | HMD5(password))
	md5Pwd := md5.Sum([]byte(sm.config.Password))

	credentials := append([]byte(clientID), md5Pwd[:]...)
	hashKey := md5.Sum(credentials)

	var h hash.Hash

	switch sm.config.Authentication {
	case HMACTypeMD5:
		h = hmac.New(md5.New, hashKey[:])
	case HMACTypeSHA1:
		h = hmac.New(sha1.New, hashKey[:])
	default:
		return nil, fmt.Errorf("unsupported HMAC type: %s", sm.config.Authentication)
	}

	debugf("Base key K = H_%s(): %x", sm.config.Authentication, hashKey)

	// Write payload (body)
	h.Write(envelope.Payload)

	// Write nonce from the envelope (the nonce we're sending in this message)
	h.Write(salt)

	return h.Sum(nil), nil
}

// ApplySecurityToEnvelope applies security to envelope according to M3DA Security Extension
// Creates nested envelopes: security envelope containing the original envelope as encrypted payload
// Returns a new protected envelope without modifying the original
func (sm *securityManager) applySecurityToEnvelope(envelope *M3daEnvelope) (*M3daEnvelope, error) {
	// Generate a fresh nonce for this message envelope
	var err error

	sm.nextNonce, err = sm.generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate envelope nonce: %w", err)
	}
	if sm.nonce == nil {
		// Use our envelope nonce for HMAC if no server nonce available
		sm.nonce = sm.nextNonce
		debugf("Bootstrap server Nonce recovery using random nonce: %x", sm.nonce)
	}

	// Step 1: Encode the original envelope (inner envelope) as payload for security envelope
	encoder := NewBysantEncoder()
	innerEnvelopeData, err := encoder.EncodeObject(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to encode inner envelope: %w", err)
	}

	debugf("Encoded inner envelope (%d bytes) for security nesting", len(innerEnvelopeData))

	// Step 2: Create security envelope (outer envelope) with inner envelope as payload
	securityEnvelope := &M3daEnvelope{
		Header: map[string]interface{}{
			HeaderKeyID:    envelope.Header[HeaderKeyID], // Copy client ID
			HeaderKeyNonce: sm.nextNonce,                 // Fresh nonce for security envelope
		},
		Payload: innerEnvelopeData, // Inner envelope becomes payload
		Footer:  make(map[string]interface{}),
	}

	// Step 3: Encrypt the inner envelope payload if required
	if sm.config.Encryption != CipherNone {
		// Get client ID from envelope
		clientID, ok := envelope.Header[HeaderKeyID].(string)
		if !ok {
			return nil, fmt.Errorf("no client ID found in envelope header for encryption")
		}

		encryptedPayload, err := sm.encryptPayloadWithNonce(innerEnvelopeData, sm.nonce, clientID)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt inner envelope payload: %w", err)
		}
		securityEnvelope.Payload = encryptedPayload
		debugf("Encrypted inner envelope payload (%d bytes)", len(encryptedPayload))
	}

	// Step 4: Calculate HMAC for the security envelope
	if sm.config.Authentication != HMACTypeNone {
		var mac []byte
		mac, err = sm.calculateHMAC(securityEnvelope, sm.nonce)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate HMAC: %w", err)
		}
		// Add HMAC to footer
		securityEnvelope.Footer[HeaderKeyMAC] = mac
		debugf("Message HMAC: %x", mac)
	}

	// Step 5: Return the new security envelope
	return securityEnvelope, nil
}

// PerformPasswordNegotiation performs password negotiation (simplified for static credentials)
func (sm *securityManager) performPasswordNegotiation(clientID string) (*M3daEnvelope, error) {
	// For static credentials, we don't need password negotiation
	// Just return a simple envelope to establish connection
	envelope := &M3daEnvelope{
		Header: map[string]interface{}{
			HeaderKeyID: clientID,
		},
		Payload: []byte{},
		Footer:  map[string]interface{}{},
	}
	return envelope, nil
}

// ProcessNegotiationResponse processes negotiation response (simplified)
func (sm *securityManager) processNegotiationResponse(envelope *M3daEnvelope) error {
	// For static credentials, just check if there's a challenge
	if _, ok := envelope.Header[HeaderKeyChallenge]; ok {
		return sm.extractAndUpdateNonces(envelope)
	}
	return nil
}

// VerifyEnvelopeSecurity verifies envelope security and extracts inner envelope from nested structure
func (sm *securityManager) verifyEnvelopeSecurity(envelope *M3daEnvelope) error {
	if sm.config.Authentication == HMACTypeNone && sm.config.Encryption == CipherNone {
		return nil // No security configured
	}

	// Get client ID from envelope (might not be present in responses)
	clientID := ""
	if id, ok := envelope.Header[HeaderKeyID].(string); ok {
		clientID = id
	}

	debugf("Verifying security for envelope with %d byte payload", len(envelope.Payload))

	// Verify HMAC if present
	if sm.config.Authentication != HMACTypeNone {
		if macValue, exists := envelope.Footer[HeaderKeyMAC]; exists {
			var receivedMAC []byte
			if macBytes, ok := macValue.([]byte); ok {
				receivedMAC = macBytes
			} else if macStr, ok := macValue.(string); ok {
				receivedMAC = []byte(macStr)
			} else {
				return fmt.Errorf("invalid MAC type in envelope footer: %T", macValue)
			}

			debugf("Nonce state: current=%x", sm.nonce)
			// Calculate expected HMAC
			expectedMAC, err := sm.calculateHMAC(envelope, sm.nonce)
			if err != nil {
				return fmt.Errorf("failed to calculate HMAC for verification: %w", err)
			}

			// Compare HMACs
			if !hmac.Equal(receivedMAC, expectedMAC) {
				return fmt.Errorf("response HMAC verification failed, computed=%x, received=%x", expectedMAC, receivedMAC)
			}

			debugf("Response HMAC verification successful: %x", receivedMAC)
		}
	}

	// Decrypt payload if encryption is configured
	var decryptedPayload []byte
	if sm.config.Encryption != CipherNone && len(envelope.Payload) > 0 {
		var err error
		decryptedPayload, err = sm.decryptPayloadWithNonce(envelope.Payload, sm.nonce, clientID)
		if err != nil {
			return fmt.Errorf("failed to decrypt payload: %w", err)
		}
		debugf("Payload decrypted successfully (%d bytes)", len(decryptedPayload))
	} else {
		decryptedPayload = envelope.Payload
	}

	// If we have a decrypted payload, try to decode it as an inner envelope
	if len(decryptedPayload) > 0 {
		// Try to decode as nested envelope
		decoder := NewBysantDecoder(bytes.NewReader(decryptedPayload))
		messages, err := decoder.Decode()
		if err != nil {
			// If decoding as envelope fails, this might be raw message data
			debugf("Payload is not a nested envelope, treating as raw data")
			envelope.Payload = decryptedPayload
		} else {
			// Look for envelope in decoded messages
			for _, msg := range messages {
				if innerEnvelope, ok := msg.(*M3daEnvelope); ok {
					infof("🔍 Successfully extracted inner envelope from nested structure")
					// Replace the outer envelope with the inner envelope contents
					*envelope = *innerEnvelope
					break
				}
			}
		}
	}

	if err := sm.extractAndUpdateNonces(envelope); err != nil {
		// Don't fail the operation, just log the warning
		warnf("Failed to update nonces from response: %v", err)
	}

	return nil
}

// DecryptPayloadWithNonce decrypts payload using M3DA Security Extension methods
func (sm *securityManager) decryptPayloadWithNonce(payload []byte, nonce []byte, clientID string) ([]byte, error) {
	if sm.config.Encryption == CipherNone {
		return payload, nil
	}

	// Derive cipher key based on encryption algorithm
	var keyLength int
	switch sm.config.Encryption {
	case CipherAESCBC128, CipherAESCTR128:
		keyLength = 16 // 128-bit
	case CipherAESCBC256, CipherAESCTR256:
		keyLength = 32 // 256-bit
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", sm.config.Encryption)
	}

	cipherKey, err := sm.deriveCipherKey(keyLength, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive cipher key: %w", err)
	}

	iv, err := sm.deriveIV()
	if err != nil {
		return nil, fmt.Errorf("failed to derive IV: %w", err)
	}

	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	switch sm.config.Encryption {
	case CipherAESCBC128, CipherAESCBC256:
		return sm.decryptCBC(block, payload, iv)
	case CipherAESCTR128, CipherAESCTR256:
		return sm.decryptCTR(block, payload, iv)
	default:
		return nil, fmt.Errorf("unsupported encryption mode: %s", sm.config.Encryption)
	}
}

// decryptCBC decrypts using AES-CBC mode and removes PKCS5 padding
func (sm *securityManager) decryptCBC(block cipher.Block, payload []byte, iv []byte) ([]byte, error) {
	if len(payload)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("payload is not a multiple of block size")
	}

	// Decrypt
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(payload))
	mode.CryptBlocks(decrypted, payload)

	// Remove PKCS5 padding
	if len(decrypted) == 0 {
		return nil, fmt.Errorf("empty decrypted payload")
	}

	padding := int(decrypted[len(decrypted)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}

	return decrypted[:len(decrypted)-padding], nil
}

// decryptCTR decrypts using AES-CTR mode
func (sm *securityManager) decryptCTR(block cipher.Block, payload []byte, iv []byte) ([]byte, error) {
	// CTR mode encryption and decryption are the same operation
	stream := cipher.NewCTR(block, iv)
	decrypted := make([]byte, len(payload))
	stream.XORKeyStream(decrypted, payload)
	return decrypted, nil
}
