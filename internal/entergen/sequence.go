package entergen

// 进场/就绪两段编排（7074 线实证相位——c.pcapng/登陆-all 窗口对拍）：
//
//	PreReady（表后 → ClientReady 前）：
//	  S2C 705 个性配置（有档；官服登录段位——先于客户端登录期 107 转储）
//	  → 0x0B 进场 → 0x0D 玩家物化（外观/身体模型集）→ 0x14/0x11 记录对
//	  （挂载→纯数字列行）→ 0x29 批量（字符串列记录，0x47 字面量包装）
//	  → 0x28 化身 → 0x10 物理流 ×36 → 0x10 基础属性批（五维/资源）
//	  → 开图双记录（MapMaskRec 迷雾行 + OpenMapRec 位图）→ NPC 物化 ×N
//	Ready（ClientReady 后）：
//	  m229 c130 NPC 物化族 ×N（场景 NPC 目录；golden 相位段首）
//	  → 0x10 AutoTrace=1（地图点击寻路闸门）→ m607/m52/m586 → m169 buff
//	  → 视图簇（空容器 + View40/43/46 + 44/45）
//	  → 战斗属性单帧族 → 键位域 0x10/快捷栏 0x11 行 → m416 九阴志域 → 0x1F 位置
//
// 会话身份（seq/obj/own/IP/时间戳）由 Identity/Session 注入，数据值取
// rolestate（role_state_json 同源）——官方 Fs 引擎 PlayerEntry 按存档
// 现场生成的等价物（docs/official/透传数据清单.md 处置台账）。

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// PreReadyFrames 进场段全量帧（表下发之后、ClientReady 之前）。
// 0x29 批量用 0x47 全字面量包装（零池依赖）；literal29=false 时字符串
// 列记录降级走 0x11 行（fork 客户端已接受方言）。
func PreReadyFrames(ctx Context, now time.Time) ([][]byte, error) {
	frames := [][]byte{}

	// S2C 705 个性配置（有档）：登录段首帧——官服位（表后、0x0B 进场前，
	// 先于客户端登录期的 107 转储）。客户端每收一帧 705 会强制
	// camera_value/game_control_mode=0，须赶在场景加载的控制初始化之前
	// 到达；无档零帧（客户端保持本地配置，首份转储落库作基线）。
	customizing, err := CustomizingFrame(ctx.State)
	if err != nil {
		return nil, err
	}
	if customizing != nil {
		frames = append(frames, customizing)
	}

	// 新手引导阶段簇（官服位：m705 后、**0x0B 场景帧前**——2026-11-26
	// 建号.pcapng 男号实证 m414 [gamestep,5,5] 与 idx129 HelpStepID 均在
	// 0x0B 之前（0x27 族夹在中间）；11-24 切景抓包同序「m705→m414→0x0B」。
	// 迟到（0x0B 之后）客户端已入正常场景态，黑幕窗不再弹出——真机两轮
	// 实证。GuideStep=0 的角色零帧。
	guide, err := GuideEntryFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, guide...)

	// 0x0B 进场（idx178/1468/1447/1448 按存档场景）。
	entry, err := EntrySceneFrame(ctx.Identity, ctx.State.Scene, ctx.State.Resource)
	if err != nil {
		return nil, err
	}
	frames = append(frames, entry)

	// 0x0D 玩家物化帧（52626 实证序 0x0B→0x0D→0x1F；身体渲染模型集：
	// 布衣三件 + Hat=Hair + 档内外观自定义，定版文档全表序）。
	appearance, err := AppearanceFrame(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, appearance)

	// 0x1F 位置 + 0x10 人物数据大批（52626 #26/#27 实证序：0x0D→0x1F→
	// 登录全量面板批→物理流单帧族；b-line sendPlayerLocationAndVitals
	// 同构——0x1F 重定位后紧跟 vitals 大批）。
	pos0 := ctx.State.Position
	frames = append(frames, LocationFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID,
		float32(pos0[0]), float32(pos0[1]), float32(pos0[2]), float32(pos0[3])))
	vital, err := PlayerVitalBatchFrame(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, vital)

	// 登录信息帧簇 m366/m623/m655（外服 0.1s 位：0x0A 表后；官服在 0x0B
	// 之前，本方言排 0x0B 后——fork/外服双验证位置不敏感）。
	info, err := LoginInfoFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, info...)

	// 记录挂载计划（MountPlan 48 项，官方顺序）：0x14 + 行推交织
	//（Task_Completed 位图 0x11 / Task_Accepted 0x29 批量 / QingGongRec
	// 逐行 0x11；其余记录 mount-only，数据源接入后在 mount.go
	// rowGenerators 注册）。
	mounts, err := MountSequenceFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, mounts...)

	// 采集技能行（CollectSkillRec 两列窄串——0x29 通道；不在挂载计划
	// 内，b-line grantCollectSkills 补发口径）。
	collect29, err := collectSkillBatch(ctx)
	if err != nil {
		return nil, err
	}
	if collect29 != nil {
		frames = append(frames, collect29)
	}

	// 0x28 化身档案（mode=0x46 明文）。
	avatar, _ := AvatarFrame(ctx, false)
	frames = append(frames, avatar)

	// 0x10 物理档案流 ×36。
	physics, err := PhysicsLoginStreamFrames(ctx.Identity.ObjectID, ctx.Identity.OwnerID)
	if err != nil {
		return nil, err
	}
	frames = append(frames, physics...)

	// 0x10 基础属性批（建号五维/资源/等级——rolestate 值）。
	base, err := BasePropertyFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, base...)

	// 开图双记录帧（b-line grantMapRecords：0x0D 后、ClientReady 前——
	// MapMaskRec 迷雾行 + OpenMapRec 24 行位图；大地图城市标签/迷雾的
	// 数据源，全缺则图层空、点击无反应）。
	maps, err := MapRecordsFrames(ctx, ctx.SceneIndex)
	if err != nil {
		return nil, err
	}
	frames = append(frames, maps...)

	// NPC 物化帧族 ×N 全量（官方登录序：玩家 0x0D→记录行→NPC 物化×N；
	// 帧由 npcfeed 预生成——默认 0x28 mode=0x46 官方言，npc.pcapng 19
	// 字段罗塞塔）。
	frames = append(frames, ctx.NPCFrames...)

	return frames, nil
}

