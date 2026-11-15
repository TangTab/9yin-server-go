package main

import (
	"encoding/binary"
	"log"
	"math"
	"os"
	"strings"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 帧替换档（B 线第十六轮，2026-11-15）：NINEYIN_REPLAY_NATIVE_FRAMES=1。
// bin 骨架（replay_p1_slim/replay_p2_slim）里已有原生生成器的帧就地替换
// 为代码生成（值取我方权威状态/结构常量），其余帧透传并打 UNKNOWN 标注
// ——标注即「仍在镜像外服数据」的欠账清单，逐个销项后 bin 退役。
//
// 全景盘点依据 tools/replay_slim_report.py（335 帧全部在 c.pcapng
// cap#110..1784 逐字节定位=纯子序列透传）。分派总表：
//
//	P1（13 帧，按序位）：
//	 #0/#2  0x27 sync 压缩簇    → UNKNOWN-1 透传（#0 明文未解/#2 已解=开关
//	                               sync 族但无编码器；LZ 帧不可部分改写）
//	 #1     0x1E m4 beginmenu   → UNKNOWN-2 透传（活动菜单编码器缺）
//	 #3/#4  0x10 idx1405        → UNKNOWN-3 透传（TodayCapitalLeave1 双帧
//	                               8B 值域 360000/350976，编码未定案）
//	 #5     0x0B 进场           → 原生 sendEntryScene（官方 4 属性形已对齐）
//	 #6     0x28 化身档案       → 原生换代=我方 41 字段+技能行生成档
//	                               （2026-11-15 01:3x「待机动作+脸部装饰」
//	                               两残留影子的根治；换心时代「只剩头」
//	                               缺口已由恒发 0x10 外观帧补上；降级=
//	                               透传+posA 改写，走查=docs/official/
//	                               外服玩家档案0x28全量帧.md）
//	 #7/#9  0x1E m2 asyn_time   → UNKNOWN-5 透传（载荷对 98/274 语义未定）
//	 #8     0x10 idx248         → UNKNOWN-6 透传（SafeLoginFlag=6 值语义未定）
//	 #11    0x10 idx694         → 原生 sceneObjectIntProperty（值=我方
//	                               qinggong.Point，非镜像）
//	 #10/#12 0x38 双时间戳帧    → UNKNOWN-7 透传（0x74d488c2×2 语义未定）
//
//	P2（322 帧，按 op）：
//	 0x15 ×30 视图容器          → 原生 serverCreateViewWithProperties（结构
//	                               常量表 officialP2ViewCreateSet，字节同形）
//	 0x1F self Location         → 原生 serverLocation（存档落点，替代
//	                               AlignSelfPos 改写）
//	 0x2A ×15 官方装备外观      → DROP-8 停发（2026-11-15 00:5x 真机
//	                               「影子覆盖 DB 装备」根因：官服玩家全套
//	                               穿着帧压在我方数据库装备之上；我方外观
//	                               走 0x10 属性通道，替换档恒发压影子）
//	 0x1E m16 ×62 removerow     → UNKNOWN-9 透传（删的是官服 AOI npc 行，
//	                               我方无行可删；保留防客户端预置行）
//	 0x1E m4/m2/m3/m5           → UNKNOWN-2/5/10/11 透传（同 P1 归类；
//	                               m3 含 DomainJiuYinZhi，我方 jyz 方言不同）
//	 0x10 ×115 官方玩家属性值   → MIRROR-12 透传（我方权威簇已覆盖大头，
//	                               105 idx 全量对账待真机再切）
//	 0x29 gemskill 预载         → UNKNOWN-13 透传
//	 0x27 trs 令牌簇            → UNKNOWN-14 透传
//	 0x38 ×2                    → UNKNOWN-7 透传

// replayNativeFramesEnabled 帧替换档开关。
func replayNativeFramesEnabled() bool {
	return os.Getenv("NINEYIN_REPLAY_NATIVE_FRAMES") == "1"
}

// officialViewSpec 一个 0x15 视图容器的结构参数（外服抓包定案，容量是
// 客户端容器结构而非玩家数据；8 组「主页 cap90+BaseCap18/扩展页 cap2」
// 与我方 bagMainPages/bagExtPages 同构）。
type officialViewSpec struct {
	id      uint16
	cap     uint16
	baseCap bool // 附 viewInt(propBaseCap, 18)
}

// officialP2ViewCreateSet 外服 Phase B 的 30 个视图容器，按 bin 帧序
// （c.pcapng P2 #5/#62-67/#74-79/#93/#102-112/#116-120/#133/#238）。
func officialP2ViewCreateSet() []officialViewSpec {
	return []officialViewSpec{
		{116, 1, false},   // 未知用途小容器
		{47, 512, false},  // 经脉
		{48, 128, false},
		{44, 512, false},  // 内修 wuxing
		{45, 128, false},  // 任务
		{43, 512, false},  // 内功
		{40, 800, false},  // 武学
		{41, 512, false},
		{170, 10, false},
		{29, 40, false},
		{1, 52, false},    // 纸娃娃
		{64, 6, false},
		{174, 90, true}, {175, 2, false},   // 背包主页/扩展页 ×8 组
		{176, 90, true}, {177, 2, false},
		{178, 90, true}, {179, 2, false},
		{180, 90, true}, {181, 2, false},
		{2, 90, true}, {3, 2, false},
		{121, 90, true}, {122, 2, false},
		{123, 90, true}, {124, 2, false},
		{125, 90, true}, {126, 2, false},
		{159, 24, false},
		{46, 512, false},  // 轻功 QingGongRec
	}
}

// p1UnknownMarker P1 透传帧的 UNKNOWN 标注（序位→欠账说明）。
func p1UnknownMarker(idx int, frame []byte) string {
	switch idx {
	case 0, 2:
		return "UNKNOWN-1 0x27 sync 压缩簇（#0 明文未解/#2 已解但无编码器）"
	case 1:
		return "UNKNOWN-2 0x1E m4 beginmenu 活动菜单（编码器缺）"
	case 3, 4:
		return "UNKNOWN-3 0x10 idx1405 TodayCapitalLeave1（8B 值域 360000/350976 编码未定案）"
	case 6:
		return "UNKNOWN-4 化身 0x28 装备外观 opaque 段（生成器缺=最后一块数据镜像；posA 改写已有）"
	case 7, 9:
		return "UNKNOWN-5 0x1E m2 asyn_time（载荷对 98/274 语义未定）"
	case 8:
		return "UNKNOWN-6 0x10 idx248 SafeLoginFlag=6（值语义未定）"
	case 10, 12:
		return "UNKNOWN-7 0x38 双时间戳帧（0x74d488c2×2 语义未定）"
	}
	return "UNKNOWN-P1-op"
}

// nativeP1MarkUnknown P1 透传帧打标（每欠账类一次，防刷屏）。
func nativeP1MarkUnknown(idx int, frame []byte, seen map[string]bool) {
	marker := p1UnknownMarker(idx, frame)
	if seen[marker] {
		return
	}
	seen[marker] = true
	log.Printf("native-replace P1#%d op=0x%02X PASSTHROUGH [%s]", idx, frame[0], marker)
}

// nativeP1Disposition P1 帧就地替换分派。返回 native=true 时调用方应发
// useEntryScene 的进场帧（或 frames 列表），不再透传原帧。
// 帧替换档下 SWAP_SELF 的两大职责全部收编：#5 0x0B→sendEntryScene、
// #6 化身 0x28→我方生成档（nativeAvatar 非空时；装备/脸面影子的根治，
// 换心时代「只剩头」缺口已由恒发 0x10 外观帧补上）或透传+posA 对齐
//（nativeAvatar=nil 的降级路径，头部形不符则原样透传保底）。
func nativeP1Disposition(idx int, frame []byte, player *playerActor, pos role.Position, nativeAvatar []byte) (frames [][]byte, useEntryScene bool, note string, native bool) {
	switch idx {
	case 5:
		if frame[0] != 0x0B {
			return nil, false, "", false
		}
		return nil, true, "sendEntryScene 官方 4 属性形", true
	case 6:
		// 化身 0x28。nativeAvatar 非空=我方生成档直接换代（ids/posA
		// 已按当前全局嵌入，零改写）；空=外服档案透传+posA 16B 对齐
		// 存档落点（UNKNOWN-4 降级路径）。
		if frame[0] != 0x28 || len(frame) <= 1000 {
			return nil, false, "", false
		}
		if len(nativeAvatar) > 0 {
			return [][]byte{nativeAvatar}, false, "我方档案 41 字段+技能行（opaque 影子段根除；外观走恒发 0x10 帧）", true
		}
		out := append([]byte(nil), frame...)
		if posOff, ok := officialFramePosAOffset(out, playerObjectID, playerOwnerID); ok {
			for i, v := range []float32{pos.X, pos.Y, pos.Z, pos.Orient} {
				binary.LittleEndian.PutUint32(out[posOff+4*i:], math.Float32bits(v))
			}
			return [][]byte{out}, false,
				"UNKNOWN-4 化身 0x28 透传+posA 对齐存档落点（装备段生成器缺=最后一块数据镜像）", true
		}
		return [][]byte{out}, false,
			"UNKNOWN-4 化身 0x28 透传（posA 偏移未定位，零改写保底）", true
	case 11:
		// 0x10 idx694 QingGongPoint 单属性帧（外服值 150=镜像；我方值=
		// 会话轻功池当前点数，租约/秒耗同源）。
		if frame[0] != 0x10 || player == nil {
			return nil, false, "", false
		}
		propIdx := clientdata.MustPlayerProperty("QingGongPoint").Index
		f := sceneObjectIntProperty(playerObjectID, playerOwnerID, propIdx, player.qinggong.Point)
		return [][]byte{f}, false, "QingGongPoint=qinggong.Point（我方权威值）", true
	}
	return nil, false, "", false
}

// p2UnknownMarker P2 透传帧的 UNKNOWN/MIRROR 标注。
func p2UnknownMarker(frame []byte) string {
	switch frame[0] {
	case 0x2A:
		return "DROP-8 官方装备外观 0x2A 停发（影子覆盖 DB 装备根因；外观改由我方 0x10 属性通道）"
	case 0x10:
		return "MIRROR-12 官方玩家属性值 0x10（我方权威簇覆盖大头，105 idx 对账待真机）"
	case 0x29:
		return "UNKNOWN-13 0x29 预载（gemskill_052 族）"
	case 0x27:
		return "UNKNOWN-14 0x27 压缩令牌簇（trs）"
	case 0x38:
		return "UNKNOWN-7 0x38 双时间戳帧"
	case 0x1E:
		if len(frame) >= 3 {
			switch binary.LittleEndian.Uint16(frame[1:]) {
			case 16:
				return "UNKNOWN-9 0x1E m16 virtualrec_removerow（删官服 AOI npc 行；保留防客户端预置行）"
			case 4:
				return "UNKNOWN-2 0x1E m4 beginmenu 活动菜单"
			case 2:
				return "UNKNOWN-5 0x1E m2 asyn_time"
			case 3:
				return "UNKNOWN-10 0x1E m3 原生域（含 DomainJiuYinZhi，我方 jyz 方言不同）"
			case 5:
				return "UNKNOWN-11 0x1E m5 addtitle（编码器缺）"
			}
		}
		return "UNKNOWN-2x 0x1E custom"
	}
	return "UNKNOWN-P2-op"
}

// replayPhase2NativeReplace Phase B 帧替换走查：0x15/0x1F 原生生成，
// 0x2A DROP-8 停发，其余透传+标注。raw/label 来自 replayPhase2FramesRaw
//（env 覆盖 > exe 内嵌）。返回发送帧数。
func replayPhase2NativeReplaceBytes(conn sceneMessageConnection, raw []byte, label string, obj, own uint32, pos role.Position) int {
	views := officialP2ViewCreateSet()
	viewIdx := 0
	seen := map[string]bool{}
	nativeCount, passCount, dropCount := 0, 0, 0
	sent := 0
	rest := raw
	for len(rest) >= 4 {
		fl := int(binary.LittleEndian.Uint32(rest))
		if fl <= 0 || fl > len(rest)-4 || fl > 0x10000 {
			break
		}
		frame := rest[4 : 4+fl]
		rest = rest[4+fl:]
		switch {
		case frame[0] == 0x2A:
			// DROP-8：官方装备外观停发（影子源头）。不写帧、只计 dropped，
			// 标注打一次。
			marker := p2UnknownMarker(frame)
			if !seen[marker] {
				seen[marker] = true
				log.Printf("native-replace P2 op=0x%02X DROP [%s]", frame[0], marker)
			}
			dropCount++
		case frame[0] == 0x15 && viewIdx < len(views):
			spec := views[viewIdx]
			viewIdx++
			binView := binary.LittleEndian.Uint16(frame[1:])
			if binView != spec.id {
				// bin 演化错位兜底：透传原帧并告警（继续消费表项重同步）。
				if !seen["VIEW-ORDER"] {
					seen["VIEW-ORDER"] = true
					log.Printf("native-replace P2 0x15 order mismatch @viewIdx=%d bin=%d table=%d; passthrough", viewIdx-1, binView, spec.id)
				}
				if err := conn.WriteFrame(frame); err != nil {
					log.Printf("replay %s: %v", label, err)
					return sent
				}
				passCount++
				sent++
				continue
			}
			var props []serverViewProperty
			if spec.baseCap {
				props = append(props, viewInt(propBaseCap, 18))
			}
			out, verr := serverCreateViewWithProperties(officialToServerViewSpec(spec), props)
			if verr != nil {
				log.Printf("native-replace P2 view %d encode: %v", spec.id, verr)
				return sent
			}
			if err := conn.WriteFrame(out); err != nil {
				log.Printf("replay %s: %v", label, err)
				return sent
			}
			nativeCount++
			sent++
		case frame[0] == 0x1F && len(frame) >= 9 && binary.LittleEndian.Uint32(frame[1:]) == obj:
			out := serverLocation(obj, own, worldcore.Transform{X: pos.X, Y: pos.Y, Z: pos.Z, Orient: pos.Orient})
			if err := conn.WriteFrame(out); err != nil {
				log.Printf("replay %s: %v", label, err)
				return sent
			}
			nativeCount++
			sent++
		default:
			marker := p2UnknownMarker(frame)
			if !seen[marker] {
				seen[marker] = true
				log.Printf("native-replace P2 op=0x%02X PASSTHROUGH [%s]", frame[0], marker)
			}
			if err := conn.WriteFrame(frame); err != nil {
				log.Printf("replay %s: %v", label, err)
				return sent
			}
			passCount++
			sent++
		}
	}
	log.Printf("native-replace P2 summary: native=%d passthrough=%d dropped=%d (unknown classes=%d) file=%s",
		nativeCount, passCount, dropCount, len(seen), strings.TrimSpace(label))
	return sent
}

// officialToServerViewSpec officialViewSpec → 0x15 编码器参数（结构常量）。
func officialToServerViewSpec(spec officialViewSpec) serverViewSpec {
	return serverViewSpec{ID: spec.id, Capacity: spec.cap}
}
