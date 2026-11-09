package neigong

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

// realScriptsDir 指向仓内真实脚本目录（scripts/module/neigong，155 本）。
func realScriptsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "scripts", "module", "neigong")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("real scripts dir: %v", err)
	}
	return dir
}

func newTestEffects(t *testing.T, dir string, sink *logSink) *Effects {
	t.Helper()
	e := New(dir, sink.printf)
	if _, err := e.Load(); err != nil {
		t.Fatalf("Load %s: %v", dir, err)
	}
	return e
}

func writeModule(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}

func probeCtx() *Ctx {
	return &Ctx{
		BookLevel: 3, BookMaxLevel: 5, StoryStage: 1,
		NowMS: 1_000_000, RProc: 0.42,
		Damage: 500, Incoming: 300, Parried: true, Crit: true, Controlled: true,
		SinceLastMS: -1,
		Self:        SelfStats{HP: 800, MaxHP: 1000, MP: 120, MaxMP: 200, Attack: 66},
		Target:      &TargetInfo{OID: 77, HP: 400, MaxHP: 900},
		AttackerOID: 88,
	}
}

func TestLoadRealScripts(t *testing.T) {
	e := newTestEffects(t, realScriptsDir(t), &logSink{})
	if e.Loaded() != 155 {
		t.Fatalf("Loaded = %d, want 155", e.Loaded())
	}
	if got := len(e.Books()); got != 155 {
		t.Fatalf("Books = %d, want 155", got)
	}
	// 目录抽查：meta 来自 M.meta，书 id 反查一致。
	book, ok := e.Book("buf_ng_bg_001")
	if !ok || book.Name != "大巧若拙经" || book.Book != "ng_bg_001" || book.MaxTier != 5 {
		t.Fatalf("Book(buf_ng_bg_001) = %+v, ok=%v", book, ok)
	}
	if id, ok := e.ByBook("ng_jh_001"); !ok || id != "buf_ng_jh_001" {
		t.Fatalf("ByBook(ng_jh_001) = (%q, %v)", id, ok)
	}
	// 排序稳定：Books 首尾可预测。
	if books := e.Books(); books[0].ID != "buf_ng_bg_001" {
		t.Fatalf("first book = %s, want buf_ng_bg_001", books[0].ID)
	}
}

func TestRealScriptsHookSmoke(t *testing.T) {
	e := newTestEffects(t, realScriptsDir(t), &logSink{})
	// ctx 按参考服分发契约构建：on_attack 必有 target（SkillCastHandler），
	// on_victim/on_parry 必有 attacker（NpcCombatAi）——脚本依赖该前提
	// （如 buf_ng_nl_001 无保护读 ctx.attacker.oid）。
	events := []struct {
		name     string
		target   bool
		attacker bool
	}{
		{"on_attack", true, false},
		{"on_victim", false, true},
		{"on_parry", false, true},
		{"on_periodic", false, false},
	}
	fired := 0
	for _, book := range e.Books() {
		for _, event := range events {
			// 覆盖有 state（模拟已开窗）与无 state 两条路径；r_proc 扫过
			// 0.05/0.5 两档，触发概率分支两侧都被执行到。
			for _, rproc := range []float64{0.05, 0.5} {
				ctx := &Ctx{
					BookLevel: 30, BookMaxLevel: 36, StoryStage: 5,
					NowMS: 1_000_000, RProc: rproc,
					Damage: 500, Incoming: 300,
					Self:  SelfStats{HP: 800, MaxHP: 1000, MP: 120, MaxMP: 200, Attack: 66},
					State: map[string]any{"until_at": int64(1_500_000), "layers": int64(2), "count": int64(1)},
				}
				if event.target {
					ctx.Target = &TargetInfo{OID: 77, HP: 400, MaxHP: 900}
				}
				if event.attacker {
					ctx.AttackerOID = 88
				}
				if effect := e.Hook(book.ID, event.name, ctx); effect != nil {
					fired++
				}
			}
		}
		if e.Passive(book.ID, probeCtx()) != nil {
			fired++
		}
	}
	if fired == 0 {
		t.Fatal("no hook produced an effect; smoke test exercised nothing")
	}
	if broken := e.Broken(); len(broken) != 0 {
		t.Fatalf("broken books after smoke: %v", broken)
	}
}

