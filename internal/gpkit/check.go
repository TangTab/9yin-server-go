// 内置 70 对检查函数、hack 警告表与 x64 CCSAuth::CheckAuthReply 复刻。
//
// 官方表 A(生成器,0x14AA0 填 .bss 0x10022CC8)与表 B(校验器,0x14D60
// 填 0x10022E58)各 70 项,对应 .rdata 0x1F160 起的 "hack 10 warning" …
// "hack 79 warning"。抽查 A[0](@0x10001A30)为 TEA 风格多轮整数混合
// (常量 0x23132312 / 0x21524111 / 0x23973235 / 0x42A48222),密钥取自
// .rdata 0x1F150 四全局:3990458D 850FE045 0761EE8A 18A164C3。
// 复刻以同构混合实现 70 个参数化变体;官方逐函数混淆常量不同,但
// .bin/.dat 载荷不随包分发,无逐位兼容对象,保持算法族一致即可。
package gpkit

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// 官方四密钥(.rdata 0x1F150..0x1F15C)。
var mixKeys = [4]uint32{0x3990458D, 0x850FE045, 0x0761EE8A, 0x18A164C3}

// 官方 A[0] 反汇编中的四个混合常量。
const (
	mixC0 = 0x23132312
	mixC1 = 0x21524111
	mixC2 = 0x23973235
	mixC3 = 0x42A48222
)

// builtinCheck 返回第 id(0..69)号内置检查(表 A/B 同族)。
// f(a, b):以 (id, a, b) 为输入做 8 轮 TEA 风格混合,返回 32 位应答。
func builtinCheck(id int) CheckFunc {
	if id < 0 || id >= 70 {
		id = 0
	}
	base := uint32(id) + 0x100 // hack 警告号 = id + 10
	return func(a, b uint32) uint32 {
		v0 := a ^ (base * 0x9E3779B9)
		v1 := b ^ mixKeys[id%4]
		sum := uint32(0)
		for i := 0; i < 8; i++ {
			sum += 0x9E3779B9
			v0 += ((v1 << 4) + mixKeys[0]) ^ (v1 + sum) ^ ((v1 >> 5) + mixKeys[1]) ^ mixC0
			v1 += ((v0 << 4) + mixKeys[2]) ^ (v0 + sum) ^ ((v0 >> 5) + mixKeys[3]) ^ mixC1
			if i&1 == 0 {
				v0 ^= v1 + mixC2
			} else {
				v1 ^= v0 + mixC3
			}
		}
		return v0 ^ v1 ^ base
	}
}

// HackWarning 复刻官方警告文案表:10..79 → "hack N warning"。
// 官方另有两条表外特殊告警:run on windows 64bit / client no admin。
func HackWarning(n int) (string, error) {
	if n < 10 || n > 79 {
		return "", fmt.Errorf("gpkit: hack warning %d out of range 10..79", n)
	}
	return fmt.Sprintf("hack %d warning", n), nil
}

// AuthReplySize 是 x64 CCSAuth::CheckAuthReply 的固定回执包长度。
const AuthReplySize = 0x80

// CheckAuthReply 复刻 x64(2013)CCSAuth::CheckAuthReply(@0x16C10):
//
//  1. len(pkt) 必须 0x80(packet length error!);
//  2. pkt[0:2](LE) 必须等于 expectIdx(CheckAuthReply Idx Exception!,防重放);
//  3. pkt[0:4] == ~pkt[4:8](packet mask check error!);
//  4. idx 分派校验表:0x0A<idx<0x15 且跳过开关关闭时直接通过;
//     idx<=0x14 且非 (0x14,0x50) 开区间且 !=0x0A 时调 table[idx];
//     其余 check error。
//
// table 为 nil 时使用内置校验表(每项恒 0 = 通过,由载荷注入真实校验)。
func CheckAuthReply(pkt []byte, expectIdx uint16, table []CheckFunc) (uint16, error) {
	if len(pkt) != AuthReplySize {
		return 0, errors.New("packet  length error!")
	}
	idx := binary.LittleEndian.Uint16(pkt[0:2])
	if idx != expectIdx {
		return 0, errors.New("CheckAuthReply Idx Exception!")
	}
	if binary.LittleEndian.Uint32(pkt[0:4]) != ^binary.LittleEndian.Uint32(pkt[4:8]) {
		return 0, errors.New("packet mask check error!")
	}
	if idx > 0x0A && idx < 0x15 && !skipWindowEnabled {
		return idx, nil // 官方:窗口期直接放行
	}
	if idx <= 0x14 && !(idx > 0x14 && idx < 0x50) && idx != 0x0A {
		var fn CheckFunc
		if table != nil && int(idx) < len(table) {
			fn = table[idx]
		} else {
			fn = builtinAuthTable[idx]
		}
		if fn != nil {
			if ret := fn(0x78, 0); ret != 0 {
				return 0, errors.New("check error")
			}
		}
		return idx, nil
	}
	return 0, errors.New("check error")
}

