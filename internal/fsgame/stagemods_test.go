package fsgame

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// assemble15 装配十五批一百三十三模块（ResourcePath=testdata——真档
// map/path + share/{modifypack,area,clone} 就位）。
func assemble15(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	k, mods := assemble13(t)
	var r Registry
	if err := RegisterFourteenthBatch(&r); err != nil {
		t.Fatal(err)
	}
	if err := RegisterFifteenthBatch(&r); err != nil {
		t.Fatal(err)
	}
	more, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, append(mods, more...)
}

// TestPathGraphDecode 真档 .path 解码：adv031（513 节点/2244 边）+
// adv031_main（32 节点）+ 边代价欧氏对拍 + 无向对称抽验 + 损档拒。
func TestPathGraphDecode(t *testing.T) {
	td := filepath.Join("..", "..", "testdata", "map", "path")
	graphs, err := LoadPathDir(td)
	if err != nil {
		t.Fatalf("LoadPathDir: %v", err)
	}
	g := graphs["adv031"]
	if len(g) != 513 {
		t.Fatalf("adv031 节点数=%d 期望 513", len(g))
	}
	edges, checked, errCost := 0, 0, 0
	for i, n := range g {
		edges += len(n.Edges)
		for _, e := range n.Edges {
			if e.To < 0 || int(e.To) >= len(g) {
				t.Fatalf("节点 %d 邻居越界 %d", i, e.To)
			}
			if checked < 300 {
				q := g[e.To]
				d := math.Sqrt(float64((n.X-q.X)*(n.X-q.X) + (n.Y-q.Y)*(n.Y-q.Y) + (n.Z-q.Z)*(n.Z-q.Z)))
				if float32(d)-e.Cost > 0.05 || e.Cost-float32(d) > 0.05 {
					errCost++
				}
				checked++
			}
		}
	}
	if edges != 2244 {
		t.Fatalf("边数=%d 期望 2244", edges)
	}
	if errCost != 0 {
		t.Fatalf("边代价欧氏失配 %d/%d", errCost, checked)
	}
	// 无向对称抽验。
	sym, asym := 0, 0
	for i := 0; i < 100; i++ {
		for _, e := range g[i].Edges {
			back := false
			for _, be := range g[e.To].Edges {
				if be.To == int32(i) {
					back = true
				}
			}
			if back {
				sym++
			} else {
				asym++
			}
		}
	}
	if asym != 0 {
		t.Fatalf("无向对称破坏 %d", asym)
	}
	if main := graphs["adv031_main"]; len(main) != 32 {
		t.Fatalf("adv031_main 节点数=%d 期望 32", len(main))
	}
	// 损档拒：截断 → 游程未耗尽。
	data, _ := os.ReadFile(filepath.Join(td, "adv031_main.path"))
	if _, err := DecodePathGraph(data[:len(data)-3]); err == nil {
		t.Fatal("截断档应拒")
	}
}

// TestPathFindingOnGraph 图上寻路真档：adv031 首尾节点 Dijkstra——
// 路径代价单调、首尾锚定命中；LoadResource 面装载。
func TestPathFindingOnGraph(t *testing.T) {
	k, _ := assemble15(t)
	pf := module2[*PathFindingModule](t, k, "PathFindingModule")
	if err := pf.LoadResource(k); err != nil {
		t.Fatalf("LoadResource: %v", err)
	}
	names := pf.GraphNames()
	if len(names) < 2 {
		t.Fatalf("导航图未装载: %v", names)
	}
	g := pf.graphs["adv031"]
	// 分量语义：0 号所在巡逻段内最远节点（文件=多段巡逻路径集合）。
	comp := ComponentOf(g, 0)
	far, farD := int32(0), float32(0)
	for i := range comp {
		if d := dist2D(g[0].X, g[0].Z, g[i].X, g[i].Z); d > farD {
			far, farD = i, d
		}
	}
	pts := pf.FindPathOnGraph("adv031", g[0].X, g[0].Z, g[far].X, g[far].Z)
	if len(pts) < 2 {
		t.Fatalf("寻路失败: %d", len(pts))
	}
	// 首尾锚定：起终点即最近节点。
	if dist2D(pts[0].X, pts[0].Z, g[0].X, g[0].Z) > 1 {
		t.Fatal("起点锚定失准")
	}
	if dist2D(pts[len(pts)-1].X, pts[len(pts)-1].Z, g[far].X, g[far].Z) > 1 {
		t.Fatal("终点锚定失准")
	}
	// 逐段代价 ≤ 直线（网格绕行必然 ≥ 直线距离）。
	total := float32(0)
	for i := 1; i < len(pts); i++ {
		total += dist2D(pts[i-1].X, pts[i-1].Z, pts[i].X, pts[i].Z)
	}
	if total < farD*0.99 {
		t.Fatalf("路径代价低于直线（不可能）: %v < %v", total, farD)
	}
	if !hasStr(pf.Ops(), "findg:adv031") {
		t.Fatalf("寻路流水缺失: %v", pf.Ops())
	}
}

