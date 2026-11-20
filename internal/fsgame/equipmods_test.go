package fsgame

import (
	"path/filepath"
	"testing"
)

// assemble13 装配十三批九十六模块（十二期 schema 面 + ToolItem/Enchase 档）。
func assemble13(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	k, mods := assemble12(t)
	ti := mustRead(t, filepath.Join("..", "..", "testdata", "struct", "item", "tool_item.xml"))
	ec := mustRead(t, filepath.Join("..", "..", "testdata", "struct", "item", "enchase.xml"))
	for cls, data := range map[string][]byte{"ToolItem": ti, "Enchase": ec} {
		if err := k.LoadStruct(cls, data); err != nil {
			t.Fatalf("LoadStruct %s: %v", cls, err)
		}
	}
	var r Registry
	if err := RegisterThirteenthBatch(&r); err != nil {
		t.Fatal(err)
	}
	more, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, append(mods, more...)
}

// TestThirteenthBatchAssemble 十三期装配序 + 注册面抽查。
func TestThirteenthBatchAssemble(t *testing.T) {
	_, mods2 := assemble13(t)
	want := []string{"CInscriptionModule", "UpgradeRuleModule", "FashionModule",
		"ItemLaveModule", "ToolItemFunc", "ItemTreasure", "ItemAppendModule",
		"InheritEnhance", "SlotItem", "EnchaseCompound", "CompoundItem",
		"DropCapitalItemModule", "ToolItemModule", "ArmorChangeModule",
		"TransformJobModule", "DepotModule", "ShortCutModule", "FightBag", "Enchase"}
	var got []string
	for _, m := range mods2 {
		for _, w := range want {
			if m.Name() == w {
				got = append(got, w)
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("装配缺模块: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
}

// TestFashionAndArmor 时装/装甲穿卸：FashionState 联动 + ArmorBox 流水。
func TestFashionAndArmor(t *testing.T) {
	k, _ := assemble13(t)
	fm := module2[*FashionModule](t, k, "FashionModule")
	am := module2[*ArmorChangeModule](t, k, "ArmorChangeModule")

	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	fashion, err := k.CreateObject("Equipment")
	if err != nil {
		t.Fatal(err)
	}
	_ = k.SetString(fashion, "ConfigID", "fashion_01")
	k.RunEvent("FashionBox", "OnAfterAdd", p, NullPersistID, VarList{fashion})
	if o.GetInt("FashionState") != 1 || !hasStr(fm.Events(), "wear:fashion_01") {
		t.Fatalf("时装穿戴异常: %d %v", o.GetInt("FashionState"), fm.Events())
	}
	k.RunEvent("FashionBox", "OnRemove", p, NullPersistID, nil)
	if o.GetInt("FashionState") != 0 || !hasStr(fm.Events(), "strip") {
		t.Fatalf("时装脱下异常: %d %v", o.GetInt("FashionState"), fm.Events())
	}
	k.RunEvent("ArmorBox", "OnAfterAdd", p, NullPersistID, nil)
	k.RunEvent("EquipBox", "OnRemove", p, NullPersistID, nil)
	if !hasStr(am.Events(), "armor-on") || !hasStr(am.Events(), "equip-off") {
		t.Fatalf("装甲流水缺失: %v", am.Events())
	}
}

// TestLaveAndTreasureAndCompound 分解/宝箱/碎片合成：ResolvePack 产出 +
// WealthPack 开箱 + FragmentComposePack 合成。
func TestLaveAndTreasureAndCompound(t *testing.T) {
	k, _ := assemble13(t)
	lm := module2[*ItemLaveModule](t, k, "ItemLaveModule")
	tr := module2[*ItemTreasure](t, k, "ItemTreasure")
	ci := module2[*CompoundItem](t, k, "CompoundItem")

	p, _ := k.CreateObject("player")
	// 分解：ResolvePack=120 → LaveOut。
	eq, _ := k.CreateObject("Equipment")
	_ = k.SetString(eq, "ConfigID", "eq_lave")
	_ = k.SetInt(eq, "ResolvePack", 120)
	got := int32(0)
	lm.LaveOut = func(player PersistID, itemID string, value int32) { got = value }
	if !lm.LaveItem(k, p, eq) || got != 120 {
		t.Fatalf("分解异常: %d", got)
	}
	// 宝箱：WealthPack=7（int 包编号——产物经配置面传入）。
	box, _ := k.CreateObject("ToolItem")
	_ = k.SetString(box, "ConfigID", "box_01")
	_ = k.SetInt(box, "WealthPack", 7)
	giftID, giftN := "", int32(0)
	tr.TreasureOut = func(player PersistID, itemID string, count int32) { giftID, giftN = itemID, count }
	if !tr.OpenTreasure(k, p, box, "item_gift", 2) || giftID != "item_gift" || giftN != 2 {
		t.Fatalf("开箱异常: %s×%d", giftID, giftN)
	}
	// 碎片合成：FragmentComposePack=9（int 包编号），8 片拒 / 10 片过。
	frag, _ := k.CreateObject("ToolItem")
	_ = k.SetString(frag, "ConfigID", "frag_sword")
	_ = k.SetInt(frag, "FragmentComposePack", 9)
	if ci.Compound(k, p, frag, 8, "item_sword", 10) {
		t.Fatal("碎片不足应拒")
	}
	prodID := ""
	ci.CompoundOut = func(player PersistID, itemID string, count int32) { prodID = itemID }
	if !ci.Compound(k, p, frag, 10, "item_sword", 10) || prodID != "item_sword" {
		t.Fatalf("合成异常: %s", prodID)
	}
}

// TestToolItemAndDropMoney 使用物品链（FuncPack + TriggerTaskID 联动）+ 掉钱拾取。
func TestToolItemAndDropMoney(t *testing.T) {
	k, _ := assemble13(t)
	tm := module2[*ToolItemModule](t, k, "ToolItemModule")
	dm := module2[*DropCapitalItemModule](t, k, "DropCapitalItemModule")

	p, _ := k.CreateObject("player")
	tool, err := k.CreateObject("ToolItem")
	if err != nil {
		t.Fatal(err)
	}
	_ = k.SetString(tool, "ConfigID", "tool_hp")
	_ = k.SetInt(tool, "FuncPack", 5)
	effect := int32(0)
	tm.EffectOut = func(player PersistID, itemID string, funcPack int32) { effect = funcPack }
	if !tm.UseItem(k, p, tool) || effect != 5 {
		t.Fatalf("使用物品异常: %d", effect)
	}
	// 命令面：ToolItem cmd 10（args 物品，from 玩家）。
	k.DispatchIntCommand("ToolItem", 10, tool, p, VarList{tool})
	if !hasStr(tm.Events(), "use:tool_hp pack=5") {
		t.Fatalf("命令使用流水缺失: %v", tm.Events())
	}
	// 掉钱：DropMoneyItem Amount=888 → cmd 851 拾取。
	drop, _ := k.CreateObject("DropMoneyItem")
	_ = k.SetInt(drop, "Amount", 888)
	money := int32(0)
	dm.MoneyOut = func(player PersistID, amount int32) { money = amount }
	k.DispatchIntCommand("player", 851, p, NullPersistID, VarList{drop})
	if money != 888 || !hasStr(dm.Events(), "pick:888") {
		t.Fatalf("掉钱拾取异常: %d %v", money, dm.Events())
	}
}

// TestTransJobAndDepot 转职 + 仓库：Job 变更流水 + 存取闭环（CantDepot 拒）。
func TestTransJobAndDepot(t *testing.T) {
	k, _ := assemble13(t)
	tj := module2[*TransformJobModule](t, k, "TransformJobModule")
	dp := module2[*DepotModule](t, k, "DepotModule")

	p, _ := k.CreateObject("player")
	if !tj.NxTransJob(k, p, 3) {
		t.Fatal("转职失败")
	}
	if k.GetGameObj(p).GetInt("Job") != 3 || !hasStr(tj.Events(), "trans:0→3") {
		t.Fatal("转职未落 Job")
	}
	if tj.NxTransJob(k, p, 3) {
		t.Fatal("同职应拒")
	}
	// 仓库：普通物存入 / CantDepot 物拒 / 取出存量扣减。
	item := newItem(k)
	_ = k.SetString(item, "ConfigID", "item_st")
	_ = k.SetInt(item, "Amount", 5)
	if !dp.Depository(k, p, item) || dp.Stock(p, "item_st") != 5 {
		t.Fatal("存仓异常")
	}
	banned := newItem(k)
	_ = k.SetInt(banned, "CantDepot", 1)
	if dp.Depository(k, p, banned) {
		t.Fatal("CantDepot 应拒")
	}
	if !dp.Withdraw(k, p, "item_st", 3) || dp.Stock(p, "item_st") != 2 {
		t.Fatal("取仓异常")
	}
	if dp.Withdraw(k, p, "item_st", 3) {
		t.Fatal("超存量应拒")
	}
}

// TestShortCutAndCrafts 快捷栏闭环 + 铭文/镶嵌/继承/宝石合成 API。
func TestShortCutAndCrafts(t *testing.T) {
	k, _ := assemble13(t)
	sm := module2[*ShortCutModule](t, k, "ShortCutModule")
	ci := module2[*CInscriptionModule](t, k, "CInscriptionModule")
	en := module2[*Enchase](t, k, "Enchase")
	ih := module2[*InheritEnhance](t, k, "InheritEnhance")
	ec := module2[*EnchaseCompound](t, k, "EnchaseCompound")
	ur := module2[*UpgradeRuleModule](t, k, "UpgradeRuleModule")

	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	// 绑定（cust 199）→ 锁定栏（类型>0）不可解 → 解普通栏。
	k.DispatchCommand("player", 199, p, NullPersistID, VarList{int32(3), "skill_01", "", int32(0)})
	k.DispatchCommand("player", 199, p, NullPersistID, VarList{int32(4), "skill_02", "", int32(1)})
	if o.Record("shortcut_rec").Rows() != 2 {
		t.Fatalf("快捷栏行数异常: %d", o.Record("shortcut_rec").Rows())
	}
	if sm.UnbindShortcut(k, p, 4) {
		t.Fatal("锁定栏不可解")
	}
	if !sm.UnbindShortcut(k, p, 3) {
		t.Fatal("普通栏应可解")
	}
	// 交换。
	k.DispatchCommand("player", 199, p, NullPersistID, VarList{int32(1), "skill_a", "", int32(0)})
	k.DispatchCommand("player", 201, p, NullPersistID, VarList{int32(1), int32(4)})
	if !hasStr(sm.Events(), "swap:1↔4") {
		t.Fatalf("交换流水缺失: %v", sm.Events())
	}
	// 铭文/镶嵌：PropPack 追加（宝石/铭文属性串由配置面传入——struct 档
	// 无 PropPack 列）。
	eq, _ := k.CreateObject("Equipment")
	_ = k.SetString(eq, "ConfigID", "eq_x")
	_ = k.SetString(eq, "PropPack", "HP,10")
	gem, _ := k.CreateObject("Enchase")
	_ = k.SetString(gem, "ConfigID", "gem_fire")
	_ = k.SetInt(gem, "JewelLevel", 2)
	if !en.EnchaseItem(k, eq, gem, "StiffAdd,3") {
		t.Fatal("镶嵌失败")
	}
	if got := o2str(k, eq, "PropPack"); got != "HP,10;StiffAdd,3" {
		t.Fatalf("镶嵌属性串异常: %s", got)
	}
	ins := newItem(k)
	if !ci.Inscribe(k, eq, ins, "MP,5") {
		t.Fatal("铭刻失败")
	}
	if got := o2str(k, eq, "PropPack"); got != "HP,10;StiffAdd,3;MP,5" {
		t.Fatalf("铭刻属性串异常: %s", got)
	}
	// 继承强化：PropBuilt 迁移。
	eq2, _ := k.CreateObject("Equipment")
	_ = k.SetString(eq, "PropBuilt", "HP,20")
	_ = k.SetString(eq2, "ConfigID", "eq_y")
	if !ih.Inherit(k, eq, eq2) || o2str(k, eq2, "PropBuilt") != "HP,20" || o2str(k, eq, "PropBuilt") != "" {
		t.Fatal("继承强化异常")
	}
	// 宝石合成：3 合 1（Enchase 类——CantCompound/JewelLevel 列所在）。
	gem2, _ := k.CreateObject("Enchase")
	_ = k.SetInt(gem2, "CantCompound", 0)
	next := ""
	ec.CompoundOut = func(player PersistID, itemID string, count int32) { next = itemID }
	if !ec.CompoundGem(k, p, gem2, "gem_fire_lv3", 3) || next != "gem_fire_lv3" {
		t.Fatal("宝石合成异常")
	}
	// 升级规则：千分比判定。
	if !ur.TryUpgrade(k, eq, 500) || ur.TryUpgrade(k, eq, 1500) {
		t.Fatal("千分比判定异常")
	}
}

// o2str 属性读串（测试面）。
func o2str(k *Kernel, id PersistID, prop string) string {
	if o := k.GetGameObj(id); o != nil {
		return o.GetString(prop)
	}
	return ""
}
