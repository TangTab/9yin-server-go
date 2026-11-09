package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/neigong"
)

// loadNeiGongDispatchHost 用临时脚本目录装载内功宿主并注入分发层全局，
// 测试结束还原为 nil。
func loadNeiGongDispatchHost(t *testing.T, dir string) {
	t.Helper()
	host := neigong.New(dir, nil)
	if _, err := host.Load(); err != nil {
		t.Fatalf("load neigong host: %v", err)
	}
	previous := neigongEffects
	neigongEffects = host
	t.Cleanup(func() { neigongEffects = previous })
}

// equipProbeBook 把玩家的装备书换成指定 buffID 的探针书。
func equipProbeBook(t *testing.T, p *playerActor, buffID string, level, maxLevel int32) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.books = []learnedNeiGong{{
		configID: "probe_book", staticData: 1, itemType: 1002,
		level: level, maxLevel: maxLevel, buffID: buffID,
	}}
	p.progress.curNeiGong = "probe_book"
	p.neigongState = nil
	p.neigongShields = nil
	p.neigongPassive = neigongPassiveCache{}
}

func writeDispatchModule(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}

// nonDodgingAttacker 在固定时刻找一个不会被基础 10% 闪避命中的攻击者 ID
// （闪避掷确定论同 combatPercentRoll salt=18）。
func nonDodgingAttacker(now time.Time) uint32 {
	for id := uint32(1); id < 100; id++ {
		if combatPercentRoll(id, now, 18) >= baseLocalDodgeProbability {
			return id
		}
	}
	return 999
}

func TestDispatchVictimReduceReflectHealState(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_victim.lua", `local M = {}
M.meta = { id = "probe_victim", name = "受击探针", book = "ng_probe_v", category = "victim", max_tier = 1 }
function M.on_victim(ctx)
  local s = ctx.state or {}
  return {
    reduce_pct = 0.5,
    reflect = { amount = 20, kind = "yang" },
    heal = 5,
    state = { hits = (s.hits or 0) + 1 },
  }
end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_victim", 6, 36)
	now := time.Unix(1_700_000_000, 0)
	attacker := nonDodgingAttacker(now)
	// 先打掉一部分血，让 heal 有落账空间（满血时 ApplyHeal 被钳）。
	player.mu.Lock()
	player.vitals.ApplyDamage(300)
	before := player.vitals.HP
	player.mu.Unlock()

	outcome := player.resolveNPCAttack(attacker, 100, now)
	if outcome.dodged || outcome.parried || outcome.dead {
		t.Fatalf("outcome = %+v", outcome)
	}
	// 50% 折减 → 实扣 50；heal +5。
	after := player.stateSnapshot()
	if after.HP != before-50+5 {
		t.Fatalf("HP = %d, want %d (100 - 50%% + heal 5)", after.HP, before-50+5)
	}
	if outcome.neigongReflect != 20 {
		t.Fatalf("neigongReflect = %d, want 20", outcome.neigongReflect)
	}
	// state 回写：第二次受击 hits=2。
	player.resolveNPCAttack(attacker, 100, now.Add(time.Second))
	player.mu.Lock()
	hits := player.neigongState["hits"]
	player.mu.Unlock()
	if hits != int64(2) {
		t.Fatalf("state.hits = %v, want 2", hits)
	}
}

func TestDispatchVictimShieldAbsorption(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_shield.lua", `local M = {}
M.meta = { id = "probe_shield", name = "护盾探针", book = "ng_probe_s", category = "victim", max_tier = 1 }
function M.on_victim(ctx)
  local s = ctx.state or {}
  if not s.armed then
    return { shield = { amount = 100, life_ms = 60000 }, state = { armed = true } }
  end
end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_shield", 6, 36)
	now := time.Unix(1_700_000_000, 0)
	attacker := nonDodgingAttacker(now)
	before := player.stateSnapshot().HP

	// 第一击：授盾 100，全吸收 60 → HP 不动。
	if outcome := player.resolveNPCAttack(attacker, 60, now); outcome.dodged || outcome.dead || !outcome.stateChanged {
		t.Fatalf("first outcome = %+v", outcome)
	}
	if hp := player.stateSnapshot().HP; hp != before {
		t.Fatalf("HP after full absorb = %d, want %d", hp, before)
	}
	// 第二击：余盾 40 吸收 → 60-40=20 落血。
	if outcome := player.resolveNPCAttack(attacker, 60, now.Add(time.Second)); outcome.dodged || outcome.dead {
		t.Fatalf("second outcome = %+v", outcome)
	}
	if hp := player.stateSnapshot().HP; hp != before-20 {
		t.Fatalf("HP after partial absorb = %d, want %d", hp, before-20)
	}
	// 第三击：无盾全额落血。
	player.resolveNPCAttack(attacker, 30, now.Add(2*time.Second))
	if hp := player.stateSnapshot().HP; hp != before-50 {
		t.Fatalf("HP after shield gone = %d, want %d", hp, before-50)
	}
}