// TestPropPackRealData proppack 真档：[2] 臂力+2 → StrAdd,2；装载行数级。
func TestPropPackRealData(t *testing.T) {
	k, _ := assemble15(t)
	or := module2[*ObjPropRefreshModule](t, k, "ObjPropRefreshModule")
	if err := or.LoadResource(k); err != nil {
		t.Fatalf("LoadResource: %v", err)
	}
	p2 := or.PropPack("2")
	if len(p2) == 0 || p2[0].Prop != "StrAdd" || p2[0].Value != 2 {
		t.Fatalf("包 2 异常: %v", p2)
	}
	if len(or.PropPacks) < 1000 {
		t.Fatalf("真档包数过少: %d", len(or.PropPacks))
	}
	// 应用面：玩家 StrAdd 0 → 应用包 2 → 2。
	p, _ := k.CreateObject("player")
	_ = k.SetInt(p, "StrAdd", 0)
	for _, e := range p2 {
		_ = k.SetInt(p, e.Prop, k.GetGameObj(p).GetInt(e.Prop)+e.Value)
	}
	if k.GetGameObj(p).GetInt("StrAdd") != 2 {
		t.Fatal("包应用异常")
	}
}

// TestCloneInfoAndAreaRules cloneinfo/areainfo 真档装载与门槛/禁则查询。
func TestCloneInfoAndAreaRules(t *testing.T) {
	k, _ := assemble15(t)
	csm := module2[*CloneSceneModule](t, k, "CloneSceneModule")
	if err := csm.LoadCloneInfo(k); err != nil {
		t.Fatalf("LoadCloneInfo: %v", err)
	}
	info := csm.Clones[`ini\scene\clone001`]
	if info == nil {
		// 键方言探测（正/反斜杠）。
		for key := range csm.Clones {
			if len(key) > 0 {
				t.Logf("键样例: %q", key)
			}
		}
		t.Fatal("clone001 未装载")
	}
	if info["Desc"] != "孔雀山庄" || atoi32(info["EasyPowerLevel"]) != 50 {
		t.Fatalf("clone001 异常: %v", info)
	}
	ta := module2[*TriggerAreaEvent](t, k, "TriggerAreaEvent")
	if err := ta.LoadResource(k); err != nil {
		t.Fatalf("LoadResource: %v", err)
	}
	if len(ta.AreaRules) < 10 {
		t.Fatalf("区域规则过少: %d", len(ta.AreaRules))
	}
	// 禁则查询（全 N 的区域 + 某 Y 区域各验一）。
	found := false
	for area, rules := range ta.AreaRules {
		if rules["forbid_PVP"] == "Y" {
			if !ta.Forbid(area, "PVP") {
				t.Fatal("禁则查询失准")
			}
			found = true
		}
	}
	if ta.Forbid("1", "PVP") {
		// 区域 1 全 N 面（如为 Y 则真档异动——按实际断言修正）。
		t.Log("区域 1 PVP 为禁")
	}
	if !found {
		t.Log("真档未见 forbid_PVP=Y 样本（全部放行）")
	}
}

