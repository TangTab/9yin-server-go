package fssolorobot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strings"
	"testing"
	"time"
)

// ---------- QuickLZ ----------

// qlzMatchHashABC = qlzHash(3 字节 "abc") = 0x457
// （((v>>12)^v)&0xFFF，v=0x636261）。
const qlzMatchHashABC = 0x457

// 存储形态（flags&1==0）：9 字节头 + 原样数据。
func TestQLZStoredFullHeader(t *testing.T) {
	raw := []byte("hello quicklz stored payload")
	src := make([]byte, 9+len(raw))
	src[0] = 0x02 // flags：9B 头、未压缩
	binary.LittleEndian.PutUint32(src[1:5], uint32(len(src)))
	binary.LittleEndian.PutUint32(src[5:9], uint32(len(raw)))
	copy(src[9:], raw)

	if n, _ := QLZSizeDecompressed(src); n != len(raw) {
		t.Fatalf("dsize = %d, want %d", n, len(raw))
	}
	dst := make([]byte, len(raw))
	if n, err := QLZDecompress(src, dst); err != nil || n != len(raw) || !bytes.Equal(dst, raw) {
		t.Fatalf("stored decompress: n=%d err=%v dst=%q", n, err, dst)
	}
}

// 存储形态（3 字节头）。
func TestQLZStoredShortHeader(t *testing.T) {
	raw := []byte("short!")
	src := []byte{0x00, 16, 6}
	src = append(src, raw...) // csize=16 覆全流，dsize=6
	dst := make([]byte, 6)
	if n, err := QLZDecompress(src, dst); err != nil || n != 6 || !bytes.Equal(dst, raw) {
		t.Fatalf("short-header stored: n=%d err=%v dst=%q", n, err, dst)
	}
	if n, _ := QLZSizeDecompressed(src); n != 6 {
		t.Fatalf("short-header dsize = %d", n)
	}
}

// qlzCompressed 组装官方 level-2 压缩流（9B 头 + 标签 + 数据），
// 供手工向量用。
func qlzCompressed(dsize int, tags uint32, core []byte) []byte {
	src := make([]byte, 9, 9+len(core))
	src[0] = 0x03 // flags：压缩 + 9B 头
	binary.LittleEndian.PutUint32(src[1:5], 0)
	binary.LittleEndian.PutUint32(src[5:9], uint32(dsize))
	src = append(src, byte(tags), byte(tags>>8), byte(tags>>16), byte(tags>>24))
	src = append(src, core...)
	return src
}

// 纯字面量小流（dsize ≤ 11 → 全程尾道）：标签 0，后跟字面量。
func TestQLZLiteralsTailPath(t *testing.T) {
	want := "ABC"
	src := qlzCompressed(3, 0, []byte("ABC"))
	dst := make([]byte, 3)
	if n, err := QLZDecompress(src, dst); err != nil || n != 3 || string(dst) != want {
		t.Fatalf("tail literals: n=%d err=%v dst=%q", n, err, dst)
	}
}

// 字面量快速道 + 尾道混合："0123456789ABCDEFGHIJ"（20 字节，dsize>11
// 时前段走快速道批拷）。
func TestQLZLiteralsFastPath(t *testing.T) {
	want := "0123456789ABCDEFGHIJ"
	// 标签 20 个 0 位（字面量）：0x00000000 + 后续补 0；
	src := qlzCompressed(len(want), 0, []byte(want))
	dst := make([]byte, len(want))
	if n, err := QLZDecompress(src, dst); err != nil || string(dst) != want {
		t.Fatalf("fast literals: n=%d err=%v dst=%q", n, err, dst)
	}
}

// 匹配向量："abc"×6（18 字节）。标签字位序（LSB 先）：000(abc)——
// 消耗后 cword==1 触发续读下一标签字，其 b0=1 即匹配位：
// 流 = [tag0=0x08]["abc"][tag1=0x01][control(2B)]。
func TestQLZMatch(t *testing.T) {
	const dsize = 18
	want := strings.Repeat("abc", 6)
	// control u16 = hash<<4 | (matchlen-2)；matchlen 15。
	ctrl := uint16(qlzMatchHashABC<<4 | (dsize - 3 - 2))
	core := append([]byte("abc"), 1, 0, 0, 0) // tag1：b0=匹配位
	core = append(core, byte(ctrl), byte(ctrl>>8))
	src := qlzCompressed(dsize, 0x08, core)
	dst := make([]byte, dsize)
	if n, err := QLZDecompress(src, dst); err != nil || string(dst) != want {
		t.Fatalf("match: n=%d err=%v dst=%q", n, err, dst)
	}
}

