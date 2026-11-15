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
//	 #0/#2  0x27 sync 压缩簇    → UNKNOWN-1 透传（#0 明文未解；#2 已解=
//	                               m414 "sync" 官方开关树 701 id 表——官方
//	家族客户端口径，与我方 203 表（fork exe 源）交集仅 63；LZ 帧不可部分改写）
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
//	 #7/#9  0x1E m98 refresh_origin → UNKNOWN-5 透传（custom_refresh_origin
//	                               id=274；P2 尾另有 m2 asyn_time µs 时戳）
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
//	                               「影子覆盖 DB 装备」根因：外服玩家全套
//	                               穿着帧压在我方数据库装备之上；我方外观
//	                               走 0x10 属性通道，替换档恒发压影子）
//	 0x1E argc16 ×62 NPC 行     → UNKNOWN-9 透传（m229 sub130 官方 AOI
//	                               NPC 行：npc\<模型>+坐标+缩放+场景 id+
//	                               名字——旧「m16 removerow」系 argc 误读 id）
//	 0x1E argc2/3/4/5 族        → UNKNOWN-2/5/10/11 透传（m4 beginmenu/
//	                               m386 限时窗/m416 DomainJiuYinZhi/m294
//	                               对钟/m681 令牌/m607 账号名等，见解析文档）
//	 0x10 ×115 官方玩家属性值   → 移动/跳跃族原生改写（MOTION-15：基础值=
//	                               我方权威生成器，Add 族=外服轻功等级加成组
//	                               常量；镜像 Add 帧本是外服 5 级轻功玩家合法
//	                               状态——「跳跃异常」真根因是瘦身把外服 34 帧
//	                               基础物理档案（JumpSpeed=5.5/Gravity=9.8/
//	                               三跳基值 6.0…）当数据行剥离+wholesale 停发
//	                               我方同款流，客户端无本地默认缺省 0，跳跃
//	                               Z 初速=0；PHYSICS-16=Phase B 后补发我方
//	                               sendPlayerPhysicsLoginStream，main.go 同点）
//	                               ；其余 idx=ATTR-SHADOW 真值优先/影子兜底
//	                               （2026-11-15：有我方值源的行就地换值，
//	                               无源行保留影子默认值；481/624 占位常量
//	                               显式保留影子——replay_attr_mirror.go）
//	 0x29 gemskill 预载         → UNKNOWN-13 透传
//	 0x27 trs 令牌簇            → UNKNOWN-14 透传
//	 0x38 ×2                    → UNKNOWN-7 透传

// replayNativeFramesEnabled 帧替换档开关。
func replayNativeFramesEnabled() bool {
	return os.Getenv("NINEYIN_REPLAY_NATIVE_FRAMES") == "1"
}

// replayBisectMode 帧级二分旋钮（NINEYIN_REPLAY_BISECT，2026-11-15
// 「重登回走」定位）：按轮丢弃回放帧集合/静默我方推送，重启（不重编）
// 即可切换。支持逗号分隔多值。
//
//	p2custom —— 丢 Phase B 全部 0x1E 透传 custom（157 帧）
//	p2mirror —— 丢 Phase B 全部 0x10 官方属性镜像
//	p2all    —— 上两者
//	p1pass   —— 丢 Phase A 全部透传帧（仅留原生生成）
//	allreplay—— p1pass+p2all（过猛：实测进不了游戏）
//	ours-switch / ours-customizing / ours-physics / ours-keybind /
//	ours-logininfo —— 静默我方对应推送簇
func replayBisectMode() string {
	return strings.TrimSpace(os.Getenv("NINEYIN_REPLAY_BISECT"))
}

func bisectActive(tag string) bool {
	for _, v := range strings.Split(replayBisectMode(), ",") {
		if strings.TrimSpace(v) == tag {
			return true
		}
	}
	return false
}

// bisectDropsP2Custom / bisectDropsP2Mirror / bisectDropsP1Pass 二分轮次判定。
func bisectDropsP2Custom() bool {
	return bisectActive("p2custom") || bisectActive("p2all") || bisectActive("allreplay")
}