func TestShieldPoolFIFOAndExpiry(t *testing.T) {
	player := newPlayerActor("探针", 0)
	now := time.Unix(1_700_000_000, 0)
	player.mu.Lock()
	player.neigongShields = []neigongShield{
		{amount: 30, expiresAt: now.Add(time.Minute)},
		{amount: 50, expiresAt: now.Add(time.Minute)},
		{amount: 999, expiresAt: now.Add(-time.Second)}, // 已过期
	}
	player.mu.Unlock()
	if left := player.absorbNeiGongShields(80, now); left != 0 {
		t.Fatalf("absorb 80 = %d, want 0 (30+50 恰好吸收)", left)
	}
	if left := player.absorbNeiGongShields(10, now); left != 10 {
		t.Fatalf("absorb 10 after empty = %d, want 10", left)
	}
	// 过期盾直接出池，不参与吸收。
	player.mu.Lock()
	player.neigongShields = []neigongShield{{amount: 100, expiresAt: now.Add(-time.Second)}}
	player.mu.Unlock()
	if left := player.absorbNeiGongShields(5, now); left != 5 {
		t.Fatalf("absorb 5 with expired shield = %d, want 5", left)
	}
	if len(player.neigongShields) != 0 {
		t.Fatalf("expired shield not dropped: %v", player.neigongShields)
	}
}

func TestDispatchParry(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_parry.lua", `local M = {}
M.meta = { id = "probe_parry", name = "格挡探针", book = "ng_probe_p", category = "victim", max_tier = 1 }
function M.on_parry(ctx)
  return { heal = 7, state = { parried = true }, mp_restore = 11 }
end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_parry", 6, 36)
	// 先打掉一部分血/蓝再格挡，验证 heal/mp 落账。
	player.mu.Lock()
	player.vitals.ApplyDamage(100)
	player.vitals.SpendMP(50)
	hpBefore, mpBefore := player.vitals.HP, player.vitals.MP
	player.mu.Unlock()
	player.setActiveParry(activeParryRequest{At: time.Unix(1_700_000_000, 0)})
	now := time.Unix(1_700_000_000, 0).Add(100 * time.Millisecond)

	outcome := player.resolveNPCAttack(7, 200, now)
	if !outcome.parried || outcome.dead || outcome.dodged {
		t.Fatalf("outcome = %+v", outcome)
	}
	player.mu.Lock()
	hp, mp := player.vitals.HP, player.vitals.MP
	state := player.neigongState
	player.mu.Unlock()
	if hp != hpBefore+7 || mp != mpBefore+11 {
		t.Fatalf("hp/mp = %d/%d, want %d/%d", hp, mp, hpBefore+7, mpBefore+11)
	}
	if state["parried"] != true {
		t.Fatalf("state = %v", state)
	}
}

func TestDispatchAttackExtraDamage(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_attack.lua", `local M = {}
M.meta = { id = "probe_attack", name = "出招探针", book = "ng_probe_a", category = "hit", max_tier = 1 }
function M.on_attack(ctx)
  local s = ctx.state or {}
  local bonus = (ctx.target and ctx.target.hp > 500) and 25 or 5
  return { extra_damage = { amount = bonus }, state = { count = (s.count or 0) + 1 } }
end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_attack", 6, 36)
	now := time.Unix(1_700_000_000, 0)

	if got := player.dispatchNeiGongAttack(9, 800, 1000, 100, now); got != 125 {
		t.Fatalf("attack damage = %d, want 125 (target.hp>500 → +25)", got)
	}
	if got := player.dispatchNeiGongAttack(9, 300, 1000, 100, now); got != 105 {
		t.Fatalf("attack damage = %d, want 105 (低血目标 → +5)", got)
	}
	player.mu.Lock()
	count := player.neigongState["count"]
	player.mu.Unlock()
	if count != int64(2) {
		t.Fatalf("state.count = %v, want 2", count)
	}
}

