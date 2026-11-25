package fssoloworld

// combatcore.go 战斗核心件：Lua 宿主懒建、玩家生存态、NPC 战斗体、帧汇
// （带可见性的帧收集与广播）、伤害落地与 buff 挂载。

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/actiondata"
	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"

	"github.com/lizhanyin/9yin-server-go/internal/script"
)

// combatLua 战斗用的 Lua 宿主与数据目录（每 Server 一份，首次用到时建）。
type combatLua struct {
	vm      *script.VM
	engine  *script.Engine
	skills  *skilldata.Catalog
	actions *actiondata.Catalog
	logf    func(format string, args ...any)
}

// combatLuaRoot 官方脚本根：NINEYIN_LUA_DIR > <数据根>/lua。
func combatLuaRoot() string {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_LUA_DIR")); v != "" {
		return v
	}
	if root := npcfeed.ResolveRoot(); root != "" {
		dir := filepath.Join(root, "lua")
		if info, err := res.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}

// combatLua 懒建宿主：技能目录 → 公式闭包装载 → 招式节拍目录。
func (s *Server) combatLua() *combatLua {
	s.luaOnce.Do(func() {
		lua := &combatLua{skills: s.skillCatalog(), logf: s.trace}
		root := combatLuaRoot()
		lua.vm = script.New(&combatWorld{s: s}, script.Options{LuaDir: root, Logf: s.trace})
		var specs []script.FormulaSpec
		var untranslatable []int
		if lua.skills != nil {
			for _, formula := range lua.skills.Formulas() {
				body, err := formula.LuaBody()
				if err != nil {
					// 源表达式超出可转译子集（如括号不配平的脏数据）：该公式
					// 不装载，用到它的击走回退面。
					untranslatable = append(untranslatable, formula.ID)
					continue
				}
				spec := script.FormulaSpec{ID: formula.ID, Body: body}
				for i, param := range formula.Params {
					spec.Params[i] = script.FormulaParam{Source: param.Source, Prop: param.Prop}
				}
				specs = append(specs, spec)
			}
		}
		lua.engine = script.NewEngine(lua.vm, specs)
		installed, skipped := lua.engine.Installed()
		if catalog, err := actiondata.Load(); err == nil {
			lua.actions = catalog
		}
		s.lua = lua
		s.trace("combat lua: 公式 %d 条（装载期跳过 %d，不可转译 %d 条 %v）脚本根=%q 招式节拍=%t",
			installed, skipped, len(untranslatable), untranslatable, root, lua.actions != nil)
	})
	return s.lua
}

// ---- 玩家生存态与面板 ----

// ensureLiveLocked 首次进入战斗时按登录面板口径初始化生存态（调用方持
// combatState.mu）。
func (s *Server) ensureLiveLocked(st *combatState, pl *Player) {
	if st.live.ready {
		return
	}
	maxHP := pl.roleState.HP
	if len(pl.roleState.NeiGong) > 0 {
		maxHP += pl.roleState.NeiGong[0].MaxHPAdd
	}
	if maxHP <= 0 {
		maxHP = 1
	}
	maxSP := int(s.playerPanel(pl)["MaxSP"])
	if maxSP <= 0 {
		maxSP = 50
	}
	st.live = playerLive{ready: true, hp: maxHP, maxHP: maxHP, maxSP: maxSP,
		parry: int(s.parryValueTotal(pl)), logic: logicNormal}
}

// playerPanel 玩家登录面板的数值属性视图（进场帧同源：装备/内功变更后
// 重新进场才刷新；战斗公式与脚本按属性名取值）。
func (s *Server) playerPanel(pl *Player) map[string]float64 {
	if pl.panel != nil {
		return pl.panel
	}
	panel := map[string]float64{}
	ctx := s.enterContext(pl, false)
	if frame, err := entergen.AppearanceFrame(ctx); err == nil {
		for name, value := range decodePropertyFrame(frame, 61) {
			panel[name] = value
		}
	}
	if frame, err := entergen.PlayerVitalBatchFrame(ctx); err == nil {
		for name, value := range decodePropertyFrame(frame, 10) {
			if _, dup := panel[name]; !dup {
				panel[name] = value
			}
		}
	}
	pl.panel = panel
	return panel
}

