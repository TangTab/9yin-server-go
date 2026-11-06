package taskdata

import (
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
