package taskdata

// steps_test.go 步骤行与奖励串（真档）：按官服引导抓包里解出的 Task_Record
// 目标行与 Task_Accepted 奖励串逐字段对拍。

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// loadRealQuest 真档任务目录：env NINEYIN_DATA_ROOT，否则仓内 testdata。
func loadRealQuest(t *testing.T) *Catalog {
	t.Helper()
	root := strings.TrimSpace(os.Getenv("NINEYIN_DATA_ROOT"))
	if root == "" {
		root = filepath.Join("..", "..", "testdata")
	}
	catalog, err := LoadQuest(root)
	if err != nil {
		t.Skipf("任务域真档缺席: %v", err)
	}
	return catalog
}

// TestStepsMatchOfficialRecords 步骤行对拍官服 Task_Record 行（引导抓包
// #1537/#1988..#1991/#2423..#2424/#3574..#3578）。
func TestStepsMatchOfficialRecords(t *testing.T) {
	catalog := loadRealQuest(t)
	type want struct {
		subtype, index, order, maxnum int
		key, scene, track             string
		link                          bool
	}
	for _, tc := range []struct {
		taskID int
		steps  []want
	}{
		{1340, []want{{StepStory, 482, 1, 1, "funnpcborn003031", "", "storyinfo_482", true}}},
		{1347, []want{
			{StepInteract, 558, 1, 1, "funnpcborn003035", "DialogID_558", "interactinfo_558", true},
			{StepInteract, 559, 2, 1, "funnpcborn003035", "DialogID_559", "interactinfo_559", true},
			{StepInteract, 560, 3, 1, "funnpcborn003035", "DialogID_560", "interactinfo_560", true},
			{StepInteract, 561, 4, 1, "funnpcborn003035", "DialogID_561", "interactinfo_561", true},
		}},
		{1341, []want{
			{StepHunter, 522, 1, 1, "NpcBorn003011", "", "hunterinfo_522", false},
			{StepHunter, 518, 2, 1, "bossborn003001", "", "hunterinfo_518", false},
		}},
		{1342, []want{
			{StepInteract, 562, 1, 1, "funnpcborn003032", "DialogID_562", "interactinfo_562", true},
			{StepSpecial, 203, 2, 1, "2061", "", "specialinfo_2061", true},
			{StepStory, 484, 3, 1, "funnpcborn003032", "", "storyinfo_484", true},
			{StepInteract, 570, 4, 1, "funnpcborn003032", "DialogID_570", "interactinfo_570", true},
			{StepInteract, 139, 5, 1, "funnpcborn003032", "DialogID_139", "interactinfo_139", true},
		}},
		{1343, []want{
			{StepHunter, 519, 1, 3, "Npcborn003005", "", "hunterinfo_519", false},
			{StepInteract, 575, 2, 1, "funnpcborn003042", "DialogID_575", "interactinfo_575", true},
			{StepHunter, 523, 3, 1, "cantsuccess", "", "hunterinfo_523", false},
			{StepHunter, 520, 4, 1, "bossborn003002", "", "hunterinfo_520", false},
		}},
		{1648, []want{{StepStory, 244, 1, 1, "Transcity05D", "", "storyinfo_244", true}}},
	} {
		got := catalog.StepsForTask(tc.taskID)
		if len(got) != len(tc.steps) {
			t.Fatalf("任务 %d 步骤行 = %+v, want %d 行", tc.taskID, got, len(tc.steps))
		}
		for i, w := range tc.steps {
			s := got[i]
			if s.Subtype != w.subtype || s.Index != w.index || s.Order != w.order || s.MaxNum != w.maxnum ||
				s.Key != w.key || s.Scene != w.scene || s.TrackInfo != w.track || s.HasLink != w.link {
				t.Fatalf("任务 %d 步骤 %d = %+v, want %+v", tc.taskID, i, s, w)
			}
			// 交互步确认行文案键（官服步骤窗 m6 [500×10⁶+TID, "okmenu_558"]）。
			okMenu := ""
			if s.Subtype == StepInteract {
				okMenu = "okmenu_" + strconv.Itoa(w.index)
			}
			if s.OkMenu != okMenu {
				t.Fatalf("任务 %d 步骤 %d OkMenu = %q, want %q", tc.taskID, i, s.OkMenu, okMenu)
			}
		}
	}
}

