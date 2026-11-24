package taskdata

// quest_source_test.go 任务目录的来源对拍（真档）：数据根同时带客户端
// share/task/task（与文案目录）和 ini/Quest 构建产物时，直读与读产物给出
// 的任务目录逐项相同——任务表、绑定序列（含顺序）。

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

// TestLoadQuestSourceParity 两种来源的任务目录逐项相同。直读需要文案目录
// （数据根自带 text/chineses，或 NINEYIN_TASK_TEXT_DIR 指定），没有即跳过。
func TestLoadQuestSourceParity(t *testing.T) {
	root := parityDataRoot(t)
	for _, probe := range []string{sharedata.QuestDomainDir, filepath.Join("ini", "Quest", "Task", "Task.xml")} {
		if _, err := res.Stat(filepath.Join(root, filepath.FromSlash(probe))); err != nil {
			t.Skipf("对拍需要两种来源同在：%s=%v", probe, err)
		}
	}
	t.Setenv("NINEYIN_DATA_SOURCE", "xml")
	product, err := LoadQuest(root)
	if err != nil {
		t.Fatal("xml:", err)
	}
	t.Setenv("NINEYIN_DATA_SOURCE", "share")
	if QuestSourceOf(root) != sharedata.SourceShare {
		t.Skip("直读需要文案目录（text/chineses 或 NINEYIN_TASK_TEXT_DIR）")
	}
	direct, err := LoadQuest(root)
	if err != nil {
		t.Fatal("share:", err)
	}
	if !reflect.DeepEqual(product.tasks, direct.tasks) {
		t.Fatalf("任务表不同：xml %d / share %d", len(product.tasks), len(direct.tasks))
	}
	if len(product.bindings) != len(direct.bindings) {
		t.Fatalf("绑定数不同：xml %d / share %d", len(product.bindings), len(direct.bindings))
	}
	for i := range product.bindings {
		if product.bindings[i] != direct.bindings[i] {
			t.Fatalf("第 %d 条绑定不同：xml %+v / share %+v", i, product.bindings[i], direct.bindings[i])
		}
	}
	t.Logf("任务目录对拍一致：任务 %d / 绑定 %d", len(direct.tasks), len(direct.bindings))
}