// 匹配 + 尾道字面量："abc"*5 + "XYZW"（19 字节）：同上布局，
// 匹配位在续读标签字 tag1 的 b0。
func TestQLZMatchThenTailLiterals(t *testing.T) {
	want := strings.Repeat("abc", 5) + "XYZW"
	ctrl := uint16(qlzMatchHashABC<<4 | 10) // matchlen = 12
	core := append([]byte("abc"), 1, 0, 0, 0)
	core = append(core, byte(ctrl), byte(ctrl>>8))
	core = append(core, "XYZW"...)
	src := qlzCompressed(19, 0x08, core)
	dst := make([]byte, 19)
	if n, err := QLZDecompress(src, dst); err != nil || string(dst) != want {
		t.Fatalf("match+tail: n=%d err=%v dst=%q", n, err, dst)
	}
}

// dsize=0 官方不读流字节即返回。
func TestQLZEmpty(t *testing.T) {
	src := []byte{0x03, 0, 0, 0, 0, 0, 0, 0, 0}
	if n, err := QLZDecompress(src, nil); err != nil || n != 0 {
		t.Fatalf("empty: n=%d err=%v", n, err)
	}
}

// ---------- 帧法 ----------

func TestEscapeRoundTrip(t *testing.T) {
	for _, payload := range [][]byte{
		{0x02},
		{0xEE},
		{0xEE, 0x00},
		{0xEE, 0xEE},
		bytes.Repeat([]byte{0xEE}, 100),
	} {
		wire := escapeFrame(payload)
		var c connect
		var got [][]byte
		c.feed(wire, func(p []byte) { got = append(got, p) })
		if len(got) != 1 || !bytes.Equal(got[0], payload) {
			t.Fatalf("round trip % X → %d 帧", payload, len(got))
		}
	}
}

// 官方宽容语义：EE 后跟非 00/EE 字节 → 两字节按字面量穿过。
func TestEscapeTolerantPassthrough(t *testing.T) {
	var c connect
	var got [][]byte
	c.feed(escapeFrame([]byte{0xEE, 0x05}), func(p []byte) { got = append(got, p) })
	// 手工构造 EE 05（未转义）+ 终止符。
	wire := []byte{0xEE, 0x05, 0xEE, 0xEE}
	got = nil
	c = connect{}
	c.feed(wire, func(p []byte) { got = append(got, p) })
	if len(got) != 1 || !bytes.Equal(got[0], []byte{0xEE, 0x05}) {
		t.Fatalf("tolerant passthrough: %v", got)
	}
}

// 空帧（仅 EE EE）：官方 len-1==0 不分发。
func TestEscapeEmptyFrameNotDispatched(t *testing.T) {
	var c connect
	dispatched := 0
	c.feed([]byte{0xEE, 0xEE, 0xEE, 0xEE}, func([]byte) { dispatched++ })
	if dispatched != 0 {
		t.Fatalf("空帧不应分发，got %d", dispatched)
	}
}

// 分片到达：转义帧任意切分喂入。
func TestEscapeFragmented(t *testing.T) {
	payload := []byte{0x04, 0xEE, 0x01, 0x02, 0xEE}
	wire := escapeFrame(payload)
	var c connect
	var got [][]byte
	for _, b := range wire {
		c.feed([]byte{b}, func(p []byte) { got = append(got, p) })
	}
	if len(got) != 1 || !bytes.Equal(got[0], payload) {
		t.Fatalf("fragmented: %v", got)
	}
}

// ---------- 报文构造 golden ----------

func TestBuildLoginGolden(t *testing.T) {
	buf := buildLogin("robot0", "robot0", "")
	if buf[0] != msgSendLogin {
		t.Fatalf("id = %#x", buf[0])
	}
	if v := binary.LittleEndian.Uint32(buf[1:5]); v != 0x300 {
		t.Fatalf("version = %#x", v)
	}
	// [5..0x15) 为 16 字节零（官方 memset 0x15 定长头）
	for i, b := range buf[5:0x15] {
		if b != 0 {
			t.Fatalf("固定头区 buf[%d] = %#x", 5+i, b)
		}
	}
	// 账号字段起于 0x15：u32 对齐长 16 + "robot0\0" 零垫
	if pad := binary.LittleEndian.Uint32(buf[0x15:0x19]); pad != 16 {
		t.Fatalf("account pad = %d", pad)
	}
	if s := buf[0x19:0x29]; !bytes.HasPrefix(s, []byte("robot0\x00")) {
		t.Fatalf("account field = %q", s)
	}
	if pad := binary.LittleEndian.Uint32(buf[0x29:0x2D]); pad != 16 {
		t.Fatalf("password pad = %d", pad)
	}
	if pad := binary.LittleEndian.Uint32(buf[0x3D:0x41]); pad != 16 {
		t.Fatalf("loginstr pad = %d", pad)
	}
	if len(buf) != 0x15+60 {
		t.Fatalf("len = %#x", len(buf))
	}
}