// TestTaskRecordColumnsMatchOfficial 主表记录行字段与奖励串对拍官服
// Task_Accepted 行（引导抓包 #1538/#1992/#2425/#3579/#4438/#5634/#6858/
// #8221/#8386/#12762）。
func TestTaskRecordColumnsMatchOfficial(t *testing.T) {
	catalog := loadRealQuest(t)
	for _, tc := range []struct {
		id                          int
		title, target, context      string
		acceptNpc, submitNpc        string
		acceptScene, submitScene    int
		submitNpcEx, completeDialog string
		prize                       string
		guide                       bool
	}{
		{1340, "title_1340", "target_1340", "context_1340", "funnpcborn003031", "funnpcborn003031", 300, 300,
			"SubmitNpcEx_1340", "complete_1340;submit_1340",
			"1000,0,repute_jianghu,10,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,5,5000,0,0", true},
		{1347, "title_1347", "target_1347", "context_1347", "funnpcborn003035", "funnpcborn003035", 300, 300,
			"SubmitNpcEx_1347", "complete_1347;submit_1347",
			"1000,0,repute_jianghu,20,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,5,10000,0,0", true},
		// 1343 是 AutoSubmit 任务：主表没有 CompleteDialogId（不走提交窗）。
		{1343, "title_1343", "target_1343", "context_1343", "funnpcborn003032", "funnpcborn003033", 300, 300,
			"SubmitNpcEx_1343", "",
			"5000,0,repute_jianghu,250,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,2,5000,0,0", true},
		{1344, "title_1344", "target_1344", "context_1344", "funnpcborn003033", "funnpcborn003033", 300, 300,
			// 官服这一行未带 submit_npc_ex，主表有值——以主表为准。
			"SubmitNpcEx_1344", "complete_1344;submit_1344",
			"3000,0,0,0,0,0,0,0,0,0,book_qinggong_7_fc,1,0,0,0,0,0,0,0,1,5000,0,0", true},
		{1346, "title_1346", "target_1346", "context_1346", "funnpcborn003038", "funnpcborn003034", 300, 300,
			"SubmitNpcEx_1346", "complete_1346;submit_1346",
			"20000,0,0,0,0,0,0,0,0,0,additem_0011,1,box_cdjh_001,1,0,0,0,0,0,5,10000,0,0", true},
		{1648, "title_1648", "target_1648", "context_1648", "", "Transcity05D", 17, 17,
			"", "complete_1648;submit_1648",
			// 1648 在引导线（Line 9）但 SearchID 是 s980107，不属于 s980301 那条链。
			"0,0,repute_jianghu,20,0,0,0,0,0,0,mount_2_001yd,1,book_riding_start,1,0,0,0,0,0,5,10000,0,0", false},
	} {
		got, ok := catalog.TaskByID(tc.id)
		if !ok {
			t.Fatalf("任务 %d 不在目录", tc.id)
		}
		if got.Line != LineGuide || got.IsGuide != tc.guide || got.TitleKey != tc.title || got.TargetKey != tc.target ||
			got.ContextKey != tc.context || got.AcceptNpc != tc.acceptNpc || got.SubmitNpc != tc.submitNpc ||
			got.AcceptScene != tc.acceptScene || got.SubmitScene != tc.submitScene ||
			got.SubmitNpcEx != tc.submitNpcEx || !strings.HasPrefix(tc.completeDialog, got.CompleteDialog) ||
			got.Prize != tc.prize {
			t.Fatalf("任务 %d 记录行列 = %+v", tc.id, got)
		}
	}
	if _, ok := catalog.TaskByID(1597); ok {
		if got, _ := catalog.TaskByID(1597); got.Prize != "" {
			t.Fatalf("1597 无 Prize 行，奖励串应为空: %q", got.Prize)
		}
	}
	if got, ok := catalog.TaskByID(81264); !ok || got.IsGuide {
		t.Fatalf("81264 非引导链: %+v", got)
	}
}
