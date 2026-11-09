package main

import (
	"time"

	"github.com/local/9yin-go-server/internal/neigong"
)

// neigongEffects 是进程级内功特效脚本宿主（scripts/module/neigong，
// internal/neigong）。main 装配 scriptHosts 后注入；nil（未启用/测试）
// 时所有分发点静默零效果——战斗链永不被脚本层阻断（契约同参考服
// NeiGongEffects 的降级与隔离）。
var neigongEffects *neigong.Effects

// neigongStoryStage 是九阴志档位门的当前阶段。九阴志系统落地前缺省 1
// （对齐参考服 -Dnineyin.neigong.storyStage 默认；cap = 3 + stage - 1，
// 脚本侧 tier_of 消费）。落地后改为读全服进度。
const neigongStoryStage = 1

// 内功特效预掷盐：与既有 combatPercentRoll 盐（17/18）错开，保证同一
// 攻击事件各判定互相独立且确定。
const (
	neigongRollVictim   = 31
	neigongRollParry    = 32
	neigongRollAttack   = 33
	neigongRollPeriodic = 34
	neigongRollDodge    = 35
)

// neigongShield 是玩家侧护盾池的一条 FIFO 吸收记录（参考服 PlayerActor
// 盾池语义：先到先吸收，到期即弃，全吸收=格挡式静默零帧）。
type neigongShield struct {
	amount    int32
	expiresAt time.Time
}

// neigongDodgeWindow 是 dodge_add 指令（料敌先机）授予的闪避加成窗：
// 期间受击闪避 +add 个百分点，到期即弃；重复授予以新窗覆盖（Proc 刷新）。
// 窗激活期间同步挂客户端可见的原生 Proc Buff（BufferInfo 槽 10）。
type neigongDodgeWindow struct {
	add       int32
	expiresAt time.Time
}

// neigongPassiveCache 是玩家侧的 M.passive 静态属性快照缓存。被动是
// 档位确定的纯函数（幻玉 evasion_add、坐忘 damage_reduction_add 等），
// 按 (宿主代数, buffID, 层位) 缓存：热重载（代数变化）/换书/升级自动失效。
type neigongPassiveCache struct {
	generation int64
	buffID     string
	level      int32
	maxLevel   int32
	values     map[string]float64
}

// neigongPassiveValues 返回装备书的 M.passive 扁平数值属性（脚本权威，
// scripts/module/neigong）。无宿主/无书/无 passive 钩子 → nil。
func (p *playerActor) neigongPassiveValues() map[string]float64 {
	host := neigongEffects
	if host == nil || !host.Enabled() {
		return nil
	}
	p.mu.Lock()
	book, ok := p.equippedNeiGongBookLocked()
	cache := p.neigongPassive
	p.mu.Unlock()
	if !ok {
		return nil
	}
	if cache.buffID == book.buffID && cache.level == book.level &&
		cache.maxLevel == book.maxLevel && cache.generation == host.Generation() {
		return cache.values
	}
	values := host.Passive(book.buffID, &neigong.Ctx{
		BookLevel:    int(book.level),
		BookMaxLevel: int(book.maxLevel),
		StoryStage:   neigongStoryStage,
	})
	p.mu.Lock()
	p.neigongPassive = neigongPassiveCache{
		generation: host.Generation(), buffID: book.buffID,
		level: book.level, maxLevel: book.maxLevel, values: values,
	}
	p.mu.Unlock()
	return values
}

// neigongPassiveEvasion 是脚本被动的常驻闪避加成（百分点；幻玉神功
// evasion_add 6..20）。并入受击裁定的基础闪避。
func (p *playerActor) neigongPassiveEvasion() int32 {
	if values := p.neigongPassiveValues(); values != nil {
		return int32(values["evasion_add"])
	}
	return 0
}

// neigongPassiveDamageReduction 是脚本被动的平减值（点；坐忘功
// damage_reduction_add 20..250）。并入受击裁定，floor 1 点。
func (p *playerActor) neigongPassiveDamageReduction() int32 {
	if values := p.neigongPassiveValues(); values != nil {
		return int32(values["damage_reduction_add"])
	}
	return 0
}

// neigongBookSnapshot 是分发所需的装备书快照（p.mu 内读取）。
type neigongBookSnapshot struct {
	buffID   string // buf_ng_*（模块键，来自 varprop BufferID 家族）
	level    int32
	maxLevel int32
}

// equippedNeiGongBook 返回当前装备内功的脚本分发快照；无书或无 buffID
// （未装备/无 varprop）时 ok=false，调用方走零效果路径。
// 调用方必须已持有 p.mu。
func (p *playerActor) equippedNeiGongBookLocked() (neigongBookSnapshot, bool) {
	book := p.progress.book(p.progress.curNeiGong)
	if book == nil || book.buffID == "" || book.level <= 0 {
		return neigongBookSnapshot{}, false
	}
	return neigongBookSnapshot{buffID: book.buffID, level: book.level, maxLevel: book.maxLevel}, true
}