func TestBuildLoginLongAccountAlignment(t *testing.T) {
	// 16 字符账号：len+1 = 17 → 对齐 32。
	buf := buildLogin(strings.Repeat("a", 16), "p", "")
	if pad := binary.LittleEndian.Uint32(buf[0x15:0x19]); pad != 32 {
		t.Fatalf("pad = %d", pad)
	}
	if pad := binary.LittleEndian.Uint32(buf[0x15+4+32 : 0x15+8+32]); pad != 16 {
		t.Fatalf("password pad = %d", pad)
	}
}

func ucs2Of(s string) []byte {
	var b []byte
	for _, r := range s {
		if r > 0xFFFF {
			r = '?'
		}
		b = append(b, byte(r), byte(r>>8))
	}
	return b
}

func TestBuildChooseCharGolden(t *testing.T) {
	buf := buildChooseChar("robert")
	if len(buf) != 0x54 || buf[0] != msgSendChooseChar {
		t.Fatalf("len=%#x id=%#x", len(buf), buf[0])
	}
	want := append(ucs2Of("robert"), 0, 0)
	if !bytes.Equal(buf[1:1+len(want)], want) {
		t.Fatalf("name = % X", buf[1:15])
	}
}

func TestBuildChooseCharTruncation(t *testing.T) {
	long := strings.Repeat("字", 40) // 40 字符 > 32：截 32 + 零字
	buf := buildChooseChar(long)
	if len(buf) != 0x54 {
		t.Fatalf("len = %#x", len(buf))
	}
	// 第 33 个 wchar 位置（+0x40 相对名字起点 +1）必须为零终结。
	if w := binary.LittleEndian.Uint16(buf[1+0x40 : 1+0x42]); w != 0 {
		t.Fatalf("terminator = %#x", w)
	}
}

func TestBuildCreateCharGolden(t *testing.T) {
	buf := buildCreateChar("robot0")
	if len(buf) != 0x49 || buf[0] != msgSendCreateChar {
		t.Fatalf("len=%#x id=%#x", len(buf), buf[0])
	}
	if v := binary.LittleEndian.Uint32(buf[1:5]); v != 0 {
		t.Fatalf("field1 = %#x", v)
	}
	if w := binary.LittleEndian.Uint16(buf[5:7]); w != 0 {
		t.Fatalf("field2 = %#x", w)
	}
	want := append(ucs2Of("robot0"), 0, 0)
	if !bytes.Equal(buf[7:7+len(want)], want) {
		t.Fatalf("name = % X", buf[7:21])
	}
}

func TestBuildMoveGolden(t *testing.T) {
	buf := buildMove(1.5, 2.5, 3.5, -1.5, -2.5)
	if len(buf) != 0x60 || buf[0] != msgSendMove || buf[1] != 1 {
		t.Fatalf("len=%#x id=%#x b1=%#x", len(buf), buf[0], buf[1])
	}
	if w := binary.LittleEndian.Uint16(buf[2:4]); w != 6 {
		t.Fatalf("word = %d", w)
	}
	for i, want := range []float32{1.5, 2.5, 3.5, -1.5, -2.5, 4.0} {
		got := math.Float32frombits(binary.LittleEndian.Uint32(buf[4+4*i : 8+4*i]))
		if got != want {
			t.Fatalf("f[%d] = %v want %v", i, got, want)
		}
	}
}

// ---------- ReceiveServerMsg 分发 ----------

type testRig struct {
	r    *Robot
	c    *connect
	sent [][]byte
}

func newTestRig() *testRig {
	rig := &testRig{}
	rig.r = NewRobot(Config{
		AccountPrefix: "robot", PasswordPrefix: "robot",
		EchoAddr: "off",
	}, nil, nil, nil)
	rig.c = &connect{index: 0, nc: &captureConn{sink: &rig.sent}}
	rig.r.connects[0] = rig.c
	rig.r.nextConnID = 1
	return rig
}

func (rig *testRig) recv(payload []byte) bool {
	return rig.r.receiveServerMsgLocked(0, rig.c, payload)
}

func (rig *testRig) lastSent() []byte {
	if len(rig.sent) == 0 {
		return nil
	}
	// captureConn 抓到的是转义后的线路帧；解回 payload（同款状态机）。
	var last byte
	frames := gwFrames(rig.sent[len(rig.sent)-1], &last, &[]byte{})
	if len(frames) == 0 {
		return nil
	}
	return frames[0]
}

