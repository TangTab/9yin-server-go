package clientdata

import "testing"

// TestHybridPropertyTable 混合表（fork 头块 0..173 + 官服共享域 174..2432）
// 结构与名字解析闸：
//  1. 行数=2433、行号严格位置绑定（idx==表位）；
//  2. 头块=fork 场景域（Type@0/Photo@2/ConfigID@7/Resource@8/PosiX@9-12）；
//  3. 共享域=官服行号（Type@181/Resource@1448/ConfigID@1447 为玩家域末现）；
//  4. 场景域首现回落头块（Type→0、Level→6）——NPC 0x0D 头行通道恢复。
func TestHybridPropertyTable(t *testing.T) {
	SwapPropertyTableForTest(t, "hybrid2433")
	if got := len(officialTable.entries); got != 2433 {
		t.Fatalf("混合表行数=%d want 2433", got)
	}
	for i, entry := range officialTable.entries {
		if int(entry.Index) != i {
			t.Fatalf("位置绑定破坏 @%d → %d", i, entry.Index)
		}
	}
	// 头块行校验
	head := map[int]string{0: "Type", 2: "Photo", 7: "ConfigID", 8: "Resource", 9: "PosiX", 10: "PosiY", 11: "PosiZ", 12: "Orient"}
	for idx, name := range head {
		if got := officialTable.entries[idx].Name; got != name {
			t.Fatalf("头块 %d=%s want %s", idx, got, name)
		}
	}
	// 共享域按官服行号（官服方言同号）
	for _, name := range []string{"Type", "Resource", "ConfigID", "Photo", "Level", "HPRatio"} {
		hybrid, ok := PlayerProperty(name)
		if !ok {
			t.Fatalf("玩家域 %s 缺席", name)
		}
		official, _ := lookupOfficial2433(t, name)
		if hybrid.Index != official {
			t.Fatalf("玩家域 %s=%d want 官服 %d", name, hybrid.Index, official)
		}
	}
	// 场景域首现落头块
	for name, want := range map[string]int{"Type": 0, "Photo": 2, "Resource": 8, "ConfigID": 7, "PosiX": 9} {
		entry, ok := SceneProperty(name)
		if !ok || int(entry.Index) != want {
			t.Fatalf("场景域 %s=%+v want idx %d", name, entry, want)
		}
	}
}

func lookupOfficial2433(t *testing.T, name string) (uint16, bool) {
	t.Helper()
	prev, prevMode := officialTable, activePropertyTableMde
	officialTable = parsePropertyTable(officialPropertyTable2433Text, "official2433")
	entry, ok := PlayerProperty(name)
	officialTable, activePropertyTableMde = prev, prevMode
	return entry.Index, ok
}
