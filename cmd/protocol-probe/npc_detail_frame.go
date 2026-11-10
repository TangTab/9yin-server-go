package main

import (
	_ "embed"
	"encoding/binary"
	"log"
	"math"
	"os"
	"strings"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

//go:embed npc_detail_0x28_template.bin
var npcDetail0x28Template []byte

// 0x28 NPC 详情帧（0x47 家族）帧头布局——c.pcapng 21 帧双模板逐字节定版
// （tmp/op28_analysis，2026-11-10）：
//
//	[0]      0x28  子操作码
//	[1]      0x47  家族标记（0x47 族：0x28 详情 / 0x29 批量 / 0x2A 行模板）
//	[2:6]    u32   len-1（NPC 233=234-1、玩家 3186=3187-1，恒成立）
//	[6:10]   u32   模板号（同模板同长同值；如 290/294/257…，非全局 serial）
//	[10:13]  0
//	[13]     状态位（0x91 常见；0xc4/0xc8/0x81 见于 0x15x-owner 族）
//	[14:18]  obj   [18:22] owner   [22:38] x/y/z/orient 四 f32
//	[38:…]   记录块（[u16 idx][值]，串=[u8 len][01 00][len 字节含 NUL]，
//	         len 通常=strlen+1；与 0x09 表版本无关的自记录方言）
//
// 三轮真机被拒根因（41d3292/c1cd1b9/9c3972d 全部 =1 替换版）：=1 把
// serial u32 写进 [1:5]，恰好覆写 [1]=0x47 家族标记（[2:6] len-1 未动），
// 客户端按错家族解析即报 size/property error——被拒的是自损帧，不是
// 0x28 通道本身。=2 原样重发从未上过真机；官服时序为 0x28 先于成对
// 0x0D（#1776→#1777），=3 按官服顺序在 0x0D 前下发。

// npcDetailProbeMode 返回 NINEYIN_NPC_0X28_PROBE 探针档位：
//
//	=2 官服帧原样重发（零字节修改，消元对照；置于 0x0D 后）
//	=3 正确帧头 + obj/owner/变换替换（诊断主模式；置于 0x0D 前，官服同序）
func npcDetailProbeMode() string {
	return strings.TrimSpace(os.Getenv("NINEYIN_NPC_0X28_PROBE"))
}

// npcDetail0x28PatchFrame 以官服模板为底，仅替换 [14:22] obj/owner 与
// [22:38] 变换四浮点；帧头（[1]=0x47/[2:6]=len-1/[6:10] 模板号/[13] 状态位）
// 与 [38:] 记录块保持官服原样。记录块仍是模板 NPC 的属性（ty_bukuai_guard/
// monm031300），真机表现会错乱——诊断用：验证 0x28 通道是否被接受、
// 是否触发客户端寻路数据挂载。
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

// writeNPCDetailProbeFrameBeforeV3 =3 档：0x0D 物化前发正确帧头的替换版
// （官服 #1776→#1777 同序）。仅当帧头不变量成立时下发，防模板损坏直传。
func writeNPCDetailProbeFrameBeforeV3(conn sceneMessageConnection, objectID, ownerID uint32, transform worldcore.Transform, remote string) {
	if npcDetailProbeMode() != "3" {
		return
	}
	if len(npcDetail0x28Template) < 0x26 || npcDetail0x28Template[0] != 0x28 || npcDetail0x28Template[1] != 0x47 ||
		binary.LittleEndian.Uint32(npcDetail0x28Template[2:]) != uint32(len(npcDetail0x28Template)-1) {
		log.Printf("%s: npc 0x28 probe v3 skipped: template header invariants broken", remote)
		return
	}
	if err := conn.WriteFrame(npcDetail0x28PatchFrame(objectID, ownerID, transform)); err != nil {
		log.Printf("%s: npc 0x28 probe v3 frame: %v", remote, err)
	}
}

// writeNPCDetailProbeFrame =2 档：官服帧原样重发（零字节修改），消元对照。
// obj/owner/坐标全是官服原值——仅验证「同字节回放是否被接受」。
func writeNPCDetailProbeFrame(conn sceneMessageConnection, objectID, ownerID uint32, remote string) {
	if npcDetailProbeMode() != "2" {
		return
	}
	if err := conn.WriteFrame(npcDetail0x28Template); err != nil {
		log.Printf("%s: npc 0x28 probe frame: %v", remote, err)
	}
}
