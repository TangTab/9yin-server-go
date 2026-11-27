package fssoloworld

// itemuse.go C2S msg31 使用道具受理链（官服抓包逐帧定版：资料\初始账号
// 登陆-完整引导-女.pcapng 8032 线 14 次 m31，tools/pcapitem 解码）。
//
// 上行（0x0A 活动信封）：[31, view, slot, count]，恰 4 值；view 是线缆
// 视图号（2 道具/121 装备/123 材料/125 任务）。
//
// 受理帧序（官服 #1~#4/#7/#9/#10 逐帧）：
//
//	m25[PrepareTime,"flow_prepare","useitem_flow"]   读条流（时长/动作取自
//	    item_func_static[FuncPack]；书类 118=1500ms life_ss_01）
//	0x10 视图对象(view, slot): LockStatus=1          使用中的行上锁
//	0x10 玩家: State="stand" → State=<PrepareAction>
//	… PrepareTime …
//	m26                                             流结束（官服多数样本带）
//	0x10 玩家: State="" ；0x10 视图对象: LockStatus=0
//	效果帧（按模板行数据分派）
//	0x19(view, slot)                                 扣 1 件；行有余量再推 0x18
//
// 拒绝方言：读条进行中/出招锁定中/冷却中/不可使用 → 零应答（官服 #11~#14
// 移动中连点无任何下行）；打坐中 → m1[2,"9402"]（「正在打坐，不能执行该
// 操作。」，sitcross.go 态机）。
//
// 状态变更在受理时同步落定（学会/入账/扣件/冷却行），读条定时器只补发
// 尾段帧：roleState 的写只发生在收包协程，定时器不碰状态（与技能链
// finishCast 同纪律）。效果面预检不通过则不读条、不扣件——无「白吃」。
//
// 效果面（internal/itemdata 取值面）：
//
//	武学书   LearnSkillPack → item_learn_skill_static.TragetID → 招式
//	         （SkillNormal/SkillLock）逐个学会：View40 追加行 + MaxLevel +
//	         CanUse + m141[技能,"Skill"] + m1[2,"9491",技能]（官服 #1~#4/#7
//	         同形；note_wuxue_rec 来历行 / m98 / m371 帮助窗未接）。目标
//	         全已学或都不是招式（内功/轻功/阵法…）→ 拒。
//	钱袋     script=CapitalItem 且 CapitalType=1（碎银）：CapitalValue 入账
//	         + 碎银帧；其余币种未建模 → 拒。
//	其余     礼盒（DropID 掉落包体官方未随资源发布）、药品（UseSkillPack
//	         回血结算未接）、带 LuaScript 的官服物品脚本（脚本本体缺失）等
//	         效果面未接 → 拒 + trace。

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/itemdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

const (
	// useItemFlowName / prepareFlowName 官服读条流名与流类型（m25 第二、三值）。
	prepareFlowName = "flow_prepare"
	useItemFlowName = "useitem_flow"
	// tipsLearnedSkill 系统提示文案键：「你学会了{@0:某项技能}！」（参数是
	// 技能 ConfigID，客户端按名取技能显示名）。
	tipsLearnedSkill = "9491"
	// tipsSitting 系统提示文案键：「正在打坐，不能执行该操作。」。
	tipsSitting = "9402"
	// capitalTypeSilver 碎银币种（roleState.Money；0/2 未建模）。
	capitalTypeSilver = 1
	// scriptCapitalItem 钱袋逻辑类（CapitalType/CapitalValue 入账）。
	scriptCapitalItem = "CapitalItem"
)

// itemUseFlow 进行中的一次使用读条（连接生命周期，不落库）。
type itemUseFlow struct {
	configID string
	view     int32 // 客户端上报的线缆视图号（回帧原样用）
	slot     int32
	done     [][]byte // 读条结束补发的尾段帧（受理时备好）
}

// itemUsePlan 一次使用的效果预检产物。
type itemUsePlan struct {
	learn   []rolestate.StarterSkill // 学会的招式（View40 追加行）
	capital int64                    // 入账碎银（钱袋）
}

