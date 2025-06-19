package m3da

import (
	"fmt"
)

func init() {
	// Register the envelope decoder
	registerCustomDecoder(OpCodeEnvelope, decodeEnvelope)
}

// M3daEnvelope represents an M3DA envelope containing header, payload, and footer
type M3daEnvelope struct {
	Header  map[string]interface{} `json:"header"`
	Payload []byte                 `json:"payload"`
	Footer  map[string]interface{} `json:"footer"`
}

// GetOpCode returns the operation code for M3DA envelopes
func (e *M3daEnvelope) GetOpCode() byte {
	return OpCodeEnvelope
}

// EncodeTo encodes the envelope using the provided encoder
func (e *M3daEnvelope) EncodeTo(encoder *BysantEncoder) error {
	// Write the envelope opcode
	encoder.buf.WriteByte(OpCodeEnvelope)

	// Encode header as map (LIST_AND_MAPS context - already correct)
	if err := encoder.encodeMapInContext(e.Header, ContextListAndMaps); err != nil {
		return err
	}

	// Encode payload as binary using UINTS_AND_STRS context
	if err := encoder.encodeBinaryInContext(e.Payload, ContextUintsAndStrs); err != nil {
		return err
	}

	// Encode footer as map (LIST_AND_MAPS context - already correct)
	return encoder.encodeMapInContext(e.Footer, ContextListAndMaps)
}

// DecodeEnvelope decodes an M3DA envelope from the decoder
func decodeEnvelope(decoder *BysantDecoder) (M3daEncodable, error) {
	// Decode header (map in LIST_AND_MAPS context)
	headerObj, err := decoder.decodeObjectInContext(ContextListAndMaps)
	if err != nil {
		return nil, fmt.Errorf("failed to decode envelope header: %w", err)
	}

	header, ok := headerObj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("envelope header is not a map")
	}

	// Decode payload (binary/string in UINTS_AND_STRS context)
	payloadObj, err := decoder.decodeObjectInContext(ContextUintsAndStrs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode envelope payload: %w", err)
	}

	var payload []byte
	if payloadObj != nil {
		if s, ok := payloadObj.(string); ok {
			payload = []byte(s)
		} else if b, ok := payloadObj.([]byte); ok {
			payload = b
		} else {
			return nil, fmt.Errorf("envelope payload is not binary data")
		}
	}

	// Decode footer (map in LIST_AND_MAPS context)
	footerObj, err := decoder.decodeObjectInContext(ContextListAndMaps)
	if err != nil {
		return nil, fmt.Errorf("failed to decode envelope footer: %w", err)
	}

	footer, ok := footerObj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("envelope footer is not a map")
	}

	return &M3daEnvelope{
		Header:  header,
		Payload: payload,
		Footer:  footer,
	}, nil
}
