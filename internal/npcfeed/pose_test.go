package npcfeed

// pose_test.go NPC 姿态/状态测试：出生状态分层、静息态与动作名校验、
// 路点到达动作、物化帧 State/BornAction 同值——并按官服姑苏两包抓包的
// 物化样本对拍（2026-11-25，规则见 pose.go）。

import (
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

func textField(fields map[string][]byte, name string) string {
	return strings.TrimRight(string(fields[name]), "\x00")
}

func TestBornActionLayering(t *testing.T) {
	placement := map[string]string{"BornAction": "ty_nod"}
	template := map[string]string{"BornAction": "ty_talk", "BornActionRec": "ty_think"}
	if got := bornActionOf(placement, template); got != "ty_nod" {
		t.Fatalf("摆放应优先: %q", got)
	}
	// 摆放显式 stand 也压过模板姿态（摆放是更具体的一层）。
	if got := bornActionOf(map[string]string{"BornAction": " stand "}, template); got != "stand" {
		t.Fatalf("摆放显式 stand 应胜出: %q", got)
	}
	if got := bornActionOf(nil, template); got != "ty_talk" {
		t.Fatalf("模板 BornAction 应先于 BornActionRec: %q", got)
	}
	if got := bornActionOf(nil, map[string]string{"BornActionRec": "ty_think"}); got != "ty_think" {
		t.Fatalf("模板 BornActionRec 应回落: %q", got)
	}
	if got := bornActionOf(map[string]string{"BornAction": " "}, nil); got != "" {
		t.Fatalf("全空应返回空: %q", got)
	}
}

// TestRestAction 静息态：姿态动作名原样；缺省 / stand / walk / run / nil /
// 0 / 非动作名（客户端表里的贴图路径脏值）一律 stand。
func TestRestAction(t *testing.T) {
	poses := []string{"ty_sitchair_drinkloop", "sitloop", "ty_sitchiar_eatloop", "h_stand_2", "29h_stand1",
		"offact337", "diedownloop", "ty_sitground_dozeloop", "Wave_2"}
	for _, pose := range poses {
		if got := (Spawn{BornAction: pose}).RestAction(); got != pose {
			t.Errorf("姿态 %q 静息态 = %q", pose, got)
		}
	}
	for _, raw := range []string{"", "  ", "stand", "walk", "run", "Run", "nil", "0",
		`icon\monster\monster105.png`, "ty talk", "a,b"} {
		if got := (Spawn{BornAction: raw}).RestAction(); got != StandAction {
			t.Errorf("出生状态 %q 静息态 = %q，应 stand", raw, got)
		}
	}
	// 分号列表：去空后恰剩一个动作就取它；多候选语义未知，不猜。
	for raw, want := range map[string]string{
		"stand;":                     StandAction,
		";":                          StandAction,
		"diedownloop;":               "diedownloop",
		"close ;":                    "close",
		"stand;sitloop":              StandAction,
		"sitloop;ty_dazuoloop":       StandAction,
		"stand;;diedownloop;sitloop": StandAction,
		"walk;":                      StandAction,
	} {
		if got := (Spawn{BornAction: raw}).RestAction(); got != want {
			t.Errorf("列表型出生状态 %q 静息态 = %q，应 %q", raw, got, want)
		}
	}
	if got := (Spawn{}).materialState(); got != StandAction {
		t.Fatalf("无姿态物化态 = %q", got)
	}
	if got := (Spawn{BornAction: "ty_lie_sleep"}).materialState(); got != "ty_lie_sleep" {
		t.Fatalf("姿态物化态 = %q", got)
	}
	// 调用方标了 State（巡逻采样在走）优先于静息态。
	if got := (Spawn{State: WalkAction, BornAction: "ty_lie_sleep"}).materialState(); got != WalkAction {
		t.Fatalf("标注的物化态应优先: %q", got)
	}
}

// TestMotionState 巡逻状态：行进 walk；停留取路点到达动作（可保持的动作）
// 否则静息态；到达动作是 stand/walk/run/脏值时不算特定动作。
func TestMotionState(t *testing.T) {
	if got := (Motion{Moving: true}).State("stand"); got != WalkAction {
		t.Fatalf("行进态 = %q", got)
	}
	for _, arrival := range []string{"", "stand", "walk", "run", "nil", "0", "a b"} {
		if got := (Motion{Arrival: arrival}).State("stand"); got != "stand" {
			t.Errorf("到达动作 %q 应回落静息态，得 %q", arrival, got)
		}
	}
	if got := (Motion{Arrival: "ty_saodiloop"}).State("stand"); got != "ty_saodiloop" {
		t.Fatalf("扫地到达动作 = %q", got)
	}
	if got := (Motion{Arrival: "interact50"}).State("stand"); got != "interact50" {
		t.Fatalf("交互到达动作 = %q", got)
	}
}

// TestParseWaypointArrivalAndRun 到达动作与判跑列随坐标起始列走：无 id
// 列的行格式（坐标从第 0 列起）两列都前移一位——旧实现判跑固定读第 8
// 列，这种行永远读成 nil。
func TestParseWaypointArrivalAndRun(t *testing.T) {
	lead, ok := parseWaypoint(",10,1,20,-10,5000,ty_saodiloop,nil,walk,nil,1,")
	if !ok || lead.Arrival != "ty_saodiloop" || lead.Run || lead.DwellMs != 5000 {
		t.Fatalf("前导空列行: %+v ok=%v", lead, ok)
	}
	id, ok := parseWaypoint("59,100,2,200,0,60000,stand,nil,run,nil,1,")
	if !ok || id.Arrival != "stand" || !id.Run {
		t.Fatalf("导航 id 行: %+v ok=%v", id, ok)
	}
	bare, ok := parseWaypoint("318.19,3.79,632.01,0,4000,h_stand_2,nil,run,nil,0,")
	if !ok || bare.Arrival != "h_stand_2" || !bare.Run || bare.DwellMs != 4000 {
		t.Fatalf("无 id 列行: %+v ok=%v（判跑/到达动作列错位）", bare, ok)
	}
	short, ok := parseWaypoint(",10,1,20")
	if !ok || short.Arrival != "" || short.Run {
		t.Fatalf("只有坐标的短行应可用且无到达动作: %+v ok=%v", short, ok)
	}
}

// TestPoseOffSwitch NINEYIN_NPC_POSE=off：静息态与物化态一律 stand（真机 A/B
// 开关）；路点到达动作不受影响。
func TestPoseOffSwitch(t *testing.T) {
	t.Setenv("NINEYIN_NPC_POSE", "off")
	spawn := Spawn{ObjectID: ObjectIDAt(2), OwnerID: OwnerIDOf(ObjectIDAt(2)),
		BornAction: "ty_sitchair_drinkloop", Template: map[string]string{"Resource": `npc\funcnpc001`}}
	if got := spawn.RestAction(); got != StandAction {
		t.Fatalf("关停后静息态 = %q", got)
	}
	fields := decode0x28Fields(t, spawn.Frame0x28())
	if state, born := textField(fields, "State"), textField(fields, "BornAction"); state != "stand" || born != "stand" {
		t.Fatalf("关停后物化帧 State/BornAction = %q/%q，应 stand/stand", state, born)
	}
	if got := (Motion{Arrival: "ty_saodiloop"}).State(spawn.RestAction()); got != "ty_saodiloop" {
		t.Fatalf("关停不影响路点到达动作，得 %q", got)
	}
}

// TestFramePoseMatchesOfficialSamples 物化帧 State/BornAction 与官服姑苏
// 物化样本同值（npc.pcapng 2026-11 姑苏场景，152 个 NPC 里 12 个姿态
// NPC 与 2 个普通 NPC；样本值抄自抓包解码）。同一模板在别的图没有
// BornAction（FuncNpc00104 在燕京为空）——取的是 city02_suzhou。
func TestFramePoseMatchesOfficialSamples(t *testing.T) {
	official := map[string]string{
		"WorldNpc01499":      "ty_kneel_entreat",
		"WorldNpc01133":      "ty_kneel_snarl",
		"WorldNpc01113":      "ty_cryout",
		"WorldNpc01108":      "ty_talk",
		"FuncNpc00104":       "ty_nod",
		"WorldNpc01212":      "ty_clap",
		"WorldNpc01206":      "ty_clap",
		"npc_mmzw_other_004": "ty_sitground_lookround",
		"npc_mmzw_other_002": "h_hurtdownloop_1",
		"WorldNpc01064":      "ty_lie_sleep",
		"npc_mmzw_other_003": "ty_sitground_dozeloop",
		"WorldNpc01081":      "ty_shake",
		"Guildfuncnpc400":    "stand",
		"WorldNpc01231":      "stand",
	}
	spawns, _, err := BuildCatalogDir(testRoot(t), "city02_suzhou")
	if err != nil {
		t.Fatalf("目录装载: %v", err)
	}
	seen := map[string]bool{}
	for i := range spawns {
		want, ok := official[spawns[i].ConfigID]
		if !ok || seen[spawns[i].ConfigID] {
			continue
		}
		seen[spawns[i].ConfigID] = true
		fields := decode0x28Fields(t, spawns[i].Frame0x28())
		if state, born := textField(fields, "State"), textField(fields, "BornAction"); state != want || born != want {
			t.Errorf("%s 物化 State/BornAction = %q/%q，官服 %q/%q", spawns[i].ConfigID, state, born, want, want)
		}
		// 0x0D 形态同一口径：头行块与官方追加块里的 State/BornAction。
		head := map[string]string{}
		for _, prop := range spawns[i].OrderedProperties() {
			if prop.Value.Type == clientdata.WireString {
				head[prop.Name] = prop.Value.Text
			}
		}
		tail := map[string]string{}
		for _, prop := range officialProps(spawns[i]) {
			tail[prop.Name] = prop.Value.Text
		}
		for _, block := range []map[string]string{head, tail} {
			if v, ok := block["State"]; ok && v != want {
				t.Errorf("%s 0x0D State = %q，官服 %q", spawns[i].ConfigID, v, want)
			}
			if v, ok := block["BornAction"]; ok && v != want {
				t.Errorf("%s 0x0D BornAction = %q，官服 %q", spawns[i].ConfigID, v, want)
			}
		}
	}
	if len(seen) != len(official) {
		t.Fatalf("官服样本只对上 %d/%d 个（缺 %v）", len(seen), len(official), missing(official, seen))
	}
}

func missing(official map[string]string, seen map[string]bool) []string {
	var out []string
	for id := range official {
		if !seen[id] {
			out = append(out, id)
		}
	}
	return out
}

// TestFramePoseCoverage 全目录姿态覆盖：真档里摆放级姿态 NPC 的物化帧
// State/BornAction 都是其 BornAction（抽样核对，防分层或校验把大量姿态
// 吃成 stand）。
func TestFramePoseCoverage(t *testing.T) {
	spawns, _, err := BuildCatalogDir(testRoot(t), "born03_yanyuzhuang")
	if err != nil {
		t.Fatalf("目录装载: %v", err)
	}
	poses := 0
	for i := range spawns {
		rest := spawns[i].RestAction()
		if rest == StandAction {
			continue
		}
		poses++
		if rest != spawns[i].BornAction {
			t.Errorf("%s 静息态 %q ≠ 出生状态 %q", spawns[i].ConfigID, rest, spawns[i].BornAction)
		}
	}
	// 实测 490（2026-11-25 全量摆放 BornAction 统计）；阈值留余量，只防
	// 分层或校验把大批姿态吃成 stand。
	if poses < 400 {
		t.Fatalf("烟雨庄姿态 NPC 只有 %d 个（实测 490，分层丢了？）", poses)
	}
}
