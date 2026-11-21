package rolestate

import (
	"strings"
	"testing"
)

// TestBuildRoleStateBorn03 钉死默认建号状态（官方口径：born03 烟雨庄 +
// RoleBaseProp 五维=3/HP70/MP40/SP80 + StarterSet 授予集 + location.ini
// BornCJ000003 出生点）。
func TestBuildRoleStateBorn03(t *testing.T) {
	state, err := BuildRoleState(DefaultBirthScene, 1, "icon\\players\\boy01_create.png,obj\\char\\b_hair\\b_hair1", 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if state.Sex != 1 || state.Str != 3 || state.Dex != 3 || state.Ing != 3 || state.Spi != 3 || state.Sta != 3 {
		t.Fatalf("五维官方口径 =3（初始号实证）： %+v", state)
	}
	if state.HP != 70 || state.MP != 40 || state.SP != 80 || state.Parry != 100 {
		t.Fatalf("资源初值 HP70/MP40/SP80/Parry100： %+v", state)
	}
	if state.Scene != `ini\scene\born03_YanYuZhuang` || state.Resource != "born03" {
		t.Fatalf("出生场景 = %q resource = %q", state.Scene, state.Resource)
	}
	if state.Position != [4]float64{482.5, 13.7, 1008, 1.56} {
		t.Fatalf("BornCJ000003 出生点 = %v", state.Position)
	}
	if state.NewbieSkill != 1 || state.InitTask != 22001 || state.BaishiTask != 49 {
		t.Fatalf("NewbieSkill/InitTask/BaishiTask = %d/%d/%d", state.NewbieSkill, state.InitTask, state.BaishiTask)
	}
	if len(state.Skills) != 27 {
		t.Fatalf("starter 武学 27 项（5 基本+7 花神+7 心佛掌+8 太极），得 %d", len(state.Skills))
	}
	if len(state.NeiGong) != 6 || state.CurNeiGong != "ng_wd_001" {
		t.Fatalf("初始内功 6 本/首本 ng_wd_001，得 %d 本 cur=%s", len(state.NeiGong), state.CurNeiGong)
	}
	if state.NeiGong[0].Total != 273200 || state.NeiGong[0].Power != 4550 || state.NeiGong[0].MaxHPAdd != 1630 {
		t.Fatalf("首本内功 30 级数值： %+v", state.NeiGong[0])
	}
	if len(state.QingGongCategories) != 18 || len(state.QingGongSkills) != 52 {
		t.Fatalf("轻功类别 18/实例 52（b-line starterQingGongSkillIDs 全集），得 %d/%d", len(state.QingGongCategories), len(state.QingGongSkills))
	}
	if len(state.CollectSkills) != 204 {
		t.Fatalf("采集技能 204 项，得 %d", len(state.CollectSkills))
	}
	// 任务门：Guide 1069..1073(5) + Alt 1344 + SchoolLead 1555..1562(8) +
	// Act 1140..1147(8) + ActAlt 51002 = 23 行（b-line facultyAcceptedTaskIDs）。
	if len(state.TaskAccepted) != 23 || state.TaskAccepted[0] != 1069 || state.TaskAccepted[len(state.TaskAccepted)-1] != 51002 {
		t.Fatalf("Task_Accepted 23 行： %v", state.TaskAccepted)
	}
	// Task_Completed：Unlock 1346 + OpenGate 49634 + FuncGate 19 项 = 21。
	if len(state.TaskCompleted) != 21 || state.TaskCompleted[0] != 1346 || state.TaskCompleted[1] != 49634 {
		t.Fatalf("Task_Completed 21 项： %v", state.TaskCompleted)
	}
}

// TestRoleStateJSONRoundTrip JSON 序列化回读（nx_roles.role_state_json 通道）。
func TestRoleStateJSONRoundTrip(t *testing.T) {
	state, err := BuildRoleState(DefaultBirthScene, 2, "icon\\players\\girl01_create.png", 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	raw, err := MarshalRoleState(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(raw, `"curNeiGong":"ng_wd_001"`) {
		t.Fatalf("json 缺内功字段: %s", raw[:120])
	}
	back, err := UnmarshalRoleState(raw)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Sex != 2 || len(back.Skills) != 27 || back.Position != state.Position {
		t.Fatalf("回读不一致： %+v", back)
	}
	empty, err := UnmarshalRoleState("")
	if err != nil || empty.Sex != 0 {
		t.Fatalf("空档兼容： %v %+v", err, empty)
	}
}

// TestInferSexFromPara 性别推断（world 侧 inferSex 同口径）。
func TestInferSexFromPara(t *testing.T) {
	cases := map[string]int{
		"icon\\players\\girl01_create.png": 2,
		"icon\\players\\boy01_create.png":  1,
		"photo,obj\\char\\g_hair\\g_hair1": 2,
		"photo,obj\\char\\b_hair\\b_hair1": 1,
		"":                                 1,
	}
	for para, want := range cases {
		if got := InferSexFromPara(para); got != want {
			t.Fatalf("InferSexFromPara(%q) = %d, want %d", para, got, want)
		}
	}
}
