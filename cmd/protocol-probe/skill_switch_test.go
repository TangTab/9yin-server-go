package main

import (
	"encoding/binary"
	"testing"
)

func TestSkillActionSwitchFrameUsesModernNativeRoute(t *testing.T) {
	frame, err := skillActionSwitchFrame()
	if err != nil {
		t.Fatalf("skillActionSwitchFrame: %v", err)
	}
	if len(frame) != 29 {
		t.Fatalf("frame length=%d want=29 bytes=%x", len(frame), frame)
	}
	if frame[0] != 0x27 {
		t.Fatalf("outer opcode=%#x want=0x27", frame[0])
	}
	if got := binary.LittleEndian.Uint16(frame[1:3]); got != 4 {
		t.Fatalf("argument count=%d want=4", got)
	}
	offset := 3
	if frame[offset] != 2 || int32(binary.LittleEndian.Uint32(frame[offset+1:])) != serverSwitchControlMessage {
		t.Fatalf("message id argument=%x", frame[offset:offset+5])
	}
	offset += 5
	if frame[offset] != 6 || binary.LittleEndian.Uint32(frame[offset+1:]) != 6 ||
		string(frame[offset+5:offset+11]) != "reset\x00" {
		t.Fatalf("command argument=%x", frame[offset:offset+11])
	}
	offset += 11
	if frame[offset] != 2 || int32(binary.LittleEndian.Uint32(frame[offset+1:])) != skillActionInputSwitch {
		t.Fatalf("switch id argument=%x", frame[offset:offset+5])
	}
	offset += 5
	if frame[offset] != 2 || binary.LittleEndian.Uint32(frame[offset+1:]) != 1 {
		t.Fatalf("switch value argument=%x", frame[offset:offset+5])
	}
}

func TestModernCustomMessageRejectsUnknownOuterOpcode(t *testing.T) {
	if _, err := serverCustomIntMessageWithOpcode(0x99, serverSwitchControlMessage); err == nil {
		t.Fatal("expected unsupported outer opcode error")
	}
}
