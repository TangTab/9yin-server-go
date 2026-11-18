package proto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// 旧分支 heartbeat_test 的 golden（fork 49840 抓包 326 对逐字节钉死）。
func TestHeartbeatGolden(t *testing.T) {
	cases := []struct{ ping, want string }{
		{"183154470000140000000000", "383154470000140000000000"},
		{"18f15b470000150000000000", "38f15b470000150000000000"},
	}
	for _, c := range cases {
		ping, _ := hex.DecodeString(c.ping)
		want, _ := hex.DecodeString(c.want)
		if got := HeartbeatReply(ping); !bytes.Equal(got, want) {
			t.Fatalf("HeartbeatReply(%x)=%x, want %x", ping, got, want)
		}
	}
}

func TestVarListGolden(t *testing.T) {
	// 单 int32 值 5：[u16 1][tag 2][04字节 LE]
	got := (&VarList{}).AddInt32(5).Encode()
	if !bytes.Equal(got, []byte{0x01, 0x00, 0x02, 0x05, 0x00, 0x00, 0x00}) {
		t.Fatalf("int32 编码错误: %x", got)
	}
	// 窄字符串 "AB"：[u16 1][tag 6][u32 len=3 含 NUL]["AB" 00]
	got = (&VarList{}).AddString("AB").Encode()
	if !bytes.Equal(got, []byte{0x01, 0x00, 0x06, 0x03, 0x00, 0x00, 0x00, 'A', 'B', 0x00}) {
		t.Fatalf("string 编码错误: %x", got)
	}
	// 宽字符串 "A"：[u16 1][tag 7][u32 (1+1)*2=4][41 00][00 00]
	got = (&VarList{}).AddWide("A").Encode()
	if !bytes.Equal(got, []byte{0x01, 0x00, 0x07, 0x04, 0x00, 0x00, 0x00, 0x41, 0x00, 0x00, 0x00}) {
		t.Fatalf("widestr 编码错误: %x", got)
	}
}

func TestVarListRoundTripAllTypes(t *testing.T) {
	obj := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	v := (&VarList{}).
		AddInt32(-42).
		AddInt64(1 << 40).
		AddFloat32(3.5).
		AddDouble(-2.25).
		AddString("hello").
		AddWide("你好").
		AddObject(obj)

	got, err := DecodeVarList(v.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Values) != 7 {
		t.Fatalf("数量错误: %d", len(got.Values))
	}
	check := func(i int, cond bool, what string) {
		t.Helper()
		if !cond {
			t.Fatalf("第 %d 个值（%s）不匹配: %+v", i, what, got.Values[i])
		}
	}
	check(0, got.Values[0].I64 == -42, "int32")
	check(1, got.Values[1].I64 == 1<<40, "int64")
	check(2, got.Values[2].F64 == 3.5, "float32")
	check(3, got.Values[3].F64 == -2.25, "double")
	check(4, got.Values[4].Text == "hello", "string")
	check(5, got.Values[5].Text == "你好", "widestr")
	check(6, got.Values[6].Raw == obj, "object")
}

func TestVarListRejectsTrailingBytes(t *testing.T) {
	enc := (&VarList{}).AddInt32(1).Encode()
	if _, err := DecodeVarList(append(enc, 0xAA)); err == nil {
		t.Fatal("尾部多余字节应报错")
	}
}

func TestCustomFrameParse(t *testing.T) {
	// 0x1E + TVarList[int32 30(移动物品), int32 视图, int32 位置, int32 目标视图, int32 目标位置]
	body := (&VarList{}).AddInt32(30).AddInt32(1).AddInt32(5).AddInt32(2).AddInt32(8)
	frame := append([]byte{MsgCustomDirect}, body.Encode()...)

	op, vl, err := ParseCustomFrame(frame)
	if err != nil || op != MsgCustomDirect {
		t.Fatalf("解析失败: %v %d", err, op)
	}
	if vl.Values[0].I64 != 30 {
		t.Fatalf("消息 ID 应为 30: %+v", vl.Values[0])
	}
	if def, ok := C2SCustomByID[30]; !ok || def.Name != "CLIENT_CUSTOMMSG_MOVEITEM" {
		t.Fatalf("目录查询失败: %+v %v", def, ok)
	}
}

func TestGeneratedCatalog(t *testing.T) {
	if len(C2SCustomByID) < 200 {
		t.Fatalf("C2S 目录过小: %d", len(C2SCustomByID))
	}
	if len(S2CCustomByKey) < 100 {
		t.Fatalf("S2C 目录过小: %d", len(S2CCustomByKey))
	}
	if id, ok := C2SCustomByName["CLIENT_CUSTOMMSG_GM"]; !ok || id != 2 {
		t.Fatalf("GM 消息应为 2: %d %v", id, ok)
	}
	if def, ok := S2CCustomByKey["sysinfo"]; !ok || def.Name != "SERVER_CUSTOMMSG_SYSINFO" {
		t.Fatalf("sysinfo 缺失: %+v", def)
	}
}