func TestRecvErrorCode(t *testing.T) {
	rig := newTestRig()
	p := make([]byte, 5)
	p[0] = msgRecvErrorCode
	binary.LittleEndian.PutUint32(p[1:5], 42)
	if !rig.recv(p) {
		t.Fatal("dispatch failed")
	}
	if rig.c.state != ConnFailed {
		t.Fatalf("state = %v", rig.c.state)
	}
	// 过短
	if rig.recv([]byte{msgRecvErrorCode, 1, 2, 3}) {
		t.Fatal("short error_code should fail")
	}
}

func TestRecvLoginSucceedChoose(t *testing.T) {
	rig := newTestRig()
	p := make([]byte, 0x50)
	p[0] = msgRecvLoginSucc
	binary.LittleEndian.PutUint32(p[0x25:0x29], 1) // 角色数 1
	// 角色条目 @0x29：word,byte,int,byte,int,byte,UCS2(name),byte,UCS2
	ar := 0x29
	binary.LittleEndian.PutUint16(p[ar:], 1)
	p[ar+2] = 2
	binary.LittleEndian.PutUint32(p[ar+3:], 3)
	p[ar+7] = 4
	binary.LittleEndian.PutUint32(p[ar+8:], 5)
	p[ar+12] = 6
	// UCS2 名字 "hero"：u32 字节长（含 \0）= 10
	binary.LittleEndian.PutUint32(p[ar+13:], 10)
	copy(p[ar+17:], ucs2Of("hero"))
	binary.LittleEndian.PutUint16(p[ar+25:], 0)
	if !rig.recv(p) {
		t.Fatal("dispatch failed")
	}
	if rig.c.state != ConnChoosing {
		t.Fatalf("state = %v", rig.c.state)
	}
	sent := rig.lastSent()
	if sent == nil || sent[0] != msgSendChooseChar || string(sent[1:9]) != string(ucs2Of("hero")) {
		t.Fatalf("choose msg = % X", sent)
	}
}

func TestRecvLoginSucceedCreate(t *testing.T) {
	rig := newTestRig()
	p := make([]byte, 0x29)
	p[0] = msgRecvLoginSucc // 角色数 @0x25 == 0
	if !rig.recv(p) {
		t.Fatal("dispatch failed")
	}
	if rig.c.state != ConnCreating {
		t.Fatalf("state = %v", rig.c.state)
	}
	sent := rig.lastSent()
	if sent == nil || sent[0] != msgSendCreateChar || len(sent) != 0x49 {
		t.Fatalf("create msg = % X", sent)
	}
	want := append(ucs2Of("robot0"), 0, 0)
	if !bytes.HasPrefix(sent[7:], want) {
		t.Fatalf("name = %q", sent[7:19])
	}
}

func TestRecvEntryScene(t *testing.T) {
	rig := newTestRig()
	p := make([]byte, 0x10)
	p[0] = msgRecvEntryScene
	binary.LittleEndian.PutUint32(p[5:9], 7)
	binary.LittleEndian.PutUint32(p[9:13], 8)
	if !rig.recv(p) {
		t.Fatal("dispatch failed")
	}
	if rig.c.ident1 != 7 || rig.c.ident2 != 8 {
		t.Fatalf("ident = %d/%d", rig.c.ident1, rig.c.ident2)
	}
	if rig.c.state != ConnInGame {
		t.Fatalf("state = %v", rig.c.state)
	}
	if sent := rig.lastSent(); sent == nil || sent[0] != msgSendEntryAck || len(sent) != 1 {
		t.Fatalf("ack = % X", sent)
	}
}

func identMsg(id byte, i1, i2 uint32, x, y, z float32, minLen int) []byte {
	p := make([]byte, minLen)
	p[0] = id
	binary.LittleEndian.PutUint32(p[1:5], i1)
	binary.LittleEndian.PutUint32(p[5:9], i2)
	binary.LittleEndian.PutUint32(p[9:13], math.Float32bits(x))
	binary.LittleEndian.PutUint32(p[13:17], math.Float32bits(y))
	binary.LittleEndian.PutUint32(p[17:21], math.Float32bits(z))
	return p
}

