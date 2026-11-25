package taskdata

// prize_test.go 奖励取数面：物品清单解析 + 真档任务/步骤的奖励字段对拍官服
// 完成簇与步骤奖励帧。

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseItemSpec(t *testing.T) {
	for _, tc := range []struct {
		spec string
		want []PrizeItem
	}{
		{"", nil},
		{"tbook_CS_jh_cqgf03-1", []PrizeItem{{"tbook_CS_jh_cqgf03", 1}}},
		{"hw_fb_rw_01003-60", []PrizeItem{{"hw_fb_rw_01003", 60}}},
		{"item_2020jq_003-1|item_2020jq_004-1", []PrizeItem{{"item_2020jq_003", 1}, {"item_2020jq_004", 1}}},
		{"TaskItemClone005025;TaskItemClone005026", []PrizeItem{{"TaskItemClone005025", 1}, {"TaskItemClone005026", 1}}},
		{"a-2|b; c-0|d-3", []PrizeItem{{"a", 2}, {"b", 1}, {"d", 3}}},
	} {
		if got := ParseItemSpec(tc.spec); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("ParseItemSpec(%q) = %+v, want %+v", tc.spec, got, tc.want)
		}
	}
}

func TestTaskPrizeAccessorsMatchOfficial(t *testing.T) {
	// 官服引导抓包：1340 完成＝碎银 1000 + 江湖声望 10 + LiveGroove_5 +5000
	//（#1931..#1939）；1344＝book_qinggong_7_fc + 碎银 3000 + LiveGroove_1
	//（#6827..#6852）；1346＝additem_0011 + box_cdjh_001 + 碎银 20000 +
	// LiveGroove_5 +10000（#12510..#12525）；1343＝LiveGroove_2（#5620）。
	root := os.Getenv("NINEYIN_DATA_ROOT")
	if root == "" {
		root = filepath.Join("..", "..", "testdata")
	}
	catalog, err := LoadQuest(root)
	if err != nil {
		t.Skipf("任务域真档缺席: %v", err)
	}
	for _, tc := range []struct {
		id          int
		money       int64
		repute      string
		reputeCount int32
		groove      int
		grooveCount int32
		items       []PrizeItem
	}{
		{1340, 1000, "repute_jianghu", 10, 5, 5000, nil},
		{1343, 5000, "repute_jianghu", 250, 2, 5000, nil},
		{1344, 3000, "", 0, 1, 5000, []PrizeItem{{"book_qinggong_7_fc", 1}}},
		{1346, 20000, "", 0, 5, 10000, []PrizeItem{{"additem_0011", 1}, {"box_cdjh_001", 1}}},
	} {
		task, ok := catalog.TaskByID(tc.id)
		if !ok {
			t.Fatalf("任务 %d 不在目录", tc.id)
		}
		if task.PrizeMoney() != tc.money {
			t.Fatalf("%d 碎银 = %d, want %d（%s）", tc.id, task.PrizeMoney(), tc.money, task.Prize)
		}
		if name, count := task.PrizeRepute(); name != tc.repute || count != tc.reputeCount {
			t.Fatalf("%d 声望 = (%q,%d), want (%q,%d)", tc.id, name, count, tc.repute, tc.reputeCount)
		}
		if index, count := task.PrizeLiveGroove(); index != tc.groove || count != tc.grooveCount {
			t.Fatalf("%d 历练 = (%d,%d), want (%d,%d)", tc.id, index, count, tc.groove, tc.grooveCount)
		}
		if got := task.PrizeItems(); !reflect.DeepEqual(got, tc.items) && !(len(got) == 0 && len(tc.items) == 0) {
			t.Fatalf("%d 物品 = %+v, want %+v", tc.id, got, tc.items)
		}
	}
	// 步骤奖励：1347 步骤 558 给武学书；1344 步骤 566 给内功书并扣暮蓼。
	steps := catalog.StepsForTask(1347)
	if len(steps) != 4 || steps[0].GetItems != "tbook_CS_jh_cqgf03-1" || steps[3].GetItems != "" {
		t.Fatalf("1347 步骤奖励 = %+v", steps)
	}
	steps = catalog.StepsForTask(1344)
	if len(steps) != 5 || steps[3].GetItems != "ng_tbook_jh_001-1" || steps[3].LostItems != "item_museborn_1-1" {
		t.Fatalf("1344 步骤 566 奖励/扣除 = %+v", steps)
	}
}
