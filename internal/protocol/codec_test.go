package protocol

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCapturedLoginFrame(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protocol", "connect-first-flight.bin"))
	if err != nil {
		t.Fatal(err)
	}
	body, err := ReadWireFrame(bytes.NewReader(raw), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	plain := XORMessage(body, InitialMessageKey)
	if len(plain) != 121 {
		t.Fatalf("decoded length = %d, want 121", len(plain))
	}
	if plain[0] != 0x02 {
		t.Fatalf("opcode = %#02x, want Login(0x02)", plain[0])
	}
	if got := EncodeWireFrame(plain, InitialMessageKey); !bytes.Equal(got, raw) {
		t.Fatal("frame did not round-trip byte-for-byte")
	}
}

func TestEscapeEE(t *testing.T) {
	plain := []byte{0x22, 0xD5, 0x80, 0x18}
	wire := EncodeWireFrame(plain, InitialMessageKey)
	if !bytes.Contains(wire, []byte{0xEE, 0x00}) {
		t.Fatalf("wire %x does not contain escaped EE", wire)
	}
	body, err := ReadWireFrame(bytes.NewReader(wire), 1024)
	if err != nil {
		t.Fatal(err)
	}
	if got := XORMessage(body, InitialMessageKey); !bytes.Equal(got, plain) {
		t.Fatalf("round trip = %x, want %x", got, plain)
	}
}