func TestDispatchPeriodicBeat(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_periodic.lua", `local M = {}
M.meta = { id = "probe_periodic", name = "周期探针", book = "ng_probe_t", category = "passive", max_tier = 1 }
M.SHIELD_INTERVAL_MS = 30000
function M.on_periodic(ctx)
  local since = ctx.since_last_ms or -1
  if since >= 0 and since < M.SHIELD_INTERVAL_MS then return nil end
  return { heal = 3, state = { last_at = ctx.now or 0 } }
end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_periodic", 6, 36)
	player.mu.Lock()
	player.vitals.ApplyDamage(30)
	hpBefore := player.vitals.HP
	player.mu.Unlock()
	now := time.Unix(1_700_000_000, 0)

	// 首拍（since=-1）生效：heal 3。
	if !player.dispatchNeiGongPeriodic(now) {
		t.Fatal("first periodic beat should fire")
	}
	// 1s 节拍门内（since<30s）静默。
	if player.dispatchNeiGongPeriodic(now.Add(time.Second)) {
		t.Fatal("periodic within cooldown should not fire")
	}
	// 冷却窗外再次生效。
	if !player.dispatchNeiGongPeriodic(now.Add(31 * time.Second)) {
		t.Fatal("periodic after cooldown should fire")
	}
	if hp := player.stateSnapshot().HP; hp != hpBefore+6 {
		t.Fatalf("HP = %d, want %d (两拍 heal 3)", hp, hpBefore+6)
	}
}

// 无宿主 / 无书：全部分发点零效果，战斗链行为与接入前逐位一致。
func TestDispatchDisabledPaths(t *testing.T) {
	player := newPlayerActor("探针", 0)
	now := time.Unix(1_700_000_000, 0)
	attacker := nonDodgingAttacker(now)
	before := player.stateSnapshot().HP

	// 宿主 nil（默认：main 之外未注入）。
	outcome := player.resolveNPCAttack(attacker, 100, now)
	if outcome.neigongReflect != 0 || outcome.stateChanged != true || player.stateSnapshot().HP != before-100 {
		t.Fatalf("nil host changed combat behavior: %+v", outcome)
	}

	// 宿主启用但玩家无书（buffID 为空）。
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_victim.lua", `local M = {}
M.meta = { id = "probe_victim", name = "受击探针", book = "ng_probe_v", category = "victim", max_tier = 1 }
function M.on_victim(ctx) return { reduce_pct = 0.5 } end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player.mu.Lock()
	player.progress.books = []learnedNeiGong{{configID: "裸书", staticData: 1, level: 1, maxLevel: 36}}
	player.progress.curNeiGong = "裸书"
	player.mu.Unlock()
	if applied, reflectAmount, changed := player.dispatchNeiGongVictim(attacker, 100, now); applied != 100 || reflectAmount != 0 || changed {
		t.Fatalf("book without buffID must be zero-effect: %d/%d/%v", applied, reflectAmount, changed)
	}

	// 未知 buffID（宿主未装载该书）同样零效果。
	equipProbeBook(t, player, "buf_ng_unknown_999", 6, 36)
	if applied, _, _ := player.dispatchNeiGongVictim(attacker, 100, now); applied != 100 {
		t.Fatalf("unknown buffID must be zero-effect, got %d", applied)
	}
}