// 真实脚本行为钉子：坐忘功（buf_ng_jh_001）——层1-3 常驻平减，层4-5 周期盾。
func TestRealScriptZuoWangGong(t *testing.T) {
	e := newTestEffects(t, realScriptsDir(t), &logSink{})

	// passive：满级书 + 阶段1 → 档位门 cap=3 → levels[3].flat=40。
	passive := e.Passive("buf_ng_jh_001", &Ctx{BookLevel: 5, BookMaxLevel: 5, StoryStage: 1})
	if passive == nil || passive["damage_reduction_add"] != 40 {
		t.Fatalf("passive = %v, want damage_reduction_add=40", passive)
	}

	// on_periodic（别名 on_tick）：阶段5 开放档位5 → 30s 之外的周期给盾。
	ctx := &Ctx{BookLevel: 5, BookMaxLevel: 5, StoryStage: 5, NowMS: 1000, SinceLastMS: 60000}
	effect := e.Hook("buf_ng_jh_001", "on_periodic", ctx)
	if effect == nil || effect.Shield == nil {
		t.Fatalf("effect = %+v, want shield", effect)
	}
	if effect.Shield.Amount != 250 || effect.Shield.LifeMS != 30000 {
		t.Fatalf("shield = %+v, want {250, 30000}", effect.Shield)
	}
	if effect.State["last_at"] != int64(1000) {
		t.Fatalf("state.last_at = %v, want 1000", effect.State["last_at"])
	}

	// 冷却窗内（since_last_ms < 30000）不给盾。
	cooling := e.Hook("buf_ng_jh_001", "on_periodic", &Ctx{
		BookLevel: 5, BookMaxLevel: 5, StoryStage: 5, NowMS: 1000, SinceLastMS: 100,
	})
	if cooling != nil {
		t.Fatalf("cooling effect = %+v, want nil", cooling)
	}
}

func TestHookConvertsAllWhitelistKinds(t *testing.T) {
	e := newTestEffects(t, "testdata", &logSink{})
	effect := e.Hook("probe_all", "on_attack", probeCtx())
	if effect == nil {
		t.Fatal("effect = nil")
	}
	if effect.ExtraDamage == nil || effect.ExtraDamage.Amount != 77 || effect.ExtraDamage.Kind != "yang" {
		t.Fatalf("ExtraDamage = %+v", effect.ExtraDamage)
	}
	if effect.ReducePct != 0.25 {
		t.Fatalf("ReducePct = %v", effect.ReducePct)
	}
	if effect.Shield == nil || effect.Shield.Amount != 200 || effect.Shield.LifeMS != 30000 {
		t.Fatalf("Shield = %+v", effect.Shield)
	}
	if effect.Heal != 60 || effect.MPRestore != 30 {
		t.Fatalf("Heal/MPRestore = %d/%d", effect.Heal, effect.MPRestore)
	}
	if effect.Reflect == nil || effect.Reflect.Amount != 45 || effect.Reflect.Kind != "yin" {
		t.Fatalf("Reflect = %+v", effect.Reflect)
	}
	if effect.ParryReduce == nil || effect.ParryReduce.Internal != 0.1 || effect.ParryReduce.External != 0.2 {
		t.Fatalf("ParryReduce = %+v", effect.ParryReduce)
	}
	if effect.State["seen"] != "on_attack" || effect.State["until_at"] != int64(1006000) {
		t.Fatalf("State = %v", effect.State)
	}
}

func TestCtxFieldsDelivered(t *testing.T) {
	e := newTestEffects(t, "testdata", &logSink{})
	ctx := probeCtx()
	ctx.State = map[string]any{"layers": int64(3)}
	effect := e.Hook("probe_ctx", "on_attack", ctx)
	if effect == nil || effect.State == nil {
		t.Fatal("probe_ctx returned no state")
	}
	s := effect.State
	want := map[string]any{
		"book_level": int64(3), "book_max_level": int64(5), "story_stage": int64(1),
		"now": int64(1_000_000), "damage": int64(500), "incoming": int64(300),
		"parried": true, "crit": true, "controlled": true, "since_last_ms": int64(-1),
		"self_hp": int64(800), "self_max_hp": int64(1000), "self_mp": int64(120),
		"self_max_mp": int64(200), "self_attack": int64(66),
		"target_oid": int64(77), "target_hp": int64(400),
		"attacker_oid": int64(88), "r_proc": 0.42,
		"has_state": true, "state_layers": int64(3),
	}
	for key, value := range want {
		if s[key] != value {
			t.Fatalf("state[%s] = %v (%T), want %v (%T)", key, s[key], s[key], value, value)
		}
	}

	// 可选段缺失：无 target/attacker/state 时脚本侧走 or 兜底，不 nil 索引。
	bare := e.Hook("probe_ctx", "on_attack", &Ctx{})
	if bare == nil || bare.State["target_oid"] != int64(-1) ||
		bare.State["attacker_oid"] != int64(-1) || bare.State["has_state"] != false {
		t.Fatalf("bare ctx state = %v", bare.State)
	}
}

