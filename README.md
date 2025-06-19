# M3DA Go Client Library

A modern Go implementation of the M3DA (Machine-to-Machine Data Access) protocol client, based on reverse engineering of the Java reference implementation.

## Overview

M3DA is a secure and bandwidth-efficient M2M protocol designed for IoT and embedded systems communication. This Go library provides:

- **Binary Protocol Support**: Custom Bysant encoding for efficient data transmission
- **TCP Transport**: Built-in TCP client with connection management
- **Data Compression**: Specialized vectors for time-series and periodic data
- **Security Features**: Optional encryption (AES) and authentication (HMAC)
- **Modern Go API**: Context-aware, concurrent-safe implementation

## Features

### Core Features ✅
- [x] M3DA envelope encoding/decoding
- [x] Bysant binary codec
- [x] TCP transport layer
- [x] Message types (Message, Response)
- [x] Status code handling
- [x] Client identification
- [x] Data compression (DeltasVector, QuasiPeriodicVector)

### Optional Features ✅
- [x] Security (encryption + authentication) - **IMPLEMENTED**
- [ ] Auto-registration - TODO
- [ ] Connection pooling - TODO
- [ ] Retry logic - TODO

## Security Features

M3DA Go client now supports comprehensive security features:

### Authentication
- **HMAC-MD5**: Message authentication using MD5 hash
- **HMAC-SHA1**: Message authentication using SHA1 hash (recommended)
- **None**: No authentication

### Encryption
- **AES-CBC-128**: 128-bit AES in CBC mode
- **AES-CBC-256**: 256-bit AES in CBC mode
- **AES-CTR-128**: 128-bit AES in CTR mode
- **AES-CTR-256**: 256-bit AES in CTR mode (recommended)
- **None**: No encryption

### Key Features
- **Cryptographic Nonces**: Automatic nonce generation for replay protection
- **Envelope Integrity**: HMAC protection of entire message envelope
- **Payload Encryption**: AES encryption of message payloads

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    m3da "github.com/SierraWireless/m3da-go"
)

func main() {
    // Create client configuration
    config := m3da.DefaultClientConfig("localhost", "my-device-001")

    // Create and connect client
    client := m3da.NewTCPClient(config)
    ctx := context.Background()

    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Send telemetry data
    data := map[string]interface{}{
        "temperature": 23.5,
        "humidity":    65.2,
        "timestamp":   time.Now().Unix(),
    }

    err := client.SendData(ctx, "@sys.telemetry", data)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Data sent successfully!")
}
```

## Secure Communication

### Basic Security Configuration

```go
// HMAC authentication only
config.SecurityConfig = &m3da.SecurityConfig{
    Authentication: m3da.HMACTypeSHA1,
    Encryption:     m3da.CipherNone,
    Password:       "your-secure-password",
}

// Full encryption + authentication
config.SecurityConfig = &m3da.SecurityConfig{
    Authentication: m3da.HMACTypeSHA1,
    Encryption:     m3da.CipherAESCTR256,
    Password:       "your-secure-password",
}
```

### Security Examples

```go
// Create secure client
config := m3da.DefaultClientConfig("secure-server.example.com", "secure-device-001")
config.SecurityConfig = &m3da.SecurityConfig{
    Authentication: m3da.HMACTypeSHA1,
    Encryption:     m3da.CipherAESCTR256,
    Password:       "my-secure-password-123",
}

client := m3da.NewTCPClient(config)

// All communication is now encrypted and authenticated
err := client.SendData(ctx, "@sys.secure.telemetry", data)
```

## API Reference

### Client Configuration

```go
config := &m3da.ClientConfig{
    Host:           "m3da-server.example.com",
    Port:           44900, // IANA assigned port
    ClientID:       "unique-client-identifier",
    ConnectTimeout: 10 * time.Second,
    ReadTimeout:    30 * time.Second,
    WriteTimeout:   10 * time.Second,
    SecurityConfig: &m3da.SecurityConfig{
        Authentication: m3da.HMACTypeSHA1,
        Encryption:     m3da.CipherAESCTR256,
        Password:       "secure-password",
    },
}

// Or use defaults
config := m3da.DefaultClientConfig("host", "clientID")
```

### Basic Operations

```go
// Connect to server
err := client.Connect(ctx)

// Send simple data
err := client.SendData(ctx, "@sys.sensors", data)

// Send message with response
response, err := client.SendMessage(ctx, "@sys.commands", requestData)

// Send multiple messages in one envelope
messages := []m3da.M3daBodyMessage{message1, message2, message3}
responses, err := client.SendEnvelope(ctx, messages...)