func TestEquipNeiGongResetsDispatchState(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_victim.lua", `local M = {}
M.meta = { id = "probe_victim", name = "受击探针", book = "ng_probe_v", category = "victim", max_tier = 1 }
function M.on_victim(ctx) return { state = { hits = 1 } } end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_victim", 6, 36)
	now := time.Unix(1_700_000_000, 0)
	attacker := nonDodgingAttacker(now)
	player.resolveNPCAttack(attacker, 10, now)
	player.mu.Lock()
	player.neigongShields = []neigongShield{{amount: 50, expiresAt: now.Add(time.Hour)}}
	player.mu.Unlock()

	// 换装另一本书：状态与盾池清空。
	player.mu.Lock()
	player.progress.books = append(player.progress.books,
		learnedNeiGong{configID: "other", staticData: 2, level: 1, maxLevel: 36, buffID: "buf_ng_other"})
	player.mu.Unlock()
	if err := player.equipNeiGong("other"); err != nil {
		t.Fatal(err)
	}
	player.mu.Lock()
	state, shields := player.neigongState, player.neigongShields
	player.mu.Unlock()
	if state != nil || shields != nil {
		t.Fatalf("dispatch state must reset on book change: %v %v", state, shields)
	}
}

// 真实脚本被动钉子（统一以参考服脚本为权威）：坐忘功（buf_ng_jh_001）
// 的 damage_reduction_add 平减；幻玉（buf_ng_yh_001）的常驻被动已由
// 真实料敌先机/移花接玉机制取代（见 TestRealHuanYuVictimDodgeHeal）。
func TestScriptPassiveEvasionAndReduction(t *testing.T) {
	loadNeiGongDispatchHost(t, filepath.Join("..", "..", "scripts", "module", "neigong"))
	now := time.Unix(1_700_000_000, 0)

	// 幻玉神功：常驻被动已移除（evasion_add 0），闪避来自 dodge_add 窗。
	huanYU := newPlayerActor("幻玉", 0)
	equipProbeBook(t, huanYU, "buf_ng_yh_001", 6, 36)
	if got := huanYU.neigongPassiveEvasion(); got != 0 {
		t.Fatalf("幻玉 evasion_add = %d, want 0（常驻近似已移除）", got)
	}
	// 裸书不得获得任何被动/窗口加成。
	var edgeCase uint32
	for id := uint32(1); id < 200; id++ {
		if roll := combatPercentRoll(id, now, 18); roll >= baseLocalDodgeProbability && roll < baseLocalDodgeProbability+20 {
			edgeCase = id
			break
		}
	}
	if edgeCase == 0 {
		t.Fatal("no attacker with roll in [10,30) found")
	}
	bare := newPlayerActor("裸身", 0)
	bare.mu.Lock()
	bare.progress.books = []learnedNeiGong{{configID: "裸书", staticData: 1, level: 1, maxLevel: 36}}
	bare.progress.curNeiGong = "裸书"
	bare.mu.Unlock()
	if outcome := bare.resolveNPCAttack(edgeCase, 50, now); outcome.dodged {
		t.Fatal("无 buffID 的书不得获得被动闪避")
	}

	// 坐忘功：层 1/5 → 档1 → damage_reduction_add 20；受击 100 → 平减后 80。
	zuoWang := newPlayerActor("坐忘", 0)
	equipProbeBook(t, zuoWang, "buf_ng_jh_001", 1, 5)
	if got := zuoWang.neigongPassiveDamageReduction(); got != 20 {
		t.Fatalf("坐忘 damage_reduction_add = %d, want 20 (tier1)", got)
	}
	attacker := nonDodgingAttacker(now)
	before := zuoWang.stateSnapshot().HP
	outcome := zuoWang.resolveNPCAttack(attacker, 100, now)
	if outcome.dodged || outcome.dead || outcome.neigongReflect != 0 {
		t.Fatalf("坐忘受击 outcome = %+v", outcome)
	}
	if hp := zuoWang.stateSnapshot().HP; hp != before-80 {
		t.Fatalf("坐忘受击后 HP = %d, want %d (100-20)", hp, before-80)
	}
	// 小伤害 floor 1 点。
	small := newPlayerActor("坐忘", 0)
	equipProbeBook(t, small, "buf_ng_jh_001", 1, 5)
	smallBefore := small.stateSnapshot().HP
	small.resolveNPCAttack(attacker, 10, now)
	if hp := small.stateSnapshot().HP; hp != smallBefore-1 {
		t.Fatalf("平减 floor = %d, want %d (10-20 → 1)", hp, smallBefore-1)
	}
}