func bisectDropsP2Mirror() bool {
	return bisectActive("p2mirror") || bisectActive("p2all") || bisectActive("allreplay")
}

func bisectDropsP1Pass() bool {
	return bisectActive("p1pass") || bisectActive("allreplay")
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
		{116, 1, false},  // 未知用途小容器
		{47, 512, false}, // 经脉
		{48, 128, false},
		{44, 512, false}, // 内修 wuxing
		{45, 128, false}, // 任务
		{43, 512, false}, // 内功
		{40, 800, false}, // 武学
		{41, 512, false},
		{170, 10, false},
		{29, 40, false},
		{1, 52, false}, // 纸娃娃
		{64, 6, false},
		{174, 90, true}, {175, 2, false}, // 背包主页/扩展页 ×8 组
		{176, 90, true}, {177, 2, false},
		{178, 90, true}, {179, 2, false},
		{180, 90, true}, {181, 2, false},
		{2, 90, true}, {3, 2, false},
		{121, 90, true}, {122, 2, false},
		{123, 90, true}, {124, 2, false},
		{125, 90, true}, {126, 2, false},
		{159, 24, false},
		{46, 512, false}, // 轻功 QingGongRec
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
		return "UNKNOWN-5 0x1E m98 custom_refresh_origin(id=274)"
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
// （nativeAvatar=nil 的降级路径，头部形不符则原样透传保底）。
func nativeP1Disposition(idx int, frame []byte, player *playerActor, pos role.Position, nativeAvatar []byte) (frames [][]byte, useEntryScene bool, note string, native bool) {
	switch idx {
	case 0, 2:
		// DROP-LZ：mode-0x47 外池透传的必垃圾帧（2026-11-15 午后定案）。
		// LZ 池是连接全局的 4096 槽哈希表（last-writer-wins）：官方这三帧
		//（#0 out9099 / #2 m414 sync 701 开关 / P2#6 0x29 gemskill）在真机
		// 客户端用「我方前置流注册的池」解压——命中内容全错但解压过程
		// 合法=每次登录灌入 ~9.3KB 垃圾内层帧，随机字节可命中任意 op
		//（0x10 随机 idx 属性/0x1E 随机子号 lua handler）——「重登回走态」
		// 等怪状态的头号嫌疑。官方语义已由我方 414 sync(203)/gem 视图
		// 等原生通道替代或本就未实现，丢弃零损失（去镜像 UNKNOWN-1 销项）。
		if frame[0] != 0x27 || frame[1] != 0x47 || len(frame) < 2 {
			return nil, false, "", false
		}
		return nil, false, "DROP-LZ 0x27 mode-0x47 外池帧（真机字典错位=垃圾内层；官方开关树由我方 414 sync 替代）", true
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
		// self 帧走 ATTR-SHADOW 换值（replay_attr_mirror.go），此处只剩
		// 场景/对象域与换值失败回落帧。
		return "MIRROR-12 官方属性值 0x10（非 self 域或影子兜底透传）"
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
				// 注：u16@1 是 argc 非 msg id——「m16 removerow」系旧误读；
				// 62 帧实为 m229 sub130 官方 AOI NPC 行（npc\<模型>+坐标+
				// 缩放串+场景 id 703/701+索引+名字，君子堂域）。
				return "UNKNOWN-9 0x1E argc16 官方 AOI NPC 行（m229 sub130；保留防客户端预置行）"
			case 4:
				return "UNKNOWN-2 0x1E argc4（m4 beginmenu/m386 限时窗/m681 令牌等）"
			case 2:
				return "UNKNOWN-5 0x1E argc2（m2 asyn_time/m98 refresh_origin/m607 账号名等）"
			case 3:
				return "UNKNOWN-10 0x1E argc3（m416 DomainJiuYinZhi/m229 NPC 行等）"
			case 5:
				return "UNKNOWN-11 0x1E argc5（m5 addtitle/m294 对钟等）"
			}
		}
		return "UNKNOWN-2x 0x1E custom"
	}
	return "UNKNOWN-P2-op"
}