// decodePropertyFrame 解 0x0D/0x10 属性帧的数值属性（countOffset = 属性数
// 所在偏移：0x0D 为 61，0x10 为 10）。
func decodePropertyFrame(frame []byte, countOffset int) map[string]float64 {
	out := map[string]float64{}
	if len(frame) < countOffset+2 {
		return out
	}
	count := int(binary.LittleEndian.Uint16(frame[countOffset:]))
	offset := countOffset + 2
	for i := 0; i < count && offset+2 <= len(frame); i++ {
		index := int(binary.LittleEndian.Uint16(frame[offset:]))
		offset += 2
		entry, ok := clientdata.PropertyByIndex(index)
		if !ok {
			break
		}
		var value float64
		switch entry.Type {
		case clientdata.WireByte:
			if offset >= len(frame) {
				return out
			}
			value = float64(frame[offset])
			offset++
		case clientdata.WireWord:
			if offset+2 > len(frame) {
				return out
			}
			value = float64(binary.LittleEndian.Uint16(frame[offset:]))
			offset += 2
		case clientdata.WireInt32:
			if offset+4 > len(frame) {
				return out
			}
			value = float64(int32(binary.LittleEndian.Uint32(frame[offset:])))
			offset += 4
		case clientdata.WireFloat32:
			if offset+4 > len(frame) {
				return out
			}
			value = float64(math.Float32frombits(binary.LittleEndian.Uint32(frame[offset:])))
			offset += 4
		case clientdata.WireString:
			if offset+4 > len(frame) {
				return out
			}
			offset += 4 + int(binary.LittleEndian.Uint32(frame[offset:]))
		case clientdata.WireWideString:
			if offset+4 > len(frame) {
				return out
			}
			offset += 4 + int(binary.LittleEndian.Uint32(frame[offset:]))
		case clientdata.WireObject:
			offset += 8
		default:
			return out
		}
		if entry.Type != clientdata.WireString && entry.Type != clientdata.WireWideString && entry.Type != clientdata.WireObject {
			out[entry.Name] = value
		}
	}
	return out
}

// ---- NPC 战斗体 ----

// combatClass 可进入战斗的 NPC 逻辑类。
func combatClass(scriptClass string) bool {
	switch strings.ToLower(strings.TrimSpace(scriptClass)) {
	case "attacknpc", "bossnpc":
		return true
	}
	return false
}

// attackable 是否可被攻击（战斗类且模板未标 CantBeAttack）。
func attackable(spawn npcfeed.Spawn) bool {
	return combatClass(spawn.Script) && strings.TrimSpace(spawn.Template["CantBeAttack"]) != "1"
}

// fighterFor 取/建场景 NPC 的战斗体（同场景共享；scene = 摆放目录名）。
func (s *Server) fighterFor(scene string, spawn npcfeed.Spawn) *npcFighter {
	catalog := s.sceneNPCCatalog(scene)
	catalog.combatMu.Lock()
	defer catalog.combatMu.Unlock()
	if f := catalog.combat[spawn.ObjectID]; f != nil {
		return f
	}
	if catalog.combat == nil {
		catalog.combat = map[uint32]*npcFighter{}
	}
	s.combatMu.Lock()
	s.nextHandle++
	handle := s.nextHandle
	s.combatMu.Unlock()
	f := &npcFighter{
		handle: handle, spawn: spawn, scene: catalog.dir, logic: logicNormal,
		pos:   [4]float64{float64(spawn.X), float64(spawn.Y), float64(spawn.Z), float64(spawn.Orient)},
		props: npcCombatProps(spawn), hp: int(npcCombatMaxHP(spawn)), hurtBy: map[uint64]bool{},
	}
	catalog.combat[spawn.ObjectID] = f
	s.combatMu.Lock()
	s.fighters[handle] = f
	s.combatMu.Unlock()
	return f
}

