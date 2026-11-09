package itemuse

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

// realScriptsDir 指向仓内真实脚本（scripts/module/item，3 本）。
func realScriptsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "scripts", "module", "item")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("real scripts dir: %v", err)
	}
	return dir
}

func newRealCatalog(t *testing.T, sink *logSink) *Catalog {
	t.Helper()
	c := New(realScriptsDir(t), sink.printf)
	if _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return c
}

func writeModule(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRealItems(t *testing.T) {
	c := newRealCatalog(t, &logSink{})
	if c.Loaded() != 3 {
		t.Fatalf("Loaded = %d, want 3", c.Loaded())
	}
	if item, ok := c.Item("yaopin_sc_1001"); !ok || item.Name != "祛风散" {
		t.Fatalf("Item(yaopin_sc_1001) = %+v, ok=%v", item, ok)
	}
	if items := c.Items(); len(items) != 3 || items[0].ID != "box_chinajoy_2026_giftbox" {
		t.Fatalf("Items = %+v", items)
	}
}

// 三本真实脚本的行为钉子。
func TestRealItemEffects(t *testing.T) {
	c := newRealCatalog(t, &logSink{})

	// 祛风散：回血 100，默认消耗。
	heal := c.Use("yaopin_sc_1001", &Ctx{Self: SelfStats{HP: 50, MaxHP: 100}})
	if heal == nil || heal.HealHP != 100 || !heal.Consume || heal.HealMP != 0 || heal.OpenBox != nil || heal.TempStat != nil || heal.GrantItems != nil {
		t.Fatalf("yaopin effect = %+v", heal)
	}

	// 惊鸿礼盒：开包 1 roll。
	box := c.Use("box_chinajoy_2026_giftbox", &Ctx{})
	if box == nil || box.OpenBox == nil ||
		box.OpenBox.DropID != "comb_ai_drop_box_chinajoy_2026_giftbox" || box.OpenBox.Rolls != 1 {
		t.Fatalf("giftbox effect = %+v", box)
	}

	// 飞云剑匣·金：限时攻击 +25、24h。
	stat := c.Use("useitemhs_dy_06", &Ctx{})
	if stat == nil || stat.TempStat == nil ||
		stat.TempStat.AttackAdd != 25 || stat.TempStat.DefenseAdd != 0 ||
		stat.TempStat.MaxHPAdd != 0 || stat.TempStat.LifeMS != 24*60*60*1000 {
		t.Fatalf("useitemhs effect = %+v", stat)
	}

	// 未知物品 / nil ctx → 回退。
	if effect := c.Use("no_such_item", &Ctx{}); effect != nil {
		t.Fatalf("unknown item = %+v", effect)
	}
	if effect := c.Use("yaopin_sc_1001", nil); effect == nil || effect.HealHP != 100 {
		t.Fatalf("nil ctx = %+v", effect)
	}
}

func TestCtxDelivered(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_ctx.lua", `local M = {}
M.meta = { id = "probe_ctx", name = "探针" }
function M.on_use(ctx)
  return { heal_hp = ctx.self.hp, heal_mp = ctx.self.max_mp,
           grant_items = { { id = ctx.item_id, amount = ctx.amount } } }
end
return M
`)
	c := New(dir, nil)
	if _, err := c.Load(); err != nil {
		t.Fatal(err)
	}
	effect := c.Use("probe_ctx", &Ctx{
		ItemID: "probe_ctx", Amount: 2, NowMS: 77,
		Self:  SelfStats{HP: 123, MaxHP: 500, MP: 10, MaxMP: 45, Silver: 9},
		RDrop: 0.1, RProc: 0.9,
	})
	if effect == nil || effect.HealHP != 123 || effect.HealMP != 45 {
		t.Fatalf("effect = %+v", effect)
	}
	if len(effect.GrantItems) != 1 || effect.GrantItems[0].ID != "probe_ctx" || effect.GrantItems[0].Amount != 2 {
		t.Fatalf("grant_items = %+v", effect.GrantItems)
	}
}

func TestWhitelistAndMalformed(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_junk.lua", `local M = {}
M.meta = { id = "probe_junk", name = "垃圾指令" }
function M.on_use(ctx)
  return { foo = 1, heal_hp = 0, heal_mp = -5,
           open_box = { drop_id = "d", rolls = 0 },
           grant_items = { { id = "", amount = 1 }, "not a table" },
           temp_stat = { attack_add = 5 },
           consume = "yes" }
end
return M
`)
	sink := &logSink{}
	c := New(dir, sink.printf)
	if _, err := c.Load(); err != nil {
		t.Fatal(err)
	}
	effect := c.Use("probe_junk", &Ctx{})
	if effect != nil {
		t.Fatalf("junk effect = %+v, want nil (all entries invalid)", effect)
	}
	for _, warn := range []string{"白名单外指令 foo", "heal_hp", "heal_mp", "open_box", "grant_items", "temp_stat", "consume"} {
		if !sink.has(warn) {
			t.Fatalf("missing warning %q in %q", warn, sink.lines)
		}
	}
	if broken := c.Broken(); len(broken) != 0 {
		t.Fatalf("malformed return must not disable module: %v", broken)
	}
}

func TestFullEffectAndConsumeFalse(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_full.lua", `local M = {}
M.meta = { id = "probe_full", name = "全指令" }
function M.on_use(ctx)
  return {
    heal_hp = 300, heal_mp = 100,
    open_box = { drop_id = "drop_x", rolls = 3 },
    grant_items = { { id = "sd_001", amount = 2 }, { id = "sd_002", amount = 1 } },
    temp_stat = { attack_add = 25, defense_add = 5, max_hp_add = 10, life_ms = 86400000 },
    consume = false,
  }
end
return M
`)
	c := New(dir, nil)
	if _, err := c.Load(); err != nil {
		t.Fatal(err)
	}
	effect := c.Use("probe_full", &Ctx{})
	if effect == nil || effect.HealHP != 300 || effect.HealMP != 100 {
		t.Fatalf("effect = %+v", effect)
	}
	if effect.OpenBox == nil || effect.OpenBox.DropID != "drop_x" || effect.OpenBox.Rolls != 3 {
		t.Fatalf("open_box = %+v", effect.OpenBox)
	}
	if len(effect.GrantItems) != 2 || effect.GrantItems[0] != (GrantItem{ID: "sd_001", Amount: 2}) {
		t.Fatalf("grant_items = %+v", effect.GrantItems)
	}
	if effect.TempStat == nil || effect.TempStat.AttackAdd != 25 || effect.TempStat.DefenseAdd != 5 || effect.TempStat.LifeMS != 86400000 {
		t.Fatalf("temp_stat = %+v", effect.TempStat)
	}
	if effect.Consume {
		t.Fatal("consume = true, want false")
	}
}

func TestBrokenIsolation(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_boom.lua", `local M = {}
M.meta = { id = "probe_boom", name = "坏脚本" }
function M.on_use(ctx) error("boom") end
return M
`)
	writeModule(t, dir, "probe_ok.lua", `local M = {}
M.meta = { id = "probe_ok", name = "好脚本" }
function M.on_use(ctx) return { heal_hp = 7 } end
return M
`)
	sink := &logSink{}
	c := New(dir, sink.printf)
	if _, err := c.Load(); err != nil {
		t.Fatal(err)
	}
	if effect := c.Use("probe_boom", &Ctx{}); effect != nil {
		t.Fatalf("boom = %+v", effect)
	}
	if broken := c.Broken(); len(broken) != 1 || broken[0] != "probe_boom" {
		t.Fatalf("Broken = %v", broken)
	}
	if !sink.has("禁用 probe_boom") {
		t.Fatalf("missing disable warning: %q", sink.lines)
	}
	if effect := c.Use("probe_ok", &Ctx{}); effect == nil || effect.HealHP != 7 {
		t.Fatalf("good = %+v", effect)
	}
	if c.Loaded() != 2 {
		t.Fatalf("Loaded = %d, want 2 (VM must not rebuild)", c.Loaded())
	}
}

func TestBadMetaSkipped(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "a_bad.lua", "not lua")
	writeModule(t, dir, "b_nometa.lua", "return {}")
	writeModule(t, dir, "c_good.lua", `local M = {}
M.meta = { id = "c_good", name = "好" }
function M.on_use(ctx) return { heal_hp = 1 } end
return M
`)
	c := New(dir, nil)
	loaded, err := c.Load()
	// b_nometa 返回表 → modulevm 计入装载（2 = b_nometa + c_good），域校验
	// 失败后进 Broken；a_bad 语法错不计入。
	if loaded != 2 {
		t.Fatalf("loaded = %d, want 2", loaded)
	}
	if err == nil || !strings.Contains(err.Error(), "a_bad.lua") || !strings.Contains(err.Error(), "b_nometa") {
		t.Fatalf("expected failures for a_bad/b_nometa, got %v", err)
	}
	if _, ok := c.Item("b_nometa"); ok {
		t.Fatal("module without meta must not enter catalog")
	}
	if broken := c.Broken(); len(broken) != 1 || broken[0] != "b_nometa" {
		t.Fatalf("Broken = %v, want [b_nometa]", broken)
	}
	if effect := c.Use("b_nometa", &Ctx{}); effect != nil {
		t.Fatalf("broken module Use = %+v, want nil", effect)
	}
	if effect := c.Use("c_good", &Ctx{}); effect == nil || effect.HealHP != 1 {
		t.Fatalf("c_good = %+v", effect)
	}
}

func TestInfiniteLoopTimesOut(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "probe_loop.lua", `local M = {}
M.meta = { id = "probe_loop", name = "死循环" }
function M.on_use(ctx) while true do end end
return M
`)
	c := New(dir, nil)
	c.SetTimeout(150 * time.Millisecond)
	if _, err := c.Load(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if effect := c.Use("probe_loop", &Ctx{}); effect != nil {
		t.Fatalf("loop = %+v", effect)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("loop took %v to cancel", elapsed)
	}
	if broken := c.Broken(); len(broken) != 1 || broken[0] != "probe_loop" {
		t.Fatalf("Broken = %v", broken)
	}
}

func TestDisabledCatalog(t *testing.T) {
	c := New("", nil)
	if c.Enabled() {
		t.Fatal("empty dir must disable")
	}
	if n, err := c.Load(); n != 0 || err != nil {
		t.Fatalf("Load = (%d, %v)", n, err)
	}
	if c.Use("yaopin_sc_1001", &Ctx{}) != nil {
		t.Fatal("disabled Use must be nil")
	}
}