func TestTierGateFollowsStoryStage(t *testing.T) {
	e := newTestEffects(t, "testdata", &logSink{})
	// 满级书：阶段1 → cap=3 → 档3（30）；阶段5 → cap=7 → 档5（50）。
	stage1 := e.Hook("probe_tier", "on_attack", &Ctx{BookLevel: 36, BookMaxLevel: 36, StoryStage: 1})
	if stage1.ExtraDamage.Amount != 30 {
		t.Fatalf("stage1 amount = %d, want 30 (tier cap 3)", stage1.ExtraDamage.Amount)
	}
	stage5 := e.Hook("probe_tier", "on_attack", &Ctx{BookLevel: 36, BookMaxLevel: 36, StoryStage: 5})
	if stage5.ExtraDamage.Amount != 50 {
		t.Fatalf("stage5 amount = %d, want 50 (tier 5)", stage5.ExtraDamage.Amount)
	}
}

func TestStateRoundTrip(t *testing.T) {
	e := newTestEffects(t, "testdata", &logSink{})
	// 首次：无 state → 开窗 heal=1。
	first := e.Hook("probe_state", "on_attack", &Ctx{NowMS: 1000})
	if first == nil || first.Heal != 1 || first.State["count"] != int64(1) {
		t.Fatalf("first = %+v", first)
	}
	// 窗内：把上次 state 回写进 ctx → 叠层 heal=5。
	second := e.Hook("probe_state", "on_attack", &Ctx{NowMS: 2000, State: first.State})
	if second == nil || second.Heal != 5 || second.State["count"] != int64(2) {
		t.Fatalf("second = %+v", second)
	}
	// 窗外：state.until_at 已过 → 重开窗。
	third := e.Hook("probe_state", "on_attack", &Ctx{NowMS: 999_999, State: second.State})
	if third == nil || third.Heal != 1 || third.State["count"] != int64(1) {
		t.Fatalf("third = %+v", third)
	}
}

func TestHookAliasAndMissing(t *testing.T) {
	e := newTestEffects(t, "testdata", &logSink{})
	// 模块只有 on_hit：on_attack 事件经别名链命中。
	alias := e.Hook("probe_alias", "on_attack", &Ctx{Damage: 123})
	if alias == nil || alias.ExtraDamage.Amount != 123 {
		t.Fatalf("alias hook = %+v", alias)
	}
	// 模块没有 on_victim 族钩子 → 零效果，不算错误。
	if effect := e.Hook("probe_alias", "on_victim", &Ctx{}); effect != nil {
		t.Fatalf("victim = %+v, want nil", effect)
	}
	// 未知 buff / 未装载事件 → 零效果。
	if effect := e.Hook("buf_ng_nope", "on_attack", &Ctx{}); effect != nil {
		t.Fatalf("unknown book = %+v, want nil", effect)
	}
	if effect := e.Hook("probe_alias", "on_parry", nil); effect != nil {
		t.Fatalf("nil ctx = %+v, want nil", effect)
	}
}

