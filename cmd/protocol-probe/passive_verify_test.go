package main

// 真实目录被动位审计（默认跳过）：NINEYIN_PASSIVE_VERIFY_ROOT 指向
// 部署根（含 resources/modern/share）时，对大场景统计 NotPositive 在
// Brain 构造后的 combat/passive 分布。守恒口径：scene01 等野外表
// NotPositive 全空 → 全主动；城内 AttackNpc.xml 大多显式置 1 → 被动。
// resolver 链路（creator 属性层覆盖模板列）若回归，此审计会先于线上
// 行为暴露（例如全城 NPC 重新变主动）。

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

func TestRealCatalogPassiveDistribution(t *testing.T) {
	root := os.Getenv("NINEYIN_PASSIVE_VERIFY_ROOT")
	if root == "" {
		t.Skip("NINEYIN_PASSIVE_VERIFY_ROOT not set")
	}
	tableDir := filepath.Join(root, "resources", "modern", "share", "npc", "npcconfig")
	creatorRoot := filepath.Join(root, "resources", "modern", "share", "creator", "npc_creator")
	tables, err := loadNPCTemplateTables(tableDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{"city05_chengdu", "city01_yanjing", "scene01_youyunshiliuzhou"} {
		spawns, stats, err := loadNPCSceneSpawnsFromTables(tables, filepath.Join(creatorRoot, folder))
		if err != nil {
			t.Fatalf("%s: %v", folder, err)
		}
		combat, passive := 0, 0
		bySource := map[string][2]int{} // NotPositive 溯源文件 → {combat, passive}
		for _, spawn := range spawns {
			services := []npcService(nil) // 交互分类只影响对话路由，战斗判定只看 ScriptClass
			// 闸门后有效种群：EventNpc 与子场景战斗 NPC（SubSceneNo）
			// 不进基础场景，不计入。
			if !npcSpawnEligible(spawn) || npcInteractionFor(spawn.resolved.ScriptClass, services) != npcInteractionCombat {
				continue
			}
			combat++
			source := "unset"
			if property, ok := spawn.resolved.Properties["NotPositive"]; ok {
				source = filepath.Base(property.Provenance.Source)
			}
			counts := bySource[source]
			counts[0]++
			if newNPCCombatState(spawn, worldcore.Transform{X: spawn.x, Y: spawn.y, Z: spawn.z}).Passive {
				passive++
				counts[1]++
			}
			bySource[source] = counts
		}
		t.Logf("%s(EFF-gate): total=%d combat=%d passive=%d active=%d stats{resolved=%d unresolved=%d}",
			folder, len(spawns), combat, passive, combat-passive, stats.Resolved, stats.Unresolved)
		subCombat, subCivil, plainCombat, plainCivil := 0, 0, 0, 0
		for _, spawn := range spawns {
			_, hasSub := spawn.resolved.Extensions["creator.SubSceneNo"]
			isCombat := npcInteractionFor(spawn.resolved.ScriptClass, nil) == npcInteractionCombat
			switch {
			case hasSub && isCombat:
				subCombat++
			case hasSub:
				subCivil++
			case isCombat:
				plainCombat++
			default:
				plainCivil++
			}
		}
		t.Logf("  SubSceneNo: combat=%d(已闸除) civilian=%d(保留) | 无SubSceneNo: combat=%d civilian=%d",
			subCombat, subCivil, plainCombat, plainCivil)
		sources := make([]string, 0, len(bySource))
		for source := range bySource {
			sources = append(sources, source)
		}
		sort.Strings(sources)
		for _, source := range sources {
			counts := bySource[source]
			t.Logf("  %s: combat=%d passive=%d", source, counts[0], counts[1])
		}
		if folder == "city05_chengdu" {
			// 闸后仍为主动的 combat NPC 按 NpcType 聚类（客户端红/白名
			// 家族），每类带样例名，定位“白名却主动咬人”的残留种群。
			type typeGroup struct {
				count int
				names map[string]int
			}
			groups := map[int32]*typeGroup{}
			for _, spawn := range spawns {
				if !npcSpawnEligible(spawn) || npcInteractionFor(spawn.resolved.ScriptClass, nil) != npcInteractionCombat {
					continue
				}
				if newNPCCombatState(spawn, worldcore.Transform{}).Passive {
					continue
				}
				npcType := spawn.int32Property("NpcType")
				group := groups[npcType]
				if group == nil {
					group = &typeGroup{names: map[string]int{}}
					groups[npcType] = group
				}
				group.count++
				group.names[spawn.textProperty("Name")]++
			}
			types := make([]int, 0, len(groups))
			for npcType := range groups {
				types = append(types, int(npcType))
			}
			sort.Ints(types)
			for _, npcType := range types {
				group := groups[int32(npcType)]
				names := make([]string, 0, len(group.names))
				for name := range group.names {
					names = append(names, name)
				}
				sort.Slice(names, func(i, j int) bool { return group.names[names[i]] > group.names[names[j]] })
				if len(names) > 6 {
					names = names[:6]
				}
				t.Logf("  ACTIVE NpcType=%d count=%d names=%v", npcType, group.count, names)
			}
		}
	}
}