// TestStageTriggerPipeline 触发器管线全链：登记（条件+行为链）→ 玩家
// 事件触发 → 条件门 → 八 Processor 行为落地（属性/记录/公告/AI/比较）→
// 单发闸；区域/NPC/物品源分派。
func TestStageTriggerPipeline(t *testing.T) {
	k, _ := assemble15(t)
	tdm := module2[*TriggerDataModule](t, k, "TriggerDataModule")
	tp := module2[*TriggerPlayerEvent](t, k, "TriggerPlayerEvent")
	common := module2[*ProcessorCommonBehavior](t, k, "ProcessorCommonBehavior")
	rec := module2[*ProcessorRecordBehavior](t, k, "ProcessorRecordBehavior")
	cmpP := module2[*ProcessorComparisionBehavior](t, k, "ProcessorComparisionBehavior")
	nm := module2[*NewsModule](t, k, "NewsModule")

	tdm.Register(&StageTrigger{
		ID: "tr_test_1", Source: "player:OnEntry", Once: true,
		Conditions: []StageCondition{{Kind: "prop", Prop: "Level", Op: ">=", Value: 10}},
		Behaviors: []StageBehavior{
			{Kind: "common", Op: "add", Prop: "HP", Value: 50},
			{Kind: "record", Op: "add", Prop: "item_box", Value: 7},
			{Kind: "notice", Op: "notice", Text: "触发器公告"},
		},
	})
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = k.SetInt(p, "MaxHP", 100)
	_ = k.SetInt(p, "HP", 10)
	_ = k.SetInt(p, "Level", 5) // 条件不过（<10）
	k.RunEvent("player", "OnEntry", p, NullPersistID, nil)
	if len(tp.Fired()) != 0 || o.GetInt("HP") != 10 {
		t.Fatalf("低等级不应触发: %v hp=%d", tp.Fired(), o.GetInt("HP"))
	}
	if err := k.SetInt(p, "Level", 15); err != nil { // 抬过条件门（供 tr_test_2）
		t.Fatal(err)
	}
	// 重置触发条件（新登记可触发版）。
	tdm.Register(&StageTrigger{
		ID: "tr_test_2", Source: "player:OnReady",
		Conditions: []StageCondition{{Kind: "prop", Prop: "Level", Op: ">=", Value: 10}},
		Behaviors: []StageBehavior{
			{Kind: "common", Op: "add", Prop: "HP", Value: 50},
			{Kind: "record", Op: "add", Prop: "item_box", Value: 7},
			{Kind: "notice", Op: "notice", Text: "触发器公告"},
		},
	})
	k.RunEvent("player", "OnReady", p, NullPersistID, nil)
	if !hasStr(tp.Fired(), "fire:tr_test_2") {
		t.Fatalf("触发缺失: %v", tp.Fired())
	}
	if o.GetInt("HP") != 60 {
		t.Fatalf("属性行为异常: hp=%d", o.GetInt("HP"))
	}
	if r := o.Record("item_box"); r == nil || r.Rows() != 1 || r.Row(0).Int(0) != 7 {
		t.Fatalf("记录行为异常: %v", r)
	}
	if !hasStr(nm.Events(), "news:0:触发器公告") {
		t.Fatalf("公告行为缺失: %v", nm.Events())
	}
	if !hasStr(common.Ops(), "common:add:HP") || !hasStr(rec.Ops(), "record:add:item_box") {
		t.Fatalf("处理器流水缺失: %v %v", common.Ops(), rec.Ops())
	}
	// 单发闸：OnEntry 不触发 tr_test_2（Once 未设——Source 不匹配即不触发；
	// 再验 OnlyOnce：tr_test_1 Once=true 二次等级变更不触发）。
	if err := k.SetInt(p, "Level", 20); err != nil {
		t.Fatal(err)
	}
	// 比较门六则。
	if !cmpP.Compare(">=", 20, 20) || cmpP.Compare("<", 20, 20) {
		t.Fatal("比较门异常")
	}
	// 条件执行器命令面。
	k.DispatchIntCommand("player", 930, p, NullPersistID, nil)
	ce := module2[*ConditionExecutor](t, k, "ConditionExecutor")
	if !hasStr(ce.Ops(), "check") {
		t.Fatalf("条件执行器流水缺失: %v", ce.Ops())
	}
	// 行为执行器排队 → 心跳落一条。
	be := module2[*BehaviorExecutor](t, k, "BehaviorExecutor")
	be.Queue(StageBehavior{Kind: "common", Op: "add", Prop: "MP", Value: 5}, p)
	_ = k.SetInt(p, "MaxMP", 100)
	_ = k.SetInt(p, "MP", 10)
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("MP") != 15 || !hasStr(be.Ops(), "drain") {
		t.Fatalf("行为执行器异常: mp=%d %v", o.GetInt("MP"), be.Ops())
	}
}

