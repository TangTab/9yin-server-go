package main

import (
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"
)

// 邮件域（RecvLetterRec 319）：十六列行编码（宽串 col0/3/4 UTF-16LE）
// + 快照 JSON（snake_case）解析形态钉死。

func TestLetterRowFrameEncodesWideColumns(t *testing.T) {
	letter := letterEntry{
		Serial: "gm1793892717215-1", Type: 101,
		Sender: "系统", SenderUID: "gm",
		Title: "系统邮件", Content: "正文内容",
		Gold: "0", Silver: "0",
		AppendixXML:     "<Root/>",
		DateUnixSeconds: 1793892717,
		ReadFlag:        true,
	}
	frame, err := letterRowFrame(letter)
	if err != nil {
		t.Fatalf("row frame: %v", err)
	}
	if frame[0] != 0x11 || binary.LittleEndian.Uint16(frame[10:]) != 319 {
		t.Fatalf("header=%x idx=%d (want 0x11/319)", frame[0], binary.LittleEndian.Uint16(frame[10:]))
	}
	// col0 发件人宽串：UTF-16LE 字节 + u32 长度（字节数+2）。
	senderLE := make([]byte, 0, 8)
	for _, unit := range utf16.Encode([]rune(letter.Sender)) {
		senderLE = binary.LittleEndian.AppendUint16(senderLE, unit)
	}
	if !strings.Contains(string(frame), string(senderLE)) {
		t.Fatalf("frame missing wide sender %q: %x", letter.Sender, frame)
	}
	// 窄串列照常（col10 序列号）。
	if !strings.Contains(string(frame), letter.Serial) {
		t.Fatalf("frame missing serial %q", letter.Serial)
	}
	// 空标题/空正文宽串列不报错（缺省空值路径）。
	if _, err := letterRowFrame(letterEntry{Serial: "s", Type: 1}); err != nil {
		t.Fatalf("sparse letter row: %v", err)
	}
}

func TestLetterSnapshotJSONDialect(t *testing.T) {
	// role_letters.snapshot 实际形态（role 618 dump）：{"letters":[{snake_case}]}。
	raw := `{"letters": [{"gold": "0", "type": 101, "title": "系统邮件", "sender": "", ` +
		`"serial": "gm1793892717215-1", "silver": "0", "content": "", "read_flag": true, ` +
		`"sender_uid": "gm", "appendix_xml": "", "date_unix_seconds": 1793892717}]}`
	var snapshot struct {
		Letters []letterEntry `json:"letters"`
	}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(snapshot.Letters) != 1 {
		t.Fatalf("letters=%d", len(snapshot.Letters))
	}
	letter := snapshot.Letters[0]
	if letter.Serial != "gm1793892717215-1" || letter.Type != 101 ||
		letter.SenderUID != "gm" || !letter.ReadFlag || letter.DateUnixSeconds != 1793892717 {
		t.Fatalf("letter=%+v", letter)
	}
}

func TestPushLetterFramesEmptyMailboxZeroFrames(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := pushLetterFrames(conn, nil); err != nil {
		t.Fatalf("push: %v", err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("frames=%d, want 0 (空邮箱零帧)", len(frames))
	}
	if err := pushLetterFrames(conn, []letterEntry{{Serial: "s1"}, {Serial: "s2"}}); err != nil {
		t.Fatalf("push rows: %v", err)
	}
	if frames := conn.Frames(); len(frames) != 2 {
		t.Fatalf("frames=%d, want 2", len(frames))
	}
}