// skipWindowEnabled 对应官方 cmp dword [rip+0x2d68b],0 的全局旁路
// (静态为 0:窗口期检查跳过生效)。
var skipWindowEnabled bool

// builtinAuthTable:内置校验表(默认全通过;真实校验由载荷注入)。
var builtinAuthTable [0x15]CheckFunc

// tableA / tableB:官方 GPKitSvr.dll(x86)70 对检查函数的 RVA 存档
// (0x14AA0 / 0x14D60 初始化代码提取,tools/gpk_probe/gpk_tables.json),
// 仅供审计比对,运行不使用。
var tableA = [70]uint32{
	0x01A30, 0x01A30, 0x02180, 0x02580, 0x029F0, 0x02DB0, 0x03200,
	0x03640, 0x03AC0, 0x03F60, 0x043A0, 0x04830, 0x04C40, 0x05050,
	0x054A0, 0x05870, 0x05CD0, 0x060B0, 0x064C0, 0x06950, 0x06E10,
	0x072D0, 0x077B0, 0x07B80, 0x07FD0, 0x08410, 0x088B0, 0x08D00,
	0x09130, 0x09560, 0x09A50, 0x09F10, 0x0A330, 0x0A750, 0x0AB50,
	0x0B060, 0x0B560, 0x0B9B0, 0x0BDD0, 0x0C2E0, 0x0C760, 0x0CB60,
	0x0CFF0, 0x0D3E0, 0x0D870, 0x0DCD0, 0x0E0D0, 0x0E4F0, 0x0E980,
	0x0EE40, 0x0F260, 0x0F710, 0x0FAF0, 0x0FF30, 0x103C0, 0x10810,
	0x10CB0, 0x11110, 0x11540, 0x11920, 0x11D70, 0x12190, 0x12630,
	0x12B00, 0x12F50, 0x13320, 0x13780, 0x13BF0, 0x13FF0, 0x14440,
}

var tableB = [70]uint32{
	0x01C40, 0x01F40, 0x02350, 0x02760, 0x02BB0, 0x02F70, 0x03420,
	0x03880, 0x03D50, 0x041A0, 0x045D0, 0x04A30, 0x04E00, 0x05250,
	0x05660, 0x05AB0, 0x05E90, 0x06270, 0x066C0, 0x06B80, 0x07040,
	0x07550, 0x07970, 0x07D70, 0x081F0, 0x08620, 0x08AB0, 0x08ED0,
	0x09350, 0x097F0, 0x09C90, 0x0A130, 0x0A520, 0x0A930, 0x0ADE0,
	0x0B2E0, 0x0B720, 0x0BB80, 0x0C050, 0x0C4E0, 0x0C950, 0x0CD90,
	0x0D1D0, 0x0D610, 0x0DAB0, 0x0DEC0, 0x0E2E0, 0x0E6F0, 0x0EBB0,
	0x0F060, 0x0F4F0, 0x0F8D0, 0x0FD20, 0x101B0, 0x105C0, 0x10A30,
	0x10E80, 0x11330, 0x11710, 0x11B10, 0x11F30, 0x123B0, 0x12870,
	0x12D00, 0x13110, 0x13520, 0x13990, 0x13DA0, 0x141B0, 0x14600,
}
