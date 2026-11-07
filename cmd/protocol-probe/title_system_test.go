package main

import (
	"encoding/binary"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// 称号（msg295）：fork title_system.go:189-202 反汇编——无所有权校验、
// 范围闸 0..500000、TitleID@360 单属性帧（官方线型 i32）。

func titleMsg(id int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomSetTitle},
		{Type: 2, Int32: id},
	}}
}

func TestSetTitleWritesTitleIDFrameAndPersists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectExec("UPDATE role_titles SET is_worn = 0").WithArgs(901).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO role_titles").WithArgs(901, 12345).WillReturnResult(sqlmock.NewResult(1, 1))

	player := newPlayerActor("称号侠", 0)
	conn := &captureMessageConnection{}
	handled, err := handleSetTitleCustom(conn, player, titleMsg(12345), "test", newTitleDBStore(db), 901)
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.titleIDValue(); got != 12345 {
		t.Fatalf("titleID=%d", got)
	}
	frames := conn.Frames()
	if len(frames) != 1 {
		t.Fatalf("frames=%d", len(frames))
	}
	frame := frames[0]
	if frame[0] != 0x10 {
		t.Fatalf("opcode=%#x", frame[0])
	}
	if got := binary.LittleEndian.Uint16(frame[12:]); got != 360 {
		t.Fatalf("property idx=%d, want 360 (TitleID)", got)
	}
	if got := binary.LittleEndian.Uint32(frame[14:]); got != 12345 {
		t.Fatalf("title value=%d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestSetTitleCancelClearsWornOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectExec("UPDATE role_titles SET is_worn = 0").WithArgs(901).WillReturnResult(sqlmock.NewResult(0, 0))

	player := newPlayerActor("称号侠", 0)
	player.setTitleIDValue(7)
	conn := &captureMessageConnection{}
	handled, err := handleSetTitleCustom(conn, player, titleMsg(0), "test", newTitleDBStore(db), 901)
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.titleIDValue(); got != 0 {
		t.Fatalf("titleID=%d, want 0 (cancelled)", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestSetTitleRejectsOutOfRangeAndMalformed(t *testing.T) {
	player := newPlayerActor("称号侠", 0)
	conn := &captureMessageConnection{}
	for _, id := range []int32{-1, 500001} {
		if handled, err := handleSetTitleCustom(conn, player, titleMsg(id), "test", nil, 901); err != nil || !handled {
			t.Fatalf("id=%d handled=%t err=%v", id, handled, err)
		}
	}
	// 缺参。
	if handled, err := handleSetTitleCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomSetTitle}}}, "test", nil, 901); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.titleIDValue(); got != 0 {
		t.Fatalf("titleID moved on reject: %d", got)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("rejects emitted %d frames", len(frames))
	}
}

func TestTitleRecRowEncodesOfficialSchema(t *testing.T) {
	frame, err := titleRecAddRow(42, true)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if frame[0] != 0x11 {
		t.Fatalf("opcode=%#x", frame[0])
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != 98 {
		t.Fatalf("record index=%d, want 98 (title_rec)", got)
	}
	// 官方列负载：u32(4)+u8(1)+i64(8)=13 字节。
	if len(frame) != 16+13 {
		t.Fatalf("frame len=%d, want 29", len(frame))
	}
	if got := binary.LittleEndian.Uint32(frame[16:]); got != 42 {
		t.Fatalf("title cell=%d", got)
	}
	if frame[16+4] != 1 {
		t.Fatalf("worn byte=%d, want 1", frame[16+4])
	}
}

func TestPushTitleFramesReplaysOwnedRowsAndWornTitle(t *testing.T) {
	player := newPlayerActor("称号侠", 0)
	player.restoreTitles([]titleRow{
		{titleID: 5, titleType: 0, worn: false},
		{titleID: 7, titleType: 0, worn: true},
	})
	conn := &captureMessageConnection{}
	if err := pushTitleFrames(conn, player); err != nil {
		t.Fatalf("push: %v", err)
	}
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("frames=%d, want 2 rec rows + 1 title frame", len(frames))
	}
	for index := 0; index < 2; index++ {
		if frames[index][0] != 0x11 {
			t.Fatalf("record frame %d=%#x", index, frames[index][0])
		}
	}
	if frames[2][0] != 0x10 || binary.LittleEndian.Uint16(frames[2][12:]) != 360 {
		t.Fatalf("worn title frame=%x", frames[2])
	}
}
