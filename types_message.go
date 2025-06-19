package m3da

import "fmt"

func init() {
	// Register the message decoder
	registerCustomDecoder(OpCodeMessage, decodeMessage)
}

// M3daMessage represents an M3DA message
type M3daMessage struct {
	Path     string                 `json:"path"`
	TicketID *uint32                `json:"ticket_id,omitempty"`
	Body     map[string]interface{} `json:"body"`
}

// GetOpCode returns the operation code for M3DA messages
func (m *M3daMessage) GetOpCode() byte {
	return OpCodeMessage
}

// EncodeTo encodes the message using the provided encoder
func (m *M3daMessage) EncodeTo(encoder *BysantEncoder) error {
	// Write the message opcode
	encoder.buf.WriteByte(OpCodeMessage)

	// Message path uses UINTS_AND_STRS context
	if err := encoder.encodeStringInContext(m.Path, ContextUintsAndStrs); err != nil {
		return err
	}

	// Message ticket ID uses UINTS_AND_STRS context (unsigned integers)
	if m.TicketID != nil {
		if err := encoder.encodeUnsignedIntegerInContext(*m.TicketID, ContextUintsAndStrs); err != nil {
			return err
		}
	} else {
		if err := encoder.encodeNull(); err != nil {
			return err
		}
	}

	// Message body uses LIST_AND_MAPS context
	return encoder.encodeMapInContext(m.Body, ContextListAndMaps)
}

// DecodeMessage decodes an M3DA message from the decoder
func decodeMessage(decoder *BysantDecoder) (M3daEncodable, error) {
	// Decode path (string in UINTS_AND_STRS context)
	pathObj, err := decoder.decodeObjectInContext(ContextUintsAndStrs)
	if err != nil {
		return nil, err
	}

	path, ok := pathObj.(string)
	if !ok {
		return nil, fmt.Errorf("expected string path, got %T", pathObj)
	}

	// Decode ticket ID (in UINTS_AND_STRS context, can be null or int64)
	ticketObj, err := decoder.decodeObjectInContext(ContextUintsAndStrs)
	if err != nil {
		return nil, err
	}

	var ticketID *uint32
	if ticketObj != nil {
		if tid, ok := ticketObj.(uint32); ok {
			ticketID = &tid
		}
	}

	// Decode body (map in LIST_AND_MAPS context)
	bodyObj, err := decoder.decodeObjectInContext(ContextListAndMaps)
	if err != nil {
		return nil, err
	}

	body, ok := bodyObj.(map[string]interface{})
	if !ok {
		body = make(map[string]interface{})
	}

	return &M3daMessage{
		Path:     path,
		TicketID: ticketID,
		Body:     body,
	}, nil
}
