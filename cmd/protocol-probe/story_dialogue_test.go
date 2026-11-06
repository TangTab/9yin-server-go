package main

import (
	"encoding/binary"
	"testing"
)

func TestEncodeNPCBubbleUsesRegisteredCustomRoute(t *testing.T) {
	frame, err := encodeNPCBubble(3494, 1, "ui_shop")
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x1E || binary.LittleEndian.Uint16(frame[1:]) != 5 {
		t.Fatalf("bubble header=%x", frame)
	}
	got := int32(binary.LittleEndian.Uint32(frame[4:]))
	if frame[3] != 2 || got != customNPCTalk {
		t.Fatalf("bubble custom ID=%d frame=%x", got, frame)
	}
}

func TestEncodeDramaPromptUsesRegisteredCustomRoute(t *testing.T) {
	frame, err := encodeDramaPrompt(gmDramaProbe())
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x1E || binary.LittleEndian.Uint16(frame[1:]) != 9 {
		t.Fatalf("drama header=%x", frame)
	}
	got := int32(binary.LittleEndian.Uint32(frame[4:]))
	if frame[3] != 2 || got != customBeginDrama {
		t.Fatalf("drama custom ID=%d frame=%x", got, frame)
	}
}

func TestEncodeDramaPromptRejectsInvalidRatings(t *testing.T) {
	prompt := gmDramaProbe()
	prompt.Explore = 6
	if _, err := encodeDramaPrompt(prompt); err == nil {
		t.Fatal("expected invalid drama rating to be rejected")
	}
}
