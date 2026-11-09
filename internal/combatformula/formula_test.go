package combatformula

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// realScriptsDir 指向仓内真实公式脚本（scripts/module/combat/formula.lua）。
func realScriptsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "scripts", "module", "combat")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("real scripts dir: %v", err)
	}
	return dir
}

func newRealFormula(t *testing.T) *Formula {
	t.Helper()
	f := New(realScriptsDir(t), nil)
	if _, err := f.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return f
}

// base 是一个必不闪避、必不暴击、无外防的基准输入（黄金值手算自
// formula.lua 常量区：未命中 10%、暴击 5%、暴伤 ×2、平减每 10 点折 1）。
func base(rMiss, rCrit float64) *Input {
	return &Input{Attack: 1000, Ratio: 1, RMiss: rMiss, RCrit: rCrit}
}

func TestLoadRealFormula(t *testing.T) {
	f := newRealFormula(t)
	if f.Loaded() != 1 {
		t.Fatalf("Loaded = %d, want 1", f.Loaded())
	}
	if len(f.Broken()) != 0 {
		t.Fatalf("Broken = %v", f.Broken())
	}
}

// 黄金用例：数值手算自 formula.lua 的系数常量区，钉住脚本行为不被意外改动。
func TestCalcGoldenCases(t *testing.T) {
	f := newRealFormula(t)
	cases := []struct {
		name string
		in   Input
		want Output
		// ReductionPct 用近似比较（对数曲线），Damage 用精确比较。
		wantReductionApprox float64
	}{
		{
			name: "未命中（10% 基础闪避，r_miss=0.05 命中掷失败）",
			in:   Input{Attack: 1000, Ratio: 1, RMiss: 0.05, RCrit: 0.5},
			want: Output{Damage: 0, Miss: true},
		},
		{
			name: "命中率高加成完全抵扣闪避（r_miss=0 也不闪）",
			in:   Input{Attack: 1000, Ratio: 1, HitRateBonus: 0.2, RMiss: 0, RCrit: 0.5},
			want: Output{Damage: 1000},
		},
		{
			name: "普攻（r_crit=0.5 > 5% 暴击）",
			in:   Input{Attack: 1000, Ratio: 1, RMiss: 0.5, RCrit: 0.5},
			want: Output{Damage: 1000},
		},
		{
			name: "暴击（基础暴伤 100% → ×2）",
			in:   Input{Attack: 1000, Ratio: 1, RMiss: 0.5, RCrit: 0.01},
			want: Output{Damage: 2000, Crit: true},
		},
		{
			name:                "外防对数减伤（10000 外防 ≈ 48.1%）",
			in:                  Input{Attack: 1000, Defense: 10000, Ratio: 1, RMiss: 0.5, RCrit: 0.5},
			want:                Output{Damage: 518},
			wantReductionApprox: (20*math.Log(100) - 44) / 100,
		},
		{
			name:                "穿刺不吃外防折减",
			in:                  Input{Attack: 1000, Defense: 10000, Pierce: 100, Ratio: 1, RMiss: 0.5, RCrit: 0.5},
			want:                Output{Damage: 618},
			wantReductionApprox: (20*math.Log(100) - 44) / 100,
		},
		{
			name: "完全招架（化劲 ≤ 格挡值 → 零伤不吃保底）",
			in:   Input{Attack: 1000, InternalConversion: 100, BlockRating: 100, Parrying: true, Ratio: 1, RMiss: 0.5, RCrit: 0.5},
			want: Output{Damage: 0, Parried: true},
		},
		{
			name: "部分招架（穿透 75%）",
			in:   Input{Attack: 1000, InternalConversion: 1000, BlockRating: 250, Parrying: true, Ratio: 1, RMiss: 0.5, RCrit: 0.5},
			want: Output{Damage: 750, Parried: true},
		},
		{
			name: "平减（25 点减免属性折 2 点）",
			in:   Input{Attack: 10, DamageReduction: 25, Ratio: 1, RMiss: 0.5, RCrit: 0.5},
			want: Output{Damage: 8},
		},
		{
			name: "保底 1 点（平减后 ≤0）",
			in:   Input{Attack: 2, DamageReduction: 25, Ratio: 1, RMiss: 0.5, RCrit: 0.5},
			want: Output{Damage: 1},
		},
		{
			name: "技能倍率 1.5",
			in:   Input{Attack: 100, Ratio: 1.5, RMiss: 0.5, RCrit: 0.5},
			want: Output{Damage: 150},
		},
		{
			name: "倍率 ≤0 按 1.0",
			in:   Input{Attack: 100, Ratio: 0, RMiss: 0.5, RCrit: 0.5},
			want: Output{Damage: 100},
		},
	}
	for _, tc := range cases {
		got := f.Calc(&tc.in)
		if got == nil {
			t.Fatalf("%s: Calc = nil", tc.name)
		}
		if got.Damage != tc.want.Damage || got.Miss != tc.want.Miss || got.Crit != tc.want.Crit || got.Parried != tc.want.Parried {
			t.Fatalf("%s: Calc = %+v, want %+v", tc.name, *got, tc.want)
		}
		if tc.wantReductionApprox != 0 && math.Abs(got.ReductionPct-tc.wantReductionApprox) > 1e-9 {
			t.Fatalf("%s: ReductionPct = %v, want ~%v", tc.name, got.ReductionPct, tc.wantReductionApprox)
		}
	}
}

