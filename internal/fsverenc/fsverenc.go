// Package fsverenc 是官方 FsVerEnc.dll（BGserverCJ 2014 快照，带 PDB 全符号）
// 的 Go 复刻。原 DLL 是 FsLoader.exe 宿主的 Lua 工具实体 "GameEncode"
// （GameEncode.bat：FsKit+FsComm+FsVerEnc，脚本 ..\Lua\GameEncode），
// 即 FsVerify.dll（服务端校验插件）与 FsVerCli.dll（客户端校验钩子）两套
// 源码的**离线生成器**。Lua 面：
//
//	属性 Console / EchoSock（PERSISTID get/set）
//	Start()  恒 true（无操作）
//	Stop()   恒 true（无操作）
//	Encode(id, complexity)  4≤complexity≤0x80 否则
//	        "[GameEncode::Encode]complex degree error"；→ GenEncodeFiles
//	Split(id, segments, length)  1≤segments≤8、0x40≤length≤0x200、
//	        segments*4*length≤0x1000 否则分段/长度/参数错；→ SplitCodeFiles
//
// GenEncodeFiles(id, complexity) @0x100429D0 管线（反汇编实证）：
//
//	srand(time(NULL))                       —— 种子非确定，官方表不可重放
//	g_pswd_mask[4]   = rand_dword ×4
//	g_pswd_key[16]   = rand_byte  ×16
//	g_rand_0..8      = 9 × 0x800000 字节 rand_byte 池
//	for i in 0..0x7FF:
//	    name = "Light is my strength" + itoa(i) + itoa(rand_dword())
//	    g_cond_value[i][16] = MD5 十六进制串前 16 个 ASCII 字符
//	    g_cond[i][j*8+b] = (cond16[i][j] >> b) & 1   —— 与 FsVerCli
//	        set_cond_value 的恒等 LSB 位映射闭环
//	CreateDirectoryA("ver%d\\"); fopen 三件 "%sMes.h"/"%sEncode.h"/
//	"%sEncode.cpp"（"wb"）→ gen_mes_h/gen_encode_h/gen_encode_cpp
//
// 发射模板为官方字符串面（MES_ACCOUNT/PASSWORD KEY/MASK 与 internal/fsverify
// 的 field1/field2 常量逐字节同源；链表达式/防调试 __asm _emit 诱饵/md5 收尾
// 模板见 docs/official）。rand_byte=rand()>>7、rand_dword=4×rand_byte 大端拼，
// 均为 MSVC LCG（seed=seed*214013+2531011，rand=(seed>>16)&0x7FFF）。
//
// 复刻边界：管线与随机流语义逐指令等价；发射器链体选取（gen_encode_cpp
// 0xBC0 字节内的池索引策略）为结构级复刻——官方产物因 time 种子本就不可
// 逐字节重放，本实现输出可编译的同族源码。
package fsverenc

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// MSVC rand（CRT LCG）
// ---------------------------------------------------------------------------

// MsvcRand 复刻 MSVCRT rand/srand 状态机。
type MsvcRand struct {
	seed uint32
}

// NewMsvcRand 对应 srand(seed)。
func NewMsvcRand(seed uint32) *MsvcRand { return &MsvcRand{seed: seed} }

// Next 对应 rand()：(seed=seed*214013+2531011)>>16 & 0x7FFF。
func (r *MsvcRand) Next() uint32 {
	r.seed = r.seed*214013 + 2531011
	return (r.seed >> 16) & 0x7FFF
}

// Seed 暴露内部状态（测试与池流随机访问用）。
func (r *MsvcRand) Seed() uint32 { return r.seed }

// RandByte 对应 rand_byte @0x10040430：rand()*256/32768 = rand()>>7。
func (r *MsvcRand) RandByte() byte { return byte(r.Next() >> 7) }

// RandDWord 对应 rand_dword @0x100404A0：4×rand_byte 大端拼。
func (r *MsvcRand) RandDWord() uint32 {
	v := uint32(r.RandByte()) << 24
	v |= uint32(r.RandByte()) << 16
	v |= uint32(r.RandByte()) << 8
	v |= uint32(r.RandByte())
	return v
}

// lcgJump 返回 LCG 迭代 n 步的 (A^n, S_n=A^0+..+A^{n-1}) mod 2^32。
// A-1 为偶数不可模逆，用倍增几何和：S(2i)=S(i)(1+A^i)。
func lcgJump(n uint64) (uint64, uint64) {
	const a = uint64(214013)
	resA, resS := uint64(1), uint64(0) // A^0, S(0)
	curA, curS := a, uint64(1)         // A^1, S(1)
	for n > 0 {
		if n&1 == 1 {
			resS = (resS + resA*curS) & 0xFFFFFFFF
			resA = resA * curA & 0xFFFFFFFF
		}
		curS = curS * (1 + curA) & 0xFFFFFFFF
		curA = curA * curA & 0xFFFFFFFF
		n >>= 1
	}
	return resA, resS
}