// 被动缓存的代数失效：热重载（generation 变化）后重算。
func TestScriptPassiveCacheInvalidatedByReload(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_passive.lua", `local M = {}
M.meta = { id = "probe_passive", name = "被动探针", book = "ng_probe_p", category = "passive", max_tier = 1 }
function M.passive(ctx) return { evasion_add = 5 } end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_passive", 1, 36)
	if got := player.neigongPassiveEvasion(); got != 5 {
		t.Fatalf("evasion = %d, want 5", got)
	}
	// 改脚本并热重载：代数变化令缓存失效。
	writeDispatchModule(t, dir, "probe_passive.lua", `local M = {}
M.meta = { id = "probe_passive", name = "被动探针", book = "ng_probe_p", category = "passive", max_tier = 1 }
function M.passive(ctx) return { evasion_add = 9 } end
return M
`)
	if _, err := neigongEffects.Reload(); err != nil {
		t.Fatal(err)
	}
	if got := player.neigongPassiveEvasion(); got != 9 {
		t.Fatalf("evasion after reload = %d, want 9 (cache must invalidate)", got)
	}
}

// procAttacker 找一个在固定时刻满足全部判定条件的攻击者：命中掷不低于
// dodge（不闪避）、proc 掷低于 procChance（料敌先机触发）。
func procAttacker(now time.Time, dodge int32, procChance int64) uint32 {
	for id := uint32(1); id < 500; id++ {
		if combatPercentRoll(id, now, 18) >= dodge && int64(combatPercentRoll(id, now, 31)) < procChance {
			return id
		}
	}
	return 0
}

// windowDodger 找一个掷值落在 [base, base+window) 的攻击者（窗内闪避、
// 无窗命中）。
func windowDodger(now time.Time, window int32) uint32 {
	for id := uint32(1); id < 500; id++ {
		if roll := combatPercentRoll(id, now, 18); roll >= baseLocalDodgeProbability && roll < baseLocalDodgeProbability+window {
			return id
		}
	}
	return 0
}

// 真实幻玉神功端到端：料敌先机（on_victim proc → dodge_add 窗 + 原生
// Buff 槽 10）→ 窗内闪避 → 移花接玉（on_dodge 回血 + 冷却）→ 窗过期清 Buff。
func TestRealHuanYuVictimDodgeHeal(t *testing.T) {
	loadNeiGongDispatchHost(t, filepath.Join("..", "..", "scripts", "module", "neigong"))
	player := newPlayerActor("幻玉", 0)
	equipProbeBook(t, player, "buf_ng_yh_001", 3, 36) // 档1：+20%/5s/36s cd、回 2%/4s cd
	now := time.Unix(1_700_000_000, 0)
	maxHP := player.stateSnapshot().MaxHP

	// 第一步：被命中且 proc 掷中（50%）→ 开窗 + 挂原生 Buff。
	attacker := procAttacker(now, baseLocalDodgeProbability, 50)
	if attacker == 0 {
		t.Fatal("no proc attacker found")
	}
	if outcome := player.resolveNPCAttack(attacker, 40, now); outcome.dodged || !outcome.stateChanged {
		t.Fatalf("proc hit outcome = %+v", outcome)
	}
	player.mu.Lock()
	window, buffSlot10 := player.neigongDodge, p_buffs_or_zero(player, neigongProcBuffSlot)
	player.mu.Unlock()
	if window.add != 20 || !window.expiresAt.Equal(now.Add(5*time.Second)) {
		t.Fatalf("dodge window = %+v, want +20/5s", window)
	}
	if buffSlot10.staticData != neigongProcBuffStaticData || buffSlot10.configID != neigongProcBuffConfigID {
		t.Fatalf("proc buff slot 10 = %+v, want static=%d config=%s", buffSlot10, neigongProcBuffStaticData, neigongProcBuffConfigID)
	}

	// 第二步：窗内 [10,30) 掷 → 闪避 → 移花接玉回 2% 最大气血。
	now2 := now.Add(time.Second)
	dodger := windowDodger(now2, 20)
	if dodger == 0 {
		t.Fatal("no window dodger found")
	}
	player.mu.Lock()
	player.vitals.ApplyDamage(300) // 腾出回血空间
	hpBeforeDodge := player.vitals.HP
	player.mu.Unlock()
	outcome := player.resolveNPCAttack(dodger, 40, now2)
	if !outcome.dodged || !outcome.stateChanged {
		t.Fatalf("window dodge outcome = %+v", outcome)
	}
	wantHeal := maxHP * 2 / 100
	if hp := player.stateSnapshot().HP; hp != hpBeforeDodge+wantHeal {
		t.Fatalf("dodge-heal HP = %d, want %d (+%d)", hp, hpBeforeDodge, wantHeal)
	}
	// state 冷却窗已回写。
	player.mu.Lock()
	healUntil, procUntil := player.neigongState["heal_until"], player.neigongState["proc_until"]
	player.mu.Unlock()
	if healUntil == nil || procUntil == nil {
		t.Fatalf("state = %v, want heal_until/proc_until", player.neigongState)
	}

	// 第三步：窗过期清扫 → Buff 摘除、闪避回落基础值。
	now3 := now.Add(6 * time.Second)
	if !player.expireNeiGongDodgeWindow(now3) {
		t.Fatal("window expiry sweep should report change")
	}
	player.mu.Lock()
	_, mounted := player.buffs[neigongProcBuffSlot]
	player.mu.Unlock()
	if mounted || player.neigongDodgeAdd(now3) != 0 {
		t.Fatal("expired window must clear buff and dodge bonus")
	}
	dodger3 := windowDodger(now3, 20)
	if outcome := player.resolveNPCAttack(dodger3, 40, now3); outcome.dodged {
		t.Fatal("dodge must fall back to base after window expiry")
	}
}