// npcCombatMaxHP 战斗行 MaxLife（模板层兜底 1）。
func npcCombatMaxHP(spawn npcfeed.Spawn) float64 {
	if spawn.Combat != nil {
		if life := spawn.Combat.Int("MaxLife"); life > 0 {
			return float64(life)
		}
	}
	return 1
}

// npcCombatProps 战斗行 → 官方属性词汇（公式与脚本的属性名）。
func npcCombatProps(spawn npcfeed.Spawn) map[string]float64 {
	props := map[string]float64{"MaxHP": npcCombatMaxHP(spawn)}
	if spawn.Combat == nil {
		return props
	}
	row := spawn.Combat
	number := func(names ...string) float64 {
		for _, name := range names {
			if f, err := strconv.ParseFloat(strings.TrimSpace(row.Attrs[name]), 64); err == nil {
				return f
			}
		}
		return 0
	}
	base, spread := number("MeleeWeaponDamageBase"), number("MeleeWeaponDamageRand")
	props["MinMeleeDamage"], props["MaxMeleeDamage"] = base, base+spread
	rbase, rspread := number("RangeWeaponDamageBase"), number("RangeWeaponDamageRand")
	props["MinShotDamage"], props["MaxShotDamage"] = rbase, rbase+rspread
	props["Level"] = number("Level")
	props["Dodge"] = number("Dodge")
	props["PhyHit"] = number("PhysicsAttackHit")
	props["MagicHit"] = number("SolarMagicHit")
	props["PhyVa"] = number("PhysicsCriticalStrike")
	props["MagicVa"] = number("SolarCriticalStrike")
	props["PhyDef"] = number("PhysicsShieldBase")
	props["MagicDef"] = number("SolarMagicDefence")
	// 五维/威力：战斗行不带，取常量基线（强度由技能公式里的武器区间承载）。
	for _, name := range []string{"Str", "Dex", "Ing", "Spi", "Sta", "StrAdd", "DexAdd", "IngAdd", "SpiAdd", "StaAdd",
		"MeleePower", "MeleePowerAdd", "ShotPower", "ShotPowerAdd", "MagicPower", "MagicPowerAdd",
		"PhyHitAdd", "DodgeAdd", "PhyVaAdd", "MagicVaAdd", "VaDef"} {
		props[name] = 0
	}
	props["PhyHit"] = math.Max(props["PhyHit"], 3)
	props["Dodge"] = math.Max(props["Dodge"], 3)
	props["PhyVa"] = math.Max(props["PhyVa"], 3)
	return props
}

// ---- 帧汇 ----

// frameSink 带可见性的帧收集：私有帧只发本人（audience），公共帧同时
// 广播给同场景看得见任一相关 NPC 的其它玩家。
type frameSink struct {
	s        *Server
	audience *Player
	ident    wireIdent     // audience 的 wire ident
	subjects []*npcFighter // 公共帧的广播主体
	frames   []combatFrame
}

// sinkFor 为玩家建帧汇（audience = 该玩家；广播主体经 subject 追加）。
func (s *Server) sinkFor(pl *Player) *frameSink {
	return &frameSink{s: s, audience: pl, ident: pl.wireIdent().orFixed()}
}

func (k *frameSink) add(frame []byte, err error, private bool) {
	if err != nil {
		k.s.trace("combat: 帧编码失败: %v", err)
		return
	}
	k.frames = append(k.frames, combatFrame{data: frame, private: private})
}

// pri / pub 多返回值帧的入汇简写。
func (k *frameSink) pri(frame []byte, err error) { k.add(frame, err, true) }
func (k *frameSink) pub(frame []byte, err error) { k.add(frame, err, false) }

// subject 登记公共帧的广播主体。
func (k *frameSink) subject(f *npcFighter) {
	for _, seen := range k.subjects {
		if seen == f {
			return
		}
	}
	k.subjects = append(k.subjects, f)
}

// priv 本人属性帧。
func (k *frameSink) priv(name string, value entergen.PropertyValue) {
	k.pri(entergen.ObjectPropertyFrame(k.ident.objectID, k.ident.ownerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty(name, value)}))
}