// hookNeiGong 是四类事件共用的派发骨架：p.mu 内取快照（书/自身/state/
// 盾），锁外调用脚本（modulevm 为叶子锁，微秒级；坏书只禁用该本），
// 再回锁应用效果。host 不可用/无书/无钩子 → 零效果。
// 返回的效果由调用方按事件语义消费。
func (p *playerActor) hookNeiGong(event string, rollSalt uint64, rollOID uint32, now time.Time, fill func(ctx *neigong.Ctx)) *neigong.Effect {
	host := neigongEffects
	if host == nil || !host.Enabled() {
		return nil
	}
	p.mu.Lock()
	book, ok := p.equippedNeiGongBookLocked()
	if !ok {
		p.mu.Unlock()
		return nil
	}
	snapshot := p.vitals.ActorState(playerObjectID, playerOwnerID)
	ctx := &neigong.Ctx{
		BookLevel:    int(book.level),
		BookMaxLevel: int(book.maxLevel),
		StoryStage:   neigongStoryStage,
		NowMS:        now.UnixMilli(),
		Self: neigong.SelfStats{
			HP: int64(snapshot.HP), MaxHP: int64(snapshot.MaxHP),
			MP: int64(snapshot.MP), MaxMP: int64(snapshot.MaxMP),
		},
		State: p.neigongState,
		RProc: float64(combatPercentRoll(rollOID, now, rollSalt)) / 100,
	}
	if fill != nil {
		fill(ctx)
	}
	p.mu.Unlock()

	effect := host.Hook(book.buffID, event, ctx)
	if effect == nil {
		return nil
	}
	p.mu.Lock()
	if effect.State != nil {
		p.neigongState = effect.State
	}
	if effect.Shield != nil {
		p.neigongShields = append(p.neigongShields, neigongShield{
			amount:    int32(effect.Shield.Amount),
			expiresAt: now.Add(time.Duration(effect.Shield.LifeMS) * time.Millisecond),
		})
	}
	if effect.DodgeAdd != nil {
		expires := now.Add(time.Duration(effect.DodgeAdd.LifeMS) * time.Millisecond)
		p.neigongDodge = neigongDodgeWindow{add: int32(effect.DodgeAdd.Amount), expiresAt: expires}
		// 客户端可见的料敌先机 Proc Buff（原生家族勘定值，见常量注释）。
		p.buffs[neigongProcBuffSlot] = activePlayerBuff{
			staticData: neigongProcBuffStaticData,
			expiresUTC: expires.UTC(),
			configID:   neigongProcBuffConfigID,
			level:      1,
		}
	}
	if effect.Heal > 0 {
		p.vitals.ApplyHeal(int32(effect.Heal))
	}
	if effect.MPRestore > 0 {
		p.vitals.RestoreMP(int32(effect.MPRestore))
	}
	p.mu.Unlock()
	return effect
}

// neigongDodgeAdd 返回当前闪避加成窗的加成（百分点）；窗过期即清
// （含原生 Proc Buff 摘除，随后战斗 tick 的 changed 帧推送清除）。
func (p *playerActor) neigongDodgeAdd(now time.Time) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.neigongDodgeAddLocked(now)
}

// neigongDodgeAddLocked 是闪避窗读取的锁内实现。
// 调用方必须已持有 p.mu。
func (p *playerActor) neigongDodgeAddLocked(now time.Time) int32 {
	if p.neigongDodge.add <= 0 || now.After(p.neigongDodge.expiresAt) {
		return 0
	}
	return p.neigongDodge.add
}

// expireNeiGongDodgeWindow 是战斗 tick 的过期清扫：窗到期后清窗并摘除
// 原生 Proc Buff。返回是否发生变化（供 changed 帧标记）。
func (p *playerActor) expireNeiGongDodgeWindow(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.neigongDodge.add <= 0 {
		return false
	}
	if now.Before(p.neigongDodge.expiresAt) {
		return false
	}
	p.neigongDodge = neigongDodgeWindow{}
	delete(p.buffs, neigongProcBuffSlot)
	return true
}

// dispatchNeiGongDodge 派发 on_dodge（成功闪避事件，本仓扩展钩子）。
// 闪避本身无伤：只落 heal/mp_restore/shield/state（移花接玉等）。
func (p *playerActor) dispatchNeiGongDodge(attackerOID uint32, incoming int32, now time.Time) bool {
	effect := p.hookNeiGong("on_dodge", neigongRollDodge, attackerOID, now, func(ctx *neigong.Ctx) {
		ctx.Incoming = int64(incoming)
		ctx.AttackerOID = attackerOID
	})
	return effect != nil
}

