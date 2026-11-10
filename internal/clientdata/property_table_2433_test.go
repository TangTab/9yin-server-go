package clientdata

import "testing"

// TestPropertyTableOfficial2433Structure 钉死官服 2433 表的结构与锚点
// （c.pcapng 7074 线 0x09 帧提取，docs/官服登录帧对照与缺口清单.md §四-2
// 第二次修正）。该表是换表工程（分支 official-2433-table §14.3）的目标
// 方言：头块 0..173 换血为家园/帮会/VIP 域、64 名缺席、共享域 +2/+7
// 插入漂移。锚点全部来自官服帧实解（0x0D #1777 / §11.4 View45）。
func TestPropertyTableOfficial2433Structure(t *testing.T) {
	table := parsePropertyTable(officialPropertyTable2433Text, "official2433-test")
	if got := len(table.entries); got != 2433 {
		t.Fatalf("official2433 entries=%d, want 2433", got)
	}
	// 头块换血实证：0=None(不可编码)、1=HomeCount；fork 场景域 0=Type。
	if table.entries[0].Name != "None" || table.entries[1].Name != "HomeCount" {
		t.Fatalf("official2433 head block changed: [0]=%s [1]=%s", table.entries[0].Name, table.entries[1].Name)
	}
	anchors := []struct {
		index int
		name  string
		tc    int
	}{
		{183, "State", 7},            // §13.5 官服 NPC State
		{214, "ActionSpeed", 5},      // 官服 0x0D #1777
		{251, "YinLevel", 3},         // §11.1-11 登录簇
		{417, "AutoSelectObject", 9}, // §四-5 官服 m431 应答
		{471, "HPRatio", 3},
		{480, "ParryRatio", 3},
		{484, "HitHPRatio", 3},
		{767, "TotalFillValue", 3}, // §11.4 View45
		{1447, "ConfigID", 7},
		{1448, "Resource", 7},
		{1472, "Level", 1},
		{1566, "NpcType", 3},
		{1597, "BossLevel", 1},
		{1602, "IDCard", 7},
		{1604, "BornAction", 7},
		{1606, "WeaponMode", 7},
		{2246, "VarPropNo", 3},
	}
	for _, a := range anchors {
		entry := table.entries[a.index]
		if entry.Name != a.name || entry.TypeCode != a.tc {
			t.Fatalf("official2433 anchor @%d = (%s, tc%d), want (%s, tc%d)", a.index, entry.Name, entry.TypeCode, a.name, a.tc)
		}
	}
	// 64 缺名抽样：fork 场景域行在官服表不存在（换表断供清单）。
	for _, absent := range []string{"BufferListStr", "NpcNickName", "Race", "PosiX", "PosiZ", "Orient", "AITemplate", "PatrolMode", "ChaseRange", "RefreshTime"} {
		if _, ok := table.byNameScene(absent, 0); ok {
			t.Fatalf("official2433 unexpectedly contains %q", absent)
		}
		if _, ok := table.player[absent]; ok {
			t.Fatalf("official2433 unexpectedly contains %q in player domain", absent)
		}
	}
}

// TestPropertyTableDualDialectResolution 同名跨表解析漂移实证：
// PlayerProperty 在两表间对共享域名字给出不同索引（fork 415 ↔ 官服 417），
// 这是 officialNPCProperties 改按名解析的依据——换表自动跟随。
func TestPropertyTableDualDialectResolution(t *testing.T) {
	forkEntry, ok := PlayerProperty("AutoSelectObject")
	if !ok || forkEntry.Index != 415 {
		t.Fatalf("fork AutoSelectObject = %+v, want index 415", forkEntry)
	}
	SwapPropertyTableForTest(t, "official2433")
	officialEntry, ok := PlayerProperty("AutoSelectObject")
	if !ok || officialEntry.Index != 417 {
		t.Fatalf("official2433 AutoSelectObject = %+v, want index 417", officialEntry)
	}
	// 官服表 NpcType 末现位 1566（fork 为 1559）——16-prop 方言的 NpcType 行。
	if entry, _ := PlayerProperty("NpcType"); entry.Index != 1566 {
		t.Fatalf("official2433 NpcType = %d, want 1566", entry.Index)
	}
}
