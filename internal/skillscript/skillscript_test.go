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
	if s.Loaded() != 7 {
		t.Fatalf("Loaded = %d, want 7 (6 overlays + effects_data)", s.Loaded())
	}
	want := []string{
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
