package skillscript

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type logSink struct{ lines []string }

func (s *logSink) printf(format string, args ...any) {
	s.lines = append(s.lines, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func (s *logSink) has(substr string) bool {
	for _, line := range s.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

// realScriptsDir 指向仓内真实脚本（scripts/module/skill：6 覆盖 + effects_data）。
func realScriptsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "scripts", "module", "skill")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("real scripts dir: %v", err)
	}
	return dir
}

func newRealScripts(t *testing.T, sink *logSink) *Scripts {
	t.Helper()
	s := New(realScriptsDir(t), sink.printf)
	if _, err := s.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return s
}

func writeModule(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRealSkillScripts(t *testing.T) {
	s := newRealScripts(t, &logSink{})
	if s.Loaded() != 10 {
		t.Fatalf("Loaded = %d, want 10 (8 overlays + effects_data + desc_effects)", s.Loaded())
	}
	want := []string{
		"CS_jh_hyjf08", "CS_jh_tqjf08",
		"CS_wd_tjq04", "CS_wd_tjq05", "CS_wd_tjq07", "CS_wd_tjq08",
		"CS_yhwq_hsqs04", "CS_yhwq_hsqs05",
	}
	if got := s.Overlays(); len(got) != len(want) {
		t.Fatalf("Overlays = %v, want %v", got, want)
	}
	for i, id := range want {
		if got := s.Overlays(); got[i] != id {
			t.Fatalf("Overlays[%d] = %s, want %s", i, got[i], id)
		}
	}
	if s.EffectDataCount() != 6433 {
		t.Fatalf("EffectDataCount = %d, want 6433", s.EffectDataCount())
	}
	if len(s.Broken()) != 0 {
		t.Fatalf("Broken = %v", s.Broken())
	}
}

// 覆盖脚本行为钉子：太极拳四式的架势轮换（守/攻双态，24h 租约，槽 7）。
func TestRealOverlayStanceCycle(t *testing.T) {
	s := newRealScripts(t, &logSink{})
	overlay := s.Overlay("CS_wd_tjq04", &BuffDef{ConfigID: "buf_CS_wd_tjq04", StaticData: 6013, Level: 1, LifetimeMS: 35000})
	if overlay == nil || overlay.StanceCycle == nil {
		t.Fatalf("overlay = %+v", overlay)
	}
	variants := overlay.StanceCycle.Variants
	if len(variants) != 2 {
		t.Fatalf("variants = %+v", variants)
	}
	if variants[0] != (StanceVariant{ID: "buf_CS_wd_tjq04_1", StaticData: 6013, Level: 1, LifeMS: 24 * 60 * 60 * 1000, Slot: 7}) {
		t.Fatalf("variant[0] = %+v", variants[0])
	}
	if variants[1].StaticData != 6055 || variants[1].ID != "buf_CS_wd_tjq04_2" {
		t.Fatalf("variant[1] = %+v", variants[1])
	}
	// 其余增量未下发。
	if overlay.Damage != nil || overlay.SelfBuffs != nil || overlay.TargetEffects != nil || overlay.RedArmor || overlay.SelfBuffSlot != 0 || overlay.SelfBuffLeaseMS != 0 {
		t.Fatalf("overlay has unexpected increments: %+v", overlay)
	}
	// 无覆盖技能 → nil（调用方走 effects_data）。
	if overlay := s.Overlay("CS_wd_tjq01", &BuffDef{}); overlay != nil {
		t.Fatalf("tjq01 overlay = %+v, want nil", overlay)
	}
	if !s.HasOverlay("CS_wd_tjq04") || s.HasOverlay("CS_wd_tjq01") {
		t.Fatal("HasOverlay mismatch")
	}
}

// effects_data 行为钉子：CS_wd_tjq01（伤害 + 目标控制 + 逐级伤害差集）。
func TestRealEffectDataEntry(t *testing.T) {
	s := newRealScripts(t, &logSink{})
	entry := s.EffectData("CS_wd_tjq01")
	if entry == nil {
		t.Fatal("EffectData(CS_wd_tjq01) = nil")
	}
	if entry.Damage == nil || entry.Damage.Amount != 112 || entry.Damage.Waves != 1 {
		t.Fatalf("damage = %+v", entry.Damage)
	}
	if len(entry.TargetEffects) != 1 {
		t.Fatalf("target_effects = %+v", entry.TargetEffects)
	}
	te := entry.TargetEffects[0]
	if te.ConfigID != "buf_CS_wd_tjq01" || te.StaticData != 6069 || te.Level != 1 || te.LifeMS != 10000 || !te.Control || te.Frame {
		t.Fatalf("target_effect = %+v", te)
	}
	if len(entry.SelfBuffs) != 0 {
		t.Fatalf("self_buffs = %+v", entry.SelfBuffs)
	}
	if got := entry.ByLevel[5]; got == nil || got.DamageAmount == nil || *got.DamageAmount != 205 {
		t.Fatalf("by_level[5] = %+v, want damage_amount=205", got)
	}
	if got := entry.ByLevel[10]; got == nil || got.DamageAmount == nil || *got.DamageAmount != 341 {
		t.Fatalf("by_level[10] = %+v, want damage_amount=341", got)
	}
	// 显式 0 有效：karma_jh_cqgf01@2 起无伤害（damage_amount/damage_waves
	// = 0 是有效差集，缺字段才沿用基准）。
	zero := s.EffectData("karma_jh_cqgf01")
	if zero == nil || zero.ByLevel[2] == nil ||
		zero.ByLevel[2].DamageAmount == nil || *zero.ByLevel[2].DamageAmount != 0 ||
		zero.ByLevel[2].DamageWaves == nil || *zero.ByLevel[2].DamageWaves != 0 {
		t.Fatalf("explicit zero delta = %+v", zero)
	}
	// 未知技能 → nil。
	if entry := s.EffectData("CS_nope_999"); entry != nil {
		t.Fatalf("unknown = %+v", entry)
	}
}

// 真实通道覆盖钉子：hyjf08（挑战.pcapng 实证）与 tqjf08（同构播种）。
func TestRealChannelOverlay(t *testing.T) {
	s := newRealScripts(t, &logSink{})
	hyjf := s.Overlay("CS_jh_hyjf08", &BuffDef{})
	if hyjf == nil || hyjf.Channel == nil {
		t.Fatalf("hyjf08 overlay = %+v", hyjf)
	}
	want := &ChannelSpec{
		Waves: 8, FirstWaveMS: 180, IntervalMS: 440, SpPerWave: 10,
		VictimBuff: "buf_CS_jh_hyjf08", SelfBuff: "buf_CS_jh_hyjf08_03", SelfBuffStatic: 9019,
	}
	if *hyjf.Channel != *want {
		t.Fatalf("hyjf08 channel = %+v, want %+v", hyjf.Channel, want)
	}
	tqjf := s.Overlay("CS_jh_tqjf08", &BuffDef{})
	if tqjf == nil || tqjf.Channel == nil {
		t.Fatalf("tqjf08 overlay = %+v", tqjf)
	}
	if tqjf.Channel.SelfBuffStatic != 32607 || tqjf.Channel.VictimBuff != "buf_CS_jh_tqjf08" {
		t.Fatalf("tqjf08 channel = %+v", tqjf.Channel)
	}
	// 通道覆盖只下发 channel 增量。
	if hyjf.Damage != nil || hyjf.SelfBuffs != nil || hyjf.TargetEffects != nil || hyjf.RedArmor {
		t.Fatalf("hyjf08 overlay has unexpected increments: %+v", hyjf)
	}
}

// channel 指令校验：任一字段畸形整条丢弃（不部分采纳）。
func TestChannelSpecValidation(t *testing.T) {
	dir := t.TempDir()
	base := `local M = {}
function M.compile(def)
  return { channel = { waves = %s, first_wave_ms = %s, interval_ms = %s, sp_per_wave = %s,
                       victim_buff = %s, self_buff = %s, self_buff_static = %s } }
end
return M
`
	writeModule(t, dir, "probe_good.lua", fmt.Sprintf(base, "8", "180", "440", "10",
		`"buf_probe"`, `"buf_probe_03"`, "9019"))
	writeModule(t, dir, "probe_waves.lua", fmt.Sprintf(base, "8000", "180", "440", "10",
		`"buf_probe"`, `"buf_probe_03"`, "9019"))
	writeModule(t, dir, "probe_interval.lua", fmt.Sprintf(base, "8", "180", "0", "10",
		`"buf_probe"`, `"buf_probe_03"`, "9019"))
	writeModule(t, dir, "probe_sp.lua", fmt.Sprintf(base, "8", "180", "440", "-1",
		`"buf_probe"`, `"buf_probe_03"`, "9019"))
	writeModule(t, dir, "probe_buffs.lua", fmt.Sprintf(base, "8", "180", "440", "10",
		`""`, `"buf_probe_03"`, "9019"))
	writeModule(t, dir, "probe_static.lua", fmt.Sprintf(base, "8", "180", "440", "10",
		`"buf_probe"`, `"buf_probe_03"`, `-1`))
	sink := &logSink{}
	s := New(dir, sink.printf)
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	good := s.Overlay("probe_good", &BuffDef{})
	if good == nil || good.Channel == nil || good.Channel.Waves != 8 || good.Channel.IntervalMS != 440 ||
		good.Channel.VictimBuff != "buf_probe" || good.Channel.SelfBuffStatic != 9019 {
		t.Fatalf("good overlay = %+v", good)
	}
	for _, id := range []string{"probe_waves", "probe_interval", "probe_sp", "probe_buffs", "probe_static"} {
		if overlay := s.Overlay(id, &BuffDef{}); overlay != nil && overlay.Channel != nil {
			t.Fatalf("%s channel = %+v, want dropped", id, overlay.Channel)
		}
		if !sink.has(id + " channel") {
			t.Fatalf("missing drop warning for %s in %q", id, sink.lines)
		}
	}
}

// 真实 desc_effects 钉子：金样本三套路（旋刀/太极剑/寒心恨雪）。
func TestRealDescEffects(t *testing.T) {
	s := newRealScripts(t, &logSink{})
	if s.DescEffectsCount() == 0 {
		t.Fatal("desc_effects 快照为空")
	}

	// 旋刀 06：招架给攻击者"血刃"（8s，10 层爆发定身 2s）+ 被攻击叠层。
	xhmdl06 := s.DescEffects("CS_by_xhmdl06")
	if xhmdl06 == nil || len(xhmdl06.OnParry) != 1 || len(xhmdl06.OnVictim) != 1 {
		t.Fatalf("xhmdl06 = %+v", xhmdl06)
	}
	parry := xhmdl06.OnParry[0]
	if parry.ApplyBuff != "buf_CS_by_xhmdl06_01" || !parry.ApplyResolved ||
		parry.ApplyLifeMS != 8000 || parry.BurstAt != 10 || parry.BurstKind != "root" || parry.BurstDurMS != 2000 {
		t.Fatalf("xhmdl06 on_parry = %+v", parry)
	}
	if xhmdl06.OnVictim[0].StackBuff != "buf_CS_by_xhmdl06_01" || xhmdl06.OnVictim[0].StackPerHit != 1 {
		t.Fatalf("xhmdl06 on_victim = %+v", xhmdl06.OnVictim[0])
	}
	if len(xhmdl06.ByLevel[2]) == 0 || xhmdl06.ByLevel[2]["on_parry.1.apply_life_ms"] != 10000 {
		t.Fatalf("xhmdl06 by_level = %+v", xhmdl06.ByLevel)
	}

	// 旋刀 01：前方 20 米扇形 + 命中未招架加"孽"（3 层爆发昏迷 2s）。
	xhmdl01 := s.DescEffects("CS_by_xhmdl01")
	if xhmdl01 == nil || xhmdl01.Compile == nil {
		t.Fatalf("xhmdl01 = %+v", xhmdl01)
	}
	if xhmdl01.Compile.AOEShape != "sector" || xhmdl01.Compile.AOERadiusM != 20 {
		t.Fatalf("xhmdl01 compile = %+v", xhmdl01.Compile)
	}
	if hit := xhmdl01.OnHit[0]; hit.Condition == nil || !hit.Condition.Unparried ||
		hit.BurstAt != 3 || hit.BurstKind != "coma" {
		t.Fatalf("xhmdl01 on_hit = %+v", hit)
	}

	// 太极剑 01：双击（max_hits=2）+ 轻功旁注；08：击退 + 减蓝；
	// 09：击落 → 5 秒禁轻功。
	tjj01 := s.DescEffects("CS_wd_tjj01")
	if tjj01 == nil || tjj01.Compile == nil || tjj01.Compile.MaxHits != 2 ||
		!tjj01.Compile.RushNote || !tjj01.Compile.CanQinggong {
		t.Fatalf("tjj01 = %+v", tjj01)
	}
	tjj08 := s.DescEffects("CS_wd_tjj08")
	if tjj08 == nil || len(tjj08.OnHit) != 1 {
		t.Fatalf("tjj08 = %+v", tjj08)
	}
	if hit := tjj08.OnHit[0]; hit.DisplacementKind != "push" || !hit.MPDrain {
		t.Fatalf("tjj08 on_hit = %+v", hit)
	}
	tjj09 := s.DescEffects("CS_wd_tjj09")
	if tjj09 == nil || len(tjj09.OnHit) != 1 || tjj09.OnHit[0].ControlKind != "ban_qinggong" ||
		tjj09.OnHit[0].ControlDurMS != 5000 {
		t.Fatalf("tjj09 = %+v", tjj09)
	}

	// 寒心恨雪：L1 斩 3 次 → L6 起斩 8 次（数值差集保真）。
	hyjf08 := s.DescEffects("CS_jh_hyjf08")
	if hyjf08 == nil || hyjf08.Compile == nil || hyjf08.Compile.MaxHits != 3 {
		t.Fatalf("hyjf08 = %+v", hyjf08)
	}
	if got := hyjf08.ByLevel[6]["compile.max_hits"]; got != 8 {
		t.Fatalf("hyjf08 by_level[6] = %d, want 8", got)
	}

	// 未知技能 → nil。
	if entry := s.DescEffects("CS_nope_999"); entry != nil {
		t.Fatalf("unknown = %+v", entry)
	}
}

// desc_effects 畸形条目降级：坏条目 WARN 跳过，合法条目照常服务。
func TestDescEffectsMalformedDegrades(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "desc_effects.lua", `local E = {}
E["probe_good"] = {
    compile = { max_hits = 4, air = "only" },
    on_hit = { { control_kind = "stun", control_dur_ms = 1500, condition = { unparried = true } } },
    unparsed = { "某段未解析" },
}
E[42] = { compile = { max_hits = 1 } }
E["probe_junk"] = "not-a-table"
E["probe_empty"] = {}
return E
`)
	sink := &logSink{}
	s := New(dir, sink.printf)
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	good := s.DescEffects("probe_good")
	if good == nil || good.Compile == nil || good.Compile.MaxHits != 4 || good.Compile.Air != "only" {
		t.Fatalf("probe_good = %+v", good)
	}
	hit := good.OnHit[0]
	if hit.ControlKind != "stun" || hit.ControlDurMS != 1500 || hit.Condition == nil || !hit.Condition.Unparried {
		t.Fatalf("probe_good on_hit = %+v", hit)
	}
	if len(good.Unparsed) != 1 || good.Unparsed[0] != "某段未解析" {
		t.Fatalf("probe_good unparsed = %+v", good.Unparsed)
	}
	for _, bad := range []string{"probe_junk", "probe_empty", "42"} {
		if entry := s.DescEffects(bad); entry != nil {
			t.Fatalf("%s = %+v, want skipped", bad, entry)
		}
	}
	if !sink.has("desc_effects") || !sink.has("已跳过") {
		t.Fatalf("missing skip warnings: %q", sink.lines)
	}
	if s.DescEffectsCount() != 1 {
		t.Fatalf("count = %d, want 1", s.DescEffectsCount())
	}
}

