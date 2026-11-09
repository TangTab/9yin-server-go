package main

import (
	_ "embed"
	"encoding/binary"
	"log"
	"os"
	"strings"
	"sync/atomic"
)

//go:embed npc_detail_0x28_template.bin
var npcDetail0x28Template []byte

// npcDetailProbeEnabled 控制 0x28 NPC 详情帧的模板镜像实验
// （NINEYIN_NPC_0X28_PROBE=1 开启，默认关）。
//
// 背景（2026-11-10 官服客户端寻路排查）：官服 NPC 物化主力是 0x28
// （进场 21 帧+移动中持续发，与 0x0D 同客户端 handler sub_1060DBC0），
// 本服此前零下发。官服客户端的寻路引擎已激活（C2S 0xA0000F01 起跑帧
// 出现）但路径计算为空——假设 0x28 触发客户端按需挂载 map_path.package
// 并初始化 path_finding。本实验用官服 234B 小帧做字节模板，仅替换
// [1:5] serial（自增）/[14:18] obj/[18:22] owner；**坐标块与属性区
// 仍是模板 NPC 的值**（ty_bukuai_guard/monm031300——真机表现会错乱，
// 纯诊断用：验证 0x28 是否为寻路数据挂载的触发器）。若假设成立，
// 下一步逆向 0x28 的坐标块与属性区生成真实帧。
var npcDetailSerial atomic.Uint32

func npcDetailProbeEnabled() bool {
	return strings.TrimSpace(os.Getenv("NINEYIN_NPC_0X28_PROBE")) == "1"
}

// npcDetail0x28Frame 生成基于模板的 0x28 帧（obj/own/serial 替换版）。
func npcDetail0x28Frame(objectID, ownerID uint32) []byte {
	frame := append([]byte(nil), npcDetail0x28Template...)
	binary.LittleEndian.PutUint32(frame[1:], 0x0000E900+npcDetailSerial.Add(1))
	binary.LittleEndian.PutUint32(frame[14:], objectID)
	binary.LittleEndian.PutUint32(frame[18:], ownerID)
	return frame
}

// writeNPCDetailProbeFrame 诊断路径：materialize NPC 后发模板 0x28。
// **仅 NINEYIN_NPC_0X28_PROBE=2（原样重发官服帧，零字节修改）有效**——
// =1 替换版已移除（06:45/06:47/06:52 三轮真机实证被拒：size/property
// error ×123 帧/轮，拒收帧轰炸污染客户端对象状态）。
func writeNPCDetailProbeFrame(conn sceneMessageConnection, objectID, ownerID uint32, remote string) {
	if strings.TrimSpace(os.Getenv("NINEYIN_NPC_0X28_PROBE")) != "2" {
		return
	}
	if err := conn.WriteFrame(npcDetail0x28Template); err != nil {
		log.Printf("%s: npc 0x28 probe frame: %v", remote, err)
	}
}
