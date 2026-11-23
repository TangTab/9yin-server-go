package taskdata

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestParseTaskTextCollectsWhitelistedPrefixes(t *testing.T) {
	raw := []byte("\uFEFF// 注释行\n" +
		"CanAcceptMenu_75701=<font face=\"Default\">拜入昆仑（可承接）</font>\n" +
		"title_75701=拜入昆仑\r\n" +
		"target_75701=找师兄报到\n" +
		"task_menu_accept_75701=接受任务\n" + // 白名单外：跳过
		"tasklua_75701=x\n" + // 白名单外：跳过
		"convoy_talk_75701=y\n" + // 白名单外：跳过
		"title_=空ID\n" + // 非法 TID：跳过
		"无法切分的行\n" +
		"title_75702=昆仑试炼\n")
	tasks := make(map[int]*Task)
	CollectTaskText(raw, tasks)
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasks))
	}
	first := tasks[75701]
	if first == nil || first.Title != "拜入昆仑" || first.Target != "找师兄报到" {
		t.Fatalf("task 75701 = %#v", first)
	}
	if first.CanAcceptMenu != `<font face="Default">拜入昆仑（可承接）</font>` {
		t.Fatalf("task 75701 can-accept menu = %q", first.CanAcceptMenu)
	}
	if second := tasks[75702]; second == nil || second.Title != "昆仑试炼" {
		t.Fatalf("task 75702 = %#v", second)
	}
}

func TestCatalogDerivesAcceptMenuKeyFromText(t *testing.T) {
	// 旧 JSON 快照链只有文案：有接取文案的任务折算出接取键（可手动接取），
	// 没有的留空（不可手动接取）；快照往返后稳定。
	catalog := NewCatalog(map[int]Task{
		75701: {CanAcceptMenu: "<font>拜入昆仑（可承接）</font>"},
		75702: {Title: "无接取文案"},
	}, nil, nil)
	first, _ := catalog.TaskByID(75701)
	second, _ := catalog.TaskByID(75702)
	if first.AcceptMenuKey != "CanAcceptMenu_75701" || second.AcceptMenuKey != "" {
		t.Fatalf("接取键折算 = %q / %q", first.AcceptMenuKey, second.AcceptMenuKey)
	}
	data, err := MarshalSnapshot(catalog.SnapshotAt(time.Unix(1700000000, 0).UTC()))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseSnapshot(data)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := parsed.TaskByID(75701); again.AcceptMenuKey != "CanAcceptMenu_75701" {
		t.Fatalf("快照往返后接取键 = %q", again.AcceptMenuKey)
	}
}

func TestParseTaskTextLaterEntryWins(t *testing.T) {
	tasks := make(map[int]*Task)
	CollectTaskText([]byte("title_100=甲\ntitle_100=乙\n"), tasks)
	if got := tasks[100].Title; got != "乙" {
		t.Fatalf("title = %q, want 乙（后写覆盖）", got)
	}
}

const storyTaskListSample = "STRING\tSTRING\tSTRING\tSTRING\tSTRING\tSTRING\tSTRING\tSTRING\tSTRING\r\n" +
	"ID\tNpcID\tcontextId\ttitleId\tLeaveTextID\tLeaveResult\tTrackInfoID\tSceneID\tTaskID\r\n" +
	"75715\tnewmp_kl_001\tstory_75715_1\tstorymenu_75715_1\t9607\t\tstoryinfo_75715\t448\t75730\r\n" +
	"75715\tnewmp_kl_001\tstory_75715_2\tstorymenu_75715_2\t9607\t\tstoryinfo_75715\t448\t75730\r\n" +
	"75717\tWorldNpc01054\tstory_75717_1\tstorymenu_75717_1\t9607\t\tstoryinfo_75717\t14\t75731\r\n" +
	"\r\n" +
	"75718\tnpc_x\tstory_75718_1\tstorymenu_75718_1\t\t\tstoryinfo_75718\t\t不是数字\r\n"

func TestLoadStoryTaskListFromBytes(t *testing.T) {
	bindings, err := LoadStoryTaskListFromBytes([]byte(storyTaskListSample), "sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 3 {
		t.Fatalf("bindings = %#v", bindings)
	}
	first := bindings[0]
	if first.TaskID != 75730 || first.NPCConfigID != "newmp_kl_001" || first.SceneID != "448" || first.Source != "storytasklist" {
		t.Fatalf("first binding = %#v", first)
	}
	if _, err := LoadStoryTaskListFromBytes([]byte("没有任何表头\n"), "broken.txt"); err == nil {
		t.Fatal("missing header should error")
	}
}

const taskNpcIniSample = "// 愆咎城任务npc配置\r\n" +
	"// taskid=npcid,conditionid\r\n" +
	"\r\n" +
	"[TaskNpc]\r\n" +
	"75765=gujian_2021_88_0301,120736\r\n" +
	"75766=gujian_2021_88_0302,120736\r\n" +
	"bad=npc_x,1\r\n" +
	"75767=,120736\r\n" +
	"[Other]\r\n" +
	"1=ignored\r\n"

