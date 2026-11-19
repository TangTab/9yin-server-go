package fsgame

import (
	"os"
	"path/filepath"
	"testing"
)

// 装配官方登录闭环：logic_class.xml + player struct + 10 模块 + 心跳泵。
func assemble(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	// testdata 根（官方 Res 副本；testdata/struct/player/player.xml 为 schema）。
	td := filepath.Join("..", "..", "testdata")
	lc, err := os.ReadFile(filepath.Join(td, "logic_class.xml"))
	if err != nil {
		t.Fatalf("读 logic_class.xml: %v", err)
	}
	px, err := os.ReadFile(filepath.Join(td, "struct", "player", "player.xml"))
	if err != nil {
		t.Fatalf("读 struct/player/player.xml: %v", err)
	}

	k := NewKernel()
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	if err := k.LoadStruct("player", px); err != nil {
		t.Fatalf("LoadStruct player: %v", err)
	}
	// exp.xml：LevelModule 经验属性面（官方由该模块 OnCreateClass 声明）。
	ex, err := os.ReadFile(filepath.Join(td, "struct", "player", "exp.xml"))
	if err != nil {
		t.Fatalf("读 struct/player/exp.xml: %v", err)
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}

	var r Registry
	if err := RegisterFirstBatch(&r); err != nil {
		t.Fatalf("RegisterFirstBatch: %v", err)
	}
	mods, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, mods
}

// TestPlayerSevenEvents 官方 player 七事件生命周期闭环：
// OnCreateClass → OnRecover → OnEntry → OnReady → OnEntryGame → OnStore → OnDisconnect。
func TestPlayerSevenEvents(t *testing.T) {
	k, _ := assemble(t)
	id, err := k.CreateObject("player")
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	// OnCreateClass 已在 LoadStruct 触发；此处跑剩余六事件（官方由宿主登录链驱动）。
	b := k.Bus()
	for _, ev := range []string{"OnRecover", "OnEntry", "OnReady", "OnEntryGame", "OnStore", "OnDisconnect"} {
		if n := k.RunEvent("player", ev, id, NullPersistID, nil); n == 0 {
			t.Fatalf("事件 %s 无处理器", ev)
		}
	}
	o := k.GetGameObj(id)
	if o == nil {
		t.Fatal("玩家对象丢失")
	}
	if v := o.Get("LeaveSceneTime"); v == nil || v == Var(int32(0)) {
		t.Fatalf("OnDisconnect 未记 LeaveSceneTime: %v", v)
	}
	if o.Get("LoginTime") == nil {
		t.Fatal("OnRecover 未记 LoginTime")
	}
	_ = b
}

// TestPlayerCommands PlayerBase 四命令 + Motion/Container/Request/Capital/建角命令路由。
func TestPlayerCommands(t *testing.T) {
	k, _ := assemble(t)
	id, _ := k.CreateObject("player")
	// PlayerBase 0xC/0x33E/0x35C/0x3F7
	for _, c := range []int32{0x0C, 0x33E, 0x35C, 0x3F7} {
		if !k.DispatchCommand("player", c, id, NullPersistID, VarList{int32(0)}) {
			t.Fatalf("命令 %#x 未命中", c)
		}
	}
	// Motion 0x03/0x0F/0x37A/0x37B
	for _, c := range []int32{0x03, 0x0F, 0x37A, 0x37B} {
		if !k.DispatchCommand("player", c, id, NullPersistID, VarList{int32(1)}) {
			t.Fatalf("Motion 命令 %#x 未命中", c)
		}
	}
	// Container 0x1E..0x29
	for c := int32(0x1E); c <= 0x29; c++ {
		if !k.DispatchCommand("player", c, id, NullPersistID, VarList{c}) {
			t.Fatalf("容器命令 %#x 未命中", c)
		}
	}
	// Request 98..104/0x6D
	for _, c := range []int32{98, 99, 100, 101, 102, 103, 104, 0x6D} {
		if !k.DispatchCommand("player", c, id, NullPersistID, nil) {
			t.Fatalf("Request 命令 %d 未命中", c)
		}
	}
	// Capital 0x2D/0x143
	for _, c := range []int32{0x2D, 0x143} {
		if !k.DispatchCommand("player", c, id, NullPersistID, nil) {
			t.Fatalf("Capital 命令 %#x 未命中", c)
		}
	}
	// CreateRole 0x375 → SaveRoleInfo → SaveOut
	saved := false
	k.SaveOut = func(role PersistID, chunk string, props map[string]Var, recs map[string][]VarList) {
		saved = true
		if chunk != "player" {
			t.Fatalf("chunk 名 %q", chunk)
		}
	}
	if !k.DispatchCommand("player", 0x375, id, NullPersistID, nil) {
		t.Fatal("建角命令 0x375 未命中")
	}
	if !saved {
		t.Fatal("0x375 未触发存档")
	}
}

