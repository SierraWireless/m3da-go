package m3da

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client represents an M3DA client interface
type Client interface {
	// Connect establishes connection to the M3DA server
	Connect(ctx context.Context) error

	// SendEnvelope sends messages in an M3DA envelope and returns received messages
	SendEnvelope(ctx context.Context, messages ...M3daBodyMessage) ([]M3daBodyMessage, error)

	// SendMessage is a convenience method to send a single message with response
	SendMessage(ctx context.Context, path string, body map[string]interface{}) (*M3daResponse, error)

	// SendData is a convenience method to send data to a specific path
	SendData(ctx context.Context, path string, data map[string]interface{}) error

	// Close closes the connection to the server
	Close() error

	// IsConnected returns true if the client is connected
	IsConnected() bool
}

// TCPClient is a TCP implementation of the M3DA client
type TCPClient struct {
	config          *ClientConfig
	conn            net.Conn
	connected       atomic.Bool
	securityManager *securityManager
	mutex           sync.RWMutex
	ticketCounter   atomic.Uint32
	negotiated      bool // Track if password negotiation is complete
}

// NewTCPClient creates a new TCP M3DA client
func NewTCPClient(config *ClientConfig) *TCPClient {
	client := &TCPClient{
		config: config,
	}

	// Initialize security manager if security is configured
	if config.SecurityConfig != nil {
		securityManager, err := newSecurityManager(config.SecurityConfig)
		if err != nil {
			// Log error but don't fail client creation
			// Security will be disabled
			warnf("Failed to initialize security: %v", err)
		} else {
			client.securityManager = securityManager
		}
	}

	return client
}

// Connect establishes a TCP connection to the M3DA server
func (c *TCPClient) Connect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.IsConnected() {
		return nil // Already connected
	}

	// Create connection with timeout
	dialer := &net.Dialer{
		Timeout: c.config.ConnectTimeout,
	}

	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	c.conn = conn
	c.connected.Store(true)
	c.negotiated = false // Reset negotiation state

	return nil
}

// performPasswordNegotiation performs the M3DA password negotiation handshake
func (c *TCPClient) performPasswordNegotiation(ctx context.Context) error {
	// FIXME: Untested unvalidated, disabled for now
	/*
		// Create negotiation envelope
		negotiationEnvelope, err := c.securityManager.PerformPasswordNegotiation(c.config.ClientID)
		if err != nil {
			return fmt.Errorf("failed to create negotiation envelope: %w", err)
		}

		// Encode and send negotiation envelope
		encoder := NewBysantEncoder()
		envelopeData, err := encoder.EncodeObject(negotiationEnvelope)
		if err != nil {
			return fmt.Errorf("failed to encode negotiation envelope: %w", err)
		}

		// Set write timeout
		if c.config.WriteTimeout > 0 {
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
		}

		// Send negotiation envelope
		_, err = c.conn.Write(envelopeData)
		if err != nil {
			return fmt.Errorf("failed to send negotiation envelope: %w", err)
		}

		// Set read timeout for negotiation response
		if c.config.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		}

		// Create decoder directly from TCP connection
		decoder := NewBysantDecoder(c.conn)
		messages, err := decoder.Decode()
		if err != nil {
			return fmt.Errorf("failed to decode negotiation response: %w", err)
		}

		if len(messages) == 0 {
			return fmt.Errorf("no negotiation response received")
		}

		// Find the envelope in the messages
		var envelope *M3daEnvelope
		for _, msg := range messages {
			if env, ok := msg.(*M3daEnvelope); ok {
				envelope = env
				break
			}
		}

		if envelope == nil {
			return fmt.Errorf("no envelope found in negotiation response")
		}

		// Process negotiation response
		err = c.securityManager.ProcessNegotiationResponse(envelope)
		if err != nil {
			return fmt.Errorf("negotiation response processing failed: %w", err)
		}

		c.negotiated = true
	*/
	return nil
}

// SendEnvelope sends messages and returns received messages
func (c *TCPClient) SendEnvelope(ctx context.Context, messages ...M3daBodyMessage) ([]M3daBodyMessage, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if password negotiation is required and completed
	// FIXME: not implemented
	/*if c.securityManager != nil && !c.negotiated {
		return nil, fmt.Errorf("password negotiation not completed")
	}*/

	// Encode messages to payload
	encoder := NewBysantEncoder()
	payload, err := encoder.Encode(messages...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode messages: %w", err)
	}

	// Create envelope
	envelope := &M3daEnvelope{
		Header: map[string]interface{}{
			HeaderKeyID: c.config.ClientID,
		},
		Payload: payload,
		Footer:  make(map[string]interface{}),
	}

	// Send envelope with retry logic for authentication challenges
	return c.sendEnvelopeWithRetryForMessages(envelope)
}