func TestLoadTaskNPCIniFromBytes(t *testing.T) {
	bindings, err := LoadTaskNPCIniFromBytes([]byte(taskNpcIniSample), "qianjiu_tasknpc.ini")
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 2 {
		t.Fatalf("bindings = %#v", bindings)
	}
	if bindings[0].TaskID != 75765 || bindings[0].NPCConfigID != "gujian_2021_88_0301" || bindings[0].Source != "qianjiu" {
		t.Fatalf("first binding = %#v", bindings[0])
	}
	if _, err := LoadTaskNPCIniFromBytes([]byte("[Nope]\n"), "x.ini"); err == nil {
		t.Fatal("missing [TaskNpc] should error")
	}
}

func TestLoadDramaCiteFromBytes(t *testing.T) {
	raw := []byte("[24001]\nLine=1\nDramaName=qdz01\nChapter=1\nRound=1\nNextTaskLimit=24002\n\n[注释节]\nDramaName=x\n")
	cites := LoadDramaCiteFromBytes(raw)
	if len(cites) != 1 {
		t.Fatalf("cites = %#v", cites)
	}
	cite, ok := cites[24001]
	if !ok || cite.DramaName != "qdz01" || cite.Chapter != 1 || cite.Round != 1 || cite.NextTaskLimit != 24002 {
		t.Fatalf("cite 24001 = %#v", cite)
	}
}

func TestCatalogDeduplicatesAndIndexes(t *testing.T) {
	catalog := NewCatalog(
		map[int]Task{75730: {Title: "虎啸疑云"}},
		[]Binding{
			{TaskID: 75731, NPCConfigID: "WorldNpc01054", Source: "storytasklist"},
			{TaskID: 75730, NPCConfigID: "NewMP_KL_001", SceneID: "448", Source: "storytasklist"},
			{TaskID: 75730, NPCConfigID: "newmp_kl_001", Source: "storytasklist"}, // 去重（大小写不敏感）
			{TaskID: 75730, NPCConfigID: "WorldNpc01054", Source: "qianjiu"},
			{TaskID: 75999, NPCConfigID: "npc_textless"}, // 只有无文本任务
			{TaskID: 0, NPCConfigID: "npc_x"},            // 非法：丢弃
			{TaskID: 5, NPCConfigID: "  "},               // 非法：丢弃
		},
		map[int]DramaCite{75730: {DramaName: "qdz01"}})
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
		t.Fatalf("worldnpc01054 只应见到有文本的 75730: %#v", got)
	}
	if got := catalog.TasksForNPC("npc_textless"); len(got) != 0 {
		t.Fatalf("75999 没有文本，不应计入 TasksForNPC: %#v", got)
	}
	if npcs := catalog.NPCsForTask(75730); len(npcs) != 2 {
		t.Fatalf("npcs for 75730 = %#v", npcs)
	}
	if cite, ok := catalog.DramaCiteByID(75730); !ok || cite.DramaName != "qdz01" {
		t.Fatalf("drama cite 75730 = %#v ok=%v", cite, ok)
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

func TestSnapshotRoundTrip(t *testing.T) {
	catalog := NewCatalog(
		map[int]Task{75730: {Title: "虎啸疑云", CanAcceptMenu: "<font>虎啸疑云（可承接）</font>"}},
		[]Binding{{TaskID: 75730, NPCConfigID: "newmp_kl_001", SceneID: "448", Source: "storytasklist"}},
		map[int]DramaCite{75730: {DramaName: "qdz01", Chapter: 1}})
	data, err := MarshalSnapshot(catalog.SnapshotAt(time.Unix(1700000000, 0).UTC()))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseSnapshot(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Count() != 1 || parsed.BindingCount() != 1 || parsed.BoundNPCCount() != 1 {
		t.Fatalf("parsed counts: tasks=%d bindings=%d npcs=%d", parsed.Count(), parsed.BindingCount(), parsed.BoundNPCCount())
	}
	tasks := parsed.TasksForNPC("newmp_kl_001")
	if len(tasks) != 1 || tasks[0].Title != "虎啸疑云" {
		t.Fatalf("tasks after roundtrip = %#v", tasks)
	}
	if cite, ok := parsed.DramaCiteByID(75730); !ok || cite.NextTaskLimit != 0 || cite.Chapter != 1 {
		t.Fatalf("drama cite after roundtrip = %#v", cite)
	}
}

func TestParseSnapshotRejectsWrongVersion(t *testing.T) {
	if _, err := ParseSnapshot([]byte(`{"version":99,"tasks":{},"bindings":[]}`)); err == nil {
		t.Fatal("version mismatch should error")
	}
	if _, err := ParseSnapshot([]byte("not json")); err == nil {
		t.Fatal("corrupt payload should error")
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
	if _, err := os.Stat(filepath.Join(dir, "Task", "Task.xml")); err != nil {
		t.Skipf("任务域 XML 构建产物缺席（tools/npcgen）: %v", err)
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
