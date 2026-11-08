package main

import (
	worldcore "github.com/local/9yin-go-server/internal/world"

	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// TestLoadStaticPropTableGolden 用仓内 testdata 样例（四节逐字拷自 F: 部署
// 的真实 staticprop.ini）钉死解析：[-1] 兜底全字段、[0]（全 4）、[43]
// （稀疏 0/1）、[205]（HPBase=1.775 浮点）。真实 745 节全量在部署端经
// 启动日志（packages=745）验证。
func TestLoadStaticPropTableGolden(t *testing.T) {
	file, err := os.Open(filepath.Join("testdata", "staticprop_golden.ini"))
	if err != nil {
		t.Fatalf("open golden fixture: %v", err)
	}
	defer file.Close()
	sections, err := scanINISections(file)
	if err != nil {
		t.Fatalf("scan golden fixture: %v", err)
	}
	table := parseStaticPropTable(sections)
	if table == nil {
		t.Fatal("staticprop 解析失败（返回 nil）")
	}
	if len(table) != 4 {
		t.Fatalf("样例节数 %d，want 4", len(table))
	}
	fallback := table[staticPropFallbackID]
	if fallback.phyDefBase != 13 || fallback.stiDefBase != 17 || fallback.staBase != 5 || fallback.strBase != 6 {
		t.Fatalf("[-1] 兜底节基数不符：%+v", fallback)
	}
	if zero := table[0]; zero.phyDefBase != 4 || zero.hpBase != 4 || zero.varLevel != 2 {
		t.Fatalf("[0] 节基数不符：%+v", zero)
	}
	if sparse := table[43]; sparse.phyDefBase != 0 || sparse.stiDefBase != 1 || sparse.randomPropType != 30 {
		t.Fatalf("[43] 节基数不符：%+v", sparse)
	}
	if boss := table[205]; math.Abs(boss.hpBase-1.775) > 1e-9 {
		t.Fatalf("[205] HPBase=%v，want 1.775（浮点基数）", boss.hpBase)
	}
}

func TestDeriveNPCCombatStats(t *testing.T) {
	original := installedStaticPropTable
	defer func() { installedStaticPropTable = original }()

	installedStaticPropTable = map[int32]staticPropRow{
		staticPropFallbackID: {phyDefBase: 13},
		1:                    {phyDefBase: 200, dexBase: 0.5},
	}
	// PowerNo=1/L50：phyDef=200×50=10000，evasion=round(0.5×50)=25
	stats := deriveNPCCombatStats(1, 50)
	if stats.phyDef != 10000 || stats.evasion != 25 {
		t.Fatalf("derive(1,50) = %+v，want phyDef=10000 evasion=25", stats)
	}
	// PowerNo 查不到 → [-1] 兜底：13×10=130
	if stats := deriveNPCCombatStats(999, 10); stats.phyDef != 130 {
		t.Fatalf("兜底 derive(999,10) phyDef=%d，want 130", stats.phyDef)
	}
	// Level 缺省 1
	if stats := deriveNPCCombatStats(1, 0); stats.phyDef != 200 {
		t.Fatalf("level 兜底 derive(1,0) phyDef=%d，want 200", stats.phyDef)
	}
	// 表缺失 → 全零（保持旧行为）
	installedStaticPropTable = nil
	if stats := deriveNPCCombatStats(1, 50); stats != (npcCombatStats{}) {
		t.Fatalf("空表 derive 应全零，得 %+v", stats)
	}
}

// TestRollSkillHitAppliesNPCDefense 接线验证：注册 Level=50/PowerNo=1 的
// 攻击 NPC（实力包 phyDefBase=200 → 外防 10000），PhyDamage 行伤害按对数
// 减免 ×(1−0.481)=×0.519：72→37；内功系（negDef=0）不受影响仍 72。
func TestRollSkillHitAppliesNPCDefense(t *testing.T) {
	originalTable := installedStaticPropTable
	installedStaticPropTable = map[int32]staticPropRow{
		staticPropFallbackID: {},
		1:                    {phyDefBase: 200},
	}
	defer func() { installedStaticPropTable = originalTable }()

	originalEngine := installedCombatCalc
	installedCombatCalc = newTestCombatEngine(t, taijiFormula494())
	defer func() { installedCombatCalc = originalEngine }()

	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 { return 0.5 }
	defer func() { combatRandomFloat64 = originalRandom }()

	player := newPlayerActor("tester", 0)
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Properties["Level"] = clientdata.PropertyValue{Value: clientdata.Int32Value(50)}
	npc.resolved.Properties["PowerNo"] = clientdata.PropertyValue{Value: clientdata.Int32Value(1)}
	if err := world.registerNPCOnRun(97, npc); err != nil {
		t.Fatal(err)
	}
	meta, ok := world.npcMeta(worldcore.EntityID(forkNPCID(97)))
	if !ok {
		t.Fatal("npc 97 meta missing")
	}
	if meta.combatStats == nil || meta.combatStats.phyDef != 10000 {
		t.Fatalf("combatStats 推导不符：%+v", meta.combatStats)
	}
	ref := skillDamageTargetRef{id: forkNPCID(97), ownerID: meta.ownerID}

	physical := combatSkillDefinition{
		damageRows: []skillDamageRow{{formulaID: 494, damageType: "PhyDamage", params: []string{"112", "1", "2"}}},
	}
	// 72×(1−(20·ln(10000/100)−44)/100)=72×0.519=37.37→37
	if roll := rollSkillHit(player, physical, ref, world, 1); roll.damage != 37 {
		t.Fatalf("外防 10000 应得 37，得 %+v", roll)
	}
	internal := combatSkillDefinition{
		damageRows: []skillDamageRow{{formulaID: 494, damageType: "NegDamage", params: []string{"112", "1", "2"}}},
	}
	// negDef=0（该实力包只配 phyDefBase）→ 无减免，仍 72
	if roll := rollSkillHit(player, internal, ref, world, 1); roll.damage != 72 {
		t.Fatalf("阴防 0 应得 72，得 %+v", roll)
	}
}
