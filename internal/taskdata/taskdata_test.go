package taskdata

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// 接取键以主表为准、原样保留：不再有「按接取文案折算」这一步。
func TestCatalogKeepsAcceptMenuKeyAsGiven(t *testing.T) {
	catalog := NewCatalog(map[int]Task{
		75701: {AcceptMenuKey: "CanAcceptMenu_75701"},
		75702: {Title: "无接取文案"},
	}, nil)
	first, _ := catalog.TaskByID(75701)
	second, _ := catalog.TaskByID(75702)
	if first.AcceptMenuKey != "CanAcceptMenu_75701" || second.AcceptMenuKey != "" {
		t.Fatalf("接取键 = %q / %q", first.AcceptMenuKey, second.AcceptMenuKey)
	}
}

func TestCatalogDeduplicatesAndIndexes(t *testing.T) {
	catalog := NewCatalog(
		map[int]Task{75730: {Title: "虎啸疑云"}},
		[]Binding{
			{TaskID: 75731, NPCConfigID: "WorldNpc01054", Source: "storytasklist"},
			{TaskID: 75730, NPCConfigID: "NewMP_KL_001", SceneID: "448", Source: "storytasklist"},
			{TaskID: 75730, NPCConfigID: "newmp_kl_001", Source: "storytasklist"}, // 去重（大小写不敏感）
			{TaskID: 75730, NPCConfigID: "WorldNpc01054", Source: "quest-submit"},
			{TaskID: 75999, NPCConfigID: "npc_textless"}, // 只绑定了目录里没有的任务
			{TaskID: 0, NPCConfigID: "npc_x"},            // 非法：丢弃
			{TaskID: 5, NPCConfigID: "  "},               // 非法：丢弃
		})
	if got := catalog.BindingCount(); got != 4 {
		t.Fatalf("binding count = %d, want 4", got)
	}
	if got := catalog.BoundNPCCount(); got != 3 {
		t.Fatalf("bound npc count = %d, want 3", got)
	}
	if got := catalog.TasksForNPC("NEWMP_KL_001"); len(got) != 1 || got[0].ID != 75730 || got[0].Title != "虎啸疑云" {
		t.Fatalf("tasks for newmp_kl_001 = %#v", got)
	}
	bindings := catalog.BindingsForNPC("worldnpc01054")
	if len(bindings) != 2 || bindings[0].TaskID != 75730 || bindings[1].TaskID != 75731 {
		t.Fatalf("bindings for WorldNpc01054 = %#v", bindings)
	}
	if got := catalog.TasksForNPC("worldnpc01054"); len(got) != 1 || got[0].ID != 75730 {
		t.Fatalf("worldnpc01054 只应见到目录里的 75730: %#v", got)
	}
	if got := catalog.TasksForNPC("npc_textless"); len(got) != 0 {
		t.Fatalf("75999 不在目录里，不应计入 TasksForNPC: %#v", got)
	}
	if npcs := catalog.NPCsForTask(75730); len(npcs) != 2 {
		t.Fatalf("npcs for 75730 = %#v", npcs)
	}
}

func TestCatalogNilReceiverIsSafe(t *testing.T) {
	var catalog *Catalog
	if got := catalog.TasksForNPC("npc"); got != nil {
		t.Fatalf("tasks = %#v, want nil", got)
	}
	if got := catalog.Count(); got != 0 {
		t.Fatalf("count = %d", got)
	}
	if _, ok := catalog.TaskByID(1); ok {
		t.Fatal("nil catalog should not resolve tasks")
	}
}

func TestStripRichText(t *testing.T) {
	input := "<font face=\"Default\" color=\"#ffffff\">拜</font><font face=\"Default\">入昆仑（可承接）</font><br/>第二行"
	if got := StripRichText(input); got != "拜 入昆仑（可承接） 第二行" {
		t.Fatalf("stripped = %q", got)
	}
	if got := StripRichText("  纯文本  "); got != "纯文本" {
		t.Fatalf("stripped = %q", got)
	}
}

