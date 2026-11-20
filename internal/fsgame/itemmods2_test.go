package fsgame

import (
	"path/filepath"
	"testing"
)

// assemble12 装配十二批七十七模块 + 物品 schema（item.xml 基类并入 item
// 与 Equipment 类——Equipment→item 继承链；player 含 includes 全链）。
func assemble12(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	it := mustRead(t, filepath.Join(td, "struct", "item", "item.xml"))
	eq := mustRead(t, filepath.Join(td, "struct", "item", "equipment.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	for cls, data := range map[string][]byte{
		"player": px, "item": it, "Equipment": eq,
	} {
		if err := k.LoadStruct(cls, data); err != nil {
			t.Fatalf("LoadStruct %s: %v", cls, err)
		}
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{
		RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch,
		RegisterFourthBatch, RegisterFifthBatch, RegisterSixthBatch,
		RegisterSeventhBatch, RegisterEighthBatch, RegisterNinthBatch,
		RegisterTenthBatch, RegisterEleventhBatch, RegisterTwelfthBatch,
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

// newItem 物化物品对象（item 基类 schema）。
func newItem(k *Kernel) PersistID {
	id, err := k.CreateObject("item")
	if err != nil {
		return NullPersistID
	}
	return id
}

// TestTwelfthBatchAssemble 十二期装配序 + 注册面 + schema 抽查。
func TestTwelfthBatchAssemble(t *testing.T) {
	k, mods := assemble12(t)
	want := []string{"ChargeModule", "ItemNotify", "ItemProCompute", "ItemLimited",
		"ItemBindModule", "ItemValidTimeModule", "ItemControlModule",
		"ModifyPackModule", "AddContainerModule"}
	var got []string
	for _, m := range mods {
		for _, w := range want {
			if m.Name() == w {
				got = append(got, w)
			}
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	for _, h := range []string{"ItemValidTimeModule::H_ValidTime",
		"ChargeModule::HB_AsyWait", "ChargeModule::HB_CheckChargeHistory"} {
		found := false
		for _, name := range k.Bus().HeartBeatNames() {
			if name == h {
				found = true
			}
		}
		if !found {
			t.Fatalf("心跳 %s 未注册", h)
		}
	}
	it := newItem(k)
	o := k.GetGameObj(it)
	for _, prop := range []string{"BindType", "BindStatus", "ValidTimeType",
		"LeftTime", "MaxCount", "CantDepot", "ExpiredTime"} {
		if _, ok := o.Class().Prop(prop); !ok {
			t.Fatalf("item 属性 %s 未装载", prop)
		}
	}
}

// TestItemBindAndValidTime 绑定 + 时效：入包即绑（BindType≠0）；相对时效
// 递减归零到期；绝对时效点折算。
func TestItemBindAndValidTime(t *testing.T) {
	k, _ := assemble12(t)
	bm := module2[*ItemBindModule](t, k, "ItemBindModule")
	vm := module2[*ItemValidTimeModule](t, k, "ItemValidTimeModule")

	// 绑定：BindType=1 入 ToolBox → BindStatus=1。
	it := newItem(k)
	_ = k.SetInt(it, "BindType", 1)
	_ = k.SetString(it, "ConfigID", "item_bind_01")
	k.RunEvent("ToolBox", "OnAfterAdd", NullPersistID, NullPersistID, VarList{it})
	if k.GetGameObj(it).GetInt("BindStatus") != 1 || len(bm.Events()) != 1 {
		t.Fatalf("入包即绑异常: %d %v", k.GetGameObj(it).GetInt("BindStatus"), bm.Events())
	}
	// 非绑定物不动。
	it2 := newItem(k)
	k.RunEvent("ToolBox", "OnAfterAdd", NullPersistID, NullPersistID, VarList{it2})
	if k.GetGameObj(it2).GetInt("BindStatus") != 0 || len(bm.Events()) != 1 {
		t.Fatal("非绑定物不应置绑定")
	}

	// 相对时效：ValidTimeType=2 ValidTime=3 → 3 拍到期。
	it3 := newItem(k)
	_ = k.SetInt(it3, "ValidTimeType", 2)
	_ = k.SetInt(it3, "ValidTime", 3)
	_ = k.SetString(it3, "ConfigID", "item_time_02")
	vm.InitLeftTime(k, it3)
	if k.GetGameObj(it3).GetInt("LeftTime") != 3 {
		t.Fatalf("相对时效折算异常: %d", k.GetGameObj(it3).GetInt("LeftTime"))
	}
	for i := 0; i < 2; i++ {
		k.Bus().TickHeartBeats(k, 1)
	}
	if len(vm.Events()) != 0 {
		t.Fatal("未到期不应流水")
	}
	k.Bus().TickHeartBeats(k, 1)
	if k.GetGameObj(it3).GetInt("LeftTime") != 0 || len(vm.Events()) != 1 {
		t.Fatalf("到期异常: left=%d events=%v", k.GetGameObj(it3).GetInt("LeftTime"), vm.Events())
	}
	// 绝对时效点：ExpiredTime「年.月.日.时.分」→ 已过时点即 0。
	it4 := newItem(k)
	_ = k.SetInt(it4, "ValidTimeType", 1)
	_ = k.SetString(it4, "ExpiredTime", "2020.1.1.0.0")
	vm.InitLeftTime(k, it4)
	if got := k.GetGameObj(it4).GetInt("LeftTime"); got > 0 {
		t.Fatalf("已过时点应无剩余: %d", got)
	}
}

// TestItemLimitedAndNotify 限制门 + 入出包通知。
func TestItemLimitedAndNotify(t *testing.T) {
	k, _ := assemble12(t)
	lm := module2[*ItemLimited](t, k, "ItemLimited")
	nm := module2[*ItemNotify](t, k, "ItemNotify")

	it := newItem(k)
	_ = k.SetInt(it, "MaxCount", 5)
	_ = k.SetInt(it, "CantDepot", 1)
	_ = k.SetInt(it, "Task", 1)
	if !lm.CanOwn(k, 4, it) || lm.CanOwn(k, 5, it) {
		t.Fatal("拥有上限门异常")
	}
	if lm.CanDepot(k, it) {
		t.Fatal("CantDepot 应拒入仓")
	}
	if !lm.CanSell(k, it) {
		t.Fatal("CanSell 默认应可贩卖")
	}
	if lm.CanDelete(k, it) {
		t.Fatal("任务道具应拒删除")
	}
	nm.NotifyAdd(NullPersistID, "item_a", 2)
	nm.NotifyRemove(NullPersistID, "item_a", 1)
	if len(nm.Events()) != 2 || !hasStr(nm.Events(), "add:item_a×2") || !hasStr(nm.Events(), "del:item_a×1") {
		t.Fatalf("通知流水异常: %v", nm.Events())
	}
}

// TestItemProComputeAndModifyPack 属性包计算 + 穿卸修正重算联动。
func TestItemProComputeAndModifyPack(t *testing.T) {
	k, _ := assemble12(t)
	pm := module2[*ItemProCompute](t, k, "ItemProCompute")
	mm := module2[*ModifyPackModule](t, k, "ModifyPackModule")
	or := module2[*ObjPropRefreshModule](t, k, "ObjPropRefreshModule")

	eq, err := k.CreateObject("Equipment")
	if err != nil {
		t.Fatal(err)
	}
	_ = k.SetString(eq, "ConfigID", "eq_001")
	_ = k.SetString(eq, "PropPack", "HP,50;MP,20")
	_ = k.SetString(eq, "PropBuilt", "HP,10")
	props := pm.ComputeItemProps(k, eq)
	if props["HP"] != 60 || props["MP"] != 20 {
		t.Fatalf("属性包计算异常: %v", props)
	}

	// 穿上 → 修正重算联动（ObjPropRefresh.Refreshes 递增）。
	p, _ := k.CreateObject("player")
	before := or.Refreshes()
	k.RunEvent("EquipBox", "OnAfterAdd", p, NullPersistID, nil)
	if or.Refreshes() != before+1 || !hasStr(mm.Events(), "equip") {
		t.Fatalf("穿装修正联动异常: %d %v", or.Refreshes(), mm.Events())
	}
	// 耐久归零 → broken 流水（先置非零再归零——同值不触发临界）。
	_ = k.SetInt(eq, "Hardiness", 50)
	_ = k.SetInt(eq, "Hardiness", 0)
	if !hasStr(mm.Events(), "broken") {
		t.Fatalf("耐久归零流水缺失: %v", mm.Events())
	}
}

// TestChargeAndContainerAndControl 充值商城 + 扩容 + 物品管控。
func TestChargeAndContainerAndControl(t *testing.T) {
	k, _ := assemble12(t)
	cm := module2[*ChargeModule](t, k, "ChargeModule")
	am := module2[*AddContainerModule](t, k, "AddContainerModule")
	ic := module2[*ItemControlModule](t, k, "ItemControlModule")

	// 商城购买（scene OnBuyItem2）→ BuyOut 出口。
	bought := ""
	cm.BuyOut = func(player PersistID, itemID string, count int32) { bought = itemID }
	scene, _ := k.CreateObject("scene")
	k.RunEvent("scene", "OnBuyItem2", scene, NullPersistID, VarList{"charge_box_01", int32(1)})
	if bought != "charge_box_01" || !hasStr(cm.Events(), "buy:charge_box_01×1") {
		t.Fatalf("商城购买异常: %q %v", bought, cm.Events())
	}
	// 背包扩容 + 容量满。
	if !am.ExpandBag(k, NullPersistID, 8) || am.Capacity(NullPersistID) != 8 {
		t.Fatal("扩容异常")
	}
	k.RunEvent("AddContainer", "OnNoAdd", NullPersistID, NullPersistID, nil)
	if !hasStr(am.Events(), "full") {
		t.Fatalf("容量满流水缺失: %v", am.Events())
	}
	// 物品管控：ban → 门命中。
	k.RunEvent("scene", "OnPublicMessage", scene, NullPersistID, VarList{"ban:item_evil"})
	if !ic.Banned("item_evil") {
		t.Fatal("管控禁用未生效")
	}
}
