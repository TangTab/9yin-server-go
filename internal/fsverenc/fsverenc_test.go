package fsverenc_test

import (
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/fsverenc"
)

// MSVC LCG 已知向量：srand(0) 的前 10 个 rand()。
func TestMsvcRandVectors(t *testing.T) {
	want := []uint32{38, 7719, 21238, 2437, 8855, 11797, 8365, 32285, 10450, 30612}
	r := fsverenc.NewMsvcRand(0)
	for i, w := range want {
		if got := r.Next(); got != w {
			t.Fatalf("rand[%d] = %d want %d", i, got, w)
		}
	}
}

// srand(1) 向量（MSVC 文档值）。
func TestMsvcRandSeedOne(t *testing.T) {
	want := []uint32{41, 18467, 6334, 26500, 19169, 15724, 11478, 29358, 26962, 24464}
	r := fsverenc.NewMsvcRand(1)
	for i, w := range want {
		if got := r.Next(); got != w {
			t.Fatalf("rand[%d] = %d want %d", i, got, w)
		}
	}
}

// rand_byte = rand()>>7（0x10040430 的 shl 8 / 0x8000 有符号除）。
func TestRandByte(t *testing.T) {
	r := fsverenc.NewMsvcRand(0)
	r2 := fsverenc.NewMsvcRand(0)
	for i := 0; i < 32; i++ {
		want := byte(r2.Next() >> 7)
		if got := r.RandByte(); got != want {
			t.Fatalf("byte[%d] = %d want %d", i, got, want)
		}
	}
}

// SeedAt 状态跳跃 = 顺序消费。
func TestSeedAtJump(t *testing.T) {
	base := fsverenc.NewMsvcRand(12345)
	seq := fsverenc.NewMsvcRand(12345)
	var states []uint32
	states = append(states, 12345)
	for i := 0; i < 1000; i++ {
		seq.Next()
		states = append(states, seqSeed(seq))
	}
	for n, want := range states {
		got := base.SeedAt(uint64(n))
		if got != want {
			t.Fatalf("SeedAt(%d) = %08x want %08x", n, got, want)
		}
	}
}

func seqSeed(r interface{ Seed() uint32 }) uint32 { return r.Seed() }

// PoolByte 与顺序消费的池字节一致（池 0 内抽查；跨池跳跃由 SeedAt 覆盖）。
func TestPoolByteMatchesSequential(t *testing.T) {
	g := fsverenc.NewGenerator(7)
	seq := fsverenc.NewMsvcRand(7)
	// prelude：4 dword + 16 byte = 32 次 rand
	for i := 0; i < 4; i++ {
		seq.RandDWord()
	}
	for i := 0; i < 16; i++ {
		seq.RandByte()
	}
	consumed := 0
	for _, idx := range []int{0, 1, 2, 0x10, 0xFF, 0x1234} {
		for j := consumed; j < idx; j++ {
			seq.RandByte()
		}
		want := seq.RandByte()
		consumed = idx + 1
		if got := g.PoolByte(0, idx); got != want {
			t.Fatalf("pool[0][%#x] = %d want %d", idx, got, want)
		}
	}
}

// cond16 全表为 ASCII 十六进制串前 16 字符（官方 "Light is my strength"+i+r）。
func TestCond16AsciiHex(t *testing.T) {
	g := fsverenc.NewGenerator(0xC0FFEE)
	for i := 0; i < 2048; i++ {
		for _, ch := range g.Cond16(i) {
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
				t.Fatalf("cond16[%d] 含非 hex 字符 %q", i, ch)
			}
		}
	}
	// 确定性：同种子同产物
	g2 := fsverenc.NewGenerator(0xC0FFEE)
	if string(g.Cond16(0)) != string(g2.Cond16(0)) || string(g.Cond16(2047)) != string(g2.Cond16(2047)) {
		t.Fatal("同种子产物不确定")
	}
	// 异种子异产物
	g3 := fsverenc.NewGenerator(1)
	if string(g.Cond16(0)) == string(g3.Cond16(0)) {
		t.Fatal("异种子产物相同")
	}
}

// Lua 面参数校验（Encode/Split 官方错误文案）。
func TestLuaSurfaceValidation(t *testing.T) {
	if err := fsverenc.Encode(1, 3); err == nil || err.Error() != "[GameEncode::Encode]complex degree error" {
		t.Fatalf("Encode(1,3) = %v", err)
	}
	if err := fsverenc.Encode(1, 0x81); err == nil {
		t.Fatal("Encode(1,0x81) 应报错")
	}
	if err := fsverenc.Encode(1, 4); err != nil {
		t.Fatalf("Encode(1,4) = %v", err)
	}
	if err := fsverenc.Encode(1, 0x80); err != nil {
		t.Fatalf("Encode(1,0x80) = %v", err)
	}
	if err := fsverenc.Split(1, 0, 0x40); err == nil || err.Error() != "[GameEncode::Split]segments error" {
		t.Fatalf("Split segments: %v", err)
	}
	if err := fsverenc.Split(1, 9, 0x40); err == nil || err.Error() != "[GameEncode::Split]segments error" {
		t.Fatalf("Split(1,9,..) 应先报 segments: %v", err)
	}
	if err := fsverenc.Split(1, 1, 0x3F); err == nil || err.Error() != "[GameEncode::Split]length error" {
		t.Fatalf("Split length: %v", err)
	}
	if err := fsverenc.Split(1, 8, 0x200); err == nil || err.Error() != "[GameEncode::Split]parameter error" {
		t.Fatalf("Split parameter: %v", err)
	}
	if err := fsverenc.Split(1, 8, 0x80); err != nil { // 8*4*0x80=0x1000 恰好通过
		t.Fatalf("Split(1,8,0x80) = %v", err)
	}
}

// 三件发射含官方关键模板行。
func TestEmitTemplates(t *testing.T) {
	g := fsverenc.NewGenerator(42)
	var ak [16]byte
	for i := range ak {
		ak[i] = byte(i * 7)
	}
	mes := g.EmitMesH(ak)
	for _, sub := range []string{
		"#define MES_MESSAGE_VERSION 300",
		"#define MES_ACCOUNT_MASK_0 0xF8EBD0A7",
		"#define MES_PASSWORD_KEY_0 0x10",
		"#define MES_PASSWORD_MASK_7 0x3FA3FB23",
		"#define MES_ACCOUNT_KEY_15 0x" + hexByte(ak[15]),
	} {
		if !strings.Contains(mes, sub) {
			t.Fatalf("Mes.h 缺模板行 %q", sub)
		}
	}
	h := g.EmitEncodeH()
	for _, sub := range []string{"extern bool login_send", "typedef BOOL (*W_ISDP)(VOID);", "set_cond_value"} {
		if !strings.Contains(h, sub) {
			t.Fatalf("Encode.h 缺 %q", sub)
		}
	}
	cpp := g.EmitEncodeCpp()
	for _, sub := range []string{"__asm __emit", "alg.Update(content, 8);", "memcpy(output, alg.Digest(), 16);", "g_cond_127 = (*(p + 15) >> 7) & 0x01;"} {
		if !strings.Contains(cpp, sub) {
			t.Fatalf("Encode.cpp 缺 %q", sub)
		}
	}
}

func hexByte(b byte) string {
	const hexd = "0123456789ABCDEF"
	return string([]byte{hexd[b>>4], hexd[b&15]})
}