// TestCreateRoleFlow 官方 OnCreateRole 属性初始化链（role 类 + skill/
// shortcut_rec/normal_attack 记录初值）。
func TestCreateRoleFlow(t *testing.T) {
	k, mods := assemble(t)
	// role 是 player 子类：LoadStruct 走 player schema。
	id, err := k.CreateObject("role")
	if err != nil {
		t.Fatalf("CreateObject role: %v", err)
	}
	n := k.RunEvent("role", "OnCreateRole", id, NullPersistID,
		VarList{"星蓝", int32(1), int32(3), int32(1)})
	if n == 0 {
		t.Fatal("OnCreateRole 无处理器")
	}
	o := k.GetGameObj(id)
	if got := firstNonEmpty(o.GetString("Name"), o.GetString("name")); got != "星蓝" {
		t.Fatalf("名字=%q", got)
	}
	if o.GetInt("Level") != 1 {
		t.Fatalf("Level=%d", o.GetInt("Level"))
	}
	if o.GetInt("Camp") != 1 || o.GetInt("Race") != 3 || o.GetInt("Job") != 1 {
		t.Fatalf("Camp/Race/Job=%d/%d/%d", o.GetInt("Camp"), o.GetInt("Race"), o.GetInt("Job"))
	}
	// 类继承：role 对象的 player 类属性面可用（schema 继承自 player 装载）。
	var cr *CreateRoleModule
	for _, m := range mods {
		if c, ok := m.(*CreateRoleModule); ok {
			cr = c
		}
	}
	if cr == nil || len(cr.Created) != 1 {
		t.Fatal("建角流水缺失")
	}
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// TestLevelCriticalChain 经验→升级临界联动（LevelModule C_OnExpChanged →
// Level 临界 → PlayerBase C_OnPlayerLevelUpChange 官方链路）。
func TestLevelCriticalChain(t *testing.T) {
	k, _ := assemble(t)
	id, _ := k.CreateObject("player")
	_ = k.SetInt(id, "Level", 1)
	_ = k.SetProp(id, "PlayerUpgradeExp", int32(260))
	_ = k.SetProp(id, "PlayerExp", int64(480)) // 480 ≥ 260：升级一次余 220
	o := k.GetGameObj(id)
	if o.GetInt("Level") != 2 {
		t.Fatalf("升级结算后 Level=%d 期望 2", o.GetInt("Level"))
	}
	if exp := varToInt64(o.Get("PlayerExp")); exp != 220 { // 480-260
		t.Fatalf("PlayerExp=%d 期望 220", exp)
	}
}

// TestSceneLifecycle 场景类 OnCreate/OnDestroy + 玩家入场联动 + 心跳。
func TestSceneLifecycle(t *testing.T) {
	k, mods := assemble(t)
	sid, err := k.CreateObject("scene")
	if err != nil {
		t.Fatalf("CreateObject scene: %v", err)
	}
	k.AddScene(SceneInfo{ID: 1, Class: "NormalScene"}, sid)
	if k.GetSceneId() != 1 || k.GetScene() != sid {
		t.Fatal("场景登记面异常")
	}
	pid, _ := k.CreateObject("player")
	// player.OnEntry 同时命中 PlayerBase 与 SceneBase 官方注册（多播）。
	if n := k.RunEvent("player", "OnEntry", pid, sid, nil); n < 2 {
		t.Fatalf("player.OnEntry 处理器 %d 个（官方 PlayerBase+SceneBase ≥2）", n)
	}
	var sb *SceneBaseModule
	for _, m := range mods {
		if s, ok := m.(*SceneBaseModule); ok {
			sb = s
		}
	}
	k.Bus().TickHeartBeats(k, 1)
	if sb == nil || sb.SyncTimeBeats == 0 {
		t.Fatal("HB_SyncCurrentTime 未跑")
	}
	k.RemoveObject(sid) // OnDestroy
}

// TestLocationDoorData LocationModule 装载官方 door_data.xml（真档）。
func TestLocationDoorData(t *testing.T) {
	k, mods := assemble(t)
	res := `F:\Game\Online\9yin\BGserverCJ\Res`
	if _, err := os.Stat(res); err != nil {
		t.Skip("官方 Res 不在本机，跳过真档校验")
	}
	k.ResourcePath = res
	var loc *LocationModule
	for _, m := range mods {
		if l, ok := m.(*LocationModule); ok {
			loc = l
		}
	}
	if err := loc.LoadResource(k); err != nil {
		t.Fatalf("door_data.xml: %v", err)
	}
	d, ok := loc.GetSwitchLocation(1)
	if !ok {
		t.Fatal("门 1 缺失")
	}
	if d.Scene != 2 || d.Name == "" {
		t.Fatalf("门 1 = %+v", d)
	}
}

// TestItemObjectLifecycle item 类 OnCreate + Amount 归零移除（官方
// C_OnAmountChanged 语义）。
func TestItemObjectLifecycle(t *testing.T) {
	k, _ := assemble(t)
	// 官方 item 类 schema（struct/item/item.xml）+ ItemBase 注册面。
	ix, err := os.ReadFile(filepath.Join("..", "..", "testdata", "struct", "item", "item.xml"))
	if err != nil {
		t.Fatalf("读 struct/item/item.xml: %v", err)
	}
	if err := k.LoadStruct("item", ix); err != nil {
		t.Fatalf("LoadStruct item: %v", err)
	}
	id, err := k.CreateObject("item")
	if err != nil {
		t.Fatalf("CreateObject item: %v", err)
	}
	o := k.GetGameObj(id)
	if _, ok := o.Class().Prop("Amount"); !ok {
		t.Fatal("item schema 无 Amount 属性")
	}
	// OnCreate 默认数量 1（官方 @0x7F600）。
	if o.GetInt("Amount") != 1 {
		t.Fatalf("新物品 Amount=%d 期望 1", o.GetInt("Amount"))
	}
	_ = k.SetInt(id, "Amount", 5)
	_ = k.SetInt(id, "Amount", 0)
	if k.GetGameObj(id) != nil {
		t.Fatal("Amount 归零未移除对象（C_OnAmountChanged 语义）")
	}
}

// TestArchiveSaveFace 存档面：save=true 属性/记录进 Archive，save=false 排除。
func TestArchiveSaveFace(t *testing.T) {
	k, _ := assemble(t)
	id, _ := k.CreateObject("player")
	o := k.GetGameObj(id)
	_ = k.SetString(id, "Photo", "gui/x.png") // save=true
	props, _ := o.Archive()
	if _, ok := props["Photo"]; !ok {
		t.Fatal("save 属性未入档")
	}
	if _, ok := props["LoginTime"]; ok {
		t.Fatal("非 save 属性入档")
	}
}

// TestModuleOrder 官方 Init 序（模块表 §8 首批相对序）。
func TestModuleOrder(t *testing.T) {
	var r Registry
	if err := RegisterFirstBatch(&r); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ContainerBaseModule", "ItemBaseModule", "CapitalModule",
		"SceneBaseModule", "LocationModule", "PlayerBaseModule",
		"CreateRoleModule", "MotionModule", "LevelModule", "RequestModule",
	}
	for i, c := range r.creators {
		if c.NameStr != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, c.NameStr, want[i])
		}
	}
}