// SeedAt 返回从当前状态再消费 n 次 rand() 后的内部种子（池流随机访问用）。
func (r *MsvcRand) SeedAt(n uint64) uint32 {
	const c = uint64(2531011)
	an, sn := lcgJump(n)
	return uint32((uint64(r.seed)*an + c*sn) & 0xFFFFFFFF)
}

// ---------------------------------------------------------------------------
// 生成器（GenEncodeFiles 管线）
// ---------------------------------------------------------------------------

// 官方常量：cond16 名字种子前缀（sprintf "%s%d%d" 的 %s）。
const condNamePrefix = "Light is my strength"

// Pool 数：g_rand_0..g_rand_8。
const numPools = 9

// 每池字节数：0x800000。
const poolSize = 0x800000

// 前置消耗：mask(4 dword=16 次 rand) + key(16 次 rand)。
const preludeCalls = 16 + 16

// Generator 复刻 GenEncodeFiles 的随机管线（种子显式注入便于测试）。
type Generator struct {
	rng      *MsvcRand
	seed0    uint32
	pswdMask [4]uint32
	pswdKey  [16]byte
	cond16   [2048][16]byte
}

// NewGenerator 对应 srand(seed) 起始的整条管线（seed 官方为 time(NULL)）。
func NewGenerator(seed uint32) *Generator {
	g := &Generator{rng: NewMsvcRand(seed), seed0: seed}
	for i := range g.pswdMask {
		g.pswdMask[i] = g.rng.RandDWord()
	}
	for i := range g.pswdKey {
		g.pswdKey[i] = g.rng.RandByte()
	}
	// 9 × 0x800000 rand 池：直接跳跃到池后状态
	g.rng.seed = (&MsvcRand{seed: g.rng.seed}).SeedAt(uint64(numPools) * poolSize)
	for i := 0; i < 2048; i++ {
		name := condNamePrefix + strconv.Itoa(i) + strconv.FormatUint(uint64(g.rng.RandDWord()), 10)
		sum := md5.Sum([]byte(name))
		hex := fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
			sum[0], sum[1], sum[2], sum[3], sum[4], sum[5], sum[6], sum[7],
			sum[8], sum[9], sum[10], sum[11], sum[12], sum[13], sum[14], sum[15])
		copy(g.cond16[i][:], hex[:16]) // 前 16 个 ASCII 十六进制字符
	}
	return g
}

// Cond16 返回 g_cond_value[i]（16 个 ASCII hex 字符）。
func (g *Generator) Cond16(i int) []byte {
	out := make([]byte, 16)
	copy(out, g.cond16[i][:])
	return out
}

// PoolByte 流式取 g_rand_k[i]（不物化 72MB 池）：
// 全局 rand 序号 = preludeCalls + k*poolSize + i。
func (g *Generator) PoolByte(k, i int) byte {
	n := uint64(preludeCalls) + uint64(k)*poolSize + uint64(i)
	seed := (&MsvcRand{seed: g.seed0}).SeedAt(n)
	r := MsvcRand{seed: seed}
	return byte(r.Next() >> 7)
}

// PswdMask / PswdKey 暴露管线产物（MES_PASSWORD_* 发射源）。
func (g *Generator) PswdMask() [4]uint32 { return g.pswdMask }
func (g *Generator) PswdKey() [16]byte   { return g.pswdKey }

// ---------------------------------------------------------------------------
// GameEncode 实体（Lua 面）
// ---------------------------------------------------------------------------

// Encode 对应 GameEncode::Encode(id, complexity)。
func Encode(id, complexity int) error {
	if complexity < 4 || complexity > 0x80 {
		return fmt.Errorf("[GameEncode::Encode]complex degree error")
	}
	if id < 0 {
		return fmt.Errorf("[GameEncode::Encode]bad id")
	}
	return nil
}

// Split 对应 GameEncode::Split(id, segments, length)。
func Split(id, segments, length int) error {
	if segments < 1 || segments > 8 {
		return fmt.Errorf("[GameEncode::Split]segments error")
	}
	if length < 0x40 || length > 0x200 {
		return fmt.Errorf("[GameEncode::Split]length error")
	}
	if segments*4*length > 0x1000 {
		return fmt.Errorf("[GameEncode::Split]parameter error")
	}
	if id < 0 {
		return fmt.Errorf("[GameEncode::Split]bad id")
	}
	return nil
}

// ---------------------------------------------------------------------------
// 三件源码发射（结构级复刻，模板取官方字符串面）
// ---------------------------------------------------------------------------

// 固定官方常量：与 internal/fsverify field1/field2 同源。
var (
	mesPasswordKey  = [16]byte{0x10, 0x6E, 0xBA, 0x55, 0x77, 0x61, 0xC7, 0x7E, 0x1E, 0xEA, 0x0D, 0xEB, 0x67, 0x62, 0x93, 0xDC}
	mesPasswordMask = [8]uint32{0xF24D83E5, 0xAB2526D8, 0x32B93688, 0xAFFDFE20, 0xE2435AE5, 0x136126B1, 0x52BA6E85, 0x3FA3FB23}
	mesAccountMask  = [8]uint32{0xF8EBD0A7, 0xC4BADC9C, 0xA84FD0A2, 0x74CF24B7, 0x26E24C92, 0x38EFDA09, 0xC3B2D047, 0x28EFD49C}
)