// custom 本人自定义帧。
func (k *frameSink) custom(messageID int32, values ...entergen.CustomValue) {
	k.pri(entergen.CustomMessageFrame(messageID, values...))
}

// npcProp 相关 NPC 的属性帧（公共）。
func (k *frameSink) npcProp(f *npcFighter, name string, value entergen.PropertyValue) {
	k.pub(entergen.ObjectPropertyFrame(f.spawn.ObjectID, f.spawn.OwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty(name, value)}))
}

// pubCustom 公共自定义帧。
func (k *frameSink) pubCustom(messageID int32, values ...entergen.CustomValue) {
	k.pub(entergen.CustomMessageFrame(messageID, values...))
}

// npcValue / playerObject 自定义消息里的对象值。
func (k *frameSink) npcValue(f *npcFighter) entergen.CustomValue {
	return entergen.CustomObject(f.spawn.ObjectID, f.spawn.OwnerID)
}

func (k *frameSink) playerObject() entergen.CustomValue {
	return entergen.CustomObject(k.ident.objectID, k.ident.ownerID)
}

// relationAdd / relationTouch / relationDel fight_relation_rec 行操作帧
// （官服方言：加行 = 0x11 整行，刷新 = 0x13 只写第 2 列，删行 = 0x12）。
func (k *frameSink) relationAdd(row uint16, target uint64, now time.Time) {
	record := clientdata.MustRecordIndexByName("fight_relation_rec")
	k.pri(entergen.RecordAddRowFrame(k.ident.objectID, k.ident.ownerID, record, row, []entergen.RecordCell{
		entergen.RecordInt64(int64(target)), entergen.RecordInt(uint32(fightRelationWord(now))),
	}))
}

func (k *frameSink) relationTouch(row uint16, now time.Time) {
	record := clientdata.MustRecordIndexByName("fight_relation_rec")
	k.pri(entergen.RecordGridFrame(k.ident.objectID, k.ident.ownerID, record, row,
		entergen.GridWord(1, fightRelationWord(now))))
}

func (k *frameSink) relationDel(row uint16) {
	record := clientdata.MustRecordIndexByName("fight_relation_rec")
	k.add(entergen.RecordDelRowFrame(k.ident.objectID, k.ident.ownerID, record, row), nil, true)
}

// fightRelationWord fight_relation_rec 第 2 列取值。
func fightRelationWord(now time.Time) uint16 {
	return uint16(int64(now.UnixMilli()+fightRelationLeadMS) & 0xFFFF)
}

// flush 依序下发：本人收全部；其它看得见相关 NPC 的玩家只收公共帧。无
// 本人（纯广播）时公共帧发给全部看得见相关 NPC 的玩家。
func (k *frameSink) flush() {
	if len(k.frames) == 0 {
		return
	}
	var public [][]byte
	for _, frame := range k.frames {
		if !frame.private {
			public = append(public, frame.data)
		}
	}
	for _, frame := range k.frames {
		if k.audience != nil {
			k.s.sendToClient(k.audience, frame.data)
		}
	}
	for _, f := range k.subjects {
		if f != nil {
			k.s.broadcastNPC(k.audience, f, public)
		}
	}
	k.frames = nil
}

// broadcastNPC 把 NPC 相关帧广播给同场景其它看得见该 NPC 的玩家。
func (s *Server) broadcastNPC(skip *Player, npc *npcFighter, frames [][]byte) {
	if npc == nil || len(frames) == 0 {
		return
	}
	for _, pl := range s.players.Snapshot() {
		if pl == skip || pl.State != PlOnline {
			continue
		}
		pl.npcMu.Lock()
		sameScene, visible := pl.npcScene == npc.scene, pl.npcVisible == nil || pl.npcVisible[npc.spawn.ObjectID]
		pl.npcMu.Unlock()
		if !sameScene || !visible {
			continue
		}
		for _, frame := range frames {
			s.sendToClient(pl, frame)
		}
	}
}

// ---- 战斗活动与脱战 ----

