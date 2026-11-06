package main

import (
	"encoding/binary"
	"math"
)

func parseMotionPosition(msg []byte) (x, y, z, orient float32, ok bool) {
	// The 67-byte position update has one stable four-float run at 0x26:
	// X, Y, Z and orientation. Longer motion packets contain destination X/Z,
	// speed and angle in the same tagged format; scanning those packets can form
	// a plausible but false coordinate tuple, so they must not update storage.
	if len(msg) != 67 {
		return 0, 0, 0, 0, false
	}
	const offset = 0x26
	if msg[offset] != 4 || msg[offset+5] != 4 || msg[offset+10] != 4 || msg[offset+15] != 4 {
		return 0, 0, 0, 0, false
	}
	x = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+1:]))
	y = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+6:]))
	z = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+11:]))
	orient = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+16:]))
	if !float32Finite(x) || !float32Finite(y) || !float32Finite(z) || !float32Finite(orient) {
		return 0, 0, 0, 0, false
	}
	if math.Abs(float64(x)) > 10000 || math.Abs(float64(y)) > 3000 || math.Abs(float64(z)) > 10000 || math.Abs(float64(orient)) > 20 {
		return 0, 0, 0, 0, false
	}
	return x, y, z, orient, true
}

func float32Finite(value float32) bool {
	v := float64(value)
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