func TestRecvPositionFamily(t *testing.T) {
	rig := newTestRig()
	rig.c.ident1, rig.c.ident2 = 7, 8

	if !rig.recv(identMsg(msgRecvAddObject, 7, 8, 1, 2, 3, 0x3F)) {
		t.Fatal("add_object failed")
	}
	if rig.c.x != 1 || rig.c.y != 2 || rig.c.z != 3 {
		t.Fatalf("add_object pos = %v/%v/%v", rig.c.x, rig.c.y, rig.c.z)
	}
	// 他人对象不更新
	if !rig.recv(identMsg(msgRecvAddObject, 9, 9, 4, 5, 6, 0x3F)) {
		t.Fatal("add_object failed")
	}
	if rig.c.x != 1 {
		t.Fatalf("foreign add_object moved self: %v", rig.c.x)
	}

	rig.recv(identMsg(msgRecvLocation, 7, 8, 7, 8, 9, 0x19))
	if rig.c.x != 7 || rig.c.y != 8 || rig.c.z != 9 {
		t.Fatalf("location pos = %v/%v/%v", rig.c.x, rig.c.y, rig.c.z)
	}

	// moving 只更新 x/z（y 保持）
	rig.recv(identMsg(msgRecvMoving, 7, 8, 10, 99, 11, 0x2D))
	if rig.c.x != 10 || rig.c.z != 11 || rig.c.y != 8 {
		t.Fatalf("moving pos = %v/%v/%v", rig.c.x, rig.c.y, rig.c.z)
	}

	// 过短检查
	if rig.recv([]byte{msgRecvMoving, 0, 0, 0}) {
		t.Fatal("short moving should fail")
	}
}

func TestRecvAllDest(t *testing.T) {
	rig := newTestRig()
	rig.c.ident1, rig.c.ident2 = 7, 8
	entries := 2
	p := make([]byte, 3+entries*0x2C+2)
	p[0] = msgRecvAllDest
	binary.LittleEndian.PutUint16(p[1:3], uint16(entries))
	e1 := p[3 : 3+0x2C]
	binary.LittleEndian.PutUint32(e1[0:], 99)
	binary.LittleEndian.PutUint32(e1[4:], 99)
	e2 := p[3+0x2C : 3+2*0x2C]
	binary.LittleEndian.PutUint32(e2[0:], 7)
	binary.LittleEndian.PutUint32(e2[4:], 8)
	binary.LittleEndian.PutUint32(e2[8:], math.Float32bits(21.5))
	binary.LittleEndian.PutUint32(e2[0x10:], math.Float32bits(22.5))
	if !rig.recv(p) {
		t.Fatal("all_dest failed")
	}
	if rig.c.x != 21.5 || rig.c.z != 22.5 {
		t.Fatalf("all_dest pos = %v/%v", rig.c.x, rig.c.z)
	}
}

// 0x28 压缩包裹（存储形态）→ add_object 语义。
func TestRecvCompressedWrapperStored(t *testing.T) {
	rig := newTestRig()
	rig.c.ident1, rig.c.ident2 = 7, 8
	const dsize = 0x3E
	body := make([]byte, dsize) // 解压体（无 id 字节：ident1@0/ident2@4/x@8/y@0xC/z@0x10）
	binary.LittleEndian.PutUint32(body[0:4], 7)
	binary.LittleEndian.PutUint32(body[4:8], 8)
	binary.LittleEndian.PutUint32(body[8:12], math.Float32bits(5.5))
	binary.LittleEndian.PutUint32(body[12:16], math.Float32bits(6.5))
	binary.LittleEndian.PutUint32(body[16:20], math.Float32bits(7.5))
	src := make([]byte, 9+dsize)
	src[0] = 0x02 // 9B 头、存储
	binary.LittleEndian.PutUint32(src[5:9], dsize)
	copy(src[9:], body)
	payload := append([]byte{msgRecvCompressed}, src...)
	if !rig.recv(payload) {
		t.Fatal("compressed dispatch failed")
	}
	if rig.c.x != 5.5 || rig.c.y != 6.5 || rig.c.z != 7.5 {
		t.Fatalf("pos = %v/%v/%v", rig.c.x, rig.c.y, rig.c.z)
	}
}

// 未知 id 忽略且返回 true。
func TestRecvUnknownIgnored(t *testing.T) {
	rig := newTestRig()
	if !rig.recv([]byte{0x15, 1, 2, 3}) {
		t.Fatal("unknown id should be ignored (true)")
	}
}

// ---------- 移动循环（同步驱动） ----------

func TestMoveLoopSendsMove(t *testing.T) {
	rig := newTestRig()
	rig.c.state = ConnInGame
	rig.c.x, rig.c.y, rig.c.z = 100, 4, 200
	rig.r.cfg.MoveSeconds = 5
	// MoveSeconds 秒内只累计不发
	for i := 0; i < 5; i++ {
		rig.r.moveLoopLocked()
	}
	if n := len(rig.sent); n != 0 {
		t.Fatalf("提前发送 %d 条", n)
	}
	rig.r.moveLoopLocked() // 第 6 秒 → wait(6) > 5 → 移动
	if n := len(rig.sent); n != 1 {
		t.Fatalf("sent = %d", n)
	}
	m := rig.lastSent()
	if m[0] != msgSendMove {
		t.Fatalf("move id = %#x", m[0])
	}
	dx := math.Float32frombits(binary.LittleEndian.Uint32(m[16:20]))
	dz := math.Float32frombits(binary.LittleEndian.Uint32(m[20:24]))
	if dx < 95 || dx > 105 || dz < 195 || dz > 205 {
		t.Fatalf("dest = %v/%v（应 ±5m）", dx, dz)
	}
	if rig.c.waitSeconds != 0 {
		t.Fatalf("wait 未清零: %d", rig.c.waitSeconds)
	}
}