// nativeP2SelfMotionOverrides P2 透传帧里 self 0x10 移动/跳跃族的权威值表
// （MOTION-15）。wholesale 进场停发我方 0x0D 大档与物理档案单帧流，
// ClientReady 战斗面板簇（officialCombatPanelProp）也不含该族——外服镜像
// 值落地后无任何我方帧覆盖。值表不发明新常量，全部取自既有权威生成器：
// baseMotionProperties（Gravity/DropHeightPub）、qinggongMotionProperties
// （跳/冲刺族基线）、motion 快照（MoveSpeed/RunSpeed）、轻功池快照
// （QingGongPoint/Max，P1#11 同源）。
//
// Add 族按「外服轻功等级加成组」取值（9yin-go-server2 qinggong_jump_physics.go
// 外服 5 级轻功抓拍定案，随 View=46 下发）：SndJumpSpeedAdd=5.1、
// Thd/DriftJumpSpeedAdd=3.0、AirRushDistAdd=5.0、DropHeightPub 10→24。
// bin 镜像值（4.8/3.0/3.0/5.0/24）即被捕获玩家的同组状态——不是坏数据，
// 但按去镜像原则改写为我方常量。二段跳 Z 初速=SndJumpSpeed(6)+Add(5.1)
// =11.1 m/s（server2 驱动链注释同源）。
func nativeP2SelfMotionOverrides(player *playerActor) map[uint16]clientdata.IndexedProperty {
	if player == nil {
		return nil
	}
	motion := player.motionSnapshot()
	overrides := make(map[uint16]clientdata.IndexedProperty, 40)
	add := func(props ...clientdata.IndexedProperty) {
		for _, prop := range props {
			overrides[prop.Index] = prop
		}
	}
	add(baseMotionProperties(motion)...)
	add(qinggongMotionProperties()...)
	f := func(name string, value float32) clientdata.IndexedProperty {
		return clientdata.IndexedProperty{Index: clientdata.MustPlayerProperty(name).Index, Name: name, Value: clientdata.Float32Value(value)}
	}
	i := func(name string, value int32) clientdata.IndexedProperty {
		return clientdata.IndexedProperty{Index: clientdata.MustPlayerProperty(name).Index, Name: name, Value: clientdata.Int32Value(value)}
	}
	// MoveSpeed/RunSpeed/RunSpeedAdd：官方档案基线 3.9/3.9/0（c.pcapng #13/#15
	// 全对 3.9；镜像 RunSpeedAdd=0.6 是外服玩家装备加成——不得继承）。
	// motion.moveSpeed 会随会话漂移（曾实测 4.8 → 真机「移动速度很快」根因，
	// 2026-11-15 午后定案：4.8+透传 0.6 叠加远超官方）。UnderWaterModifySpeed：
	// 外服物理档案流同值 1.0（改写=闭环防漂移）。
	add(f("MoveSpeed", 3.9), f("RunSpeed", 3.9), f("RunSpeedAdd", 0), f("UnderWaterModifySpeed", 1))
	// 轻功等级加成组（外服 5 级轻功值；覆盖 qinggongMotionProperties 的
	// Add=0 基线——跳跃高度主要由这组 Add 决定，清零=二段跳只有基值 6）。
	add(f("SndJumpSpeedAdd", officialQingGongLevelAdds.sndJump),
		f("ThdJumpSpeedAdd", officialQingGongLevelAdds.thdJump),
		f("DriftJumpSpeedAdd", officialQingGongLevelAdds.driftJump),
		f("AirRushDistAdd", officialQingGongLevelAdds.airRushDist))
	state := player.stateSnapshot()
	add(i("QingGongPoint", state.QingGongPoint), i("MaxQingGongPoint", state.MaxQingGongPoint), i("MaxQingGongPointAdd", state.MaxQingGongPointAdd))
	return overrides
}