// ReadyFrames 就绪段全量帧（ClientReady 之后）。
func ReadyFrames(ctx Context, now time.Time) ([][]byte, error) {
	frames := [][]byte{}

	// 0x10 AutoTrace=1（开图三件套之三，b-line pushAutoTrace：ClientReady
	// 簇 m52 前——引擎侧地图点击寻路闸门，缺则点击地图不发起自动寻路）。
	autoTrace, err := AutoTraceFrame(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, autoTrace)

	// m607/m52/m586。
	info, err := ReadyInfoFrames(ctx, now)
	if err != nil {
		return nil, err
	}
	frames = append(frames, info...)

	// m169 登录 buff（组标志 + 装备内功）。
	buffs, err := LoginBuffFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, buffs...)

	// 视图簇：空容器族 + View40 武学 + View43 内功 + View46 轻功。
	empty, err := EmptyViewFrames()
	if err != nil {
		return nil, err
	}
	frames = append(frames, empty...)
	// 纸娃娃行 + 背包物品行（空容器族建视图之后逐行推送；2433 线
	// inventoryFrames 进场整推序：纸娃娃行先于背包行；空装备/空背包零帧）。
	dollRows, err := EquipViewFrames(ctx.State)
	if err != nil {
		return nil, err
	}
	frames = append(frames, dollRows...)
	bagItems, err := BagItemViewFrames(ctx.State)
	if err != nil {
		return nil, err
	}
	frames = append(frames, bagItems...)
	skills, err := SkillViewFrames(ctx.State)
	if err != nil {
		return nil, err
	}
	frames = append(frames, skills...)
	books, err := NeiGongViewFrames(ctx.State)
	if err != nil {
		return nil, err
	}
	frames = append(frames, books...)
	qg, err := QingGongViewFrames(ctx.State)
	if err != nil {
		return nil, err
	}
	frames = append(frames, qg...)

	// 战斗属性单帧族（建号可计算子集：五维/等级/资源上限）。
	combat, err := CombatAttributeFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, combat...)

	// 玩家对象域配置：键位域 0x10（198/199/200/223/201）→ 快捷栏
	// shortcut_rec 0x11 行（视图簇之后；无档零帧，starter 打坐条目由
	// EffectiveShortcuts 推导）。fork 客户端本地不落盘，缺发即「快捷
	// 键重登全丢」。
	// 705 不在本段：2026-11-23 线路抓包 48 会话实证客户端在 0x09 之前
	// 已上传 107 共 2–4 次，ClientReady 位的 705 赶不上登录转储；现行
	// 下行位 = 进场段首帧（PreReadyFrames / 切图 0x0B 前），上传零应答
	//（官服口径，docs/official/自定义配置705推送卡顿排查.md §八）。
	playerObject, err := PlayerObjectConfigFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, playerObject...)

	// m416 九阴志域声明。
	jyz, err := JyzDomainFrame(6)
	if err != nil {
		return nil, err
	}
	frames = append(frames, jyz)

	// 0x1F 位置（建号出生点）。
	pos := ctx.State.Position
	frames = append(frames, LocationFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID,
		float32(pos[0]), float32(pos[1]), float32(pos[2]), float32(pos[3])))

	return frames, nil
}

// collectSkillBatch CollectSkillRec（ConfigID+等级）0x29 批量。
func collectSkillBatch(ctx Context) ([]byte, error) {
	if len(ctx.State.CollectSkills) == 0 {
		return nil, nil
	}
	rec := clientdata.MustRecordIndexByName("CollectSkillRec")
	rows := make([][]RecordCell, 0, len(ctx.State.CollectSkills))
	for _, id := range ctx.State.CollectSkills {
		rows = append(rows, []RecordCell{RecordString(id), RecordInt(1)})
	}
	return recordBatch0x29(ctx, rec, rows)
}

