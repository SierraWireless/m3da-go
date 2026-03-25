package m3da

import "fmt"

func init() {
	// Register the response decoder
	registerCustomDecoder(OpCodeResponse, decodeResponse)
}

// M3daResponse represents an M3DA response
type M3daResponse struct {
	TicketID uint32 `json:"ticket_id"`
	Status   int64  `json:"status"`
	Message  string `json:"message,omitempty"`
}

// GetOpCode returns the operation code for M3DA responses
func (r *M3daResponse) GetOpCode() byte {
	return OpCodeResponse
}

// EncodeTo encodes the response using the provided encoder
func (r *M3daResponse) EncodeTo(encoder *BysantEncoder) error {
	// Write the response opcode
	encoder.buf.WriteByte(OpCodeResponse)

	// Encode ticket ID
	if err := encoder.encodeUnsignedIntegerInContext(r.TicketID, ContextUintsAndStrs); err != nil {
		return err
	}

	// Encode status
	if err := encoder.encodeIntegerInContext(r.Status, ContextNumber); err != nil {
		return err
	}

	// Encode message
	return encoder.encodeString(r.Message, ContextUintsAndStrs)
}

// DecodeResponse decodes an M3DA response from the decoder
func decodeResponse(decoder *BysantDecoder) (M3daEncodable, error) {
	// Decode ticket ID (in UINT AND STRINGS context)
	ticketObj, err := decoder.decodeObjectInContext(ContextUintsAndStrs)
	if err != nil {
		return nil, err
	}

	ticketID, ok := ticketObj.(uint32)
	if !ok {
		return nil, fmt.Errorf("invalid ticket ID type: got %T", ticketObj)
	}

	// Decode status (in Number context)
	statusObj, err := decoder.decodeObjectInContext(ContextNumber)
	if err != nil {
		return nil, err
	}

	status, ok := statusObj.(int64)
	if !ok {
		return nil, fmt.Errorf("invalid status type: got %T", statusObj)
	}

	// Decode message (string in GLOBAL context)
	messageObj, err := decoder.decodeObjectInContext(ContextGlobal)
	if err != nil {
		return nil, err
	}

	message, ok := messageObj.(string)
	if !ok {
		message = ""
	}

	return &M3daResponse{
		TicketID: ticketID,
		Status:   status,
		Message:  message,
	}, nil
}
