package main

import "encoding/binary"

// clientObjectRequest is the observed fixed C2S 0x07 reference packet. The
// client sends it when an in-world object is clicked. Its surrounding request
// type has not yet been named by the modern client, so only the proven identity
// fields are exposed here rather than pretending it is a dialogue packet.
type clientObjectRequest struct {
	Sequence uint32
	ObjectID uint32
	OwnerID  uint32
	FuncID   int32
}

func parseClientObjectRequest(message []byte) (clientObjectRequest, bool) {
	// Captures are exactly 33 bytes. The identity begins at offsets 21 and 25;
	// accept a longer compatible packet but never inspect an incomplete one.
	if len(message) < 29 || message[0] != 0x07 {
		return clientObjectRequest{}, false
	}
	request := clientObjectRequest{
		Sequence: binary.LittleEndian.Uint32(message[17:]),
		ObjectID: binary.LittleEndian.Uint32(message[21:]),
		OwnerID:  binary.LittleEndian.Uint32(message[25:]),
		FuncID:   int32(binary.LittleEndian.Uint32(message[29:])),
	}
	if request.ObjectID == 0 || request.OwnerID == 0 {
		return clientObjectRequest{}, false
	}
	return request, true
}