// officialQingGongLevelAdds 外服轻功等级加成组常量（server2
// qinggong_jump_physics.go 外服 5 级轻功抓拍；驱动链=跳跃 Z 初速的 Add 项）。
type officialQingGongLevelAddsSpec struct {
	sndJump     float32 // SndJumpSpeedAdd：二段跳加成
	thdJump     float32 // ThdJumpSpeedAdd：三段跳加成
	driftJump   float32 // DriftJumpSpeedAdd：水面跳加成
	airRushDist float32 // AirRushDistAdd：空中冲刺距离加成
}

var officialQingGongLevelAdds = officialQingGongLevelAddsSpec{
	sndJump:     5.1,
	thdJump:     3.0,
	driftJump:   3.0,
	airRushDist: 5.0,
}

// replayPhase2NativeReplace Phase B 帧替换走查：0x15/0x1F 原生生成，
// 0x2A DROP-8 停发，self 0x10 移动/跳跃族 MOTION-15 改写，其余透传+标注。
// raw/label 来自 replayPhase2FramesRaw（env 覆盖 > exe 内嵌）。返回发送帧数。
func replayPhase2NativeReplaceBytes(conn sceneMessageConnection, raw []byte, label string, obj, own uint32, pos role.Position, player *playerActor) int {
	views := officialP2ViewCreateSet()
	viewIdx := 0
	seen := map[string]bool{}
	motionOverrides := nativeP2SelfMotionOverrides(player)
	attrOverrides := replayAttrOverrides(player)
	nativeCount, passCount, dropCount, motionCount := 0, 0, 0, 0
	attrSwapped, attrShadow := 0, 0
	sent := 0
	rest := raw
	for len(rest) >= 4 {
		fl := int(binary.LittleEndian.Uint32(rest))
		if fl <= 0 || fl > len(rest)-4 || fl > 0x10000 {
			break
		}
		frame := rest[4 : 4+fl]
		rest = rest[4+fl:]
		// MOTION-15：self 单属性 0x10 命中移动/跳跃族 → 原生改写为
		// 我方权威值（镜像帧形 [0x10][0][obj][own][1][idx][value]，
		// 官方镜像与原生编码同形，值域即全部差异）。表外 idx 落回
		// switch 的 MIRROR-12 透传口径。
		if frame[0] == 0x10 && len(frame) >= 18 &&
			binary.LittleEndian.Uint32(frame[2:]) == obj && binary.LittleEndian.Uint16(frame[10:]) == 1 {
			if prop, ok := motionOverrides[binary.LittleEndian.Uint16(frame[12:])]; ok {
				out, oerr := sceneObjectProperties(obj, own, 0, []clientdata.IndexedProperty{prop})
				if oerr != nil {
					log.Printf("native-replace P2 motion prop %d encode: %v", prop.Index, oerr)
					return sent
				}
				if err := conn.WriteFrame(out); err != nil {
					log.Printf("replay %s: %v", label, err)
					return sent
				}
				if motionCount == 0 {
					log.Printf("native-replace P2 op=0x10 self 移动/跳跃族 -> 我方权威值 [MOTION-15 基础值+轻功加成组；基础物理流缺失=「跳跃异常」真根因，PHYSICS-16 补发]")
				}
				nativeCount++
				motionCount++
				sent++
				continue
			}
		}
		switch {
		case frame[0] == 0x10 && len(frame) >= 12 && frame[1] == 0 &&
			binary.LittleEndian.Uint32(frame[2:]) == obj:
			// ATTR-SHADOW：self 0x10 影子属性帧真值优先/影子兜底
			// （2026-11-15 用户拍板）。帧级二分 p2mirror 丢帧口径优先。
			if bisectDropsP2Mirror() {
				dropCount++
				continue
			}
			if out, swapped, _, aok := replayAttrShadowSwap(frame, obj, attrOverrides); aok {
				if err := conn.WriteFrame(out); err != nil {
					log.Printf("replay %s: %v", label, err)
					return sent
				}
				if swapped > 0 {
					if attrSwapped == 0 {
						log.Printf("native-replace P2 op=0x10 self 属性帧 -> 真值优先/影子兜底 [有源 idx 换我方值，无源 idx=影子默认值；481/624 保留影子]")
					}
					nativeCount++
					attrSwapped++
				} else {
					if attrShadow == 0 {
						log.Printf("native-replace P2 op=0x10 self 属性帧纯影子透传（无我方值源 idx=影子默认值，暂时兜底）")
					}
					passCount++
					attrShadow++
				}
				sent++
				continue
			}
			// aok=false（解析错位/round-trip 不过）：落 default 透传。
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
			attrShadow++
			sent++
		case frame[0] == 0x1E && len(frame) >= 14 && isSelfM24StateMirror(frame, obj, own):
			// DROP-m24：外服玩家状态镜帧（bin P2#122 实证 m24 obj(self) i33
			// ——c.pcapng 抓拍时外服玩家所处的状态值，重登即灌给我们的玩家；
			// c1=33 疑移动/走态镜像，2026-11-15「重登回走」终局嫌疑。外服
			// 玩家数据行一律不得镜像（DROP-8 同族），状态权威在我方。
			marker := "DROP-m24 外服玩家状态镜帧（m24 obj(self) c1=33 疑走态镜像；重登回走终局嫌疑）"
			if !seen[marker] {
				seen[marker] = true
				log.Printf("native-replace P2 op=0x1E DROP [%s]", marker)
			}
			dropCount++
		case frame[0] == 0x2A:
			// DROP-8：官方装备外观停发（影子源头）。不写帧、只计 dropped，
			// 标注打一次。
			marker := p2UnknownMarker(frame)
			if !seen[marker] {
				seen[marker] = true
				log.Printf("native-replace P2 op=0x%02X DROP [%s]", frame[0], marker)
			}
			dropCount++
		case frame[0] == 0x29 && frame[1] == 0x47:
			// DROP-LZ：0x29 gemskill 预载为 mode-0x47 外池帧（真机字典
			// 错位=垃圾内层，与 P1#0/#2 同族；gem 视图我方未实现，丢弃
			// 零损失——2026-11-15 午后定案）。
			marker := "DROP-LZ 0x29 mode-0x47 外池帧（gemskill 预载，真机字典错位）"
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
			// 帧级二分（NINEYIN_REPLAY_BISECT）：按轮丢弃整类透传集合。
			if bisectDropsP2Custom() && frame[0] == 0x1E {
				dropCount++
				continue
			}
			if bisectDropsP2Mirror() && frame[0] == 0x10 {
				dropCount++
				continue
			}
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
	log.Printf("native-replace P2 summary: native=%d (motion=%d attrSwapped=%d) passthrough=%d (attrShadow=%d) dropped=%d (unknown classes=%d) file=%s",
		nativeCount, motionCount, passCount, attrShadow, attrSwapped, dropCount, len(seen), strings.TrimSpace(label))
	return sent
}

// officialToServerViewSpec officialViewSpec → 0x15 编码器参数（结构常量）。
func officialToServerViewSpec(spec officialViewSpec) serverViewSpec {
	return serverViewSpec{ID: spec.id, Capacity: spec.cap}
}

// isSelfM24StateMirror 判定 0x1E m24 自靶状态镜帧：[1E][argc u16]
// [tag2 24][tag8 obj=自身][tag2 c1]。
func isSelfM24StateMirror(frame []byte, obj, own uint32) bool {
	if binary.LittleEndian.Uint16(frame[1:]) < 3 || frame[3] != 2 ||
		binary.LittleEndian.Uint32(frame[4:]) != 24 {
		return false
	}
	p := 8
	if p+9 > len(frame) || frame[p] != 8 {
		return false
	}
	return binary.LittleEndian.Uint32(frame[p+1:]) == obj &&
		binary.LittleEndian.Uint32(frame[p+5:]) == own
}
