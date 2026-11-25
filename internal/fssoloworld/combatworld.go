package fssoloworld

// combatworld.go Lua World 适配：对象注册表与 nx_* 战斗动作面的落点
// （对象视图的前半——玩家/NPC/技能对象——在 combat.go）。

import (
	"math"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/script"
)

// ---- 辅助对象（技能/buff）视图：记录表 ----

func (o *auxObject) AddRecordRow(record string, values []script.Value) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.records[record] = append(o.records[record], values)
	return len(o.records[record]) - 1
}

func (o *auxObject) QueryRecord(record string, row, col int) (script.Value, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	rows := o.records[record]
	if row < 0 || row >= len(rows) || col < 0 || col >= len(rows[row]) {
		return script.Nil(), false
	}
	return rows[row][col], true
}

func (o *auxObject) RecordRows(record string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.records[record])
}

// sceneObject 场景对象视图（nx_get_scene）。
type sceneObject struct{}

func (o *sceneObject) Type() int                         { return int(script.KindScene) }
func (o *sceneObject) Config() string                    { return "" }
func (o *sceneObject) Query(string) (script.Value, bool) { return script.Nil(), false }
func (o *sceneObject) Set(string, script.Value) bool     { return false }

// ---- World 适配 ----

type combatWorld struct{ s *Server }

var sceneHandle = script.Handle{Kind: script.KindScene, ID: 1}

func (w *combatWorld) Lookup(h script.Handle) (script.Object, bool) {
	s := w.s
	switch h.Kind {
	case script.KindScene:
		return &sceneObject{}, true
	case script.KindPlayer:
		pl := s.playerByKey(h.ID)
		if pl == nil {
			return nil, false
		}
		return &playerObject{s: s, pl: pl}, true
	case script.KindNPC:
		if f := s.fighterByHandle(h.ID); f != nil {
			return &npcObject{s: s, f: f}, true
		}
	case script.KindHelper:
		if aux := s.auxByHandle(h.ID); aux != nil {
			return aux, true
		}
	}
	return nil, false
}

func (w *combatWorld) Scene() script.Handle { return sceneHandle }

// CreateFromConfig 按 ConfigID 建技能/buff 辅助对象。
func (w *combatWorld) CreateFromConfig(configID string) (script.Handle, bool) {
	s := w.s
	lua := s.combatLua()
	if lua == nil || lua.skills == nil {
		return script.NullHandle, false
	}
	if buff, err := lua.skills.Buff(configID, 1); err == nil {
		return s.registerAux(&auxObject{kind: script.HelperBuffer, config: configID, script: buff.LuaScript,
			props: map[string]script.Value{
				"ConfigID": script.String(configID), "Level": script.Int(1),
				"LifeTime": script.Int(int64(buff.LifeTimeMS)), "EffectType": script.Int(int64(buff.EffectType)),
				"IsDamage": script.Int(int64(btoi(buff.IsDamage))), "EffectTimer": script.Int(int64(buff.EffectTimerMS)),
				"EffectRatio": script.Int(int64(buff.EffectRatio)), "Category": script.Int(int64(buff.Category)),
			},
			records: map[string][][]script.Value{}}), true
	}
	if def, err := lua.skills.Definition(configID, 1); err == nil {
		return s.registerAux(&auxObject{kind: script.HelperSkill, config: configID, script: def.LuaScript, def: def,
			props: map[string]script.Value{}, records: map[string][][]script.Value{}}), true
	}
	return script.NullHandle, false
}

func (w *combatWorld) Destroy(h script.Handle) bool {
	w.s.combatMu.Lock()
	aux := w.s.aux[h.ID]
	delete(w.s.aux, h.ID)
	w.s.combatMu.Unlock()
	if aux != nil && aux.kind == script.HelperBuffer {
		w.s.detachBuffByHandle(h.ID)
	}
	return aux != nil
}

func (w *combatWorld) AddBuffer(target, caster, buffer script.Handle) bool {
	return w.s.attachBuffAux(target, caster, buffer.ID)
}

func (w *combatWorld) RemoveBuffer(target, buffer script.Handle) bool {
	w.s.detachBuffByHandle(buffer.ID)
	return true
}

func (w *combatWorld) FindBuffer(target script.Handle, configID string) (script.Handle, bool) {
	return w.s.findBuff(target, configID)
}

func (w *combatWorld) Damage(attacker, defender script.Handle, damageType int, amount float64) float64 {
	return w.s.applyDamage(attacker, defender, script.HitResult{Damage: amount}, damageSource{class: "Buffer", byBuff: true})
}

// DamageByBuffer nx_damage_target：按 buff 脚本的 get_damage 结算伤害。
func (w *combatWorld) DamageByBuffer(attacker, target, buffer script.Handle) float64 {
	return w.s.damageByBuffer(attacker, target, buffer)
}

// BreakFlow nx_break_any_flow / nx_stop：打断当前施放。
func (w *combatWorld) BreakFlow(h script.Handle) {
	w.s.breakFlow(h)
}