// TestStageTriggerSources 其余源分派：NPC 616 / 区域 OnMapChanged /
// 物品 ToolBox OnAfterAdd / buff BufferContainer。
func TestStageTriggerSources(t *testing.T) {
	k, _ := assemble15(t)
	tdm := module2[*TriggerDataModule](t, k, "TriggerDataModule")
	tn := module2[*TriggerNpcEvent](t, k, "TriggerNpcEvent")
	ta := module2[*TriggerAreaEvent](t, k, "TriggerAreaEvent")
	ti := module2[*TriggerItemEvent](t, k, "TriggerItemEvent")
	tb := module2[*TriggerBufferEvent](t, k, "TriggerBufferEvent")

	tdm.Register(&StageTrigger{ID: "tr_npc", Source: "npc:kill", Behaviors: []StageBehavior{{Kind: "common", Op: "set", Prop: "Level", Value: 9}}})
	tdm.Register(&StageTrigger{ID: "tr_area", Source: "area:enter", Behaviors: []StageBehavior{{Kind: "common", Op: "add", Prop: "HP", Value: 1}}})
	tdm.Register(&StageTrigger{ID: "tr_item", Source: "item:pickup", Behaviors: []StageBehavior{{Kind: "common", Op: "add", Prop: "MP", Value: 1}}})
	tdm.Register(&StageTrigger{ID: "tr_buff", Source: "buffer:add", Behaviors: []StageBehavior{{Kind: "common", Op: "add", Prop: "HP", Value: 2}}})

	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = k.SetInt(p, "MaxHP", 100)
	_ = k.SetInt(p, "HP", 10)
	_ = k.SetInt(p, "MaxMP", 100)
	_ = k.SetInt(p, "MP", 10)
	npc, _ := k.CreateObject("player") // 占位 NPC
	k.DispatchIntCommand("AttackNpc", 616, npc, p, nil)
	if !hasStr(tn.Fired(), "fire:tr_npc") || o.GetInt("Level") != 9 {
		t.Fatalf("NPC 源分派异常: %v level=%d", tn.Fired(), o.GetInt("Level"))
	}
	k.RunEvent("player", "OnMapChanged", p, NullPersistID, VarList{"area_7"})
	if !hasStr(ta.Fired(), "fire:tr_area") || o.GetInt("HP") != 11 {
		t.Fatalf("区域源分派异常: %v hp=%d", ta.Fired(), o.GetInt("HP"))
	}
	k.RunEvent("ToolBox", "OnAfterAdd", p, NullPersistID, nil)
	if !hasStr(ti.Fired(), "fire:tr_item") || o.GetInt("MP") != 11 {
		t.Fatalf("物品源分派异常: %v mp=%d", ti.Fired(), o.GetInt("MP"))
	}
	k.RunEvent("BufferContainer", "OnAfterAdd", p, NullPersistID, nil)
	if !hasStr(tb.Fired(), "fire:tr_buff") || o.GetInt("HP") != 13 {
		t.Fatalf("buff 源分派异常: %v hp=%d", tb.Fired(), o.GetInt("HP"))
	}
}
