// Package transport implements the byte-stream framing used by the game
// connection. It deliberately contains no login or game-message semantics.
package transport

import "encoding/binary"

const InitialKeyValue uint32 = 0xFA6D3BCC

// Key is an immutable four-byte repeating XOR key. Keeping its bytes private
// prevents callers from accidentally changing a live connection's key through
// a shared slice.
type Key struct {
	bytes [4]byte
}

// NewKey constructs a key from the little-endian value used by FxClient.
func NewKey(value uint32) Key {
	var key Key
	binary.LittleEndian.PutUint32(key.bytes[:], value)
	return key
}

// InitialKey is the key observed on the first modern-client login flight.
var InitialKey = NewKey(InitialKeyValue)

// Uint32 returns the key in the representation used by the client binary.
func (k Key) Uint32() uint32 {
	return binary.LittleEndian.Uint32(k.bytes[:])
}

// Apply returns a transformed copy of src. It never aliases or modifies src.
// XOR is symmetric, so the same operation encodes and decodes a frame body.
func (k Key) Apply(src []byte) []byte {
	dst := make([]byte, len(src))
	for i, b := range src {
		dst[i] = b ^ k.bytes[i&3]
	}
	return dst
}
