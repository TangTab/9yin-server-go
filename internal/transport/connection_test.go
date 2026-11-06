package transport

import (
	"bytes"
	"net"
	"testing"
)

func TestConnectionOwnsPersistentReaderAndDirectionalKeys(t *testing.T) {
	leftStream, rightStream := net.Pipe()
	leftConfig := ConnectionConfig{MaxFrameSize: 1024, ReadKey: NewKey(0x22222222), WriteKey: NewKey(0x11111111)}
	rightConfig := ConnectionConfig{MaxFrameSize: 1024, ReadKey: NewKey(0x11111111), WriteKey: NewKey(0x22222222)}
	left, err := NewConnection(leftStream, leftConfig)
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewConnection(rightStream, rightConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer left.Close()
	defer right.Close()

	assertExchange := func(sender, receiver *Connection, want []byte) {
		t.Helper()
		writeDone := make(chan error, 1)
		go func() { writeDone <- sender.WriteFrame(want) }()
		got, readErr := receiver.ReadFrame()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if writeErr := <-writeDone; writeErr != nil {
			t.Fatal(writeErr)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("received % X, want % X", got, want)
		}
	}

	assertExchange(left, right, []byte("left to right"))
	assertExchange(right, left, []byte("right to left"))

	rotatedLeftToRight := NewKey(0x33333333)
	left.SetWriteKey(rotatedLeftToRight)
	right.SetReadKey(rotatedLeftToRight)
	assertExchange(left, right, []byte("rotated"))
}
