package protocol

import (
	"bufio"
	"errors"
	"io"
)

const InitialMessageKey uint32 = 0xFA6D3BCC

var ErrFrameTooLarge = errors.New("protocol frame too large")

// ReadWireFrame reads bytes through the unescaped EE EE terminator.
func ReadWireFrame(r io.Reader, limit int) ([]byte, error) {
	br := bufio.NewReader(r)
	out := make([]byte, 0, 256)
	for {
		b, err := br.ReadByte()
		if err != nil {
			return nil, err
		}
		if b != 0xEE {
			out = append(out, b)
		} else {
			next, err := br.ReadByte()
			if err != nil {
				return nil, err
			}
			switch next {
			case 0x00:
				out = append(out, 0xEE)
			case 0xEE:
				return out, nil
			default:
				return nil, errors.New("invalid EE escape")
			}
		}
		if len(out) > limit {
			return nil, ErrFrameTooLarge
		}
	}
}

func XORMessage(buf []byte, key uint32) []byte {
	keyBytes := [4]byte{byte(key), byte(key >> 8), byte(key >> 16), byte(key >> 24)}
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = b ^ keyBytes[i&3]
	}
	return out
}

func EncodeWireFrame(plain []byte, key uint32) []byte {
	encoded := XORMessage(plain, key)
	out := make([]byte, 0, len(encoded)+2)
	for _, b := range encoded {
		out = append(out, b)
		if b == 0xEE {
			out = append(out, 0x00)
		}
	}
	return append(out, 0xEE, 0xEE)
}