// objectValue 句柄对象的 wire 值（玩家 / NPC；其余为 None）。
func (s *Server) objectValue(h script.Handle) entergen.CustomValue {
	ident, _, _ := s.identOf(h)
	return entergen.CustomObject(ident.objectID, ident.ownerID)
}

func (w *combatWorld) Custom(self script.Handle, args ...script.Value) {
	if len(args) == 0 {
		return
	}
	pl := w.s.playerByKey(self.ID)
	if pl == nil {
		return
	}
	values := make([]entergen.CustomValue, 0, len(args))
	for _, arg := range args[1:] {
		values = append(values, customValueOf(arg))
	}
	if frame, err := entergen.CustomMessageFrame(int32(args[0].IntValue()), values...); err == nil {
		w.s.sendToClient(pl, frame)
	}
}

// SysInfo 提示信息：m1[提示类型, 文本ID, 参数…]（物品链 m1[2,"9402"] 同形）。
func (w *combatWorld) SysInfo(self script.Handle, tipType int, args ...script.Value) {
	pl := w.s.playerByKey(self.ID)
	if pl == nil {
		return
	}
	values := []entergen.CustomValue{entergen.CustomInt(int32(tipType))}
	for _, arg := range args {
		values = append(values, customValueOf(arg))
	}
	b := newFrameBuilder(pl)
	b.custom(1, values...)
	w.s.sendFrames(pl, b.frames)
}

// Around 周围对象（玩家自身 + 同场景半径内的 NPC；战斗脚本的用法是范围
// 伤害/治疗目标）。
func (w *combatWorld) Around(center script.Handle, radius float64, types, limit int) []script.Handle {
	s := w.s
	if limit <= 0 {
		limit = 100
	}
	var out []script.Handle
	if center.Kind == script.KindPlayer {
		if pl := s.playerByKey(center.ID); pl != nil && types&int(script.KindPlayer) != 0 {
			out = append(out, playerHandle(pl))
		}
	}
	if types&int(script.KindNPC) == 0 {
		return out
	}
	origin, scene, ok := s.centerOf(center)
	if !ok {
		return out
	}
	for _, other := range s.sceneFighters(scene) {
		other.mu.Lock()
		pos, dead := other.pos, other.dead
		other.mu.Unlock()
		if dead || math.Hypot(pos[0]-origin[0], pos[2]-origin[2]) > radius {
			continue
		}
		out = append(out, npcHandle(other))
		if len(out) >= limit {
			break
		}
	}
	return out
}

// centerOf 句柄对象的位置与场景（NPC：战斗体位；玩家：上报位 + 当前场景）。
func (s *Server) centerOf(h script.Handle) (pos [4]float64, scene string, ok bool) {
	switch h.Kind {
	case script.KindNPC:
		f := s.fighterByHandle(h.ID)
		if f == nil {
			return pos, "", false
		}
		f.mu.Lock()
		pos, scene = f.pos, f.scene
		f.mu.Unlock()
	case script.KindPlayer:
		pl := s.playerByKey(h.ID)
		if pl == nil {
			return pos, "", false
		}
		pl.npcMu.Lock()
		scene = pl.npcScene
		pl.npcMu.Unlock()
		pos = pl.roleState.Position
	}
	return pos, scene, h.Kind == script.KindNPC || h.Kind == script.KindPlayer
}

// customValueOf script.Value → 自定义消息值。
func customValueOf(v script.Value) entergen.CustomValue {
	switch v.Kind {
	case script.KindString, script.KindWide:
		return entergen.CustomString(v.Text)
	case script.KindObject:
		return entergen.CustomObject(uint32(v.Obj.ID), uint32(v.Obj.ID>>32))
	case script.KindFloat:
		return entergen.CustomFloat32(float32(v.Num))
	default:
		return entergen.CustomInt(int32(v.IntValue()))
	}
}

// ---- 注册表 ----

func (s *Server) registerAux(aux *auxObject) script.Handle {
	s.combatMu.Lock()
	defer s.combatMu.Unlock()
	s.nextHandle++
	s.aux[s.nextHandle] = aux
	return script.Handle{Kind: script.KindHelper, Helper: aux.kind, ID: s.nextHandle}
}

func (s *Server) auxByHandle(id uint64) *auxObject {
	s.combatMu.Lock()
	defer s.combatMu.Unlock()
	return s.aux[id]
}

// playerByKey 连接键（persistKey：conn1 在高 32 位）→ 玩家。
func (s *Server) playerByKey(key uint64) *Player {
	return s.players.FindByConnect(uint32(key>>32), uint32(key))
}

func (s *Server) fighterByHandle(handle uint64) *npcFighter {
	s.combatMu.Lock()
	defer s.combatMu.Unlock()
	return s.fighters[handle]
}

func (s *Server) sceneFighters(scene string) []*npcFighter {
	s.combatMu.Lock()
	defer s.combatMu.Unlock()
	var out []*npcFighter
	for _, f := range s.fighters {
		if f.scene == scene {
			out = append(out, f)
		}
	}
	return out
}

func hpRatio(hp, maxHP int) int32 {
	if maxHP <= 0 {
		return 0
	}
	return int32(hp * 100 / maxHP)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi && hi >= lo {
		return hi
	}
	return v
}