// 非 ingame 连接只累计等待秒（GetConnectWaitSeconds 口径）。
func TestMoveLoopNonInGameOnlyWaits(t *testing.T) {
	rig := newTestRig()
	rig.c.state = ConnLogging
	for i := 0; i < 3; i++ {
		rig.r.moveLoopLocked()
	}
	if rig.c.waitSeconds != 3 || len(rig.sent) != 0 {
		t.Fatalf("wait=%d sent=%d", rig.c.waitSeconds, len(rig.sent))
	}
}

// ---------- 模块状态机 ----------

func TestModuleStateMachine(t *testing.T) {
	r := NewRobot(Config{EchoAddr: "off"}, nil, nil, nil)
	if r.module != ModuleUnknown || r.module.String() != "Unknown" {
		t.Fatalf("initial = %v", r.module)
	}
	if !r.stateMachine(moduleEventStart, 0, 0) || r.module != ModuleOpened {
		t.Fatalf("start → %v", r.module)
	}
	if !r.stateMachine(moduleEventPause, 0, 0) || r.module != ModulePaused {
		t.Fatalf("pause → %v", r.module)
	}
	if !r.stateMachine(moduleEventStart, 0, 0) || r.module != ModuleOpened {
		t.Fatalf("resume → %v", r.module)
	}
	if r.stateMachine(99, 0, 0) {
		t.Fatal("未知事件应拒绝")
	}
	if !r.stateMachine(moduleEventTick, 0, 0) || r.module != ModuleOpened {
		t.Fatalf("tick 不迁移，got %v", r.module)
	}
}

// ---------- 配置 ----------

func TestParseGameRobotINI(t *testing.T) {
	ini := []byte(`
[System]
Sleep=-1
MainEntity=SoloRobot
MainScript=main

[Module]
fskit.dll
fscomm.dll
fssolorobot.dll

[Script]
Path=..\lua\solorobot
Reload=1

[Main]
MaxPlayer=2000
ConnectConcurrent=50
ConnectSpeed=100
//MessageEncodeKey=A5375CAE
MoveSeconds=7

[Outer]
Addr=10.1.2.3
Port=3001

[Echo]
Addr=127.0.0.1
Port=15010
`)
	cfg, err := ParseGameRobotINI(ini)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MainEntity != "SoloRobot" || len(cfg.Modules) != 3 || cfg.ScriptPath != `..\lua\solorobot` {
		t.Fatalf("system/module: %+v", cfg)
	}
	if cfg.MaxConnects != 2000 || cfg.ConnectConcurrent != 50 || cfg.ConnectSpeed != 100 || cfg.MoveSeconds != 7 {
		t.Fatalf("main: %+v", cfg)
	}
	if cfg.OuterAddr != "10.1.2.3" || cfg.OuterPort != 3001 || cfg.EchoPort != 15010 {
		t.Fatalf("net: %+v", cfg)
	}
	if cfg.MessageEncodeKey != "" {
		t.Fatalf("注释键不应生效: %q", cfg.MessageEncodeKey)
	}
}