// handleUseItem C2S 31 [31, view, slot, count]：官服一次使用恒扣 1 件
// （count 是客户端连续使用的申报数，官服按次回读条——连点由读条进行中
// 闸门逐次拦下）。
func (s *Server) handleUseItem(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) != 4 {
		s.trace("item use: 畸形 values=%d client=%d", len(message.Values), pl.Conn1)
		return
	}
	view, okView := message.Values[1].ExactInt32()
	slot, okSlot := message.Values[2].ExactInt32()
	count, okCount := message.Values[3].ExactInt32()
	if !okView || !okSlot || !okCount || slot < 1 || slot > 90 || count < 1 {
		s.trace("item use: 越界 view=%d slot=%d count=%d client=%d", view, slot, count, pl.Conn1)
		return
	}
	page := bagViewIDOfWire(view)
	if page == 0 {
		s.trace("item use: 未知背包页 view=%d client=%d", view, pl.Conn1)
		return
	}
	// 读条进行中：静默（官服连点零应答）。
	pl.itemUseMu.Lock()
	busy := pl.itemUse != nil
	pl.itemUseMu.Unlock()
	if busy {
		s.trace("item use: 读条进行中忽略 view=%d slot=%d client=%d", view, slot, pl.Conn1)
		return
	}
	st := &pl.roleState
	index := bagRowIndex(st, page, int(slot))
	if index < 0 {
		s.trace("item use: 无行 view=%d slot=%d client=%d", view, slot, pl.Conn1)
		return
	}
	item := st.BagItems[index]
	def, ok := s.itemCatalogOf().Definition(item.ConfigID)
	if !ok {
		s.trace("item use: %s 无模板行（物品目录为设计态 ini 回退或未知物品）client=%d", item.ConfigID, pl.Conn1)
		return
	}
	// 出招/锁定中：静默。
	pl.combat.mu.Lock()
	casting := pl.combat.curSkill != "" || pl.combat.locked
	pl.combat.mu.Unlock()
	if casting {
		s.trace("item use: %s 出招进行中忽略 client=%d", item.ConfigID, pl.Conn1)
		return
	}
	// 打坐中：m1[2,"9402"]，坐姿不动。
	if pl.sitting() {
		b := newFrameBuilder(pl)
		b.custom(1, entergen.CustomInt(2), entergen.CustomString(tipsSitting))
		s.sendFrames(pl, b.frames)
		s.trace("item use: %s 打坐中拒绝 client=%d", item.ConfigID, pl.Conn1)
		return
	}
	funcPack, ok := def.FuncPack()
	if !ok {
		s.trace("item use: %s 不可使用（无功能包）client=%d", item.ConfigID, pl.Conn1)
		return
	}
	// 冷却闸：行携 CoolDownCategory/CoolDownTime 且冷却表该类别有未过期行
	// → 静默（技能链同款）。
	nowMS := filetimeMS(combatNow())
	category, coolDownMS := def.Int("CoolDownCategory"), def.Int("CoolDownTime")
	pl.combat.mu.Lock()
	cooling := category > 0 && coolDownMS > 0 && cooldownLive(pl.combat.cooldowns, uint32(category), nowMS)
	pl.combat.mu.Unlock()
	if cooling {
		s.trace("item use: %s 冷却中忽略 client=%d", item.ConfigID, pl.Conn1)
		return
	}
	// 效果预检：不成立不读条不扣件。
	plan, ok := s.planItemUse(pl, def)
	if !ok {
		s.trace("item use: %s（script=%s FuncPack=%d）效果面未接或无可生效目标 client=%d",
			item.ConfigID, def.Script(), funcPack.ID, pl.Conn1)
		return
	}

	// ---- 受理：效果/冷却/扣件同步落定，尾段帧备好 ----
	tail := newFrameBuilder(pl)
	s.applyItemUse(pl, &plan, tail)
	if category > 0 && coolDownMS > 0 {
		tail.frames = append(tail.frames, s.startItemCooldown(pl, category, coolDownMS, nowMS)...)
	}
	tail.frames = append(tail.frames, s.consumeItemFrame(pl, index, view, slot)...)
	s.sendRoleSave(pl)
	s.trace("item use: %s（%s）读条 %dms 学会 %d 件/入账 %d client=%d", item.ConfigID, def.Str("Name"),
		funcPack.PrepareTimeMS, len(plan.learn), plan.capital, pl.Conn1)

	if funcPack.PrepareTimeMS <= 0 {
		// 瞬时使用（功能包无 PrepareTime）：无读条流，效果帧即发。
		s.sendFrames(pl, tail.frames)
		return
	}
	b := newFrameBuilder(pl)
	b.custom(25, entergen.CustomInt(int32(funcPack.PrepareTimeMS)),
		entergen.CustomString(prepareFlowName), entergen.CustomString(useItemFlowName))
	lock, err := useItemLockFrame(view, slot, 1)
	if err != nil {
		s.trace("item use: 锁帧编码失败: %v", err)
		s.sendFrames(pl, tail.frames)
		return
	}
	b.frames = append(b.frames, lock)
	b.prop("State", entergen.StringValue("stand"))
	if funcPack.PrepareAction != "" {
		b.prop("State", entergen.StringValue(funcPack.PrepareAction))
	}
	s.sendFrames(pl, b.frames)

	flow := &itemUseFlow{configID: item.ConfigID, view: view, slot: slot}
	// 读条结束帧序（官服）：m26 → State="" → 行解锁 → 效果 → 冷却 → 扣件。
	done := newFrameBuilder(pl)
	done.custom(26)
	done.prop("State", entergen.StringValue(""))
	unlock, err := useItemLockFrame(view, slot, 0)
	if err == nil {
		done.frames = append(done.frames, unlock)
	}
	flow.done = append(done.frames, tail.frames...)
	pl.itemUseMu.Lock()
	pl.itemUse = flow
	pl.itemUseMu.Unlock()
	time.AfterFunc(time.Duration(funcPack.PrepareTimeMS)*time.Millisecond, func() { s.finishItemUse(pl, flow) })
}