// Close connection
err := client.Close()
```

### Message Types

#### M3DA Message
```go
message := &m3da.M3daMessage{
    Path:     "@sys.telemetry.temperature",
    TicketID: &ticketID, // Optional, for request-response correlation
    Body: map[string]interface{}{
        "value":     23.5,
        "timestamp": time.Now().Unix(),
        "sensor_id": "temp-001",
    },
}
```

#### M3DA Response
```go
response := &m3da.M3daResponse{
    TicketID: requestTicketID,
    Status:   200, // HTTP-like status codes
    Message:  "OK",
}
```

### Data Compression

#### Deltas Vector
Efficient encoding for similar sequential values:

```go
temperaturesDeltas := &m3da.M3daDeltasVector{
    Factor: 0.1,  // Scale factor
    Start:  235,  // Starting value (23.5°C * 10)
    Deltas: []float64{1, -2, 3, 0, -1, 2}, // Small variations
}

// Reconstruct original values
originalValues := temperaturesDeltas.AsFlatList()
// Result: [23.5, 23.6, 23.4, 23.7, 23.7, 23.6, 23.8]
```

#### Quasi-Periodic Vector
Efficient encoding for periodic data with variations:

```go
timestamps := &m3da.M3daQuasiPeriodicVector{
    Period: 60,                          // 60 seconds between measurements
    Start:  float64(time.Now().Unix()),  // Starting timestamp
    Shifts: []float64{5, -2, 3, 1, 2, -1, 4, 0}, // Timing variations
}

// Reconstruct original timestamps
originalTimestamps := timestamps.AsFlatList()
```

## Status Codes

M3DA uses HTTP-like status codes:

| Code | Status | Description |
|------|--------|-------------|
| 200  | OK | Everything went fine |
| 400  | BAD_REQUEST | Malformed request |
| 401  | UNAUTHORIZED | Incorrect credentials |
| 403  | FORBIDDEN | System not allowed |
| 407  | AUTHENTICATION_REQUIRED | No credentials provided |
| 450  | ENCRYPTION_NEEDED | Payload must be encrypted |
| 500  | UNEXPECTED_ERROR | Server error |
| 503  | SERVICE_UNAVAILABLE | Server unavailable |

## Examples

See the `examples/` directory for complete working examples:

- **Basic Client**: Simple data sending
- **Compressed Data**: Using deltas and quasi-periodic vectors
- **Request-Response**: Command and response pattern
- **Batch Operations**: Multiple messages per envelope

## Protocol Documentation

For detailed protocol information, see [M3DA_PROTOCOL_DOCUMENTATION.md](M3DA_PROTOCOL_DOCUMENTATION.md).

## Testing with Reference Server

Start the Java reference server:

```bash
git clone github.com/SierraWireless/m3da-server
cd m3da-server
mvn clean install
cd server
mvn assembly:assembly -DdescriptorId=jar-with-dependencies
java -jar target/m3da-server-1.0-SNAPSHOT-jar-with-dependencies.jar
```

The server will listen on:
- **M3DA TCP**: `localhost:44900`
- **HTTP REST**: `localhost:8080`

Then run the Go client examples:

```bash
go run examples/basic_client/main.go
```

## REST API Integration

The reference server provides REST endpoints for integration:

```bash
# View received data
curl http://localhost:8080/clients/my-device-001/data

# Send data to client
curl -X POST http://localhost:8080/clients/my-device-001/data \
  -H "Content-Type: application/json" \
  -d '{"settings": [{"key": "@sys.commands.test", "value": "hello"}]}'

# List connected clients
curl http://localhost:8080/clients
```

## Architecture

```
┌─────────────────┐    TCP/44900     ┌─────────────────┐
│   Go Client     │ ◄──────────────► │   M3DA Server   │
│                 │                  │                 │
│ ┌─────────────┐ │                  │ ┌─────────────┐ │
│ │   M3DA      │ │   M3DA Protocol  │ │   M3DA      │ │
│ │ TCP Client  │ │   (Binary/Bysant)│ │   Handler   │ │
│ └─────────────┘ │                  │ └─────────────┘ │
│                 │                  │                 │
│ ┌─────────────┐ │                  │ ┌─────────────┐ │
│ │   Bysant    │ │                  │ │   REST      │ │
│ │   Codec     │ │                  │ │   API       │ │
│ └─────────────┘ │                  │ └─────────────┘ │
└─────────────────┘                  └─────────────────┘
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the Eclipse Public License v1.0 - see the [LICENSE-EPLv1.0.html](LICENSE-EPLv1.0.html) file for details.

## References

- [M3DA Specification](http://wiki.eclipse.org/Mihini/M3DA_Specification)
- [Eclipse Mihini Project](http://www.eclipse.org/mihini)
- [Original Java Implementation](https://github.com/eclipse/mihini-m3da-server)

## Acknowledgments

This implementation is based on reverse engineering of the Eclipse Mihini M3DA Java reference implementation. Special thanks to the Sierra Wireless and Eclipse Mihini teams for creating the original protocol and reference implementation.