package transport

import (
	"bytes"
	"testing"
)

func TestKeyUsesLittleEndianBytesAndDoesNotMutateInput(t *testing.T) {
	key := NewKey(0xFA6D3BCC)
	input := []byte{0, 0, 0, 0, 1}
	wantInput := append([]byte(nil), input...)
	got := key.Apply(input)
	if want := []byte{0xCC, 0x3B, 0x6D, 0xFA, 0xCD}; !bytes.Equal(got, want) {
		t.Fatalf("Apply() = % X, want % X", got, want)
	}
	if !bytes.Equal(input, wantInput) {
		t.Fatalf("Apply mutated input: % X", input)
	}
	got[0] = 0
	if second := key.Apply(input); second[0] != 0xCC {
		t.Fatalf("returned buffer aliases key state: % X", second)
	}
	if key.Uint32() != 0xFA6D3BCC {
		t.Fatalf("Uint32() = %#08x", key.Uint32())
	}
}

func TestKeyApplyIsSymmetric(t *testing.T) {
	plain := []byte("nineyin transport")
	encoded := InitialKey.Apply(plain)
	if got := InitialKey.Apply(encoded); !bytes.Equal(got, plain) {
		t.Fatalf("round trip = %q, want %q", got, plain)
	}
}