// finishItemUse 读条结束：清读条态并补发尾段帧。流已被清掉/替换（测试
// 直接驱动过、或会话重入）则空转——真实时钟的定时器晚到也无害。
func (s *Server) finishItemUse(pl *Player, flow *itemUseFlow) {
	pl.itemUseMu.Lock()
	if pl.itemUse != flow {
		pl.itemUseMu.Unlock()
		return
	}
	pl.itemUse = nil
	pl.itemUseMu.Unlock()
	if pl.State != PlOnline {
		return // 离线：帧经 entry 丢弃；状态已在受理时落定并存档
	}
	s.sendFrames(pl, flow.done)
}

// planItemUse 效果预检：按模板行数据给出读条结束时落定的效果。
func (s *Server) planItemUse(pl *Player, def *itemdata.Definition) (itemUsePlan, bool) {
	if targets := def.LearnTargets(); len(targets) > 0 {
		return s.planLearnSkill(pl, targets)
	}
	if def.Script() == scriptCapitalItem && def.Int("CapitalType") == capitalTypeSilver {
		value := def.Int("CapitalValue")
		if value <= 0 {
			return itemUsePlan{}, false
		}
		return itemUsePlan{capital: int64(value)}, true
	}
	return itemUsePlan{}, false
}

// planLearnSkill 武学书预检：学习包目标里挑出「招式且未学」者。
func (s *Server) planLearnSkill(pl *Player, targets []string) (itemUsePlan, bool) {
	catalog := s.skillCatalog()
	if catalog == nil {
		return itemUsePlan{}, false
	}
	var plan itemUsePlan
	for _, id := range targets {
		known := false
		for _, have := range pl.roleState.Skills {
			if have.ConfigID == id {
				known = true
				break
			}
		}
		if known {
			continue
		}
		skill, err := catalog.Definition(id, 1)
		if err != nil || skill.Script != skilldata.ScriptNormal && skill.Script != skilldata.ScriptLock {
			continue // 内功/轻功/阵法等非招式目标：对应容器（View43/46…）未接
		}
		plan.learn = append(plan.learn, rolestate.StarterSkill{
			ConfigID:   skill.ID,
			StaticData: skill.StaticData,
			ItemType:   skill.ItemType,
			Level:      1,
			MaxLevel:   max(skill.Levels, 1),
			PauseTime:  float64(skill.PauseTimeMS),
		})
	}
	return plan, len(plan.learn) > 0
}

