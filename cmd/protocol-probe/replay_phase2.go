package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"os"
	"strings"
	"sync/atomic"

	"github.com/local/9yin-go-server/cmd/protocol-probe/replaydata"
	"github.com/local/9yin-go-server/internal/role"
)

// B 线两阶段回放（2026-11-14 第十轮）。外服抓包时序定案（c/登陆/登陆2/
// 初始账号 四份交叉验证）：0x0B+化身 0x28+首爆发在进场即发；客户端读条
// 3~10s 完成后回发单字节 0x09，服务端 ~65ms 后发第二爆发（m561/m681 族、
// op15 建视图 + 0x18 视图行 + 0x2A 外观），再 ~1s 起按 1s tick 推 AOI
// 物化簇（0x28+0x0D）。把全序列在单点灌入时，Phase A 里的外服 0x0B 会
// 让客户端进入新一轮读条，其后帧全部落在读条窗口——场景对象表未建，
// AddObject 查表失败全丢。因此拆两段对齐官方相位：
//   Phase A = NINEYIN_REPLAY_FRAMES_FILE：现有 clientActivity 钩子发送
//             （0x0B+化身 0x28+首爆发）；
//   Phase B = NINEYIN_REPLAY_PHASE2_FILE：客户端对 Phase A 的 0x0B 回发
//             的下一个显式 0x09 ClientReady 时发送（第二爆发+物化簇）。
var replayPhase2Armed atomic.Bool

// replayReplaceEntryEnabled：B 线第十一轮 wholesale 判别开关
// （NINEYIN_REPLAY_REPLACE_ENTRY=1）。真机 trace 定案（2026-11-14
// 13:11 会话）：我方 city05 0x0B 进场未走完时灌入外服 0x0B = 双进场
// 结构性毒药——unload_terrain 与 entry_stage_main finish 之间的窗口
// 吞掉化身帧，客户端报 game_visual:GetPlayer not exist。本开关让进场
// 整体外服化：角色选定后我方进场帧（0x0B/0x0D/0x1F/0x10/背包/解锁/
// 开图/挂件/场景对象物化）全不发，改发 NINEYIN_REPLAY_FRAMES_FILE
// （Phase A），客户端 0x09 后只发 Phase B——复刻外服登录的单次进场
// 结构。判据：通了=缺帧/相位假说终局；不通=缺口在 S2C 序列之外。
func replayReplaceEntryEnabled() bool {
	return os.Getenv("NINEYIN_REPLAY_REPLACE_ENTRY") == "1"
}

// replayNativeOverlayEnabled：wholesale 骨架上的我方数据覆盖开关
//（NINEYIN_REPLAY_NATIVE_OVERLAY=1）。放开两处压制：显式 0x09 后的
// 我方 ClientReady 推送簇（AutoTrace/轻功/个性配置/键位/外观/邮件/
// 坐骑/兵录/九阴志/开关树）照常下发；clientActivity 恢复我方场景对
// 象物化（仅跳过回放 env 钩子——Phase A 已在进场位发过）。镜像数据
// 先落、我方数据后到覆盖，逐步把 bin 拐杖里可替代的部分替换成代码。
func replayNativeOverlayEnabled() bool {
	return os.Getenv("NINEYIN_REPLAY_NATIVE_OVERLAY") == "1"
}

// replayExplicitReadySeen：本连接是否已观察到显式 0x09（stage_main
// ClientReady）。overlay 档下 clientActivity 会在兼容晋升期（读条窗口，
// m301 触发）提前到达——此时物化的帧会被客户端丢弃且「已物化」记账
// 生效，读条后 delta=0 永不重发（15:04 会话 NPC 全丢的根因）。因此
// 物化必须推迟到显式 0x09 之后的首个 custom。
var replayExplicitReadySeen atomic.Bool

// replayEntryFramesPath：Phase A 文件路径（wholesale 模式下即进场帧）。
func replayEntryFramesPath() string {
	return strings.TrimSpace(os.Getenv("NINEYIN_REPLAY_FRAMES_FILE"))
}

// officialFramePosAOffset 在外服透传帧头部定位变换块 posA（X/Y/Z/
// Orient 4×f32）的字节偏移。头部形（c#124 化身 0x28 实测 obj@14/own@18/
// posA@22；Phase B 0x1F Location 实测 obj@1/own@5/posA@9——两形同为
// … obj u32, own u32, posA 紧邻）：obj/own 限定帧头 64 字节内首个紧邻
// 匹配对，防长帧字段区的值碰撞；形不符返回 ok=false（调用方原样透传
// 保底，零改写——身份借用的透传帧任何多余改动都是回归风险）。
func officialFramePosAOffset(frame []byte, obj, own uint32) (int, bool) {
	limit := 64
	if len(frame) < limit {
		limit = len(frame)
	}
	head := frame[:limit]
	objOff := bytes.Index(head, binary.LittleEndian.AppendUint32(nil, obj))
	if objOff < 0 || objOff+8 > limit {
		return 0, false
	}
	if binary.LittleEndian.Uint32(frame[objOff+4:]) != own {
		return 0, false
	}
	if objOff+8+16 > len(frame) {
		return 0, false
	}
	return objOff + 8, true
}