// p_buffs_or_zero 读取槽位 buff（缺失返回零值），仅供测试断言。
func p_buffs_or_zero(p *playerActor, slot uint16) activePlayerBuff {
	buff, ok := p.buffs[slot]
	if !ok {
		return activePlayerBuff{}
	}
	return buff
}

// 移花接玉充能（层 6-7 语义，档位门内不可达故用探针复刻）：闪避充能 →
// 下一击 on_attack 兑现 +60% 附伤并清充能。
func TestDodgeChargedNextHit(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_charge.lua", `local M = {}
M.meta = { id = "probe_charge", name = "充能探针", book = "ng_probe_c", category = "victim", max_tier = 1 }
function M.on_dodge(ctx)
  local s = ctx.state or {}
  if s.charged then return nil end
  return { heal = 1, state = { charged = true } }
end
function M.on_attack(ctx)
  local s = ctx.state or {}
  if not s.charged then return nil end
  return { extra_damage = { amount = math.floor((ctx.damage or 0) * 0.6), kind = "yu" },
           state = {} }
end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_charge", 1, 36)
	now := time.Unix(1_700_000_000, 0)

	if !player.dispatchNeiGongDodge(7, 50, now) {
		t.Fatal("on_dodge should fire and charge")
	}
	if got := player.dispatchNeiGongAttack(9, 500, 1000, 100, now); got != 160 {
		t.Fatalf("charged hit = %d, want 160 (100 + 60%%)", got)
	}
	// 充能已消费：再次出招无附伤。
	if got := player.dispatchNeiGongAttack(9, 500, 1000, 100, now); got != 100 {
		t.Fatalf("uncharged hit = %d, want 100", got)
	}
}

// dodge_add 指令畸形值整条丢弃（白名单转换层）。
func TestDodgeAddMalformed(t *testing.T) {
	dir := t.TempDir()
	writeDispatchModule(t, dir, "probe_bad_dodge.lua", `local M = {}
M.meta = { id = "probe_bad_dodge", name = "畸形窗", book = "ng_probe_b", category = "victim", max_tier = 1 }
function M.on_victim(ctx)
  return { dodge_add = { amount = 0, life_ms = 5000 } }
end
return M
`)
	loadNeiGongDispatchHost(t, dir)
	player := newPlayerActor("探针", 0)
	equipProbeBook(t, player, "probe_bad_dodge", 1, 36)
	now := time.Unix(1_700_000_000, 0)
	attacker := nonDodgingAttacker(now)
	player.resolveNPCAttack(attacker, 50, now)
	player.mu.Lock()
	window := player.neigongDodge
	_, mounted := player.buffs[neigongProcBuffSlot]
	player.mu.Unlock()
	if window.add != 0 || mounted {
		t.Fatalf("malformed dodge_add must be dropped: window=%+v mounted=%v", window, mounted)
	}
}
