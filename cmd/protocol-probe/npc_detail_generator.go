package main

import (
	"encoding/binary"
	"math"
	"os"
	"strings"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// npcDetail0x28Enabled 控制 0x28 详情记录生成下发（NINEYIN_NPC_DETAIL_0X28=1）。
// 2026-11-10 真机 sparse 二分定案：官服表方言下 NPC 渲染必需 0x28（0x0D
// 与官服样本同构也不够）——本生成器是官服模式 NPC 渲染的载体。
func npcDetail0x28Enabled() bool {
	return strings.TrimSpace(os.Getenv("NINEYIN_NPC_DETAIL_0X28")) == "1"
}

// npcDetail0x28Frame 以官服模板为底生成 0x28 详情记录（§14.1 帧头定版）。
// 策略=模板常量照抄 + 身份字段按 idx 模式定位重写：
//   - [14:22] obj/owner、[22:38] 变换四浮点
//   - State(183) 动作名 / Resource(1448) / ConfigID(1447) / Photo(184)
//     四个身份串按 [u16 idx][u8 len][01 00][GBK+NUL] 重写（帧长随之伸缩，
//     [2:6] len-1 重算）
//   - 其余段（[38:51] 未解区、Type 与 Resource 间变体字节、尾段 d7 01 …
//     常量族）为模板常量照抄——=3 探针崩溃根因是身份串与本对象冲突，
//     数值常量按模板类复制（生成器验收标准即身份一致性）。
func npcDetail0x28Frame(objectID, ownerID uint32, transform worldcore.Transform, resource, configID string) []byte {
	frame := append([]byte(nil), npcDetail0x28Template...)
	binary.LittleEndian.PutUint32(frame[14:], objectID)
	binary.LittleEndian.PutUint32(frame[18:], ownerID)
	binary.LittleEndian.PutUint32(frame[22:], math.Float32bits(transform.X))
	binary.LittleEndian.PutUint32(frame[26:], math.Float32bits(transform.Y))
	binary.LittleEndian.PutUint32(frame[30:], math.Float32bits(transform.Z))
	binary.LittleEndian.PutUint32(frame[34:], math.Float32bits(transform.Orient))

	frame = rewrite0x28StringField(frame, 183, "stand")
	if resource != "" {
		frame = rewrite0x28StringField(frame, 1448, resource)
		frame = rewrite0x28StringField(frame, 184, "icon\\func"+resource+".png")
	}
	if configID != "" {
		frame = rewrite0x28StringField(frame, 1447, configID)
	}
	binary.LittleEndian.PutUint32(frame[2:], uint32(len(frame)-1))
	return frame
}

// rewrite0x28StringField 定位串字段并整段重写。两种形态：
//   - 规整形：[u16 idx][u8 len][01 00][len 字节含 NUL]
//   - mangled 形（模板 Resource 实测）：[u16 idx][≤8B 前缀变体][01 00]
//     [NUL 结尾串，无 len 字节]
//
// 找不到（模板缺该字段/形状不符）时原样返回——宁可少改不可错改。
func rewrite0x28StringField(frame []byte, index uint16, value string) []byte {
	want := []byte(value)
	for p := 38; p+5 <= len(frame); p++ {
		if binary.LittleEndian.Uint16(frame[p:]) != index {
			continue
		}
		n := int(frame[p+2])
		if frame[p+3] == 0x01 && frame[p+4] == 0x00 && p+5+n <= len(frame) && p+5+n > p+5 {
			// 规整形：len 字节有效（≥1，值以 NUL 收尾）。
			out := append([]byte(nil), frame[:p]...)
			out = append(out, byte(index), byte(index>>8), byte(len(want)+1), 0x01, 0x00)
			out = append(out, want...)
			out = append(out, 0)
			out = append(out, frame[p+5+n:]...)
			return out
		}
		// mangled 形（模板 Resource 实测）：idx 后 8 字节内找 0x01 标记
		//（单字节，无 00 尾随——与干净帧的 01 00 双标记不同），其后的
		// NUL 结尾串整段替换。
		for q := p + 3; q <= p+8 && q+1 < len(frame); q++ {
			if frame[q] != 0x01 {
				continue
			}
			s := q + 1
			if s < len(frame) && frame[s] == 0x00 {
				s++
			}
			end := s
			for end < len(frame) && frame[end] != 0 {
				end++
			}
			if end >= len(frame) || end == s {
				continue
			}
			out := append([]byte(nil), frame[:s]...)
			out = append(out, want...)
			out = append(out, 0)
			out = append(out, frame[end+1:]...)
			return out
		}
	}
	return frame
}