func TestLoadQuestXMLDir(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "ini", "Quest")
	if _, err := res.Stat(filepath.Join(dir, "Task", "Task.xml")); err != nil {
		t.Skipf("任务域 XML 构建产物缺席（tools/datagen npc）: %v", err)
	}
	catalog, err := LoadQuestXMLDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Count() == 0 || catalog.BindingCount() == 0 {
		t.Fatalf("目录为空: tasks=%d bindings=%d", catalog.Count(), catalog.BindingCount())
	}
	// 主表 Title 是 npcgen 按 TitleId 解析的客户端显示文本；客户端无文案的
	// 任务（Title 留空）不入目录。
	task, ok := catalog.TaskByID(1340)
	if !ok || task.Title != "武林高手" {
		t.Fatalf("task 1340 = %+v ok=%v（Title 应为 TitleId 解析的显示文本）", task, ok)
	}
	if _, ok := catalog.TaskByID(103); ok {
		t.Fatal("无文案任务 103 不应入目录")
	}
	// 接取/链进字段（主表行带出）：1340 无接取文案（官服从不发它的接取
	// 行）、链根（首谈即接）、剧情走完自动提交、提交后链式接取 1347；
	// 1648 同为链根；27509 虽无前置但是 1648 的链式目标，不算链根。
	for _, tc := range []struct {
		id         int
		acceptKey  string
		chainRoot  bool
		autoSubmit bool
		autoNext   int
	}{
		{1340, "", true, true, 1347},
		{1648, "", true, false, 27509},
		{27509, "", false, false, 27510},
		{81264, "CanAcceptMenu_81264", true, false, 0},
	} {
		got, ok := catalog.TaskByID(tc.id)
		if !ok || got.AcceptMenuKey != tc.acceptKey || got.ChainRoot != tc.chainRoot ||
			got.AutoSubmit != tc.autoSubmit || got.AutoAcceptID != tc.autoNext {
			t.Fatalf("task %d 链字段 = %+v ok=%v, want key=%q root=%v autoSubmit=%v next=%d",
				tc.id, got, ok, tc.acceptKey, tc.chainRoot, tc.autoSubmit, tc.autoNext)
		}
	}
	// 引导链 1340→1347→1341（官服 0x29 记录帧对拍）：主表 AcceptNpc/
	// SubmitNpc 绑定到 NPC（接/交同 NPC 时两侧绑定按（NPC, 任务）去重为一）。
	quests := questBindingsFor(catalog, "FunNpcBorn003031", 1340)
	if len(quests) != 1 || quests[0].Source != "quest-accept" || quests[0].SceneID != "300" {
		t.Fatalf("funnpcborn003031 的 1340 主表绑定 = %+v, want 一条带 AcceptSceneID", quests)
	}
	// SubFunction(story)×StoryTaskList 剧情段：1340 → 组 482 五段（引导抓包
	// story_482_1..5），行序即段序，词汇同 txt 链。源表只在首段行写 NpcID，
	// 续段行沿用——五段都应绑到 funnpcborn003031。
	segments := catalog.StoryBindingsFor("funnpcborn003031", 1340)
	if len(segments) != 5 {
		t.Fatalf("1340 剧情段 = %+v, want 5 段", segments)
	}
	for i, seg := range segments {
		n := strconv.Itoa(i + 1)
		if seg.StoryID != "482" || seg.ContextID != "story_482_"+n || seg.TitleID != "storymenu_482_"+n || seg.LeaveTextID != "9607" {
			t.Fatalf("段 %d = %+v", i, seg)
		}
	}
	if _, err := LoadQuestXMLDir(filepath.Join(dir, "..", "NoQuest")); err == nil {
		t.Fatal("缺 Task.xml 应报错")
	}
}

// questBindingsFor 某 NPC 某任务的主表绑定（quest-accept/quest-submit），
// 按目录行序。
func questBindingsFor(catalog *Catalog, configID string, taskID int) []Binding {
	var out []Binding
	for _, binding := range catalog.BindingsForNPC(configID) {
		if binding.TaskID == taskID && binding.Source != "storytasklist" {
			out = append(out, binding)
		}
	}
	return out
}
