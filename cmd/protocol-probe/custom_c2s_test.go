package main

import (
	"encoding/binary"
	"testing"
)

func TestParseClientCustomMessageShopBuy(t *testing.T) {
	frame := []byte{0x1E, 5, 0}
	appendInt := func(value int32) {
		frame = append(frame, 2)
		frame = binary.LittleEndian.AppendUint32(frame, uint32(value))
	}
	appendText := func(value string) {
		frame = append(frame, 6)
		frame = binary.LittleEndian.AppendUint32(frame, uint32(len(value)+1))
		frame = append(frame, value...)
		frame = append(frame, 0)
	}
	appendInt(70)
	appendText("Shop_yaopin_00100")
	appendInt(1)
	appendInt(2)
	appendInt(3)

	got, err := parseClientCustomMessage(frame)
	if err != nil {
		t.Fatal(err)
	}
	if got.Opcode != 0x1E || len(got.Values) != 5 || got.Values[0].Int32 != 70 {
		t.Fatalf("decoded header=%+v", got)
	}
	if got.Values[1].Type != 6 || got.Values[1].Text != "Shop_yaopin_00100" {
		t.Fatalf("shop id=%+v", got.Values[1])
	}
	if got.Values[2].Int32 != 1 || got.Values[3].Int32 != 2 || got.Values[4].Int32 != 3 {
		t.Fatalf("buy args=%+v", got.Values)
	}
}

func TestParseClientCustomMessageRejectsTrailingBytes(t *testing.T) {
	frame := []byte{0x1E, 1, 0, 2, 31, 0, 0, 0, 0}
	if _, err := parseClientCustomMessage(frame); err == nil {
		t.Fatal("expected trailing-byte error")
	}
}

func TestParseClientActivityCustomMessageQingGong(t *testing.T) {
	// Captured current-client shape: 0x0A + opaque 20-byte activity header +
	// TVarList(216, "qinggong_8", 1).
	frame := make([]byte, 21)
	frame[0] = 0x0A
	frame = binary.LittleEndian.AppendUint16(frame, 3)
	frame = append(frame, 2)
	frame = binary.LittleEndian.AppendUint32(frame, 216)
	frame = append(frame, 6)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len("qinggong_8")+1))
	frame = append(frame, "qinggong_8"...)
	frame = append(frame, 0)
	frame = append(frame, 2)
	frame = binary.LittleEndian.AppendUint32(frame, 1)

	got, ok, err := parseClientActivityCustomMessage(frame)
	if err != nil || !ok {
		t.Fatalf("decode activity custom ok=%t err=%v", ok, err)
	}
	if got.Opcode != 0x0A || len(got.Values) != 3 || got.Values[0].Int32 != 216 || got.Values[1].Text != "qinggong_8" || got.Values[2].Int32 != 1 {
		t.Fatalf("decoded qinggong=%+v", got)
	}
}