func TestHitRatio(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_hit.lua", `local M = {}
function M.on_hit(ctx)
  if ctx.level >= 5 and ctx.rand.r_adjust < 0.30 then
    return { ratio = ctx.ratio * 1.5 }
  end
  if ctx.level >= 5 and ctx.rand.r_adjust < 0.60 then
    return { ratio = 250 }   -- 上限夹取
  end
end
function M.compile(def) return { self_buff_lease_ms = 35000, self_buff_slot = 5, red_armor = true } end
return M
`)
	s := New(dir, nil)
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	// 5 级 + 低掷：×1.5。
	if got := s.HitRatio("probe_hit", &HitCtx{Ratio: 1.0, Level: 5, RAdjust: 0.1}); got == nil || *got != 1.5 {
		t.Fatalf("hit ratio = %v, want 1.5", got)
	}
	// 高于上限：夹 100。
	if got := s.HitRatio("probe_hit", &HitCtx{Ratio: 1.0, Level: 5, RAdjust: 0.4}); got == nil || *got != 100 {
		t.Fatalf("hit ratio = %v, want 100 (clamped)", got)
	}
	// 不触发分支：nil = 不调整（RNG 零扰动语义）。
	if got := s.HitRatio("probe_hit", &HitCtx{Ratio: 1.0, Level: 5, RAdjust: 0.9}); got != nil {
		t.Fatalf("hit ratio = %v, want nil", got)
	}
	// 低级不触发。
	if got := s.HitRatio("probe_hit", &HitCtx{Ratio: 1.0, Level: 1, RAdjust: 0.1}); got != nil {
		t.Fatalf("hit ratio = %v, want nil", got)
	}
	// 无钩子技能 → nil；nil ctx 合法。
	if got := s.HitRatio("no_such", nil); got != nil {
		t.Fatalf("unknown = %v", got)
	}

	// compile 白名单：租约/槽位/红霸体增量。
	overlay := s.Overlay("probe_hit", &BuffDef{})
	if overlay == nil || overlay.SelfBuffLeaseMS != 35000 || overlay.SelfBuffSlot != 5 || !overlay.RedArmor {
		t.Fatalf("overlay = %+v", overlay)
	}
}

func TestOverlayWhitelistAndMalformed(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_junk.lua", `local M = {}
function M.compile(def)
  return { foo = 1, self_buff_lease_ms = 0, self_buff_slot = -1, red_armor = 1,
           stance_cycle = { variants = { { id = "a", life_ms = 100 } } },
           damage = { amount = -5, waves = 0 },
           self_buffs = { { config_id = "", life_ms = 100 } },
           target_effects = { { config_id = "buf_x", life_ms = 0 } } }
end
return M
`)
	sink := &logSink{}
	s := New(dir, sink.printf)
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if overlay := s.Overlay("probe_junk", &BuffDef{}); overlay != nil {
		t.Fatalf("junk overlay = %+v, want nil", overlay)
	}
	for _, warn := range []string{"白名单外指令 foo", "self_buff_lease_ms", "self_buff_slot", "red_armor", "stance_cycle", "damage", "self_buffs", "target_effects"} {
		if !sink.has(warn) {
			t.Fatalf("missing warning %q in %q", warn, sink.lines)
		}
	}
}

func TestOverlayFullSpec(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_full.lua", `local M = {}
function M.compile(def)
  return {
    damage = { amount = 540, waves = 4 },
    self_buffs = { { config_id = "buf_x", static_data = 6013, level = 1, life_ms = 35000, slot = 5 } },
    target_effects = { { config_id = "buf_y", static_data = 7001, level = 1, life_ms = 8000, control = true, frame = true } },
  }
end
return M
`)
	s := New(dir, nil)
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	overlay := s.Overlay("probe_full", &BuffDef{})
	if overlay == nil || overlay.Damage == nil || overlay.Damage.Amount != 540 || overlay.Damage.Waves != 4 {
		t.Fatalf("overlay damage = %+v", overlay)
	}
	if len(overlay.SelfBuffs) != 1 || overlay.SelfBuffs[0].Slot != 5 || overlay.SelfBuffs[0].ConfigID != "buf_x" {
		t.Fatalf("self_buffs = %+v", overlay.SelfBuffs)
	}
	if len(overlay.TargetEffects) != 1 || !overlay.TargetEffects[0].Control || !overlay.TargetEffects[0].Frame {
		t.Fatalf("target_effects = %+v", overlay.TargetEffects)
	}
}

func TestBrokenIsolation(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_boom.lua", `local M = {}
function M.compile(def) error("boom") end
return M
`)
	writeModule(t, dir, "probe_ok.lua", `local M = {}
function M.compile(def) return { self_buff_lease_ms = 1000 } end
return M
`)
	sink := &logSink{}
	s := New(dir, sink.printf)
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if overlay := s.Overlay("probe_boom", &BuffDef{}); overlay != nil {
		t.Fatalf("boom overlay = %+v", overlay)
	}
	if broken := s.Broken(); len(broken) != 1 || broken[0] != "probe_boom" {
		t.Fatalf("Broken = %v", broken)
	}
	if !sink.has("禁用 probe_boom") {
		t.Fatalf("missing disable warning: %q", sink.lines)
	}
	if overlay := s.Overlay("probe_ok", &BuffDef{}); overlay == nil || overlay.SelfBuffLeaseMS != 1000 {
		t.Fatalf("good overlay = %+v", overlay)
	}
	// effects_data 查询不受运行期坏脚本影响（快照独立）。
	if s.EffectData("whatever") != nil {
		t.Fatal("no effects_data in this dir")
	}
}

func TestEffectsDataMissingDegrades(t *testing.T) {
	// 目录只有覆盖脚本、无 effects_data：装载成功，EffectData 恒 miss。
	dir := t.TempDir()
	writeModule(t, dir, "probe_only.lua", `local M = {}
function M.compile(def) return { red_armor = true } end
return M
`)
	s := New(dir, nil)
	if n, err := s.Load(); err != nil || n != 1 {
		t.Fatalf("Load = (%d, %v)", n, err)
	}
	if s.EffectDataCount() != 0 || s.EffectData("probe_only") != nil {
		t.Fatal("EffectData must miss without effects_data.lua")
	}
}

func TestInfiniteLoopTimesOut(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_loop.lua", `local M = {}
function M.compile(def) while true do end end
return M
`)
	s := New(dir, nil)
	s.SetTimeout(150 * time.Millisecond)
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if overlay := s.Overlay("probe_loop", &BuffDef{}); overlay != nil {
		t.Fatalf("loop overlay = %+v", overlay)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("loop took %v to cancel", elapsed)
	}
}

func TestDisabled(t *testing.T) {
	s := New("", nil)
	if s.Enabled() {
		t.Fatal("empty dir must disable")
	}
	if n, err := s.Load(); n != 0 || err != nil {
		t.Fatalf("Load = (%d, %v)", n, err)
	}
	if s.Overlay("CS_wd_tjq04", &BuffDef{}) != nil || s.HitRatio("CS_wd_tjq04", &HitCtx{}) != nil || s.EffectData("CS_wd_tjq01") != nil {
		t.Fatal("disabled queries must be nil")
	}
}
