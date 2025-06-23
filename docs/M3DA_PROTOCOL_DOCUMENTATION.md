# M3DA Protocol Documentation

## Overview

M3DA (Machine-to-Machine Data Access) is a secure and bandwidth-efficient protocol designed for IoT and embedded systems communication. This document provides comprehensive documentation of the M3DA protocol based on reverse engineering of the Java reference implementation.

## Protocol Stack

```
┌─────────────────────────────────────────────────┐
│                Application Layer                │
├─────────────────────────────────────────────────┤
│ M3DA Messages (Message, Response, Vectors)     │
├─────────────────────────────────────────────────┤
│ M3DA Envelope (Header, Payload, Footer)        │
├─────────────────────────────────────────────────┤
│ Bysant Binary Serialization                    │
├─────────────────────────────────────────────────┤
│ TCP Transport (Length-Prefixed Frames)         │
├─────────────────────────────────────────────────┤
│ Security Layer (Optional AES + HMAC)           │
└─────────────────────────────────────────────────┘
```

## Bysant Binary Serialization

M3DA uses the Bysant binary serialization format for efficient data encoding. Bysant is a self-describing, context-aware binary format.

### Bysant Contexts

The Bysant protocol defines 7 different encoding contexts:

- **Context 0 (Global)**: Default context for general objects
- **Context 1 (UIS)**: Unsigned Integers and Strings (uint32 max)
- **Context 2 (Numbers)**: Optimized for numeric values
- **Context 3 (Int32)**: 32-bit signed integers only
- **Context 4 (Float)**: 32-bit floating point numbers
- **Context 5 (Double)**: 64-bit floating point numbers
- **Context 6 (Lists & Maps)**: Container objects

## M3DA Envelope Structure

Every M3DA communication is wrapped in an envelope:

```go
type M3daEnvelope struct {
    Header  map[string]interface{} // Control information
    Payload []byte                 // Bysant-encoded messages
    Footer  map[string]interface{} // Additional metadata
}
```

### Standard Header Keys

| Key | Type | Description |
|-----|------|-------------|
| `id` | string | Client identifier |
| `status` | int64 | Response status code |
| `nonce` | int64 | Cryptographic nonce |
| `challenge` | []byte | Authentication challenge |
| `mac` | []byte | HMAC signature |

## M3DA Message Types

### M3DA Message

Standard data message for sending telemetry or commands:

```go
type M3daMessage struct {
    Path     string                 // Message path (e.g., "@sys.telemetry")
    TicketID *int64                 // Optional correlation ID
    Body     map[string]interface{} // Message payload
}
```

### M3DA Response

Response to a message with ticket ID:

```go
type M3daResponse struct {
    TicketID int64  // Correlation ID from request
    Status   int64  // HTTP-like status code
    Message  string // Optional status message
}
```

## Data Compression Vectors

M3DA provides specialized data structures for efficient compression of time-series data.

### Deltas Vector

Compresses sequences of similar values using deltas:

```go
type M3daDeltasVector struct {
    Factor interface{}   // Scale factor
    Start  interface{}   // Starting value
    Deltas []interface{} // Delta values
}
```

**Example**: Temperature readings [23.5, 23.6, 23.4, 23.7] can be compressed as:
- Factor: 0.1
- Start: 235 (23.5 * 10)
- Deltas: [1, -2, 3] (representing +0.1, -0.2, +0.3)

### Quasi-Periodic Vector

Compresses periodic data with variations:

```go
type M3daQuasiPeriodicVector struct {
    Period interface{}   // Base period
    Start  interface{}   // Starting value
    Shifts []interface{} // Timing variations
}
```

**Example**: Timestamps with 60-second intervals but slight variations.

## Security Features

M3DA supports comprehensive security through encryption and authentication.

### Authentication (HMAC)

- **HMAC-MD5**: Legacy support
- **HMAC-SHA1**: Recommended standard
- **None**: No authentication

### Encryption (AES)

- **AES-CBC-128**: 128-bit AES in CBC mode
- **AES-CBC-256**: 256-bit AES in CBC mode
- **AES-CTR-128**: 128-bit AES in CTR mode
- **AES-CTR-256**: 256-bit AES in CTR mode (recommended)
- **None**: No encryption

## Status Codes

M3DA uses HTTP-like status codes:

| Code | Status | Description |
|------|--------|-------------|
| 200 | OK | Success |
| 400 | BAD_REQUEST | Malformed request |
| 401 | UNAUTHORIZED | Invalid credentials |
| 403 | FORBIDDEN | Access denied |
| 407 | AUTHENTICATION_REQUIRED | No credentials |
| 450 | ENCRYPTION_NEEDED | Encryption required |
| 500 | UNEXPECTED_ERROR | Server error |
| 503 | SERVICE_UNAVAILABLE | Server unavailable |

## Transport Layer

M3DA uses TCP with length-prefixed framing:

1. **Length Prefix**: 4-byte big-endian length
2. **Envelope Data**: Bysant-encoded envelope
3. **Connection**: Persistent TCP connection on port 44900

## Protocol Flow

### Basic Communication

1. Client connects to server (TCP port 44900)
2. Client sends envelope with message(s)
3. Server processes and responds with envelope
4. Connection remains open for subsequent exchanges

### Secure Communication

1. Client configures security (encryption + authentication)
2. For each envelope:
   - Generate cryptographic nonce
   - Encrypt payload (if configured)
   - Calculate HMAC over envelope
   - Send secured envelope
3. Server verifies HMAC and decrypts payload
4. Server responds with secured envelope

## Implementation Notes

### Type Handling

- **Integers**: Use int64 for all integer values
- **Floats**: Support both float32 and float64
- **Strings**: UTF-8 encoded, binary data as []byte
- **Nonces**: Use int64, not uint64 (Bysant limitation)

### Context Usage

- **Message paths**: Use UIS context (Context 1)
- **Message bodies**: Use Lists & Maps context (Context 6)
- **Numeric data**: Use Numbers context (Context 2) when appropriate

### Error Handling

- Always check status codes in responses
- Handle security errors (450, 407) appropriately
- Implement proper timeout handling

## References

- [M3DA Specification](http://wiki.eclipse.org/Mihini/M3DA_Specification)
- [Bysant Serializer PDF](https://wiki.eclipse.org/images/6/6a/M3DABysantSerializer.pdf)
- [Eclipse Mihini Project](http://www.eclipse.org/mihini)

## Reverse Engineering Notes

This documentation is based on analysis of the Java reference implementation, including:

- Packet capture analysis
- Java source code examination
- Protocol behavior observation
- Binary format reverse engineering

The Go implementation maintains compatibility with the Java reference while providing a modern, idiomatic Go API.