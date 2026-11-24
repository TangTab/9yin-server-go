package fsgame

// source_parity_test.go 来源对拍（真档）：NPC 资源登记与九阴任务域在
// 「直读客户端 share 树」与「读 XML 构建产物」两种来源下的装载结果逐项
// 相同。来源开关见 internal/npcfeed/source.go；两处改道点是
// NpcResourceModule.LoadResource 与 loadPropRows。

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

// parityDataRoot 对拍用的数据根：env NINEYIN_DATA_ROOT，否则仓内 testdata。
func parityDataRoot(t *testing.T) string {
	t.Helper()
	if v := strings.TrimSpace(os.Getenv("NINEYIN_DATA_ROOT")); v != "" {
		return v
	}
	return filepath.Join("..", "..", "testdata")
}

// requireBothSources 两种来源同在才对拍，否则跳过。
func requireBothSources(t *testing.T, root string, probes ...string) {
	t.Helper()
	for _, probe := range probes {
		if _, err := res.Stat(filepath.Join(root, filepath.FromSlash(probe))); err != nil {
			t.Skipf("对拍需要两种来源同在：%s=%v", probe, err)
		}
	}
}

// loadNpcResourceUnder 用装配好的内核在指定数据根上装载 NPC 资源表。
func loadNpcResourceUnder(t *testing.T, k *Kernel, root string) *NpcResourceModule {
	t.Helper()
	k.ResourcePath = root
	m := &NpcResourceModule{}
	if err := m.Init(k); err != nil {
		t.Fatal(err)
	}
	if err := m.LoadResource(k); err != nil {
		t.Fatal(err)
	}
	return m
}

// TestNpcResourceSourceParity NPC 资源登记（npcID → 物化类）两路相同，
// 预载档数相同。
func TestNpcResourceSourceParity(t *testing.T) {
	k, _ := assemble8(t)
	root := parityDataRoot(t)
	requireBothSources(t, root, sharedata.NPCConfigDir, filepath.Join("ini", "Npc", "NpcConfig"))

	t.Setenv("NINEYIN_DATA_SOURCE", "xml")
	product := loadNpcResourceUnder(t, k, root)
	t.Setenv("NINEYIN_DATA_SOURCE", "share")
	direct := loadNpcResourceUnder(t, k, root)

	if product.loaded != direct.loaded {
		t.Fatalf("预载档数不同：xml %d / share %d", product.loaded, direct.loaded)
	}
	if !reflect.DeepEqual(product.npcs, direct.npcs) {
		onlyXML, onlyShare, differ := 0, 0, 0
		for id, class := range product.npcs {
			other, ok := direct.npcs[id]
			if !ok {
				onlyXML++
			} else if other != class {
				differ++
			}
		}
		for id := range direct.npcs {
			if _, ok := product.npcs[id]; !ok {
				onlyShare++
			}
		}
		t.Fatalf("NPC 登记不同：只在 xml %d / 只在 share %d / 类不同 %d", onlyXML, onlyShare, differ)
	}
	t.Logf("NPC 资源登记对拍一致：%d 档 / %d NPC", direct.loaded, len(direct.npcs))
}

// TestNineyinQuestSourceParity 九阴任务域（引擎词汇两表 + 奖励 + NPC 索引 +
// 统计）与任务文案两路相同。直读需要文案目录（text/chineses 或
// NINEYIN_TASK_TEXT_DIR），没有即跳过。
func TestNineyinQuestSourceParity(t *testing.T) {
	root := parityDataRoot(t)
	requireBothSources(t, root, sharedata.QuestDomainDir, filepath.Join("ini", "Quest", "Task", "Task.xml"))

	t.Setenv("NINEYIN_DATA_SOURCE", "xml")
	product, err := loadNineyinQuests(root)
	if err != nil {
		t.Fatal("xml:", err)
	}
	productText, err := loadPropRows(root, "ini/Text/TaskText.xml")
	if err != nil {
		t.Fatal("xml text:", err)
	}
	t.Setenv("NINEYIN_DATA_SOURCE", "share")
	if sharedata.QuestSource(root) != sharedata.SourceShare {
		t.Skip("直读需要文案目录（text/chineses 或 NINEYIN_TASK_TEXT_DIR）")
	}
	direct, err := loadNineyinQuests(root)
	if err != nil {
		t.Fatal("share:", err)
	}
	directText, err := loadPropRows(root, "ini/Text/TaskText.xml")
	if err != nil {
		t.Fatal("share text:", err)
	}

	for _, part := range []struct {
		name     string
		product  any
		direct   any
		countXML int
		countDir int
	}{
		{"任务表", product.Quests, direct.Quests, len(product.Quests), len(direct.Quests)},
		{"子模板", product.Criteria, direct.Criteria, len(product.Criteria), len(direct.Criteria)},
		{"奖励", product.Awards, direct.Awards, len(product.Awards), len(direct.Awards)},
		{"NPC→接取任务", product.NpcAccept, direct.NpcAccept, len(product.NpcAccept), len(direct.NpcAccept)},
		{"NPC→提交任务", product.NpcSubmit, direct.NpcSubmit, len(product.NpcSubmit), len(direct.NpcSubmit)},
	} {
		if !reflect.DeepEqual(part.product, part.direct) {
			t.Fatalf("%s 不同：xml %d / share %d", part.name, part.countXML, part.countDir)
		}
	}
	if product.Stats != direct.Stats {
		t.Fatalf("统计不同：xml %+v / share %+v", product.Stats, direct.Stats)
	}
	if len(productText) != len(directText) {
		t.Fatalf("任务文案行数不同：xml %d / share %d", len(productText), len(directText))
	}
	for i := range productText {
		if !reflect.DeepEqual(productText[i], directText[i]) {
			t.Fatalf("任务文案第 %d 行不同：xml %v / share %v", i, productText[i], directText[i])
		}
	}
	t.Logf("九阴任务域对拍一致：任务 %d / 子模板 %d / 奖励 %d / 文案 %d 行；统计 %+v",
		len(direct.Quests), len(direct.Criteria), len(direct.Awards), len(directText), direct.Stats)
}
