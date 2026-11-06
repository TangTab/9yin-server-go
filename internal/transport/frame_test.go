package transport

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCapturedLoginFrameRoundTrip(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protocol", "connect-first-flight.bin"))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewFrameReader(bytes.NewReader(raw), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := reader.ReadFrame(InitialKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 121 || plain[0] != 0x02 {
		t.Fatalf("decoded frame length/opcode = %d/%#02x", len(plain), plain[0])
	}
	wire, err := EncodeFrame(plain, InitialKey, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wire, raw) {
		t.Fatal("captured frame did not round-trip byte-for-byte")
	}
}

func TestFrameReaderPreservesPrefetchedFollowingFrames(t *testing.T) {
	first, _ := EncodeFrame([]byte{1, 2, 3}, InitialKey, 1024)
	second, _ := EncodeFrame([]byte{4, 5, 6}, InitialKey, 1024)
	reader, err := NewFrameReader(bytes.NewReader(append(first, second...)), 1024)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range [][]byte{{1, 2, 3}, {4, 5, 6}} {
		got, readErr := reader.ReadFrame(InitialKey)
		if readErr != nil {
			t.Fatalf("frame %d: %v", i, readErr)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("frame %d = % X, want % X", i, got, want)
		}
	}
}

type oneByteReader struct{ source io.Reader }

func (r oneByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.source.Read(p)
}

func TestFrameReaderAcceptsFragmentedStreamAndEscapedEE(t *testing.T) {
	plain := []byte{0x22, 0xD5, 0x80, 0x18, 0xEE}
	wire, err := EncodeFrame(plain, InitialKey, 1024)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewFrameReader(oneByteReader{bytes.NewReader(wire)}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reader.ReadFrame(InitialKey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("ReadFrame() = % X, want % X", got, plain)
	}
}

func TestFrameReaderRejectsMalformedAndTruncatedFrames(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
		want error
	}{
		{name: "invalid escape", wire: []byte{1, 0xEE, 0x7F}, want: ErrInvalidEscape},
		{name: "truncated escape", wire: []byte{1, 0xEE}, want: io.ErrUnexpectedEOF},
		{name: "missing terminator", wire: []byte{1, 2}, want: io.ErrUnexpectedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewFrameReader(bytes.NewReader(tt.wire), 1024)
			if err != nil {
				t.Fatal(err)
			}
			_, err = reader.ReadEncoded()
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}

func TestFrameReaderDistinguishesCleanEOF(t *testing.T) {
	reader, err := NewFrameReader(bytes.NewReader(nil), 1024)
	if err != nil {
		t.Fatal(err)
	}
	_, err = reader.ReadEncoded()
	if !errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("empty stream error = %v, want clean EOF", err)
	}
}

func TestFrameSizeLimit(t *testing.T) {
	reader, _ := NewFrameReader(bytes.NewReader([]byte{1, 2, 3, 0xEE, 0xEE}), 2)
	if _, err := reader.ReadEncoded(); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("oversize read error = %v", err)
	}

	exact, _ := NewFrameReader(bytes.NewReader([]byte{1, 2, 0xEE, 0xEE}), 2)
	if got, err := exact.ReadEncoded(); err != nil || !bytes.Equal(got, []byte{1, 2}) {
		t.Fatalf("exact-limit read = % X, %v", got, err)
	}

	if _, err := EncodeFrame([]byte{1, 2, 3}, NewKey(0), 2); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("oversize encode error = %v", err)
	}
	if _, err := NewFrameReader(bytes.NewReader(nil), 0); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("zero reader limit error = %v", err)
	}
	if _, err := NewFrameWriter(io.Discard, -1); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("negative writer limit error = %v", err)
	}
}

type chunkWriter struct {
	bytes.Buffer
	max int
}

func (w *chunkWriter) Write(p []byte) (int, error) {
	if len(p) > w.max {
		p = p[:w.max]
	}
	return w.Buffer.Write(p)
}

func TestFrameWriterCompletesPartialWrites(t *testing.T) {
	destination := &chunkWriter{max: 2}
	writer, err := NewFrameWriter(destination, 1024)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte{0x22, 0xD5, 0x80, 0x18, 9, 10}
	if err := writer.WriteFrame(plain, InitialKey); err != nil {
		t.Fatal(err)
	}
	want, _ := EncodeFrame(plain, InitialKey, 1024)
	if !bytes.Equal(destination.Bytes(), want) {
		t.Fatalf("written wire = % X, want % X", destination.Bytes(), want)
	}
}

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

func TestFrameWriterRejectsNoProgress(t *testing.T) {
	writer, _ := NewFrameWriter(zeroWriter{}, 1024)
	if err := writer.WriteFrame([]byte{1}, InitialKey); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteFrame error = %v, want io.ErrShortWrite", err)
	}
}