// dispatchNeiGongVictim 派发 on_victim（玩家被 NPC 命中、未格挡）。
// 返回折减与吸收后的实扣伤害；heal/mp 由效果骨架直接落账（含 vital），
// reflectAmount 交由调用方反打 NPC（击杀走 NPC 常规死亡链）。
func (p *playerActor) dispatchNeiGongVictim(attackerOID uint32, incoming int32, now time.Time) (applied, reflectAmount int32, changed bool) {
	effect := p.hookNeiGong("on_victim", neigongRollVictim, attackerOID, now, func(ctx *neigong.Ctx) {
		ctx.Incoming = int64(incoming)
		ctx.AttackerOID = attackerOID
	})
	if effect == nil {
		return p.absorbNeiGongShields(incoming, now), 0, false
	}
	damage := incoming
	if effect.ReducePct > 0 {
		damage = int32(float64(damage) * (1 - effect.ReducePct))
	}
	damage = p.absorbNeiGongShields(damage, now)
	if effect.Reflect != nil && effect.Reflect.Amount > 0 {
		reflectAmount = int32(effect.Reflect.Amount)
	}
	return damage, reflectAmount, true
}

// dispatchNeiGongParry 派发 on_parry（玩家格挡下 NPC 攻击）。当前管线
// 格挡零伤，只落 state/heal/mp/shield（对齐参考服语义）。
func (p *playerActor) dispatchNeiGongParry(attackerOID uint32, incoming int32, now time.Time) bool {
	effect := p.hookNeiGong("on_parry", neigongRollParry, attackerOID, now, func(ctx *neigong.Ctx) {
		ctx.Incoming = int64(incoming)
		ctx.AttackerOID = attackerOID
		ctx.Parried = true
	})
	return effect != nil
}

// dispatchNeiGongAttack 派发 on_attack（玩家技能实中目标，miss 不触发）。
// 返回计入附伤后的落血值。
func (p *playerActor) dispatchNeiGongAttack(targetOID uint32, targetHP, targetMaxHP, damage int32, now time.Time) int32 {
	effect := p.hookNeiGong("on_attack", neigongRollAttack, targetOID, now, func(ctx *neigong.Ctx) {
		ctx.Damage = int64(damage)
		ctx.Target = &neigong.TargetInfo{OID: targetOID, HP: int64(targetHP), MaxHP: int64(targetMaxHP)}
	})
	if effect == nil || effect.ExtraDamage == nil || effect.ExtraDamage.Amount <= 0 {
		return damage
	}
	return damage + int32(effect.ExtraDamage.Amount)
}

// dispatchNeiGongPeriodic 派发 on_periodic（战斗期 1s 节拍，脱战自停）。
// 返回效果是否落账（供调用方标记属性帧变化）。盾/heal/mp/state 由骨架
// 直接应用。
func (p *playerActor) dispatchNeiGongPeriodic(now time.Time) bool {
	p.mu.Lock()
	last := p.neigongPeriodicAt
	p.mu.Unlock()
	if !last.IsZero() {
		if since := now.Sub(last); since < time.Second {
			return false
		}
	}
	sinceMS := int64(-1)
	if !last.IsZero() {
		sinceMS = now.Sub(last).Milliseconds()
	}
	effect := p.hookNeiGong("on_periodic", neigongRollPeriodic, playerObjectID, now, func(ctx *neigong.Ctx) {
		ctx.SinceLastMS = sinceMS
	})
	p.mu.Lock()
	p.neigongPeriodicAt = now
	p.mu.Unlock()
	return effect != nil
}

// absorbNeiGongShields 用盾池 FIFO 吸收伤害：过期盾直接出池，吸收到 0
// 为止（全吸收=静默零帧）。返回剩余伤害。
func (p *playerActor) absorbNeiGongShields(damage int32, now time.Time) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.absorbNeiGongShieldsLocked(damage, now)
}

// absorbNeiGongShieldsLocked 是盾吸收的锁内实现。
// 调用方必须已持有 p.mu。
func (p *playerActor) absorbNeiGongShieldsLocked(damage int32, now time.Time) int32 {
	alive := p.neigongShields[:0]
	for _, shield := range p.neigongShields {
		if now.After(shield.expiresAt) {
			continue
		}
		if damage <= 0 {
			alive = append(alive, shield)
			continue
		}
		if shield.amount > damage {
			alive = append(alive, neigongShield{amount: shield.amount - damage, expiresAt: shield.expiresAt})
			damage = 0
			continue
		}
		damage -= shield.amount
	}
	for i := len(alive); i < len(p.neigongShields); i++ {
		p.neigongShields[i] = neigongShield{}
	}
	p.neigongShields = alive
	return damage
}
