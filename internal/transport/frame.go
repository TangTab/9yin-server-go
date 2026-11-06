package transport

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

const DefaultMaxFrameSize = 1 << 20

var (
	ErrFrameTooLarge = errors.New("transport frame too large")
	ErrInvalidEscape = errors.New("invalid EE escape")
	ErrInvalidLimit  = errors.New("invalid frame size limit")
)

// FrameReader owns one persistent buffered reader for the lifetime of a
// stream. Recreating a bufio.Reader per frame can discard bytes prefetched from
// the following frame.
type FrameReader struct {
	reader *bufio.Reader
	limit  int
}

func NewFrameReader(r io.Reader, limit int) (*FrameReader, error) {
	if r == nil {
		return nil, fmt.Errorf("nil frame reader")
	}
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	return &FrameReader{reader: bufio.NewReader(r), limit: limit}, nil
}

// ReadEncoded reads and unescapes one encoded body. The EE EE terminator is
// consumed but not returned. An EE 00 sequence contributes one EE body byte.
func (r *FrameReader) ReadEncoded() ([]byte, error) {
	capacity := 256
	if r.limit < capacity {
		capacity = r.limit
	}
	body := make([]byte, 0, capacity)
	started := false
	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && started {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
		started = true
		if b != 0xEE {
			body = append(body, b)
		} else {
			next, err := r.reader.ReadByte()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil, io.ErrUnexpectedEOF
				}
				return nil, err
			}
			switch next {
			case 0x00:
				body = append(body, 0xEE)
			case 0xEE:
				return body, nil
			default:
				return nil, fmt.Errorf("%w: EE %02X", ErrInvalidEscape, next)
			}
		}
		if len(body) > r.limit {
			return nil, fmt.Errorf("%w: limit %d", ErrFrameTooLarge, r.limit)
		}
	}
}

func (r *FrameReader) ReadFrame(key Key) ([]byte, error) {
	body, err := r.ReadEncoded()
	if err != nil {
		return nil, err
	}
	return key.Apply(body), nil
}

// EncodeFrame XORs, escapes, and terminates one plaintext message.
func EncodeFrame(plain []byte, key Key, limit int) ([]byte, error) {
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	if len(plain) > limit {
		return nil, fmt.Errorf("%w: size %d, limit %d", ErrFrameTooLarge, len(plain), limit)
	}
	body := key.Apply(plain)
	wire := make([]byte, 0, len(body)+2)
	for _, b := range body {
		wire = append(wire, b)
		if b == 0xEE {
			wire = append(wire, 0x00)
		}
	}
	return append(wire, 0xEE, 0xEE), nil
}

// FrameWriter serializes complete frames to an io.Writer and handles writers
// that accept fewer bytes than requested.
type FrameWriter struct {
	writer io.Writer
	limit  int
}

func NewFrameWriter(w io.Writer, limit int) (*FrameWriter, error) {
	if w == nil {
		return nil, fmt.Errorf("nil frame writer")
	}
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	return &FrameWriter{writer: w, limit: limit}, nil
}

func (w *FrameWriter) WriteFrame(plain []byte, key Key) error {
	wire, err := EncodeFrame(plain, key, w.limit)
	if err != nil {
		return err
	}
	for len(wire) > 0 {
		n, writeErr := w.writer.Write(wire)
		if n < 0 || n > len(wire) {
			return fmt.Errorf("invalid write count %d for %d bytes", n, len(wire))
		}
		wire = wire[n:]
		if writeErr != nil {
			return writeErr
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
