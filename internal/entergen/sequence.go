package entergen

// 进场/就绪两段编排（7074 线实证相位——c.pcapng/登陆-all 窗口对拍）：
//
//	PreReady（表后 → ClientReady 前）：
//	  0x0B 进场 → 0x14/0x11 记录对（挂载→纯数字列行）→ 0x29 批量
//	  （字符串列记录，0x47 字面量包装）→ 0x28 化身 → 0x10 物理流 ×36
//	  → 0x10 基础属性批（五维/资源）
//	Ready（ClientReady 后）：
//	  m607/m52/m586 → m169 buff → 视图簇（空容器 + View40/43/46 + 44/45）
//	  → 战斗属性单帧族 → m416 九阴志域 → 0x1F 位置
//
// 会话身份（seq/obj/own/IP/时间戳）由 Identity/Session 注入，数据值取
// rolestate（role_state_json 同源）——官方 Fs 引擎 PlayerEntry 按存档
// 现场生成的等价物（docs/official/透传数据清单.md 处置台账）。

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// recordMountPairs 就绪前要挂载+推行的纯数字列记录（rec 名 → 行集）。
// 位图记录（Task_Completed）走 taskCompletedBit 行展开。
type recordMountPair struct {
	recordName string
	rows       func(state rolestate.RoleState) ([][]RecordCell, error)
}

// PreReadyFrames 进场段全量帧（表下发之后、ClientReady 之前）。
// 0x29 批量用 0x47 全字面量包装（零池依赖）；literal29=false 时字符串
// 列记录降级走 0x11 行（fork 客户端已接受方言）。
func PreReadyFrames(ctx Context, now time.Time) ([][]byte, error) {
	frames := [][]byte{}

	// 0x0B 进场（idx178/1468/1447/1448 按存档场景）。
	entry, err := EntrySceneFrame(ctx.Identity, ctx.State.Scene, ctx.State.Resource)
	if err != nil {
		return nil, err
	}
	frames = append(frames, entry)

	// 登录信息帧簇 m366/m623/m655（外服 0.1s 位：0x0A 表后）。
	info, err := LoginInfoFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, info...)

	// 记录挂载 + 行推（0x14 + 0x11）：Task_Completed 位图行。
	if err := appendRecordMounts(&frames, ctx); err != nil {
		return nil, err
	}

	// Task_Accepted 行（官方 28 列字符串记录——0x29 批量通道；
	// 字面量 0x47 包装）。
	accepted29, err := taskAcceptedBatch(ctx)
	if err != nil {
		return nil, err
	}
	if accepted29 != nil {
		frames = append(frames, accepted29)
	}

	// 采集技能行（CollectSkillRec 两列窄串——0x29 通道）。
	collect29, err := collectSkillBatch(ctx)
	if err != nil {
		return nil, err
	}
	if collect29 != nil {
		frames = append(frames, collect29)
	}

	// 轻功类别记录 QingGongRec（单列窄串——逐行 0x11 官方形态）。
	for _, cat := range ctx.State.QingGongCategories {
		cell := cat
		row, err := RecordAddCellsFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID,
			clientdata.MustRecordIndexByName("QingGongRec"), []RecordCell{RecordString(cell)})
		if err != nil {
			return nil, err
		}
		frames = append(frames, row)
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
	base, err := basePropertyFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, base...)

	return frames, nil
}

// ReadyFrames 就绪段全量帧（ClientReady 之后）。
func ReadyFrames(ctx Context, now time.Time) ([][]byte, error) {
	frames := [][]byte{}

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
	combat, err := combatAttributeFrames(ctx)
	if err != nil {
		return nil, err
	}
	frames = append(frames, combat...)

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

// appendRecordMounts 0x14 挂载 + 0x11 位图行（Task_Completed 串行位映射
// 与官方 row = (id-1)/32 / mask = 1<<((id-1)%32) 同构，b-line
// taskCompletedRows 口径——含 final-row 前的全部零值行）。
func appendRecordMounts(frames *[][]byte, ctx Context) error {
	rec := clientdata.MustRecordIndexByName("Task_Completed")
	*frames = append(*frames, RecordMountFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, rec))
	if len(ctx.State.TaskCompleted) == 0 {
		return nil
	}
	maxRow := 0
	rows := map[int]uint32{}
	for _, id := range ctx.State.TaskCompleted {
		if id == 0 {
			continue
		}
		row := int((id - 1) / 32)
		bit := (id - 1) % 32
		rows[row] |= 1 << bit
		if row > maxRow {
			maxRow = row
		}
	}
	for row := 0; row <= maxRow; row++ {
		v := rows[row]
		cell := v
		frame, err := RecordAddCellsFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, rec,
			[]RecordCell{RecordInt(cell)})
		if err != nil {
			return err
		}
		*frames = append(*frames, frame)
	}
	return nil
}

// taskAcceptedBatch Task_Accepted 28 列字符串记录的 0x29 批量帧
// （0x47 全字面量包装）。内层明文（初始账号方言定版）：
// [0]=0 [12]=变体 [14]=0 [15:19]=obj [19:23]=own [23:27]=rec u32
// [27:29]=行数 u16 [29:]=行体（每行按 schema 列型，首 7 列官方方言）。
func taskAcceptedBatch(ctx Context) ([]byte, error) {
	if len(ctx.State.TaskAccepted) == 0 {
		return nil, nil
	}
	rec := clientdata.MustRecordIndexByName("Task_Accepted")
	rows := make([][]RecordCell, 0, len(ctx.State.TaskAccepted))
	for _, taskID := range ctx.State.TaskAccepted {
		rows = append(rows, []RecordCell{
			RecordInt(taskID), RecordInt(0), RecordString(""),
			RecordInt(0), RecordString(""), RecordString(""), RecordInt(0),
		})
	}
	return recordBatch0x29(ctx, rec, rows)
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

// basePropertyFrames 建号基础属性批（0x10 单帧大批：五维/资源/等级）。
func basePropertyFrames(ctx Context) ([][]byte, error) {
	state := ctx.State
	props := []IndexedProperty{
		NamedProperty("Level", ByteValue(1)),
		NamedProperty("CurLevel", ByteValue(1)),
		NamedProperty("Str", Int32Value(int32(state.Str))),
		NamedProperty("Dex", Int32Value(int32(state.Dex))),
		NamedProperty("Ing", Int32Value(int32(state.Ing))),
		NamedProperty("Spi", Int32Value(int32(state.Spi))),
		NamedProperty("Sta", Int32Value(int32(state.Sta))),
		NamedProperty("MaxHP", Int32Value(int32(state.HP))),
		NamedProperty("MaxMP", Int32Value(int32(state.MP))),
		NamedProperty("MaxSP", Int32Value(int32(state.SP))),
		NamedProperty("HP", Int32Value(int32(state.HP))),
		NamedProperty("MP", Int32Value(int32(state.MP))),
		NamedProperty("SP", Int32Value(int32(state.SP))),
		NamedProperty("MaxParry", Int32Value(int32(state.Parry))),
	}
	frame, err := ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0, props)
	if err != nil {
		return nil, err
	}
	return [][]byte{frame}, nil
}

// combatAttributeFrames 战斗面板单帧族（外服形态：单属性 0x10 逐帧；
// 建号可计算子集——等级内功加成后面板值）。
func combatAttributeFrames(ctx Context) ([][]byte, error) {
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