func TestParseGameRobotINIDefaults(t *testing.T) {
	cfg, err := ParseGameRobotINI([]byte("[Main]\nMaxPlayer=3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConnects != 3 || cfg.ConnectConcurrent != 10 || cfg.ConnectSpeed != 50 ||
		cfg.MoveSeconds != 5 || cfg.AccountPrefix != "robot" || cfg.OuterPort != 2001 || cfg.EchoPort != 15009 {
		t.Fatalf("defaults: %+v", cfg)
	}
}

// ---------- echo 控制台 ----------

func TestEchoConsole(t *testing.T) {
	ec := NewEchoConsole(func(line string) (string, bool) {
		if line == "end" {
			return "", true
		}
		return "echo:" + line, false
	})
	if err := ec.Start("127.0.0.1", 0); err != nil {
		t.Fatal(err)
	}
	// 取实际端口
	ec.mu.Lock()
	ln := ec.listener
	ec.mu.Unlock()
	port := ln.Addr().(*net.TCPAddr).Port

	c1, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	welcome := make([]byte, 64)
	n, err := c1.Read(welcome)
	if err != nil || string(welcome[:n]) != "welcome game robot\r" {
		t.Fatalf("welcome = %q err=%v", welcome[:n], err)
	}
	c1.Write([]byte("hello\r"))
	buf := make([]byte, 64)
	n, _ = c1.Read(buf)
	if string(buf[:n]) != ">echo:hello\r" {
		t.Fatalf("reply = %q", buf[:n])
	}
	// 单客户端：第二个连接被拒
	c2, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	c2.SetReadDeadline(time.Now().Add(2 * time.Second))
	if n2, err := c2.Read(buf); n2 > 0 || err == nil {
		t.Fatalf("second client should be kicked: n=%d err=%v", n2, err)
	}
	// end 断开
	c1.Write([]byte("end\r"))
	c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	if n3, err := c1.Read(buf); n3 > 0 && err == nil {
		// 服务器侧关闭 → EOF
		_ = n3
	}
	ec.Stop()
}

// ---------- 全链：net.Pipe 假网关 ----------

// nullConn / captureConn：协议分发测试用的最小 net.Conn。
type nullConn struct{ net.Conn }

func (n *nullConn) Read(b []byte) (int, error)  { return 0, nil }
func (n *nullConn) Write(b []byte) (int, error) { return len(b), nil }
func (n *nullConn) Close() error                { return nil }

type captureConn struct {
	net.Conn
	sink *[][]byte
}

func (c *captureConn) Write(b []byte) (int, error) {
	*c.sink = append(*c.sink, append([]byte(nil), b...))
	return len(b), nil
}
func (c *captureConn) Read(b []byte) (int, error) { return 0, nil }
func (c *captureConn) Close() error               { return nil }

// gwFrame 在假网关侧解帧（与 robot.feed 同款状态机的函数形态）。
func gwFrames(data []byte, last *byte, buf *[]byte) [][]byte {
	var out [][]byte
	for _, b := range data {
		if b == 0xEE && *last == 0xEE {
			if n := len(*buf) - 1; n > 0 {
				out = append(out, append([]byte(nil), (*buf)[:n]...))
			}
			*buf = (*buf)[:0]
			*last = 0
			continue
		}
		if !(b == 0x00 && *last == 0xEE) {
			*buf = append(*buf, b)
		}
		*last = b
	}
	return out
}

func TestRobotEndToEndWithFakeGateway(t *testing.T) {
	gwConnCh := make(chan net.Conn, 4)
	dial := func(addr string, port int) (net.Conn, error) {
		c1, c2 := net.Pipe()
		select {
		case gwConnCh <- c1:
			return c2, nil
		default:
			c1.Close()
			c2.Close()
			return nil, fmt.Errorf("no gateway slot")
		}
	}

	cfg := Config{
		MaxConnects:       1,
		ConnectConcurrent: 1,
		ConnectSpeed:      1000, // 起连间隔 1ms
		MoveSeconds:       1,
		AccountPrefix:     "rb",
		PasswordPrefix:    "rb",
		EchoAddr:          "off",
	}
	robot := NewRobot(cfg, nil, dial, nil)
	if err := robot.Start(); err != nil {
		t.Fatal(err)
	}
	defer robot.Stop()

	gw := <-gwConnCh
	defer gw.Close()

	// 1) 读 login 帧
	var frames [][]byte
	var last byte
	var acc []byte
	deadline := time.Now().Add(5 * time.Second)
	readMore := func(want int) bool {
		buf := make([]byte, 4096)
		for time.Now().Before(deadline) && len(frames) < want {
			gw.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := gw.Read(buf)
			if n > 0 {
				frames = append(frames, gwFrames(buf[:n], &last, &acc)...)
			}
			if err != nil {
				time.Sleep(10 * time.Millisecond)
			}
		}
		return len(frames) >= want
	}
	if !readMore(1) {
		t.Fatal("未收到 login 帧")
	}
	login := frames[0]
	if login[0] != msgSendLogin || binary.LittleEndian.Uint32(login[1:5]) != 0x300 {
		t.Fatalf("login = % X", login[:8])
	}
	acctLen := int(binary.LittleEndian.Uint32(login[0x15:0x19]))
	if acct := login[0x19 : 0x19+acctLen]; !bytes.HasPrefix(acct, []byte("rb0\x00")) {
		t.Fatalf("account = %q", acct)
	}

	// 2) 回 login_succeed（角色数 0）→ 期待 create_char
	ls := make([]byte, 0x29)
	ls[0] = msgRecvLoginSucc
	gw.Write(escapeFrame(ls))
	if !readMore(2) {
		t.Fatal("未收到 create 帧")
	}
	create := frames[1]
	if create[0] != msgSendCreateChar || len(create) != 0x49 {
		t.Fatalf("create = % X…", create[:8])
	}
	want := append(ucs2Of("rb0"), 0, 0)
	if !bytes.HasPrefix(create[7:], want) {
		t.Fatalf("create name = % X", create[7:19])
	}

	// 3) 回 entry_scene → 期待 1 字节 ack
	es := make([]byte, 0x10)
	es[0] = msgRecvEntryScene
	binary.LittleEndian.PutUint32(es[5:9], 55)
	binary.LittleEndian.PutUint32(es[9:13], 66)
	gw.Write(escapeFrame(es))
	if !readMore(3) {
		t.Fatal("未收到 entry ack")
	}
	if ack := frames[2]; ack[0] != msgSendEntryAck || len(ack) != 1 {
		t.Fatalf("ack = % X", ack)
	}

	// 4) 回 location 设初始坐标 → 等 move（1s tick，wait>1 → 约 2s）
	loc := make([]byte, 0x1A)
	loc[0] = msgRecvLocation
	binary.LittleEndian.PutUint32(loc[1:5], 55)
	binary.LittleEndian.PutUint32(loc[5:9], 66)
	binary.LittleEndian.PutUint32(loc[9:13], math.Float32bits(10))
	binary.LittleEndian.PutUint32(loc[13:17], math.Float32bits(0))
	binary.LittleEndian.PutUint32(loc[17:21], math.Float32bits(20))
	gw.Write(escapeFrame(loc))

	if !readMore(4) {
		t.Fatal("未收到 move 帧")
	}
	mv := frames[3]
	if mv[0] != msgSendMove {
		t.Fatalf("move = % X", mv[:4])
	}
	x := math.Float32frombits(binary.LittleEndian.Uint32(mv[4:8]))
	z := math.Float32frombits(binary.LittleEndian.Uint32(mv[12:16]))
	dx := math.Float32frombits(binary.LittleEndian.Uint32(mv[16:20]))
	dz := math.Float32frombits(binary.LittleEndian.Uint32(mv[20:24]))
	if x != 10 || z != 20 || dx < 5 || dx > 15 || dz < 15 || dz > 25 {
		t.Fatalf("move cur=(%v,%v) dest=(%v,%v)", x, z, dx, dz)
	}

	// 5) 状态与统计
	if s := robot.GetConnectState(0); s != "ingame" {
		t.Fatalf("state = %q", s)
	}
	total, conn, inGame, failed, module := robot.stats()
	if total != 1 || conn != 1 || inGame != 1 || failed != 0 || module != ModuleOpened {
		t.Fatalf("stats = %d/%d/%d/%d/%v", total, conn, inGame, failed, module)
	}
}

// 拨号失败 → failed 计数、槽清空、不重连（total 不回落）。
func TestRobotDialFailure(t *testing.T) {
	dial := func(string, int) (net.Conn, error) { return nil, fmt.Errorf("refused") }
	cfg := Config{MaxConnects: 2, ConnectConcurrent: 2, ConnectSpeed: 1000, EchoAddr: "off"}
	robot := NewRobot(cfg, nil, dial, nil)
	if err := robot.Start(); err != nil {
		t.Fatal(err)
	}
	defer robot.Stop()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		total, _, _, failed, _ := robot.stats()
		if total == 2 && failed == 2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	total, _, _, failed, _ := robot.stats()
	t.Fatalf("total=%d failed=%d（应 2/2，且不再重拨）", total, failed)
}

// OnClose → 槽清空、计数保持。
func TestRobotServerClose(t *testing.T) {
	gwConnCh := make(chan net.Conn, 4)
	dial := func(string, int) (net.Conn, error) {
		c1, c2 := net.Pipe()
		gwConnCh <- c1
		return c2, nil
	}
	cfg := Config{MaxConnects: 1, ConnectConcurrent: 1, ConnectSpeed: 1000, MoveSeconds: 5, EchoAddr: "off"}
	robot := NewRobot(cfg, nil, dial, nil)
	if err := robot.Start(); err != nil {
		t.Fatal(err)
	}
	defer robot.Stop()
	gw := <-gwConnCh
	gw.Close() // 服务端断开
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s := robot.GetConnectState(0); s == "" {
			total, conn, _, failed, _ := robot.stats()
			if total == 1 && conn == 1 && failed == 0 {
				return // 槽已清、计数保持
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	total, conn, _, failed, _ := robot.stats()
	t.Fatalf("close 后状态=%q stats=%d/%d/%d", robot.GetConnectState(0), total, conn, failed)
}
