package fsgame

import (
	"math/rand"
	"path/filepath"
	"testing"
)

// lastTraces 测试观测面（assemble5 装配 k.Trace）。
var lastTraces []string

// fixedRand 恒 0.05 随机源（直落 0.1 概率门必过；组落权重取首组）。
type fixedRand struct{}

func (fixedRand) Int63() int64 { return int64(0.05 * float64(1<<63)) }
func (fixedRand) Seed(int64)   {}

// assemble5 装配五批二十九模块 + item/Equipment schema。
func assemble5(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	ix := mustRead(t, filepath.Join(td, "struct", "item", "item.xml"))
	eq := mustRead(t, filepath.Join(td, "struct", "item", "equipment.xml"))
	nb := mustRead(t, filepath.Join(td, "struct", "container", "npcdropbox.xml"))

	k := NewKernel()
	k.ResourcePath = td
	lastTraces = lastTraces[:0]
	k.Trace = func(msg string) { lastTraces = append(lastTraces, msg) }
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	if err := k.LoadStruct("player", px); err != nil {
		t.Fatalf("LoadStruct player: %v", err)
	}
	if err := k.LoadStruct("item", ix); err != nil {
		t.Fatalf("LoadStruct item: %v", err)
	}
	if err := k.LoadStruct("Equipment", eq); err != nil {
		t.Fatalf("LoadStruct Equipment: %v", err)
	}
	if err := k.LoadStruct("NpcDropBox", nb); err != nil {
		t.Fatalf("LoadStruct NpcDropBox: %v", err)
	}
	_ = lastTraces
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{
		RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch,
		RegisterFourthBatch, RegisterFifthBatch,
	} {
		if err := reg(&r); err != nil {
			t.Fatal(err)
		}
	}
	mods, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, mods
}