func TestReflectNumberForm(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_reflect_num.lua", `local M = {}
M.meta = { id = "probe_reflect_num", name = "反打数值", book = "ng_prn", category = "hit", max_tier = 1 }
function M.on_victim(ctx) return { reflect = 123 } end
return M
`)
	e := newTestEffects(t, dir, &logSink{})
	effect := e.Hook("probe_reflect_num", "on_victim", &Ctx{})
	if effect == nil || effect.Reflect == nil || effect.Reflect.Amount != 123 || effect.Reflect.Kind != "" {
		t.Fatalf("effect = %+v, want reflect 123", effect)
	}
}

func TestPassiveFlatAttributes(t *testing.T) {
	e := newTestEffects(t, "testdata", &logSink{})
	passive := e.Passive("probe_passive", &Ctx{})
	want := map[string]float64{"attack_add": 25, "max_mp_add": 100.5, "damage_reduction_add": 20}
	if len(passive) != len(want) {
		t.Fatalf("passive = %v", passive)
	}
	for key, value := range want {
		if passive[key] != value {
			t.Fatalf("passive[%s] = %v, want %v", key, passive[key], value)
		}
	}
	// 无 passive 钩子的书 → nil。
	if passive := e.Passive("probe_all", &Ctx{}); passive != nil {
		t.Fatalf("passive = %v, want nil", passive)
	}
}

func TestBrokenScriptIsolation(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_boom.lua", `local M = {}
M.meta = { id = "probe_boom", name = "坏书", book = "ng_boom", category = "hit", max_tier = 1 }
function M.on_attack(ctx) error("boom") end
return M
`)
	writeModule(t, dir, "probe_ok.lua", `local M = {}
M.meta = { id = "probe_ok", name = "好书", book = "ng_ok", category = "hit", max_tier = 1 }
function M.on_attack(ctx) return { heal = 7 } end
return M
`)
	sink := &logSink{}
	e := newTestEffects(t, dir, sink)

	if effect := e.Hook("probe_boom", "on_attack", &Ctx{}); effect != nil {
		t.Fatalf("boom effect = %+v, want nil", effect)
	}
	if broken := e.Broken(); len(broken) != 1 || broken[0] != "probe_boom" {
		t.Fatalf("Broken = %v, want [probe_boom]", broken)
	}
	if !sink.has("禁用 probe_boom") {
		t.Fatalf("missing disable warning: %q", sink.lines)
	}
	// 再次调用：零效果且不重复告警。
	before := len(sink.lines)
	if effect := e.Hook("probe_boom", "on_attack", &Ctx{}); effect != nil {
		t.Fatalf("boom retry = %+v, want nil", effect)
	}
	if len(sink.lines) != before {
		t.Fatalf("disable warned twice: %q", sink.lines)
	}
	// 坏书不连坐：其余书照常、装载数不变、VM 不重建（Loaded 不变）。
	if effect := e.Hook("probe_ok", "on_attack", &Ctx{}); effect == nil || effect.Heal != 7 {
		t.Fatalf("good book = %+v", effect)
	}
	if e.Loaded() != 2 {
		t.Fatalf("Loaded = %d, want 2 (VM must not rebuild)", e.Loaded())
	}
	// 被禁的书换钩子调用同样零效果。
	if effect := e.Hook("probe_boom", "on_victim", &Ctx{}); effect != nil {
		t.Fatalf("broken other hook = %+v, want nil", effect)
	}
}

func TestInfiniteLoopHookTimesOut(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_loop.lua", `local M = {}
M.meta = { id = "probe_loop", name = "死循环", book = "ng_loop", category = "hit", max_tier = 1 }
function M.on_attack(ctx) while true do end end
return M
`)
	writeModule(t, dir, "probe_ok.lua", `local M = {}
M.meta = { id = "probe_ok", name = "好书", book = "ng_ok", category = "hit", max_tier = 1 }
function M.on_attack(ctx) return { heal = 7 } end
return M
`)
	e := New(dir, nil)
	e.SetTimeout(150 * time.Millisecond)
	if _, err := e.Load(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if effect := e.Hook("probe_loop", "on_attack", &Ctx{}); effect != nil {
		t.Fatalf("loop effect = %+v, want nil", effect)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("loop took %v to cancel", elapsed)
	}
	if broken := e.Broken(); len(broken) != 1 || broken[0] != "probe_loop" {
		t.Fatalf("Broken = %v, want [probe_loop]", broken)
	}
	// 超时禁用后 VM 仍可用：其他书照常（指令预算按调用重置，不累积）。
	if effect := e.Hook("probe_ok", "on_attack", &Ctx{}); effect == nil || effect.Heal != 7 {
		t.Fatalf("good book after timeout = %+v", effect)
	}
}

func TestBadFilesSkippedAtLoad(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "a_bad.lua", "not lua at all")
	writeModule(t, dir, "b_notable.lua", "return 42")
	writeModule(t, dir, "c_good.lua", `local M = {}
M.meta = { id = "c_good", name = "好", book = "ng_c", category = "hit", max_tier = 1 }
function M.on_attack(ctx) return { heal = 1 } end
return M
`)
	sink := &logSink{}
	e := New(dir, sink.printf)
	loaded, err := e.Load()
	if loaded != 1 {
		t.Fatalf("loaded = %d, want 1", loaded)
	}
	if err == nil || !strings.Contains(err.Error(), "a_bad.lua") || !strings.Contains(err.Error(), "b_notable.lua") {
		t.Fatalf("expected both failures in error, got %v", err)
	}
	if effect := e.Hook("c_good", "on_attack", &Ctx{}); effect == nil {
		t.Fatal("good script should still load")
	}
}

func TestMetaIDMismatchKeysByFilename(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_named.lua", `local M = {}
M.meta = { id = "other_id", name = "名不符", book = "ng_mis", category = "hit", max_tier = 1 }
function M.on_attack(ctx) return { heal = 1 } end
return M
`)
	sink := &logSink{}
	e := newTestEffects(t, dir, sink)
	if _, ok := e.Book("other_id"); ok {
		t.Fatal("meta.id must not be used as key")
	}
	if effect := e.Hook("probe_named", "on_attack", &Ctx{}); effect == nil {
		t.Fatal("hook by filename stem should work")
	}
	if !sink.has("meta.id") {
		t.Fatalf("missing mismatch warning: %q", sink.lines)
	}
}

func TestUnknownAndMalformedDropped(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_junk.lua", `local M = {}
M.meta = { id = "probe_junk", name = "垃圾指令", book = "ng_junk", category = "hit", max_tier = 1 }
function M.on_attack(ctx)
  return { foo = 1, heal = 0, reduce_pct = 2, shield = { amount = -5 },
           extra_damage = "not a table", state = 7 }
end
return M
`)
	sink := &logSink{}
	e := newTestEffects(t, dir, sink)
	if effect := e.Hook("probe_junk", "on_attack", &Ctx{}); effect != nil {
		t.Fatalf("junk effect = %+v, want nil (all entries invalid)", effect)
	}
	for _, warn := range []string{"白名单外指令 foo", "reduce_pct", "shield", "extra_damage", "state"} {
		if !sink.has(warn) {
			t.Fatalf("missing warning %q in %q", warn, sink.lines)
		}
	}
	// 同一坏书重复调用：告警去重（warnOnce），热路径不刷屏。
	before := len(sink.lines)
	e.Hook("probe_junk", "on_attack", &Ctx{})
	e.Hook("probe_junk", "on_attack", &Ctx{})
	if got := len(sink.lines); got != before {
		t.Fatalf("warnings not deduped: %d → %d", before, got)
	}
	// 坏返回值不禁用这本书（仍可调用，只是零效果）。
	if broken := e.Broken(); len(broken) != 0 {
		t.Fatalf("malformed return must not disable book: %v", broken)
	}
}

func TestReloadPicksUpNewModule(t *testing.T) {
	dir := t.TempDir()
	e := newTestEffects(t, dir, &logSink{})
	if _, ok := e.Book("probe_fresh"); ok {
		t.Fatal("unexpected book")
	}
	writeModule(t, dir, "probe_fresh.lua", `local M = {}
M.meta = { id = "probe_fresh", name = "新书", book = "ng_fresh", category = "hit", max_tier = 1 }
function M.on_attack(ctx) return { heal = 9 } end
return M
`)
	if n, err := e.Reload(); err != nil || n != 1 {
		t.Fatalf("Reload = (%d, %v)", n, err)
	}
	if effect := e.Hook("probe_fresh", "on_attack", &Ctx{}); effect == nil || effect.Heal != 9 {
		t.Fatalf("fresh = %+v", effect)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if effect := e.Hook("probe_fresh", "on_attack", &Ctx{}); effect != nil {
		t.Fatal("Hook after Close should be nil")
	}
}

func TestDisabledHost(t *testing.T) {
	e := New("", nil)
	if e.Enabled() {
		t.Fatal("empty dir must disable")
	}
	if n, err := e.Load(); n != 0 || err != nil {
		t.Fatalf("Load = (%d, %v), want (0, nil)", n, err)
	}
	if effect := e.Hook("buf_ng_jh_001", "on_attack", &Ctx{}); effect != nil {
		t.Fatal("disabled Hook must be nil")
	}
	if passive := e.Passive("buf_ng_jh_001", &Ctx{}); passive != nil {
		t.Fatal("disabled Passive must be nil")
	}
}
