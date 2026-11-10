package main

import (
	_ "embed"
	"encoding/binary"
	"math"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

//go:embed npc_detail_0x28_template.bin
var npcDetail0x28Template []byte

// 0x28 NPC 详情帧（0x47 家族）帧头布局——c.pcapng 21 帧双模板逐字节定版
// （tmp/op28_analysis，2026-11-10）：
//
//	[0]      0x28  子操作码
//	[1]      0x47  家族标记（0x47 族：0x28 详情 / 0x29 批量 / 0x2A 行模板）
//	[2:6]    u32   len-1（NPC 233=234-1、玩家 3187=3187-1，恒成立）
//	[6:10]   u32   模板号（同模板同长同值；如 290/294/257…，非全局 serial）
//	[10:13]  0
//	[13]     状态位（0x91 常见；0xc4/0xc8/0x81 见于 0x15x-owner 族）
//	[14:18]  obj   [18:22] owner   [22:38] x/y/z/orient 四 f32
//	[38:…]   记录块（[u16 idx][值]，串=[u8 len][01 00][len 字节含 NUL]，
//	         len 通常=strlen+1；与 0x09 表版本无关的自记录方言）
//
// 探针史（2026-11-10 真机三轮+第四轮定案）：
//   - =1 替换版三轮被拒：根因是 serial 写进 [1:5] 覆写了 0x47 家族标记
//     （自损帧，客户端按错家族解析报 size/property error）。
//   - =3 正帧版（正确帧头+obj/owner/变换替换、0x0D 前下发）：真机实证
//     **客户端崩溃**——通道被消费（对比 =1 的静默拒收），但记录块仍是
//     官服模板 NPC 的身份（State="ty_bukuai_guard" 动作/ConfigID=
//     monm031300/icon 路径）套在本服对象上，身份冲突即崩。
//   - 寻路假设证伪：不开 0x28 时地图点击寻路正常工作——0x28 不是寻路
//     数据挂载的触发器。
//   结论：模板重发路线退役（探针已移除防再崩）；0x28 只能走 P3
//   按对象生成记录块（npc_detail_frame 生成器，docs/官服登录帧对照
//   与缺口清单.md §14.2/§14.3-P3）。

// npcDetail0x28PatchFrame 以官服模板为底，仅替换 [14:22] obj/owner 与
// [22:38] 变换四浮点；帧头与记录块保持原样。**不得直接下发**（记录块
// 身份冲突真机实证崩溃）——P3 生成器在此基础上重写记录块字段后使用。
func npcDetail0x28PatchFrame(objectID, ownerID uint32, transform worldcore.Transform) []byte {
	frame := append([]byte(nil), npcDetail0x28Template...)
	binary.LittleEndian.PutUint32(frame[14:], objectID)
	binary.LittleEndian.PutUint32(frame[18:], ownerID)
	binary.LittleEndian.PutUint32(frame[22:], math.Float32bits(transform.X))
	binary.LittleEndian.PutUint32(frame[26:], math.Float32bits(transform.Y))
	binary.LittleEndian.PutUint32(frame[30:], math.Float32bits(transform.Z))
	binary.LittleEndian.PutUint32(frame[34:], math.Float32bits(transform.Orient))
	return frame
}