// TestFifthBatchAssemble 五期装配序 + 六掉落表/强化限制真档装载。
func TestFifthBatchAssemble(t *testing.T) {
	k, mods := assemble5(t)
	want := []string{"EquipmentModule", "Strengthen", "ToolBoxModule", "PickUpModule", "DropModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "EquipmentModule", "Strengthen", "ToolBoxModule", "PickUpModule", "DropModule":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	dm := module2[*DropModule](t, k, "DropModule")
	if len(dm.DropItem) == 0 || len(dm.DropTeam) == 0 || len(dm.WorldSimple) == 0 ||
		len(dm.WorldComplex) == 0 || len(dm.SpecSimple) == 0 || len(dm.SpecComplex) == 0 {
		t.Fatalf("掉落六表装载异常: %d/%d/%d/%d/%d/%d",
			len(dm.DropItem), len(dm.DropTeam), len(dm.WorldSimple),
			len(dm.WorldComplex), len(dm.SpecSimple), len(dm.SpecComplex))
	}
	// 真档抽查：WDrop_0001 直落行集（同包多行）/ W_LZ_001 组落行。
	if rows := dm.WorldSimple["WDrop_0001"]; len(rows) == 0 || rows[0]["ItemID"] == "" {
		t.Fatalf("World_Drop_Simple 真档异常: %v", dm.WorldSimple["WDrop_0001"])
	}
	if w := dm.WorldComplex["W_LZ_001"]; w == nil || w["FstTeamID"] == "" {
		t.Fatalf("World_Drop_Complex 真档异常: %v", dm.WorldComplex["W_LZ_001"])
	}
	st := module2[*Strengthen](t, k, "Strengthen")
	if st.EnhanceLimit != 15 {
		t.Fatalf("enhance_limit StartLevel=%d 期望 15", st.EnhanceLimit)
	}
}

// TestLootLoop 掉落→拾取→入背包闭环：固定随机源掷取 → NPC 死亡箱物化 →
// 归属门 → 拾取出包（PickOut→ToolBox.AddItem）→ 箱空销毁。
func TestLootLoop(t *testing.T) {
	k, _ := assemble5(t)
	dm := module2[*DropModule](t, k, "DropModule")
	pm := module2[*PickUpModule](t, k, "PickUpModule")
	tb := module2[*ToolBoxModule](t, k, "ToolBoxModule")

	p, _ := k.CreateObject("player")
	npc, _ := k.CreateObject("player") // 占位 AttackNpc（记录 schema 需 AttackNpc 类，Go 面以 player 承载）
	k.RunEvent("player", "OnRecover", p, NullPersistID, nil)

	// 固定小值随机源：直落必过 0.1 概率门（fixedRand 恒 0.05）。
	dm.rnd = rand.New(fixedRand{})
	entries := dm.RollDrop("WDrop_0001", true)
	if len(entries) == 0 || entries[0].ItemID == "" {
		t.Fatalf("直落掷取异常: %v", entries)
	}
	// 组落链：W_LZ_001 → T_LZ_001 → LZ_001 物品组。
	complexEntries := dm.RollDrop("W_LZ_001", true)
	if len(complexEntries) == 0 {
		t.Fatalf("组落掷取空: %v（真档链 W_LZ_001→Drop_Team→Drop_Item）", complexEntries)
	}
	for _, e := range complexEntries {
		if e.ItemID == "" || e.Count <= 0 {
			t.Fatalf("组落产物异常: %+v", e)
		}
	}

	// NPC 死亡 → 掉落箱物化 + 注入。
	k.RunEvent("CommonNpc", "OnDestroy", npc, NullPersistID, nil)
	box := pm.DropBoxOf(npc)
	if !box.Valid() {
		t.Fatal("掉落箱未物化")
	}
	pm.DropInto(k, npc, entries)
	// 拾取出口 → 背包。
	pm.PickOut = func(pl PersistID, itemID string, count int32) {
		tb.AddItem(k, pl, itemID, count)
	}
	k.DispatchCommand("player", 380, p, NullPersistID, VarList{npc, "x"})
	if len(pm.Picked()) != 1 {
		t.Fatalf("拾取流水 %v", pm.Picked())
	}
	gotItem := pm.Picked()[0]
	if got := tb.CountOf(p, gotItem); got != 1 {
		t.Fatalf("拾取入包 %s=%d 期望 1", gotItem, got)
	}
	// 直落包多行（fixedRand 全过）——循环拾取至箱空销毁。
	for i := 0; pm.DropBoxOf(npc).Valid() && i < 64; i++ {
		k.DispatchCommand("player", 380, p, NullPersistID, VarList{npc, "x"})
	}
	if pm.DropBoxOf(npc).Valid() {
		t.Fatal("箱空未销毁")
	}
	// 归属门：open_player_rec 非空且不含他人 → 拒。
	npc2, _ := k.CreateObject("player")
	k.RunEvent("CommonNpc", "OnDestroy", npc2, NullPersistID, nil)
	pm.DropInto(k, npc2, []DropItemEntry{{"S_Drug_003C", 2}})
	other, _ := k.CreateObject("player")
	// AttackNpc 的 open_player_rec 在 npc2（player 占位类无该记录 → 通配开放），
	// 直接以归属记录路径验证：给 npc2 挂归属表需 AttackNpc schema——此处验证
	// 通配开放面（他人可拾）。
	k.DispatchCommand("player", 380, other, NullPersistID, VarList{npc2, "S_Drug_003C"})
	if got := tb.CountOf(other, "S_Drug_003C"); got != 2 {
		t.Fatalf("通配拾取=%d 期望 2", got)
	}
	// 用道具（cust 31）消耗。
	k.DispatchCommand("player", 31, other, NullPersistID, VarList{"S_Drug_003C"})
	if got := tb.CountOf(other, "S_Drug_003C"); got != 1 {
		t.Fatalf("用道具后=%d 期望 1", got)
	}
}

// TestEquipAndStrengthen 穿戴→强化闭环：装备物化（EquipType/Level/StarLevel）
// → 穿戴（栏位表/换武器临界）→ 卸下 → 强化台（NPC 登记→选装备等级门→
// 执行 StarLevel+1 → 临界联动）。
func TestEquipAndStrengthen(t *testing.T) {
	k, _ := assemble5(t)
	em := module2[*EquipmentModule](t, k, "EquipmentModule")
	st := module2[*Strengthen](t, k, "Strengthen")

	p, _ := k.CreateObject("player")
	_ = k.SetInt(p, "Level", 30)

	// 武器物化（Equipment 类 schema：EquipType/StarLevel/Level）。
	w, err := k.CreateObject("Equipment")
	if err != nil {
		t.Fatalf("CreateObject Equipment: %v", err)
	}
	wo := k.GetGameObj(w)
	_ = wo.Set("EquipType", "Weapon")
	_ = wo.Set("Level", int32(20))
	_ = wo.Set("StarLevel", int32(0))

	// 穿戴：栏位表 + Weapon 临界链。
	if !em.Equip(k, p, w) {
		t.Fatal("穿戴失败")
	}
	if em.EquippedAt(p, "Weapon") != w {
		t.Fatal("栏位表未登记")
	}
	// Weapon object 属性在真档 dump schema 缺档（官方运行期有）——以栏位表为准。
	// 同栏换装：旧装卸下。
	w2, _ := k.CreateObject("Equipment")
	_ = k.GetGameObj(w2).Set("EquipType", "Weapon")
	_ = k.GetGameObj(w2).Set("Level", int32(20))
	if !em.Equip(k, p, w2) {
		t.Fatal("换装失败")
	}
	if em.EquippedAt(p, "Weapon") != w2 {
		t.Fatal("换装后栏位未更新")
	}
	// 卸下。
	if !em.Unequip(k, p, w2) {
		t.Fatal("卸下失败")
	}
	if em.EquippedAt(p, "Weapon").Valid() {
		t.Fatal("卸下后栏位未清")
	}

	// 强化台：NPC 选中（self=NPC、from=玩家）。
	npc, _ := k.CreateObject("player")
	k.RunEvent("CommonNpc", "OnSelect", npc, p, VarList{"npc_smith"})
	if st.BenchOf(p) == nil {
		t.Fatal("强化台未登记")
	}
	// 选装备（等级门：enhance_limit StartLevel=15，装备 Level 20 过）。
	k.DispatchCommand("player", 83, p, NullPersistID, VarList{w})
	if st.BenchOf(p).Equip != w {
		t.Fatal("选装备未上台")
	}
	// 执行 → StarLevel+1。
	k.DispatchCommand("player", 84, p, NullPersistID, VarList{})
	if got := k.GetGameObj(w).GetInt("StarLevel"); got != 1 {
		t.Fatalf("StarLevel=%d 期望 1", got)
	}
	if bench := st.BenchOf(p); bench.Ticket != 100 || bench.Blackgold != 1 {
		t.Fatalf("消耗记账 %+v", bench)
	}
	// 等级门拒：低级装备。
	low, _ := k.CreateObject("Equipment")
	_ = k.GetGameObj(low).Set("EquipType", "Weapon")
	_ = k.GetGameObj(low).Set("Level", int32(10))
	k.DispatchCommand("player", 83, p, NullPersistID, VarList{low})
	if st.BenchOf(p).Equip == low {
		t.Fatal("低级装备过等级门")
	}
	if !hasTpl(st.SysMsgs(), "sys_item_0070") {
		t.Fatal("等级门未通知 sys_item_0070")
	}
	// 锁定拒（sys_depot_0016）。
	_ = k.GetGameObj(low).Set("LockStatus", int32(1))
	_ = k.GetGameObj(low).Set("Level", int32(20))
	k.DispatchCommand("player", 83, p, NullPersistID, VarList{low})
	if !hasTpl(st.SysMsgs(), "sys_depot_0016") {
		t.Fatal("锁定装备未拒 sys_depot_0016")
	}
	// Lua 面 + 关台。
	st.NxStrengthen(k, w, 5)
	if k.GetGameObj(w).GetInt("StarLevel") != 5 {
		t.Fatal("nx_strengthen 未生效")
	}
	k.DispatchCommand("player", 82, p, NullPersistID, VarList{})
	if st.BenchOf(p) != nil {
		t.Fatal("关台未清")
	}
}
