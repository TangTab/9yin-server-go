package main

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"sync/atomic"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// npcDetail0x28Enabled 控制 0x28 详情记录生成下发（NINEYIN_NPC_DETAIL_0X28=1）。
// 2026-11-10 真机 sparse 二分定案：官服表方言下 NPC 渲染必需 0x28（0x0D
// 与官服样本同构也不够）——本生成器是官服模式 NPC 渲染的载体。
func npcDetail0x28Enabled() bool {
	return strings.TrimSpace(os.Getenv("NINEYIN_NPC_DETAIL_0X28")) == "1"
}

// npcDetail0x28Event 为每次物化生成的事件 u32（官服 [38:42] 每物化事件
// 一个值、同对象重发幂等——l.pcapng 218 帧实证）。
var npcDetail0x28Event atomic.Uint32

// npcDetail0x28Frame 以官服 234B 模板为底生成 0x28 详情记录。
//
// 模板（c.pcapng #1776）逐字节全解定版（2026-11-11，l.pcapng 218 帧
// 交叉验证）。结构：
//
//	[0:14]   头（28 47 / len-1 / 模板号 290 / 0 / 状态位 0x91）
//	[14:38]  obj / owner / x y z orient（4×f32）
//	[38:51]  前奏：事件 u32（每物化一值）+ 9B 常量
//	[51:57]  CollideRadius(179) f32 0.939
//	[57:78]  State(183) 串 [len][01 00]["ty_bukuai_guard\0"]
//	[78:81]  Type(181) byte 4
//	[81:105] Resource(1448) mangled 区（前缀 7B + NUL 结尾串）
//	[105:126] RotatePara(1565) 串（非身份，保原样）
//	[126:143] ConfigID(1447) 串 ["monm031300\0"]
//	[143:177] Photo(184) 串（官方 blob 值，len=29 含内嵌二进制）
//	[177:180] Level(1472) byte 25
//	[180:183] BossLevel(1597) byte 1
//	[183:213] IDCard(1602) 串（blob 值，非身份，保原样）
//	[213:220] 未定段（含 3d 06 01 前缀，保原样）
//	[220:228] HPRatio(471) 短值形 [64][01 00] + 3B
//	[228:234] HitHPRatio(484) raw i32 100
//
// 生成策略＝精确切片替换：仅重写 [14:38] 与四个身份串（State/Resource/
// ConfigID/Photo），全部用干净 ASCII 形 [len=strlen+1][01 00][值+NUL]；
// 其余段（含 RotatePara/IDCard/尾部常量族）逐字节保原样——字段数不变，
// 官方 blob 边界不破坏（=4 探针崩溃根因即 mangled 区边界错位）。
func npcDetail0x28Frame(objectID, ownerID uint32, transform worldcore.Transform, resource, configID string) []byte {
	frame := append([]byte(nil), npcDetail0x28Template...)
	binary.LittleEndian.PutUint32(frame[14:], objectID)
	binary.LittleEndian.PutUint32(frame[18:], ownerID)
	binary.LittleEndian.PutUint32(frame[22:], math.Float32bits(transform.X))
	binary.LittleEndian.PutUint32(frame[26:], math.Float32bits(transform.Y))
	binary.LittleEndian.PutUint32(frame[30:], math.Float32bits(transform.Z))
	binary.LittleEndian.PutUint32(frame[34:], math.Float32bits(transform.Orient))
	// 事件 u32：每物化一值（同对象重复物化也递增——官服幂等重发场景
	// 我们不产生，重进视口重发取同帧语义由调用方缓存实现，此处从简）。
	binary.LittleEndian.PutUint32(frame[38:], 0x0000E900+npcDetail0x28Event.Add(1))

	photo := ""
	if resource != "" {
		photo = "icon\\func" + resource + ".png"
	}
	// 身份串切片替换（干净形），从后往前拼接保持偏移有效。
	out := make([]byte, 0, len(frame)+32)
	out = append(out, frame[:59]...)
	out = appendClean0x28String(out, 183, "stand") // State：len@59 起
	out = append(out, frame[78:81]...)             // Type
	if resource == "" {
		out = append(out, frame[81:104]...) // 无资源：保模板原段
	} else {
		out = appendClean0x28String(out, 1448, resource) // Resource 整段替换 [81:104)
		out = append(out, frame[104:126]...)             // RotatePara+尾 3B 原样
	}
	if configID == "" {
		out = append(out, frame[126:143]...)
	} else {
		out = appendClean0x28String(out, 1447, configID) // ConfigID [126:143)
	}
	if photo == "" {
		out = append(out, frame[143:177]...)
	} else {
		out = appendClean0x28String(out, 184, photo) // Photo [143:177)
	}
	out = append(out, frame[177:]...) // Level/BossLevel/IDCard/尾段原样
	binary.LittleEndian.PutUint32(out[2:], uint32(len(out)-1))
	return out
}

// appendClean0x28String 追加干净串字段：[u16 idx][u8 len=strlen+1][01 00]
// [值+NUL]（官服 State/ConfigID/WeaponMode 同形实证；l.pcapng 交叉验证）。
func appendClean0x28String(out []byte, index uint16, value string) []byte {
	out = binary.LittleEndian.AppendUint16(out, index)
	out = append(out, byte(len(value)+1), 0x01, 0x00)
	out = append(out, value...)
	return append(out, 0)
}