func TestCalcInputDelivered(t *testing.T) {
	// 命中率/暴击率路径依赖预掷与守方属性：坚韧压制暴击、闪避压制命中。
	f := newRealFormula(t)
	suppressed := f.Calc(&Input{Attack: 1000, Ratio: 1, Evasion: 1000, Accuracy: 0, RMiss: 0.2, RCrit: 0.01})
	if suppressed == nil || !suppressed.Miss {
		t.Fatalf("evasion 1000 vs accuracy 0 应未命中（24%% 上限内），got %+v", suppressed)
	}
	noCrit := f.Calc(&Input{Attack: 1000, Ratio: 1, CritValue: 100, Resilience: 100, CritRateReduction: 1, RMiss: 0.5, RCrit: 0.001})
	if noCrit == nil || noCrit.Crit {
		t.Fatalf("暴击率被坚韧/减免压到 0 后 r_crit 再小也不暴击，got %+v", noCrit)
	}
	if noCrit.Damage != 1000 {
		t.Fatalf("noCrit damage = %d, want 1000", noCrit.Damage)
	}
}

func writeModule(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCalcUnavailablePaths(t *testing.T) {
	// 禁用态。
	disabled := New("", nil)
	if disabled.Enabled() || disabled.Calc(base(0.5, 0.5)) != nil {
		t.Fatal("disabled Calc must be nil")
	}

	// 目录缺 formula 模块 → Load 报错。
	empty := New(t.TempDir(), nil)
	if _, err := empty.Load(); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("empty dir Load err = %v, want missing module", err)
	}

	// 模块缺 calc 函数 → 域校验失败。
	noCalc := t.TempDir()
	writeModule(t, noCalc, "formula.lua", "return {}\n")
	nc := New(noCalc, nil)
	if _, err := nc.Load(); err == nil || !strings.Contains(err.Error(), "calc") {
		t.Fatalf("no-calc Load err = %v", err)
	}

	// 运行期出错 → 只禁用该模块，之后 Calc 恒 nil。
	boom := t.TempDir()
	writeModule(t, boom, "formula.lua", `local M = {}
function M.calc(c) error("boom") end
return M
`)
	b := New(boom, nil)
	if _, err := b.Load(); err != nil {
		t.Fatal(err)
	}
	if got := b.Calc(base(0.5, 0.5)); got != nil {
		t.Fatalf("boom Calc = %+v, want nil", got)
	}
	if broken := b.Broken(); len(broken) != 1 || broken[0] != "formula" {
		t.Fatalf("Broken = %v, want [formula]", broken)
	}

	// 返回畸形（缺字段/类型错）→ nil。
	malformed := t.TempDir()
	writeModule(t, malformed, "formula.lua", "return { calc = function(c) return { damage = 1 } end }\n")
	m := New(malformed, nil)
	if _, err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if got := m.Calc(base(0.5, 0.5)); got != nil {
		t.Fatalf("malformed Calc = %+v, want nil", got)
	}
}

func TestInfiniteLoopTimesOut(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "formula.lua", `local M = {}
function M.calc(c) while true do end end
return M
`)
	f := New(dir, nil)
	f.SetTimeout(150 * time.Millisecond)
	if _, err := f.Load(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if got := f.Calc(base(0.5, 0.5)); got != nil {
		t.Fatalf("loop Calc = %+v, want nil", got)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("loop took %v to cancel", elapsed)
	}
	if broken := f.Broken(); len(broken) != 1 {
		t.Fatalf("Broken = %v", f.Broken())
	}
}
