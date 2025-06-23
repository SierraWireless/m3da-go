package m3da

import (
	"fmt"
	"time"
)

// StatusCode represents M3DA protocol status codes
type StatusCode int

const (
	StatusOK                     StatusCode = 200
	StatusBadRequest             StatusCode = 400
	StatusUnauthorized           StatusCode = 401
	StatusForbidden              StatusCode = 403
	StatusAuthenticationRequired StatusCode = 407
	StatusEncryptionNeeded       StatusCode = 450
	StatusShortcutMapError       StatusCode = 451
	StatusUnexpectedError        StatusCode = 500
	StatusServiceUnavailable     StatusCode = 503
)

// String returns the string representation of the status code
func (s StatusCode) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusBadRequest:
		return "BAD_REQUEST"
	case StatusUnauthorized:
		return "UNAUTHORIZED"
	case StatusForbidden:
		return "FORBIDDEN"
	case StatusAuthenticationRequired:
		return "AUTHENTICATION_REQUIRED"
	case StatusEncryptionNeeded:
		return "ENCRYPTION_NEEDED"
	case StatusShortcutMapError:
		return "SHORTCUT_MAP_ERROR"
	case StatusUnexpectedError:
		return "UNEXPECTED_ERROR"
	case StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(s))
	}
}

// M3DAError represents an M3DA protocol error
type M3DAError struct {
	StatusCode StatusCode
	Message    string
}

func (e *M3DAError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("M3DA error %d (%s): %s", int(e.StatusCode), e.StatusCode.String(), e.Message)
	}
	return fmt.Sprintf("M3DA error %d (%s)", int(e.StatusCode), e.StatusCode.String())
}

// HeaderKey constants for M3DA envelope headers
const (
	HeaderKeyID            = "id"
	HeaderKeyStatus        = "status"
	HeaderKeyNonce         = "nonce"
	HeaderKeyChallenge     = "challenge"
	HeaderKeyMAC           = "mac"
	HeaderKeyAutoregSalt   = "autoreg_salt"
	HeaderKeyAutoregPubkey = "autoreg_pubkey"
	HeaderKeyAutoregCtext  = "autoreg_ctext"
	HeaderKeyAutoregMAC    = "autoreg_mac"
)

// Bysant OpCodes
const (
	OpCodeTinyString          = 0x03
	OpCodeSmallString         = 0x24
	OpCodeLargeString         = 0x28
	OpCodeChunkedString       = 0x29
	OpCodeEnvelope            = 0x60
	OpCodeMessage             = 0x61
	OpCodeResponse            = 0x62
	OpCodeDeltasVector        = 0x63
	OpCodeQuasiPeriodicVector = 0x64
	OpCodeInt64               = 0xFD
	OpCodeFloat32             = 0xFE
	OpCodeFloat64             = 0xFF
)

// Numeric constraint defines the supported numeric types for deltas vectors
type Numeric interface {
	~int32 | ~int64 | ~float32 | ~float64
}

// EncodingContext represents the context for Bysant encoding
type EncodingContext int

const (
	ContextGlobal       EncodingContext = 0
	ContextUintsAndStrs EncodingContext = 1
	ContextNumber       EncodingContext = 2
	ContextInt32        EncodingContext = 3
	ContextFloat        EncodingContext = 4
	ContextDouble       EncodingContext = 5
	ContextListAndMaps  EncodingContext = 6
)

// M3daEncodable represents any M3DA data structure that can be encoded/decoded
type M3daEncodable interface {
	GetOpCode() byte
	EncodeTo(encoder *BysantEncoder) error
}

// M3daBodyMessage is the interface for all M3DA body messages
// It extends M3daEncodable to ensure all message types can be encoded consistently
type M3daBodyMessage interface {
	M3daEncodable
}

// ClientConfig represents M3DA client configuration
type ClientConfig struct {
	Host           string
	Port           int
	ClientID       string
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	SecurityConfig *SecurityConfig
}

// SecurityConfig represents M3DA security configuration
type SecurityConfig struct {
	Authentication   HMACType
	Encryption       CipherAlgorithm
	Password         string
	AutoRegistration bool
	ServerID         string
}

// HMACType represents HMAC authentication types
type HMACType string

const (
	HMACTypeNone HMACType = "NONE"
	HMACTypeMD5  HMACType = "HMAC_MD5"
	HMACTypeSHA1 HMACType = "HMAC_SHA1"
)

// CipherAlgorithm represents encryption algorithms
type CipherAlgorithm string

const (
	CipherNone      CipherAlgorithm = "NONE"
	CipherAESCBC128 CipherAlgorithm = "AES_CBC_128"
	CipherAESCBC256 CipherAlgorithm = "AES_CBC_256"
	CipherAESCTR128 CipherAlgorithm = "AES_CTR_128"
	CipherAESCTR256 CipherAlgorithm = "AES_CTR_256"
)

// DefaultPort is the IANA assigned port for M3DA
const DefaultPort = 44900

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig(host, clientID string) *ClientConfig {
	return &ClientConfig{
		Host:           host,
		Port:           DefaultPort,
		ClientID:       clientID,
		ConnectTimeout: 10 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   10 * time.Second,
	}
}
