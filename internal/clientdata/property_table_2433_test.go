package clientdata

import "testing"

// TestPropertyTableOfficial2433Structure 钉死外服私服 2433 表的结构与锚点
// （c.pcapng 7074 线 0x09 帧提取，docs/official/外服登录帧对照与缺口清单.md §四-2
// 第二次修正）。头块 0..173 换血为家园/帮会/VIP 域、共享域 +2/+7 插入漂移。
// 2026-11-14 表尾追：末尾追加 61 行 fork 头块缺席名（PosiX/Orient/Race…
// 官方 2433 行零改动），fork 客户端渲染头行通道复活——锚点全部来自
// 外服私服帧实解（0x0D #1777 / §11.4 View45）。
func TestPropertyTableOfficial2433Structure(t *testing.T) {
	table := parsePropertyTable(officialPropertyTable2433Text, "official2433-test")
	if got := len(table.entries); got != 2494 {
		t.Fatalf("official2433+tail entries=%d, want 2494", got)
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
		{183, "State", 7},            // §13.5 外服私服 NPC State
		{214, "ActionSpeed", 5},      // 外服私服 0x0D #1777
		{251, "YinLevel", 3},         // §11.1-11 登录簇
		{417, "AutoSelectObject", 9}, // §四-5 外服私服 m431 应答
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
	// 尾追段断言：fork 头块缺席名落官方 2433 行之后的连续尾位
	//（官方行零改动，外服抓包 diff/回放不受影响）。
	tailAnchors := []struct {
		index int
		name  string
		tc    int
	}{
		{2433, "Race", 3},
		{2434, "PosiX", 5},
		{2435, "PosiY", 5},
		{2436, "PosiZ", 5},
		{2437, "Orient", 5},
		{2493, "Prop173", 7},
	}
	for _, a := range tailAnchors {
		entry := table.entries[a.index]
		if entry.Name != a.name || entry.TypeCode != a.tc {
			t.Fatalf("tail anchor @%d = (%s, tc%d), want (%s, tc%d)", a.index, entry.Name, entry.TypeCode, a.name, a.tc)
		}
	}
	// StopTime 刻意不追加（fork 双列审定语义已退役，保持断供）。
	if _, ok := table.byNameScene("StopTime", 0); ok {
		t.Fatal("StopTime unexpectedly present (intentionally not tail-added)")
	}
}

// TestPropertyTablePlayerDomainResolution 玩家域末现位断言
// （原双方言漂移实证收敛为唯一表锚点）：AutoSelectObject=417
// （fork 415 已退役）、NpcType 末现位 1566（fork 1559）——
// officialNPCProperties 按名解析的回归依据。
func TestPropertyTablePlayerDomainResolution(t *testing.T) {
	entry, ok := PlayerProperty("AutoSelectObject")
	if !ok || entry.Index != 417 {
		t.Fatalf("official2433 AutoSelectObject = %+v, want index 417", entry)
	}
	if entry, _ := PlayerProperty("NpcType"); entry.Index != 1566 {
		t.Fatalf("official2433 NpcType = %d, want 1566", entry.Index)
	}
}
