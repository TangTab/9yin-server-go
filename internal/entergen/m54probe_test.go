package entergen

import "testing"

// 官方女号抓包 m54 帧（0x0A 活动信封原样字节）。
func TestParseActivityCustomMessageM54Official(t *testing.T) {
	frame := []byte{
		0x0a, 0xdb, 0x6e, 0x61, 0x19, 0x8f, 0xf1, 0xee, 0xb9, 0xcf, 0x21, 0x03, 0x53, 0x89, 0x46, 0x7d,
		0x15, 0x3a, 0x01, 0x00, 0x00, 0x02, 0x00, 0x02, 0x36, 0x00, 0x00, 0x00, 0x03, 0xca, 0x0e, 0x86,
		0x37, 0x01, 0x00, 0x00, 0x00,
	}
	msg, ok, err := ParseActivityCustomMessage(frame)
	if !ok || err != nil {
		t.Fatalf("官方 m54 帧未解析: ok=%v err=%v", ok, err)
	}
	if len(msg.Values) != 2 || msg.Values[0].Type != 2 || msg.Values[0].Int32 != 54 {
		t.Fatalf("首值应 int32 54: %+v", msg.Values)
	}
	if msg.Values[1].Type != 3 || msg.Values[1].Int64 != 5226499786 {
		t.Fatalf("次值应 int64 5226499786: %+v", msg.Values[1])
	}
}