// recordBatch0x29 行集 → 0x29/0x47 全字面量批量帧。
func recordBatch0x29(ctx Context, recordIndex uint16, rows [][]RecordCell) ([]byte, error) {
	body := make([]byte, 29, 64)
	body[0] = 0
	body[12] = 1 // 变体
	body[14] = 0
	body = putU32(body, 15, ctx.Identity.ObjectID)
	body = putU32(body, 19, ctx.Identity.OwnerID)
	body = putU32(body, 23, uint32(recordIndex))
	body = putU16(body, 27, uint16(len(rows)))
	for _, row := range rows {
		// 行体复用 0x11 的列编码（与 schema 列型一致）——裸拼进明文。
		rowBytes, err := recordRowBody(recordIndex, row)
		if err != nil {
			return nil, err
		}
		body = append(body, rowBytes...)
	}
	return WrapAs47Literal(0x29, body), nil
}

// recordRowBody 按 schema 列型编码行体（缺尾列补零/空串——与
// RecordAddCellsFrame 的行体段相同，不带 16B 帧头）。
func recordRowBody(recordIndex uint16, cells []RecordCell) ([]byte, error) {
	full, err := RecordAddCellsFrame(0, 0, recordIndex, cells)
	if err != nil {
		return nil, err
	}
	return full[16:], nil
}

// BasePropertyFrames 建号基础属性批（0x10 单帧大批：五维/资源/等级）。
func BasePropertyFrames(ctx Context) ([][]byte, error) {
	state := ctx.State
	props := []IndexedProperty{
		NamedProperty("Level", ByteValue(1)),
		// 属性表 CurLevel（idx 765）是 int32：按字节发会让其后的属性在按表
		// 解码时错位（0x0D 与人物数据大批同属性也走 Int32）。
		NamedProperty("CurLevel", Int32Value(1)),
		NamedProperty("Str", Int32Value(int32(state.Str))),
		NamedProperty("Dex", Int32Value(int32(state.Dex))),
		NamedProperty("Ing", Int32Value(int32(state.Ing))),
		NamedProperty("Spi", Int32Value(int32(state.Spi))),
		NamedProperty("Sta", Int32Value(int32(state.Sta))),
		NamedProperty("MaxHP", Int32Value(int32(state.HP))),
		NamedProperty("MaxMP", Int32Value(int32(state.MP))),
		NamedProperty("MaxSP", Int32Value(RageCap(ctx))),
		NamedProperty("HP", Int32Value(int32(state.HP))),
		NamedProperty("MP", Int32Value(int32(state.MP))),
		NamedProperty("SP", Int32Value(int32(ctx.SP))),
		NamedProperty("MaxParry", Int32Value(int32(state.Parry))),
		// 碎银（CapitalType1，2433 线 silverUpdate 同名同型 Int64）。
		NamedProperty("CapitalType1", Int64Value(state.Money)),
	}
	frame, err := ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0, props)
	if err != nil {
		return nil, err
	}
	return [][]byte{frame}, nil
}

// CombatAttributeFrames 战斗面板单帧族（外服形态：单属性 0x10 逐帧；
// 建号可计算子集——等级内功加成后面板值）。
func CombatAttributeFrames(ctx Context) ([][]byte, error) {
	state := ctx.State
	str := int32(state.Str)
	sta := int32(state.Sta)
	dex := int32(state.Dex)
	ing := int32(state.Ing)
	spi := int32(state.Spi)
	// 首本内功 varprop 加成叠加（b-line combatPanelProperties 口径）。
	for i := range state.NeiGong {
		if state.NeiGong[i].ConfigID == state.CurNeiGong {
			b := &state.NeiGong[i]
			str += int32(b.StrAdd)
			sta += int32(b.StaAdd)
			dex += int32(b.DexAdd)
			ing += int32(b.IngAdd)
			spi += int32(b.SpiAdd)
			break
		}
	}
	specs := []struct {
		name  string
		value int32
	}{
		{"Str", str}, {"Sta", sta}, {"Dex", dex}, {"Ing", ing}, {"Spi", spi},
		{"MaxHPAdd", int32(state.NeiGong[0].MaxHPAdd)},
		{"MaxMPAdd", int32(state.NeiGong[0].MaxMPAdd)},
		{"MaxParryAdd", int32(state.NeiGong[0].MaxParryAdd)},
	}
	frames := make([][]byte, 0, len(specs))
	for _, spec := range specs {
		frame, err := ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0,
			[]IndexedProperty{NamedProperty(spec.name, Int32Value(spec.value))})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func putU32(b []byte, off int, v uint32) []byte {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
	b[off+2] = byte(v >> 16)
	b[off+3] = byte(v >> 24)
	return b
}

func putU16(b []byte, off int, v uint16) []byte {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
	return b
}