// sendEnvelopeWithRetryForMessages sends envelope and retries on 401, returning decoded messages
func (c *TCPClient) sendEnvelopeWithRetryForMessages(envelope *M3daEnvelope) ([]M3daBodyMessage, error) {
	var messages []M3daBodyMessage
	var err error

	// Apply security if configured
	if c.securityManager != nil {
		var protectedEnvelope *M3daEnvelope
		if protectedEnvelope, err = c.securityManager.applySecurityToEnvelope(envelope); err != nil {
			return nil, fmt.Errorf("failed to apply security: %w", err)
		}
		// First attempt
		messages, err = c.sendEnvelopeAndReadMessages(protectedEnvelope)
	} else {
		// First attempt
		messages, err = c.sendEnvelopeAndReadMessages(envelope)
	}

	if err != nil {
		// Check if it's a 401 error with challenge
		if m3daErr, ok := err.(*M3DAError); ok && (m3daErr.StatusCode == StatusUnauthorized || m3daErr.StatusCode == StatusAuthenticationRequired) && c.securityManager != nil {
			debugf("Got 401 challenge, retrying with server nonce...")
			var protectedEnvelope *M3daEnvelope

			// Re-apply security with the server nonce we just extracted
			if protectedEnvelope, err = c.securityManager.applySecurityToEnvelope(envelope); err != nil {
				return nil, fmt.Errorf("failed to re-apply security with server nonce: %w", err)
			}

			// Retry the request
			debugf("Sending retry request...")
			return c.sendEnvelopeAndReadMessages(protectedEnvelope)
		}
		return nil, err
	}

	return messages, nil
}

// sendEnvelopeAndReadMessages sends envelope and reads response messages
func (c *TCPClient) sendEnvelopeAndReadMessages(envelope *M3daEnvelope) ([]M3daBodyMessage, error) {
	// Encode envelope directly using BysantEncoder
	encoder := NewBysantEncoder()
	envelopeData, err := encoder.EncodeObject(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to encode envelope: %w", err)
	}

	// Set write timeout
	if c.config.WriteTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	}

	// Send envelope
	_, err = c.conn.Write(envelopeData)
	if err != nil {
		return nil, fmt.Errorf("failed to send envelope: %w", err)
	}

	// Set read timeout
	if c.config.ReadTimeout > 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	}

	// Read response
	return c.readResponse()
}

// readResponse reads and decodes the server response directly from TCP connection
func (c *TCPClient) readResponse() ([]M3daBodyMessage, error) {
	debugf("Waiting for response")

	// Create decoder directly from TCP connection
	decoder := NewBysantDecoder(c.conn)
	message, err := decoder.decodeMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Find the envelope in the messages
	var envelope *M3daEnvelope
	if env, ok := message.(*M3daEnvelope); ok {
		envelope = env
	} else {
		return []M3daBodyMessage{message}, nil // Return raw messages if no envelope found
	}

	// Update nonces from successful response for future communications
	if c.securityManager != nil {
		if err := c.verifyEnvelopeSecurity(envelope); err != nil {
			return nil, fmt.Errorf("failed to verify security: %w", err)
		}
	}

	// Check status
	if status, ok := envelope.Header[HeaderKeyStatus].(int64); ok {
		if StatusCode(status) != StatusOK {
			return nil, &M3DAError{
				StatusCode: StatusCode(status),
				Message:    fmt.Sprintf("Server returned status %d", status),
			}
		}
	}

	// Decode payload if present
	if len(envelope.Payload) == 0 {
		return []M3daBodyMessage{}, nil
	}

	// Decode body messages
	decoder = NewBysantDecoder(bytes.NewReader(envelope.Payload))
	return decoder.Decode()
}

// Close closes the connection
func (c *TCPClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.IsConnected() {
		return nil
	}

	c.connected.Store(false)

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

// IsConnected returns true if the client is connected
func (c *TCPClient) IsConnected() bool {
	return c.connected.Load()
}

// nextTicketID generates a unique ticket ID
func (c *TCPClient) nextTicketID() uint32 {
	return c.ticketCounter.Add(1)
}

// verifyEnvelopeSecurity verifies the security of received envelope
func (c *TCPClient) verifyEnvelopeSecurity(envelope *M3daEnvelope) error {
	if c.securityManager == nil {
		return nil // No security configured
	}

	return c.securityManager.verifyEnvelopeSecurity(envelope)
}

// SendMessage is a convenience method to send a single message
func (c *TCPClient) SendMessage(ctx context.Context, path string, body map[string]interface{}) (*M3daResponse, error) {
	ticketID := c.nextTicketID()

	message := &M3daMessage{
		Path:     path,
		TicketID: &ticketID,
		Body:     body,
	}

	responses, err := c.SendEnvelope(ctx, message)
	if err != nil {
		return nil, err
	}

	// Look for response with matching ticket ID
	for _, resp := range responses {
		if response, ok := resp.(*M3daResponse); ok && response.TicketID == ticketID {
			return response, nil
		}
	}

	return nil, fmt.Errorf("no response received for ticket ID %d", ticketID)
}

// SendData is a convenience method to send data to a specific path
func (c *TCPClient) SendData(ctx context.Context, path string, data map[string]interface{}) error {
	message := &M3daMessage{
		Path: path,
		Body: data,
	}

	_, err := c.SendEnvelope(ctx, message)
	return err
}

// closeConnectionUnsafe closes the connection without acquiring mutex (for internal use)
func (c *TCPClient) closeConnectionUnsafe() {
	c.connected.Store(false)
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
