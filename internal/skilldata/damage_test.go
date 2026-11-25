package skilldata

// damage_test.go 伤害计算包 / 公式 / buff 联查测试。真值取自官服设计态表
// （share/skill 的 damage_calculate.ini / formula.ini / buff_*.ini，经
// tools/datagen skill 构建）；仓库根 testdata 的产物缺席即跳过。

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// realCatalog 仓库根 testdata 的构建产物（进程级缓存同 Load）。
func realCatalog(t *testing.T) *Catalog {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(root)
	if err != nil {
		t.Skipf("技能构建产物缺席（go run ./tools/datagen skill）: %v", err)
	}
	return catalog
}

// 伤害计算包：普攻一条 r 行；长拳锁定技；五行包按 r/r_2..r_5 取全，伤害
// 类型记号映射到 DAMAGE_TYPE_*（外功 0 / 刚 1 / 柔 2 / 阴 3 / 阳 4）。
func TestDamageCalculate(t *testing.T) {
	catalog := realCatalog(t)
	basic := catalog.DamageCalculate(52)
	if len(basic) != 1 || basic[0].Formula != 372 || basic[0].DamageType != 0 ||
		basic[0].TypeName != "PhyDamage" || strings.Join(basic[0].Args, "|") != "5|1||" {
		t.Fatalf("普攻伤害包 52 不符: %+v", basic)
	}
	lock := catalog.DamageCalculate(2998)
	if len(lock) != 1 || lock[0].Formula != 237 || lock[0].DamageType != 0 ||
		strings.Join(lock[0].Args, "|") != "25|1|3|" {
		t.Fatalf("锁定技伤害包 2998 不符: %+v", lock)
	}
	multi := catalog.DamageCalculate(4526)
	if len(multi) != 5 {
		t.Fatalf("五行包 4526 应取全 5 条 r 行: %+v", multi)
	}
	for i, want := range []struct {
		name string
		code int
		base string
	}{{"StiDamage", 1, "5"}, {"JujDamage", 2, "5"}, {"NegDamage", 3, "5"}, {"MasDamage", 4, "5"}, {"PhyDamage", 0, "40"}} {
		if multi[i].TypeName != want.name || multi[i].DamageType != want.code || multi[i].Formula != 24 ||
			len(multi[i].Args) != 4 || multi[i].Args[0] != want.base {
			t.Errorf("第 %d 条应为 %s 基值 %s: %+v", i, want.name, want.base, multi[i])
		}
	}
	dual := catalog.DamageCalculate(3125)
	if len(dual) != 2 || dual[0].Formula != 81 || dual[0].DamageType != 0 || dual[1].DamageType != 4 ||
		strings.Join(dual[0].Args, "|") != "|||" {
		t.Errorf("双行包 3125 不符: %+v", dual)
	}
	if got := catalog.DamageCalculate(1 << 30); got != nil {
		t.Errorf("缺席的包应返回 nil: %+v", got)
	}
}

// 公式行：P<n> 的来源绑定（女号抓包五个技能用到的两条）。
func TestFormulaParams(t *testing.T) {
	catalog := realCatalog(t)
	normal, ok := catalog.Formula(372)
	if !ok || normal.Expr != "(P1+((P3+P4)*P2*0.1))/P5" {
		t.Fatalf("公式 372 不符: %+v", normal)
	}
	for slot, want := range map[int]FormulaParam{
		0: {Source: SrcArg}, 1: {Source: SrcArg},
		2: {Source: SrcSelfProp, Prop: "MeleePower"}, 3: {Source: SrcSelfProp, Prop: "MeleePowerAdd"},
		4: {Source: SrcSkillContext, Prop: "HurtCount"},
	} {
		if normal.Params[slot] != want {
			t.Errorf("公式 372 的 P%d = %+v, want %+v", slot+1, normal.Params[slot], want)
		}
	}
	single, ok := catalog.Formula(237)
	if !ok || single.Expr != "(P1+((P4+P5)*P2*0.3))/P3" || single.Params[2] != (FormulaParam{Source: SrcArg}) ||
		single.Params[3] != (FormulaParam{Source: SrcSelfProp, Prop: "MeleePower"}) ||
		single.Params[4] != (FormulaParam{Source: SrcSelfProp, Prop: "MeleePowerAdd"}) {
		t.Errorf("公式 237 不符: %+v", single)
	}
	if _, ok := catalog.Formula(1 << 30); ok {
		t.Error("缺席的公式应返回 false")
	}
}