// EmitMesH 生成服务端宏头（FsVerify 侧）。账号键由生成器随机产出
// （官方 MES_ACCOUNT_KEY_%d 0x%02X 模板）。
func (g *Generator) EmitMesH(accountKey [16]byte) string {
	var b strings.Builder
	b.WriteString("#ifndef _MES_H\r\n#define _MES_H\r\n")
	fmt.Fprintf(&b, "#define MES_MESSAGE_VERSION %d\r\n", 300)
	fmt.Fprintf(&b, "#define MES_MAX_CLASS_NUM 0x%02X\r\n", 0x08)
	for i := 7; i >= 0; i-- {
		fmt.Fprintf(&b, "#define MES_ACCOUNT_MASK_%d 0x%08X\r\n", i, mesAccountMask[i])
	}
	for i := 15; i >= 0; i-- {
		fmt.Fprintf(&b, "#define MES_ACCOUNT_KEY_%d 0x%02X\r\n", i, accountKey[i])
	}
	for i := 7; i >= 0; i-- {
		fmt.Fprintf(&b, "#define MES_PASSWORD_MASK_%d 0x%08X\r\n", i, mesPasswordMask[i])
	}
	for i := 15; i >= 0; i-- {
		fmt.Fprintf(&b, "#define MES_PASSWORD_KEY_%d 0x%02X\r\n", i, mesPasswordKey[i])
	}
	// MES_ENCODE_ORDER / MES_COND_ENCODE / MES_ENCODE_N 宏体（结构级）
	fmt.Fprintf(&b, "#define MES_ENCODE_ORDER UI32 g_encode_order[%d] = { ", 2048)
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, "0x%03X, ", i) // 结构占位：官方为随机置换
	}
	b.WriteString("};\r\n")
	b.WriteString("#endif // _MES_H\r\n")
	return b.String()
}

// EmitEncodeH 生成客户端头（FsVerCli 侧 Encode.h）。
func (g *Generator) EmitEncodeH() string {
	var b strings.Builder
	b.WriteString("#ifndef _ENCODE_H\r\n#define _ENCODE_H\r\n#include <windows.h>\r\n")
	fmt.Fprintf(&b, "#define MES_MAX_CLASS_NUM_1 0x%02X\r\n", 0x08)
	b.WriteString("typedef BOOL (*W_ISDP)(VOID);\r\nextern W_ISDP g_isdp;\r\n")
	b.WriteString("extern void set_cond_value(unsigned char* p);\r\n")
	b.WriteString("extern bool login_send(const void* pdata, int len);\r\n")
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, "extern bool encode_func_%d(unsigned char* input, unsigned char* output);\r\n", i)
	}
	b.WriteString("#endif // _ENCODE_H\r\n")
	return b.String()
}

// EmitEncodeCpp 生成客户端实现（FsVerCli 侧 Encode.cpp 前置段 + 单函数骨架）。
func (g *Generator) EmitEncodeCpp() string {
	var b strings.Builder
	b.WriteString("#include <string.h>\r\n#include \"../public/Macros.h\"\r\n")
	b.WriteString("#include \"../server/OuterMsg.h\"\r\n#include \"../tools/d3des.h\"\r\n")
	b.WriteString("#include \"../tools/md5.h\"\r\n#include \"CoreApi.h\"\r\n#include \"Encode.h\"\r\n")
	b.WriteString("typedef unsigned char UI08;\r\ntypedef unsigned int UI32;\r\nextern VAPI* _CORE_API;\r\n")
	b.WriteString("W_ISDP g_isdp = NULL;\r\nUI08 g_content[16];\r\n")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&b, "UI32 g_cond_%d = 0x00;\r\n", i)
	}
	b.WriteString("void set_cond_value(UI08* p)\r\n{\r\n")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&b, "\tg_cond_%d = (*(p + %d) >> %d) & 0x01;\r\n", i, i/8, i%8)
	}
	b.WriteString("}\r\n")
	b.WriteString("bool encode_func_0(UI08* input, UI08* output)\r\n{\r\n")
	b.WriteString("\tif (input == NULL) { __asm __emit 0x00 }\r\n")
	b.WriteString("\tUI08 content[8];\r\n")
	b.WriteString("\tcontent[0] = UI08(input[0] ^ 0x00);\r\n")
	b.WriteString("\tmd5 alg;\r\n\talg.Update(content, 8);\r\n\talg.Finalize();\r\n")
	b.WriteString("\tmemcpy(output, alg.Digest(), 16);\r\n\treturn true;\r\n}\r\n")
	return b.String()
}