// noteFightLocked 记一次战斗活动并刷新脱战计时；返回是否刚进入战斗态
// （LogicState 帧由调用方按帧序入汇）。调用方持 combatState.mu。
func noteFightLocked(st *combatState, now time.Time) (entered bool) {
	if st.live.fightAt.IsZero() {
		entered = true
	}
	st.live.fightAt = now
	if st.live.logic != logicFighting {
		st.live.logic = logicFighting
		entered = true
	}
	return entered
}

// parryRegen 招架耐力回复：进入过战斗且未在招架中，每秒 +1 至上限，变化
// 即推 ParryRatio/ParryValue（抓包 226.834 → 232.883：67 → 72）。
func (s *Server) parryRegen(pl *Player) [][]byte {
	st := &pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.live.ready {
		return nil
	}
	pl.parryMu.Lock()
	parrying := pl.parrying
	pl.parryMu.Unlock()
	maxParry := int(s.parryValueTotal(pl))
	if parrying || st.live.parry >= maxParry {
		return nil
	}
	st.live.parry++
	b := newFrameBuilder(pl)
	b.prop("ParryRatio", entergen.Int32Value(hpRatio(st.live.parry, maxParry)))
	b.prop("ParryValue", entergen.Int32Value(int32(st.live.parry)))
	return b.frames
}

// playerCalm 脱战检查（由 armCalmLocked 排在最后一次战斗活动
// playerLeaveFight 之后）：满 playerLeaveFight → 警戒态并清 FP/SP，
// 再过 playerCalmDelay − playerLeaveFight → 回常态、关系行全删、对手 NPC
// 同步脱战（抓包 228.138 → 231.152 → 231.303）。
func (s *Server) playerCalm(pl *Player) {
	st := &pl.combat
	st.mu.Lock()
	if !st.live.ready || st.live.fightAt.IsZero() {
		st.calmAt = time.Time{}
		st.mu.Unlock()
		return
	}
	idle := combatNow().Sub(st.live.fightAt)
	b := newFrameBuilder(pl)
	switch {
	case idle >= playerCalmDelay:
		st.live.logic = logicNormal
		st.live.fightAt, st.calmAt = time.Time{}, time.Time{}
		b.prop("LogicState", entergen.ByteValue(logicNormal))
	case idle >= playerLeaveFight && st.live.logic == logicFighting:
		st.live.logic = logicWarning
		if st.live.fp > 0 {
			b.prop("FP", entergen.Int32Value(0))
		}
		if st.live.sp > 0 {
			b.prop("SP", entergen.Int32Value(0))
		}
		st.live.fp, st.live.sp = 0, 0
		b.prop("LogicState", entergen.ByteValue(logicWarning))
		s.after(playerCalmDelay-playerLeaveFight, func() { s.playerCalm(pl) })
	default:
		// 更晚的活动已另排检查，本次作废。
		st.calmAt = time.Time{}
	}
	relations := st.live.relations
	if idle >= playerCalmDelay {
		st.live.relations = nil
	}
	st.mu.Unlock()
	if idle < playerLeaveFight {
		return
	}
	if idle >= playerCalmDelay {
		for _, row := range relations {
			if f := s.npcByKey(row.target); f != nil {
				s.npcCalm(f, "player calm")
			}
		}
		// 关系行倒序删（每次删行后余行前移）。
		record := clientdata.MustRecordIndexByName("fight_relation_rec")
		ident := pl.wireIdent().orFixed()
		for i := len(relations) - 1; i >= 0; i-- {
			b.frames = append(b.frames, entergen.RecordDelRowFrame(ident.objectID, ident.ownerID, record, uint16(i)))
		}
	}
	s.sendFrames(pl, b.frames)
}

// npcByKey fight_relation_rec 行里的对手（对象键 → 战斗体）。
func (s *Server) npcByKey(key uint64) *npcFighter {
	objID, owner := uint32(key), uint32(key>>32)
	s.combatMu.Lock()
	defer s.combatMu.Unlock()
	for _, f := range s.fighters {
		if f.spawn.ObjectID == objID && f.spawn.OwnerID == owner {
			return f
		}
	}
	return nil
}