// 公式表达式 → Lua 函数体：官服 tips_func_skill.lua 的 process_formula
// 实证子集 + 嵌套调用 + 小写参数位 + #属性 直引 + 全角括号；子集之外、
// 括号不配平的脏行报错。
func TestTranspile(t *testing.T) {
	cases := map[string]string{
		"(P1+((P3+P4)*P2*0.1))/P5":    "(PV[1]+((PV[3]+PV[4])*PV[2]*0.1))/PV[5]",
		"(R[P1,P2]*1.2)+(P3*5)":       "(R(PV[1],PV[2])*1.2)+(PV[3]*5)",
		"(MAX[P1,P2])*0.04":           "(MAX(PV[1],PV[2]))*0.04",
		"R[P1,P2]/P3":                 "R(PV[1],PV[2])/PV[3]",
		"R[10,20]":                    "R(10,20)",
		"MIN[5,40]*0.05+20":           "MIN(5,40)*0.05+20",
		"10+(MIN[16,P1]*2.5)":         "10+(MIN(16,PV[1])*2.5)",
		"MIN[2.1,MAX[0,P10]]":         "MIN(2.1,MAX(0,PV[10]))",
		"R[MIN[1,P1],MAX[P2,2]]/P3+1": "R(MIN(1,PV[1]),MAX(PV[2],2))/PV[3]+1",
		"6000/#CurTargetCount":        `6000/Q("CurTargetCount")`,
		"p1*(p2+p3)+p4":               "PV[1]*(PV[2]+PV[3])+PV[4]",
		"（P1+P2）*2":                   "(PV[1]+PV[2])*2",
	}
	for expr, want := range cases {
		got, err := Transpile(expr)
		if err != nil || got != want {
			t.Errorf("Transpile(%q) = %q, %v; want %q", expr, got, err, want)
		}
	}
	for _, expr := range []string{
		"", "   ", // 空表达式
		"R[1,2", "MIN[P1", // 方括号未闭合
		"((P1+2)*0.5", "P1)", // 圆括号不配平（源表脏行 formula.ini [1116] 的原形）
		"P13+1", "P0", // 参数位越界
		"R 1,2]", "MAX", "#", // 记号残缺
		"X[1,2]", "1 & 2", // 子集之外
	} {
		if _, err := Transpile(expr); err == nil {
			t.Errorf("Transpile(%q) 应报错", expr)
		}
	}
}

// 全量公式可转译：仓库产物里的每条公式都应转出合法函数体。源表 formula.ini
// [1116] 括号不配平的脏值由构建器按审定值修正（tools/datagen/skill/mapping.go
// Fixups），产物里不留不可转译的行。
func TestAllFormulasTranspile(t *testing.T) {
	catalog := realCatalog(t)
	formulas := catalog.Formulas()
	if len(formulas) < 1000 {
		t.Fatalf("公式行数 %d 异常（产物不完整？）", len(formulas))
	}
	for _, formula := range formulas {
		if _, err := formula.LuaBody(); err != nil {
			t.Errorf("公式 %d %q 转译失败: %v", formula.ID, formula.Expr, err)
		}
	}
	if fixed, ok := catalog.Formula(1116); !ok || fixed.Expr != "(P1+((P2+P3+P4+P5)*0.065))" {
		t.Errorf("公式 1116 应为审定修正值: %+v, %t", fixed, ok)
	}
}

// buff 定义：长拳锁定技的 buf_CS_jh_cqgf01（抓包 m169：Category 73、
// LifeTime 5000ms）；等级行缺席时退回首行；未知 buff 返回 ErrNotFound。
func TestBuffDefinition(t *testing.T) {
	catalog := realCatalog(t)
	buff, err := catalog.Buff("buf_CS_jh_cqgf01", 1)
	if err != nil {
		t.Fatal(err)
	}
	if buff.Script != "Buffer" || buff.StaticData != 5035 || buff.Category != 73 ||
		buff.LifeTimeMS != 5000 || buff.EffectType != 1 || buff.Level != 1 || buff.EffectRatio != 100 ||
		!buff.Visible || !buff.IsDamage || !buff.DeadClear || !buff.OffLineClear || buff.EndFightClear {
		t.Fatalf("buf_CS_jh_cqgf01 不符: %+v", buff)
	}
	if level5, err := catalog.Buff("buf_CS_jh_cqgf01", 5); err != nil || level5.Level != 5 || level5.LifeTimeMS != 5000 {
		t.Errorf("5 级行不符: %+v, %v", level5, err)
	}
	// 静态包 5035 的逐级行只到 12 级：越界等级取首行（1 级）的数值，等级
	// 字段仍记请求值。
	if beyond, err := catalog.Buff("buf_CS_jh_cqgf01", 20); err != nil || beyond.Level != 20 || beyond.LifeTimeMS != 5000 {
		t.Errorf("越界等级应退回首行: %+v, %v", beyond, err)
	}
	if level0, _ := catalog.Buff("buf_CS_jh_cqgf01", 0); level0.Level != 1 {
		t.Errorf("等级 <1 应按 1: %+v", level0)
	}
	if !catalog.BuffExists("buf_CS_jh_cqgf01") || catalog.BuffExists("buf_no_such_buff") {
		t.Error("BuffExists 判定不符")
	}
	if _, err := catalog.Buff("buf_no_such_buff", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("缺席的 buff 应返回 ErrNotFound: %v", err)
	}
}

// 静态包号 → 技能 ID（NPC 战斗行技能列的联查键）。
func TestSkillIDByStaticData(t *testing.T) {
	catalog := realCatalog(t)
	if id, ok := catalog.SkillIDByStaticData(1643); !ok || id != "Clone007_bingkuai" {
		t.Errorf("静态包 1643 → %q, %t", id, ok)
	}
	if _, ok := catalog.SkillIDByStaticData(1 << 30); ok {
		t.Error("缺席的静态包号应返回 false")
	}
}