// replayFramesPackAlignSelfPos：replayFramesPack 的 self 0x1F 对齐变体。
// p2_slim 内有一帧外服 0x1F Location（25B 无 mode 字节：0x1f+obj@1+
// own@5+posA@9=出生点）在显式 0x09 后随 Phase B 下发——会把进场位化身
// 0x28 的 posA 改写成果拽回外服出生点（重进位置修复的漏网帧）。逐帧
// 把 self 0x1F 的 posA 对齐存档落点，其余帧原样。
func replayFramesPackAlignSelfPos(conn sceneMessageConnection, path string, obj, own uint32, pos role.Position) int {
	return replayFramesPackTransform(conn, path, selfPosAlignTransform(obj, own, pos))
}

// replayFramesPackAlignSelfPosBytes 字节版（内嵌/env 已读入的包体）。
func replayFramesPackAlignSelfPosBytes(conn sceneMessageConnection, raw []byte, label string, obj, own uint32, pos role.Position) int {
	return replayFramesPackTransformBytes(conn, raw, label, selfPosAlignTransform(obj, own, pos))
}

// selfPosAlignTransform self 0x1F 的 posA 对齐变换（两版共用）。
func selfPosAlignTransform(obj, own uint32, pos role.Position) func([]byte) []byte {
	return func(frame []byte) []byte {
		if frame[0] != 0x1F {
			return frame
		}
		posOff, ok := officialFramePosAOffset(frame, obj, own)
		if !ok {
			return frame
		}
		out := append([]byte(nil), frame...)
		for i, v := range []float32{pos.X, pos.Y, pos.Z, pos.Orient} {
			binary.LittleEndian.PutUint32(out[posOff+4*i:], math.Float32bits(v))
		}
		return out
	}
}

// replayFramesPack 原样回放 [u32 len][frame]×N 打包文件（tools 打包器
// 产出）；裸单帧文件回退。返回发送帧数。
func replayFramesPack(conn sceneMessageConnection, path string) int {
	return replayFramesPackTransform(conn, path, nil)
}

// replayEntryFramesRaw Phase A 帧源：env 覆盖（换包实验）> exe 内嵌
// bin（replaydata）。2026-11-15 起生产 bat 不再设 FRAMES_FILE——骨架
// 数据已进 exe，外置路径只剩实验用途。
func replayEntryFramesRaw() (raw []byte, label string) {
	if p := replayEntryFramesPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			return b, p
		} else {
			log.Printf("replay entry file %s unreadable (%v); using embedded bin", p, err)
		}
	}
	return replaydata.Phase1, "embedded:replay_p1_slim.bin"
}

// replayPhase2FramesRaw Phase B 帧源：同 replayEntryFramesRaw 口径。
func replayPhase2FramesRaw() (raw []byte, label string) {
	if p := strings.TrimSpace(os.Getenv("NINEYIN_REPLAY_PHASE2_FILE")); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			return b, p
		} else {
			log.Printf("replay phase-2 file %s unreadable (%v); using embedded bin", p, err)
		}
	}
	return replaydata.Phase2, "embedded:replay_p2_slim.bin"
}

// replayFramesPackTransform 逐帧回放打包文件，transform 非 nil 时每帧
// 先经其改写（返回切片可直接发送；返回原帧=零改写）。裸单帧文件回退。
func replayFramesPackTransform(conn sceneMessageConnection, path string, transform func([]byte) []byte) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("replay file %s unreadable: %v", path, err)
		return 0
	}
	return replayFramesPackTransformBytes(conn, raw, path, transform)
}

// replayFramesPackTransformBytes 字节版走查（内嵌/env 已读入的包体）。
func replayFramesPackTransformBytes(conn sceneMessageConnection, raw []byte, label string, transform func([]byte) []byte) int {
	sent := 0
	rest := raw
	for len(rest) >= 4 {
		fl := int(binary.LittleEndian.Uint32(rest))
		if fl <= 0 || fl > len(rest)-4 || fl > 0x10000 {
			break
		}
		frame := rest[4 : 4+fl]
		if transform != nil {
			frame = transform(frame)
		}
		if err := conn.WriteFrame(frame); err != nil {
			log.Printf("replay %s: %v", label, err)
			break
		}
		sent++
		rest = rest[4+fl:]
	}
	// 兼容裸单帧文件
	if sent == 0 && len(raw) > 0 {
		if err := conn.WriteFrame(raw); err != nil {
			log.Printf("replay %s: %v", label, err)
			return 0
		}
		sent = 1
	}
	return sent
}
