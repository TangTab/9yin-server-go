package combatcalc

import (
	"path/filepath"
	"testing"
)

// engineDir 指向仓内引擎脚本（scripts/module/combat/formula.lua）。
func engineDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "scripts", "module", "combat")
}

// taijiSpec 是 formula.ini [494]（太极拳系威力系数公式）的规格镜像：
// (P1+((P4+P5+P6+P7)*P2*0.195))/P3，P4..P7=近战威力/威力加成/武器下限/上限。
func taijiSpec() FormulaSpec {
	return FormulaSpec{
		ID:      494,
		Formula: "(P1+((P4+P5+P6+P7)*P2*0.195))/P3",
		Src:     map[int]int{1: 5, 2: 5, 3: 5, 4: 0, 5: 0, 6: 0, 7: 0},
		Prop: map[int]string{
			4: "MeleePower", 5: "MeleePowerAdd",
			6: "MinMeleeDamage", 7: "MaxMeleeDamage",
		},
	}
}

func testProps() map[string]int64 {
	return map[string]int64{
		"MeleePower": 80, "MeleePowerAdd": 30,
		"MinMeleeDamage": 20, "MaxMeleeDamage": 35,
	}
}

func TestEngineCalcHitGoldenTaiji(t *testing.T) {
	engine := New(engineDir(t), nil)
	installed, err := engine.Load([]FormulaSpec{taijiSpec()})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if installed != 1 {
		t.Fatalf("installed = %d, want 1", installed)
	}
	// 太极拳一式 L1：r=494,NegDamage,112,1,2 →
	// (112+((80+30+20+35)*1*0.195))/2 = (112+32.175)/2 = 72.08 → 72
	result, ok := engine.CalcHit(&HitInput{
		FormulaID: 494,
		Params:    []string{"112", "1", "2"},
		Props:     testProps(),
		MissRoll:  0.5, CritRoll: 0.5,
	})
	if !ok {
		t.Fatalf("CalcHit 未接手")
	}
	if result.Miss || result.Crit {
		t.Fatalf("期望命中非暴击，得 miss=%t crit=%t", result.Miss, result.Crit)
	}
	if result.Damage != 72 {
		t.Fatalf("damage = %d, want 72", result.Damage)
	}
}

func TestEngineCalcHitMissCritFloor(t *testing.T) {
	engine := New(engineDir(t), nil)
	if _, err := engine.Load([]FormulaSpec{taijiSpec()}); err != nil {
		t.Fatalf("Load: %v", err)
	}
	base := HitInput{
		FormulaID: 494,
		Params:    []string{"112", "1", "2"},
		Props:     testProps(),
	}

	// miss：基础 10%，掷 0.05 → 未命中
	in := base
	in.MissRoll, in.CritRoll = 0.05, 0.5
	if result, ok := engine.CalcHit(&in); !ok || !result.Miss || result.Damage != 0 {
		t.Fatalf("miss 用例得 %+v ok=%t", result, ok)
	}
	// crit：基础 5%，掷 0.01 → ×2.0：72.0875×2=144.175 → 144
	in = base
	in.MissRoll, in.CritRoll = 0.5, 0.01
	if result, ok := engine.CalcHit(&in); !ok || !result.Crit || result.Damage != 144 {
		t.Fatalf("crit 用例得 %+v ok=%t", result, ok)
	}
	// 保底：零属性+零参数 → raw<1 → 保底 1
	weak := base
	weak.MissRoll, weak.CritRoll = 0.5, 0.5
	weak.Props = map[string]int64{"MeleePower": 0, "MeleePowerAdd": 0, "MinMeleeDamage": 0, "MaxMeleeDamage": 0}
	weak.Params = []string{"0", "1", "2"}
	if result, ok := engine.CalcHit(&weak); !ok || result.Damage != 1 {
		t.Fatalf("保底用例得 %+v ok=%t", result, ok)
	}
}

// TestEngineCalcHitRangeParam 验证 Src=6 的 "a;b" 区间参数与公式内 R[a,b]
// 都吃预掷序列（公式 [9] R[P1,P2]：P1/P2 为字面量）。
func TestEngineCalcHitRangeParam(t *testing.T) {
	engine := New(engineDir(t), nil)
	specs := []FormulaSpec{
		{ID: 9, Formula: "R[P1,P2]", Src: map[int]int{1: 5, 2: 5}},
		{ID: 1, Formula: "P1+P2+((P4+R[P5,P6])/P3)",
			Src:  map[int]int{1: 5, 2: 6, 3: 5, 4: 0, 5: 0, 6: 0},
			Prop: map[int]string{4: "MeleePower", 5: "MinMeleeDamage", 6: "MaxMeleeDamage"}},
	}
	if _, err := engine.Load(specs); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// R[10,20] 掷 0.5 → 10+floor(0.5*11)=15
	result, ok := engine.CalcHit(&HitInput{
		FormulaID: 9, Params: []string{"10", "20"},
		MissRoll: 0.5, CritRoll: 0.5, ExtraRolls: []float64{0.5},
	})
	if !ok || result.Damage != 15 {
		t.Fatalf("R[10,20] 得 %+v ok=%t", result, ok)
	}
	// 公式 [1]：P1=100、P2="-5;5" 掷 0.5 → -5+floor(0.5×11)=0、P3=2、威力 80、
	// 武器 R[20,35] 掷 0.5 → 20+floor(0.5×16)=28
	// → 100+0+(80+28)/2 = 154（区间掷消费顺序：Src=6 先于 R[·,·]）
	result, ok = engine.CalcHit(&HitInput{
		FormulaID: 1, Params: []string{"100", "-5;5", "2"},
		Props:     map[string]int64{"MeleePower": 80, "MinMeleeDamage": 20, "MaxMeleeDamage": 35},
		MissRoll:  0.5, CritRoll: 0.5,
		ExtraRolls: []float64{0.5, 0.5},
	})
	if !ok || result.Damage != 154 {
		t.Fatalf("公式[1] 得 %+v ok=%t", result, ok)
	}
}