// applyItemUse 效果落账 + 效果帧（官服武学书帧序：View40 追加行 →
// MaxLevel → CanUse → m141 → m1）。
func (s *Server) applyItemUse(pl *Player, plan *itemUsePlan, b *frameBuilder) {
	st := &pl.roleState
	for _, skill := range plan.learn {
		index := len(st.Skills)
		row, err := entergen.SkillRowFrame(index, skill)
		if err != nil {
			s.trace("item use: 武学行帧编码失败 %s: %v", skill.ConfigID, err)
			continue
		}
		st.Skills = append(st.Skills, skill)
		b.frames = append(b.frames, row)
		for _, prop := range []entergen.IndexedProperty{
			entergen.NamedProperty("MaxLevel", entergen.Int32Value(int32(max(skill.MaxLevel, 1)))),
			entergen.NamedProperty("CanUse", entergen.ByteValue(1)),
		} {
			if frame, err := entergen.ObjectPropertyFrame(uint32(entergen.ViewportSkill), uint32(index+1), 1,
				[]entergen.IndexedProperty{prop}); err == nil {
				b.frames = append(b.frames, frame)
			}
		}
		b.custom(141, entergen.CustomString(skill.ConfigID), entergen.CustomString("Skill"))
		b.custom(1, entergen.CustomInt(2), entergen.CustomString(tipsLearnedSkill), entergen.CustomString(skill.ConfigID))
	}
	if plan.capital > 0 {
		st.Money += plan.capital
		b.prop("CapitalType1", entergen.Int64Value(st.Money))
	}
}

// startItemCooldown 冷却表追加一行个人冷却（类别 = 行 CoolDownCategory，
// 时长 = 行 CoolDownTime；技能链个人行同形）。调用方在收包协程。
func (s *Server) startItemCooldown(pl *Player, category, coolDownMS int, nowMS int64) [][]byte {
	pl.combat.mu.Lock()
	defer pl.combat.mu.Unlock()
	ident := pl.wireIdent()
	w := &cooldownRowWriter{
		objectID: ident.objectID, ownerID: ident.ownerID,
		record: clientdata.MustRecordIndexByName("cooldown_rec"),
		rows:   pl.combat.cooldowns,
	}
	if err := w.insert(cooldownRow{category: uint32(category), start: nowMS,
		end: nowMS + int64(coolDownMS), team: cooldownTeamNone}); err != nil {
		return nil
	}
	pl.combat.cooldowns = w.rows
	return w.frames
}

// consumeItemFrame 扣 1 件并给出扣件帧：行有余量 → 入包三元组（0x19
// 预清 + 0x18 减堆行 + 0x19 镜像清位，购销同方言）；空行 → 删行对
// （0x19 + 镜像 0x19；官服书类单件样本：`19 02 00 05 00`）。
func (s *Server) consumeItemFrame(pl *Player, index int, view, slot int32) [][]byte {
	st := &pl.roleState
	if index < 0 || index >= len(st.BagItems) {
		return nil
	}
	wire := uint16(view)
	remove := entergen.BagRowRemoveFrames(wire, uint16(slot))
	if st.BagItems[index].Amount > 1 {
		st.BagItems[index].Amount--
		push, err := entergen.BagItemPushFrames(st.BagItems[index])
		if err != nil {
			return remove
		}
		return push // 三元组自带预清，无需再前缀删行对
	}
	st.BagItems = append(st.BagItems[:index], st.BagItems[index+1:]...)
	return remove
}

// useItemLockFrame 使用中的行上锁/解锁：0x10 视图对象帧（isViewObject=1，
// obj=线缆视图号，own=行号），LockStatus=1/0（官服 `10 01 02 00 00 00
// 05 00 00 00 01 00 72 07 01 00 00 00`）。
func useItemLockFrame(view, slot, status int32) ([]byte, error) {
	return entergen.ObjectPropertyFrame(uint32(view), uint32(slot), 1,
		[]entergen.IndexedProperty{entergen.NamedProperty("LockStatus", entergen.Int32Value(status))})
}
