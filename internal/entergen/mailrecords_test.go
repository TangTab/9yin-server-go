package entergen

// 邮箱记录行单测（对照 2433 mail_records_test.go / Java MailFlowTest 帧锚）：
// RecvLetterRec 官方下标、16 列宽窄串/int64 列型、空表零帧、附件 XML 形态。

import (
	"encoding/binary"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func sampleLetter() rolestate.Letter {
	return rolestate.Letter{
		Serial:   "gm1700000000000-1",
		Type:     LetterSystemPost,
		Sender:   "GM",
		Title:    "GM 测试邮件",
		Content:  "欢迎来到九阴",
		Appendix: AppendixXML("tool_an_0001", 3),
		DateUnix: 1700000000,
	}
}

// TestLetterRowFrameShape 0x11 帧头 + recordIndex=RecvLetterRec + 宽串标题
// 在帧内（UTF-16LE）+ 16 列行体。
func TestLetterRowFrameShape(t *testing.T) {
	letter := sampleLetter()
	frame, err := LetterRowFrame(1, 1, letter)
	if err != nil {
		t.Fatalf("letter row frame: %v", err)
	}
	if frame[0] != 0x11 {
		t.Fatalf("op = %#x, want 0x11", frame[0])
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != RecordRecvLetter {
		t.Fatalf("recordIndex = %d, want %d (RecvLetterRec)", got, RecordRecvLetter)
	}
	if RecordRecvLetter != 319 { // 2433 线 mail_records.go 实测锚——官方表换代即警觉
		t.Fatalf("RecvLetterRec index = %d, official dialect anchor is 319", RecordRecvLetter)
	}
	if binary.LittleEndian.Uint16(frame[14:]) != 1 {
		t.Fatalf("rowCount != 1")
	}
	if !containsUTF16(frame, letter.Title) {
		t.Fatal("wide-string title not found in frame")
	}
	if !containsUTF16(frame, letter.Content) {
		t.Fatal("wide-string content not found in frame")
	}
	if !containsASCII(frame, letter.Appendix) {
		t.Fatal("appendix xml not found in frame")
	}
}

// TestLetterRowFramesEmptyTable 空邮箱零帧（登录/重入不下发）。
func TestLetterRowFramesEmptyTable(t *testing.T) {
	frames, err := LetterRowFrames(1, 1, nil)
	if err != nil || len(frames) != 0 {
		t.Fatalf("empty table: frames=%d err=%v, want 0/nil", len(frames), err)
	}
}

// TestLetterRowFramesOrder 行序=下发序。
func TestLetterRowFramesOrder(t *testing.T) {
	letters := []rolestate.Letter{sampleLetter(), {Serial: "gm2", Title: "B", Type: LetterSystemPost}}
	frames, err := LetterRowFrames(1, 1, letters)
	if err != nil {
		t.Fatalf("frames: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	if !containsUTF16(frames[0], letters[0].Title) || !containsUTF16(frames[1], letters[1].Title) {
		t.Fatal("row order mismatch")
	}
}

// TestAppendixXMLFormat 客户端 ParseXmlData 口径单物件形态。
func TestAppendixXMLFormat(t *testing.T) {
	got := AppendixXML("tool_an_0001", 3)
	want := `<Root><Object Config="tool_an_0001"><Prop Amount="3" ColorLevel="0"/></Object></Root>`
	if got != want {
		t.Fatalf("appendix xml = %q, want %q", got, want)
	}
}

// TestLetterDelRowFrame 0x12 帧（14B，末 u16=行号）。
func TestLetterDelRowFrame(t *testing.T) {
	frame := LetterDelRowFrame(1, 1, 2)
	if frame[0] != 0x12 {
		t.Fatalf("op = %#x, want 0x12", frame[0])
	}
	if got := binary.LittleEndian.Uint16(frame[12:]); got != 2 {
		t.Fatalf("row = %d, want 2", got)
	}
	if binary.LittleEndian.Uint16(frame[10:]) != RecordRecvLetter {
		t.Fatalf("recordIndex mismatch")
	}
}

func containsUTF16(frame []byte, text string) bool {
	// 窄串变长列在前，宽串可奇数起始——逐字节扫描（非 2 对齐步进）。
	units := []rune(text)
	if len(units) == 0 {
		return false
	}
	for off := 0; off+2*len(units) <= len(frame); off++ {
		match := true
		for i, r := range units {
			if binary.LittleEndian.Uint16(frame[off+i*2:]) != uint16(r) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func containsASCII(frame []byte, text string) bool {
	return len(text) > 0 && indexOf(frame, text) >= 0
}

func indexOf(haystack []byte, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