func TestEngineFallbacks(t *testing.T) {
	engine := New(engineDir(t), nil)
	if _, err := engine.Load([]FormulaSpec{taijiSpec()}); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 未注入的公式号 → 不接手
	if _, ok := engine.CalcHit(&HitInput{FormulaID: 12345, Params: []string{"1"}, MissRoll: 0.5, CritRoll: 0.5}); ok {
		t.Fatalf("未注入公式应回退")
	}
	// 含不支持 Src（11=buff 释放者）的公式装载期被跳过
	bad := FormulaSpec{ID: 390, Formula: "P1+((P3+P4)*0.1)",
		Src: map[int]int{1: 5, 3: 11, 4: 11}, Prop: map[int]string{3: "Str", 4: "StrAdd"}}
	engine2 := New(engineDir(t), nil)
	installed, err := engine2.Load([]FormulaSpec{bad, taijiSpec()})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if installed != 1 || engine2.Supports(390) || !engine2.Supports(494) {
		t.Fatalf("bad 公式应被跳过：installed=%d supports390=%t", installed, engine2.Supports(390))
	}
	// 空目录引擎：不 Ready，CalcHit 恒回退
	empty := New("", nil)
	if empty.Ready() || empty.Enabled() {
		t.Fatalf("空目录引擎应禁用")
	}
	if _, ok := empty.CalcHit(&HitInput{FormulaID: 494}); ok {
		t.Fatalf("空引擎应回退")
	}
}

// TestEngineCalcHitAttackerDefenderStats 验证攻方暴击/命中真实值与守方
// 三字段（§十六②④管线）：默认 16 暴击值 vs 零封劲 ≈14.4%（§七 PVE 实测
// 14–15%），封劲 16 压回 5%，1 万防御 →48.1% 减免，闪避 50 vs 命中 50 →10%。
func TestEngineCalcHitAttackerDefenderStats(t *testing.T) {
	engine := New(engineDir(t), nil)
	if _, err := engine.Load([]FormulaSpec{taijiSpec()}); err != nil {
		t.Fatalf("Load: %v", err)
	}
	base := HitInput{
		FormulaID: 494,
		Params:    []string{"112", "1", "2"},
		Props:     testProps(),
		CritValue: 16, Accuracy: 50,
		MissRoll: 0.5, CritRoll: 0.5,
	}

	// 暴击边界：critChance(16,0)=5%+10%×16/17=14.41%
	in := base
	in.CritRoll = 0.14
	if r, ok := engine.CalcHit(&in); !ok || !r.Crit || r.Damage != 144 {
		t.Fatalf("crit@0.14 得 %+v ok=%t（want crit=144）", r, ok)
	}
	in.CritRoll = 0.15
	if r, ok := engine.CalcHit(&in); !ok || r.Crit || r.Damage != 72 {
		t.Fatalf("no-crit@0.15 得 %+v ok=%t（want 72）", r, ok)
	}

	// 守方封劲 16 → critChance(16,16)=5%：0.04 暴 / 0.06 不暴
	in = base
	in.Defender = &DefenderStats{Resilience: 16}
	in.CritRoll = 0.04
	if r, ok := engine.CalcHit(&in); !ok || !r.Crit {
		t.Fatalf("crit@0.04/res16 得 %+v ok=%t", r, ok)
	}
	in.CritRoll = 0.06
	if r, ok := engine.CalcHit(&in); !ok || r.Crit {
		t.Fatalf("no-crit@0.06/res16 得 %+v ok=%t", r, ok)
	}

	// 守方防御 10000 → 减免 20·ln(100)−44=48.1%：72×0.519=37.37→37
	in = base
	in.Defender = &DefenderStats{Defense: 10000}
	if r, ok := engine.CalcHit(&in); !ok || r.Damage != 37 {
		t.Fatalf("def10000 得 %+v ok=%t（want 37）", r, ok)
	}

	// 守方闪避 50 vs 命中 50 → missChance=10%：0.05 未命中
	in = base
	in.Defender = &DefenderStats{Evasion: 50}
	in.MissRoll = 0.05
	if r, ok := engine.CalcHit(&in); !ok || !r.Miss || r.Damage != 0 {
		t.Fatalf("eva50/acc50 miss 得 %+v ok=%t", r, ok)
	}

	// 攻方命中 50 vs 守方闪避 0：永不 miss（真实 PVE 口径——低级怪近零闪避）
	in = base
	in.MissRoll = 0.001
	if r, ok := engine.CalcHit(&in); !ok || r.Miss {
		t.Fatalf("eva0 应永不 miss 得 %+v ok=%t", r, ok)
	}
}
