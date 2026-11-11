package main

// 宝物洗炼 688 主链测试：
//   - 纯逻辑：0x12 视图寻址清行帧、TempTersureRec 行型（官服 schema #402
//     8 列）、开窗帧（线名=lua on_custom_show_tersure + 字符串功能号）、
//     计费表（[taolu]→[special_cost] 火浣台 / [normal_cost] 文）；
//   - 合成目录：属性洗同池重掷（条数守恒）、武学洗门派/江湖池过滤
//     （CS_jh_* 启发式）、锁定行保留；
//   - 会话流：锁 0 → 属性洗 8（预览+计费）→ 存 9（快照/帧重写）→
//     隐式重锁 → 解锁 1 的帧序列与内存态（nil store 纯内存降级）。

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"
)

func tersureMsg(sub int32, args ...int32) clientCustomMessage {
	values := []clientCustomValue{{Type: 2, Int32: clientCustomTersure}, {Type: 2, Int32: sub}}
	for _, arg := range args {
		values = append(values, clientCustomValue{Type: 2, Int32: arg})
	}
	return clientCustomMessage{Values: values}
}

// framesSince 自某计数起的帧切片（洗炼流断言用）。
func (r *frameRecorder) framesSince(from int) [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if from >= len(r.frames) {
		return nil
	}
	return append([][]byte(nil), r.frames[from:]...)
}

// syntheticWashCatalog 合成宝物掷值目录：模板 7 → 包 70（属性 11..14 +
// 技能 21 门派/22 江湖），limitelist 900/901 单行带宽。
func syntheticWashCatalog() *randomEquipCatalog {
	resetRandomEquipCatalogForTest()
	randomEquipCatalogOnce.Do(func() {})
	c := &randomEquipCatalog{
		itemFields: map[string]randomItemFields{
			"treasure_test": {equipType: "Treasure", colorLevel: 6, level: 64, templateID: 7},
		},
		templates: map[int][]int{7: {70}},
		tstore: map[int]tstorePack{
			70: {num: 99, entries: []tstoreEntry{
				{kind: 1, id: 11}, {kind: 1, id: 12}, {kind: 1, id: 13}, {kind: 1, id: 14},
				{kind: 2, id: 21}, {kind: 2, id: 22},
			}},
		},
		propstore: map[int]propstoreEntry{
			11: {limitNo: 900}, 12: {limitNo: 900}, 13: {limitNo: 900}, 14: {limitNo: 900},
		},
		skstore: map[int]skillstoreEntry{
			21: {subs: [][2]string{{"CS_wd_test", "PropA"}}, limitNo: 901, taolu: "CS_wd_test"},
			22: {subs: [][2]string{{"CS_jh_test", "PropB"}}, limitNo: 901, taolu: "CS_jh_test"},
		},
		limitRows: map[int][][2]int{
			900: {{10, 20}},
			901: {{5, 8}},
		},
		maxalvl: map[int][]int{64: {1, 1, 1, 1, 1, 1, 1}},
	}
	randomEquipCat = c
	return c
}

// syntheticCostTables 注入合成计费表（门派技=普通文价、江湖技=火浣台）。
func syntheticCostTables() {
	tersureCostOnce.Do(func() {})
	tersureCostLoaded = &tersureCostTables{
		taoluNum:    map[string]int{"CS_wd_test": 2224, "CS_jh_test": 2614},
		specialCost: map[int]int32{2224: 10, 2614: 3},
		normalCost:  []int32{10, 10, 10, 10, 10, 100, 100},
	}
}

// resetTersureCostTablesForTest 重置计费表懒加载（真表用例前调）。
func resetTersureCostTablesForTest() {
	tersureCostOnce = sync.Once{}
	tersureCostLoaded = nil
}

func TestServerRecordDelRowViewEncoding(t *testing.T) {
	frame := serverRecordDelRowView(121, 5, recordTempTersure, 2)
	if len(frame) != 16 {
		t.Fatalf("帧长 %d want 16", len(frame))
	}
	if frame[0] != 0x12 || frame[1] != 1 {
		t.Fatalf("op/flag = %#x/%d want 0x12/1", frame[0], frame[1])
	}
	if view := binary.LittleEndian.Uint32(frame[2:]); view != 121 {
		t.Errorf("view=%d want 121", view)
	}
	if obj := binary.LittleEndian.Uint32(frame[6:]); obj != 5 {
		t.Errorf("obj=%d want 5", obj)
	}
	if rec := binary.LittleEndian.Uint16(frame[10:]); rec != recordTempTersure {
		t.Errorf("rec=%d want %d", rec, recordTempTersure)
	}
	if row := binary.LittleEndian.Uint16(frame[12:]); row != 2 {
		t.Errorf("row=%d want 2", row)
	}
	if count := binary.LittleEndian.Uint16(frame[14:]); count != 1 {
		t.Errorf("count=%d want 1", count)
	}
}

func TestTersureTempRowLayout(t *testing.T) {
	prop := tersureTempPropRow(randomPropLine{PropID: 15, Value: 12.5, Flag: 1, Quality: 3, Level: 10}, true)
	if len(prop) != 8 {
		t.Fatalf("属性行列数 %d want 8（#402 schema）", len(prop))
	}
	if got := *prop[0].intValue; got != 1 {
		t.Errorf("col0 type=%d want 1（属性）", got)
	}
	if got := *prop[1].intValue; got != 15 {
		t.Errorf("col1 propID=%d want 15", got)
	}
	if got := *prop[2].doubleValue; got != 12.5 {
		t.Errorf("col2 value=%v want 12.5", got)
	}
	if got := *prop[3].intValue; got != 1 {
		t.Errorf("col3 lock=%d want 1（锁定行前置标记）", got)
	}
	if got := *prop[4].intValue; got != 1 || *prop[5].intValue != 3 || *prop[6].intValue != 10 {
		t.Errorf("col4-7 flag/quality/level/tail = %d/%d/%d/%d want 1/3/10/0",
			*prop[4].intValue, *prop[5].intValue, *prop[6].intValue, *prop[7].intValue)
	}

	skill := tersureTempSkillRow(randomSkillLine{StoreID: 22, Value: 6})
	if got := *skill[0].intValue; got != 0 {
		t.Errorf("技能行 col0 type=%d want 0", got)
	}
	if got := *skill[1].intValue; got != 22 || *skill[3].intValue != 0 {
		t.Errorf("col1 store/col3 sub = %d/%d want 22/0", got, *skill[3].intValue)
	}
}

func TestTersureWashOpenFrame(t *testing.T) {
	for form, want := range map[string]int32{"new": tersureFormResetAddID, "old": tersureFormResetID} {
		frame, err := tersureWashOpenFrame(form)
		if err != nil {
			t.Fatalf("form=%s: %v", form, err)
		}
		// 0x27 现代壳 | u16 计数=2 | tag2 msgId=517 | tag2 subType=功能号
		//（真机 GM 探针实证的数字寻址方言）。
		if frame[0] != 0x27 {
			t.Fatalf("form=%s op=%#x want 0x27", form, frame[0])
		}
		if count := binary.LittleEndian.Uint16(frame[1:]); count != 2 {
			t.Fatalf("form=%s 值计数=%d want 2", form, count)
		}
		if frame[3] != 2 {
			t.Fatalf("form=%s msgId 线型 tag=%d want 2（int32）", form, frame[3])
		}
		if got := int32(binary.LittleEndian.Uint32(frame[4:])); got != 517 {
			t.Errorf("form=%s msgId=%d want 517", form, got)
		}
		if got := int32(binary.LittleEndian.Uint32(frame[9:])); got != want {
			t.Errorf("form=%s subType=%d want %d", form, got, want)
		}
	}
	if _, err := tersureWashOpenFrame("bogus"); err == nil {
		t.Error("非法 form 应报错")
	}
}

func TestTersureSkillWashCost(t *testing.T) {
	syntheticCostTables()
	// 江湖技在 [taolu]/[special_cost] → 火浣台 3 个。
	if huantai, wen, _ := tersureSkillWashCost("CS_jh_test"); huantai != 3 || wen != 0 {
		t.Errorf("CS_jh_test cost = %d/%d want 3/0", huantai, wen)
	}
	// 不在表 → [normal_cost] 首档 10 文。
	if huantai, wen, _ := tersureSkillWashCost("CS_unknown"); huantai != 0 || wen != 10 {
		t.Errorf("CS_unknown cost = %d/%d want 0/10", huantai, wen)
	}
	if got := tersureNormalTierSilver(9); got != 100 {
		t.Errorf("越界档 = %d want 末档 100", got)
	}
}

func TestRollTreasurePropsForWashKeepsCount(t *testing.T) {
	c := syntheticWashCatalog()
	rnd := rand.New(rand.NewSource(7))
	rolled, err := c.rollTreasurePropsForWash("treasure_test", 3, rnd)
	if err != nil {
		t.Fatalf("roll: %v", err)
	}
	if len(rolled) != 3 {
		t.Fatalf("重掷 %d 条 want 3", len(rolled))
	}
	seen := map[int]bool{}
	for _, line := range rolled {
		if line.PropID < 11 || line.PropID > 14 {
			t.Errorf("propID %d 越出合成池 11..14", line.PropID)
		}
		if seen[line.PropID] {
			t.Errorf("propID %d 重复（不放回语义）", line.PropID)
		}
		seen[line.PropID] = true
		if line.Value < 10 || line.Value > 20 {
			t.Errorf("value %v 越出 limitelist 900 带宽 [10,20]", line.Value)
		}
	}
	if _, err := c.rollTreasurePropsForWash("treasure_missing", 1, rnd); err == nil {
		t.Error("未知物品应报错")
	}
}

func TestRollTreasureSkillForWashPools(t *testing.T) {
	c := syntheticWashCatalog()
	for _, skillType := range []int{0, 2} { // 门派（老/新界面同值域）
		line, err := c.rollTreasureSkillForWash("treasure_test", skillType, rand.New(rand.NewSource(1)))
		if err != nil {
			t.Fatalf("门派 type=%d: %v", skillType, err)
		}
		if line.StoreID != 21 {
			t.Errorf("门派 type=%d 掷出 store=%d want 21（CS_wd_*）", skillType, line.StoreID)
		}
	}
	for _, skillType := range []int{1, 3} { // 江湖
		line, err := c.rollTreasureSkillForWash("treasure_test", skillType, rand.New(rand.NewSource(1)))
		if err != nil {
			t.Fatalf("江湖 type=%d: %v", skillType, err)
		}
		if line.StoreID != 22 {
			t.Errorf("江湖 type=%d 掷出 store=%d want 22（CS_jh_*）", skillType, line.StoreID)
		}
	}
	if _, err := c.rollTreasureSkillForWash("treasure_missing", 0, rand.New(rand.NewSource(1))); err == nil {
		t.Error("未知物品应报错")
	}
}

// TestTersureWashFlow 会话流（nil store 纯内存）：锁 → 属性洗 → 存 → 隐式
// 重锁 → 解锁。断言帧序列（LockStatus 属性帧 / 0x12 清 / 0x11 加 / 结果
// 通知）与玩家内存态。
func TestTersureWashFlow(t *testing.T) {
	syntheticWashCatalog()
	syntheticCostTables()

	player := &playerActor{}
	player.mu.Lock()
	player.bag = []bagRow{{slot: 3, configID: "treasure_test", itemType: 141, amount: 1, viewID: 2, equipType: "Treasure"}}
	player.silver = 1000
	player.mu.Unlock()

	recorder := &frameRecorder{}
	h := newTestItemSession(recorder, player, nil)
	const view, slot = uint16(121), int32(3)

	// 放入锁定（sub 0）。
	if handled, err := handleTersureCustom(h, tersureMsg(0, int32(view), slot)); !handled || err != nil {
		t.Fatalf("lock: handled=%v err=%v", handled, err)
	}
	player.mu.Lock()
	locked := player.tersure.active && player.tersure.configID == "treasure_test"
	player.mu.Unlock()
	if !locked {
		t.Fatal("锁后会话未绑定 treasure_test")
	}
	if last := recorder.last(); last == nil || last[0] != 0x10 {
		t.Fatalf("锁帧 op=%v want 0x10（LockStatus 属性帧）", last)
	}

	// 属性洗（sub 8）：预览帧 + 结果通知 sub 2。
	before := recorder.count()
	if handled, err := handleTersureCustom(h, tersureMsg(8, int32(view), slot, -1)); !handled || err != nil {
		t.Fatalf("wash: handled=%v err=%v", handled, err)
	}
	player.mu.Lock()
	pending := player.tersure.pendingProp
	player.mu.Unlock()
	if len(pending) == 0 {
		t.Fatal("属性洗后无 pending 候选")
	}
	frames := recorder.framesSince(before)
	var hasTempAdd, hasNotify bool
	for _, frame := range frames {
		switch frame[0] {
		case 0x11:
			if binary.LittleEndian.Uint16(frame[10:]) == recordTempTersure {
				hasTempAdd = true
			}
		case 0x1E:
			if bytes.Contains(frame, []byte("on_custom_show_tersuer_result")) {
				hasNotify = true
			}
		}
	}
	if !hasTempAdd || !hasNotify {
		t.Errorf("洗帧缺件 tempAdd=%v notify=%v", hasTempAdd, hasNotify)
	}
	// 计费：10 文。
	if got := player.silverBalance(); got != 1000-tersurePropWashCostSilver {
		t.Errorf("属性洗后碎银 %d want %d", got, 1000-tersurePropWashCostSilver)
	}

	// 存（sub 9）：RandomPropRec 重写 + 清预览 + 通知 sub 3 + 会话快照更新。
	before = recorder.count()
	if handled, err := handleTersureCustom(h, tersureMsg(9, int32(view), slot)); !handled || err != nil {
		t.Fatalf("save: handled=%v err=%v", handled, err)
	}
	player.mu.Lock()
	snapshot := player.tersure.snapshot
	pendingCleared := player.tersure.pendingProp == nil
	player.mu.Unlock()
	if snapshot == nil || !pendingCleared {
		t.Fatalf("存后快照=%v pending清空=%v", snapshot != nil, pendingCleared)
	}
	if len(snapshot.PropRows) != len(pending) {
		t.Errorf("存后快照行数 %d want %d（=pending 全量）", len(snapshot.PropRows), len(pending))
	}
	frames = recorder.framesSince(before)
	var delCount, addRandom int
	for _, frame := range frames {
		switch frame[0] {
		case 0x12:
			delCount++
		case 0x11:
			if binary.LittleEndian.Uint16(frame[10:]) == recordRandomProp {
				addRandom++
			}
		}
	}
	if delCount == 0 || addRandom == 0 {
		t.Errorf("存帧缺件 del=%d randomAdd=%d", delCount, addRandom)
	}

	// 解锁（sub 1）：清预览行 + LockStatus=0 + 会话结束。
	before = recorder.count()
	if handled, err := handleTersureCustom(h, tersureMsg(1, int32(view), slot)); !handled || err != nil {
		t.Fatalf("unlock: handled=%v err=%v", handled, err)
	}
	player.mu.Lock()
	stillActive := player.tersure.active
	player.mu.Unlock()
	if stillActive {
		t.Error("解锁后会话应结束")
	}
	for _, frame := range recorder.framesSince(before) {
		if frame[0] == 0x12 && binary.LittleEndian.Uint16(frame[10:]) == recordTempTersure {
			t.Error("解锁前预览行未清空")
		}
	}

	// 未锁会话直接洗：静默拒绝（无帧）。
	before = recorder.count()
	handleTersureCustom(h, tersureMsg(8, int32(view), slot, -1))
	if recorder.count() != before {
		t.Error("无会话洗炼不应发帧")
	}
}

// TestTersureWashSkillFlowBillsHuantai 武学洗计费走火浣台（江湖技
// [special_cost]=3 个 gn_Procon_001）。
func TestTersureWashSkillFlowBillsHuantai(t *testing.T) {
	syntheticWashCatalog()
	syntheticCostTables()

	player := &playerActor{}
	player.mu.Lock()
	player.bag = []bagRow{
		{slot: 3, configID: "treasure_test", itemType: 141, amount: 1, viewID: 2, equipType: "Treasure"},
		{slot: 7, configID: tersureHuantaiItem, itemType: 1, amount: 5, viewID: 1},
	}
	player.mu.Unlock()

	recorder := &frameRecorder{}
	h := newTestItemSession(recorder, player, nil)
	const view, slot = uint16(121), int32(3)

	handleTersureCustom(h, tersureMsg(0, int32(view), slot))
	// 江湖向（新界面 sub 12, skill_type 3）→ 必掷 CS_jh_test（store 22）→
	// 火浣台 3 个。
	if handled, err := handleTersureCustom(h, tersureMsg(12, int32(view), slot, 3)); !handled || err != nil {
		t.Fatalf("skill wash: handled=%v err=%v", handled, err)
	}
	if got := player.countBagConfig(tersureHuantaiItem); got != 2 {
		t.Errorf("洗后火浣台 %d want 2（扣 3）", got)
	}
	player.mu.Lock()
	pending := player.tersure.pendingSkill
	player.mu.Unlock()
	if pending == nil || pending.StoreID != 22 {
		t.Fatalf("pendingSkill=%v want store 22", pending)
	}
	// 存（sub 13）：主武学行（Unlock==0 首行）替换 + 通知 sub 9。
	// 首掷合成包 num=99 全取 → 原快照技能行 [21,22]，替换首行后 = [22,22]。
	handleTersureCustom(h, tersureMsg(13, int32(view), slot))
	player.mu.Lock()
	snapshot := player.tersure.snapshot
	player.mu.Unlock()
	if snapshot == nil || len(snapshot.SkillRows) != 2 || snapshot.SkillRows[0].StoreID != 22 {
		t.Fatalf("存后技能行 = %+v want [22,22]（首行=主武学被替换）", snapshot.SkillRows)
	}
}

// TestTersureWashInsufficientFunds 余额不足：洗静默拒绝（无预览帧/无扣费）。
func TestTersureWashInsufficientFunds(t *testing.T) {
	syntheticWashCatalog()
	syntheticCostTables()

	player := &playerActor{}
	player.mu.Lock()
	player.bag = []bagRow{{slot: 3, configID: "treasure_test", itemType: 141, amount: 1, viewID: 2, equipType: "Treasure"}}
	player.silver = 5 // 属性洗要 10 文
	player.mu.Unlock()

	recorder := &frameRecorder{}
	h := newTestItemSession(recorder, player, nil)
	handleTersureCustom(h, tersureMsg(0, 121, 3))

	before := recorder.count()
	handleTersureCustom(h, tersureMsg(8, 121, 3, -1))
	if recorder.count() != before {
		t.Error("余额不足不应发帧")
	}
	if got := player.silverBalance(); got != 5 {
		t.Errorf("碎银被动了: %d want 5", got)
	}
}

// TestTersureWashRealTables 真表（NINEYIN_SERVER_ROOT 门控）：黄帝金简宝物
// treasure_31111 的洗炼池与计费——技能包 2014（stores 38..42 全 CS_jh_tlq
// 江湖）→ 江湖洗可行/门派洗空池；计费 CS_jh_tlq=2014 不在 [special_cost]
// → 10 文；CS_jh_cqgf=2614 → 10 火浣台对照。
func TestTersureWashRealTables(t *testing.T) {
	c := randomCatalogForTest(t)
	resetTersureCostTablesForTest()

	// 技能池：门派空、江湖 5 条（38..42）。
	if pool, err := c.treasureWashPool("treasure_31111", 2); err != nil || len(pool) != 5 {
		t.Fatalf("skill pool = %d err=%v want 5（pack 2014 stores 38..42）", len(pool), err)
	}
	rnd := rand.New(rand.NewSource(20261109))
	line, err := c.rollTreasureSkillForWash("treasure_31111", 3, rnd) // 新界面·江湖
	if err != nil {
		t.Fatalf("江湖洗: %v", err)
	}
	if line.StoreID < 38 || line.StoreID > 42 {
		t.Errorf("store=%d 越出 38..42", line.StoreID)
	}
	if _, err := c.rollTreasureSkillForWash("treasure_31111", 2, rnd); err == nil {
		t.Error("门派洗应空池报错（pack 2014 全 CS_jh_*）")
	}

	// 计费：CS_jh_tlq=2014 ∉ special_cost → 10 文；CS_jh_cqgf=2614 → 10 火浣台。
	if huantai, wen, billed := tersureSkillWashCost(c.skillStoreTaoLu(line.StoreID)); billed != "CS_jh_tlq" || huantai != 0 || wen != 10 {
		t.Errorf("CS_jh_tlq 计费 = %d/%d (%s) want 0/10 文", huantai, wen, billed)
	}
	if huantai, wen, _ := tersureSkillWashCost("CS_jh_cqgf"); huantai != 10 || wen != 0 {
		t.Errorf("CS_jh_cqgf 计费 = %d/%d want 10 火浣台/0", huantai, wen)
	}

	// 属性洗：3 条同池重掷、值域为正。
	rolled, err := c.rollTreasurePropsForWash("treasure_31111", 3, rnd)
	if err != nil {
		t.Fatalf("prop wash: %v", err)
	}
	if len(rolled) != 3 {
		t.Fatalf("prop wash = %d 条 want 3", len(rolled))
	}
	for _, prop := range rolled {
		if prop.Value <= 0 {
			t.Errorf("prop %d value %v 非正", prop.PropID, prop.Value)
		}
	}
}

// ---- 娃娃淬炼族（sub 14/15 锁 + 21 强化 + 20 替换）----

// syntheticWawaCatalog 合成娃娃目录（NewTreasure：newtemplate 7 → 基础
// + 附加两条管线，与 rollWawa 表族对齐）。
func syntheticWawaCatalog() *randomEquipCatalog {
	c := syntheticWashCatalog()
	c.itemFields["wawa_test"] = randomItemFields{
		equipType: "NewTreasure", colorLevel: 6, level: 64, templateID: 7,
	}
	c.newtpl = map[int]int{7: 1}
	c.baseRul = map[int][]rollRange{1: {{lo: 0, hi: 9999, id: 101}}}
	c.baseLvl = map[int]map[int]int{6: {64*10000 + 101: 50}}
	c.boxWawa = map[int]int{6: 2}
	c.addRul = map[string][]rollRange{
		"Prop":   {{lo: 0, hi: 4999, id: 201}, {lo: 5000, hi: 9999, id: 202}},
		"Color":  {{lo: 0, hi: 9999, id: 3}},
	}
	c.addLvl = map[int]map[int][2]int{3: {1*10000 + 201: {10, 20}, 1*10000 + 202: {15, 25}}}
	return c
}

func TestTersureWawaLockAndIntensify(t *testing.T) {
	syntheticWawaCatalog()
	syntheticCostTables()

	player := &playerActor{}
	player.mu.Lock()
	player.bag = []bagRow{{slot: 6, configID: "wawa_test", itemType: 193, amount: 1, viewID: 2, equipType: "NewTreasure"}}
	player.silver = 1000
	player.mu.Unlock()
	recorder := &frameRecorder{}
	h := newTestItemSession(recorder, player, nil)
	const view, slot = uint16(121), int32(6)

	// sub 14 娃娃锁（复用 0/1 链）。
	if handled, err := handleTersureCustom(h, tersureMsg(14, int32(view), slot)); !handled || err != nil {
		t.Fatalf("wawa lock: handled=%v err=%v", handled, err)
	}
	player.mu.Lock()
	locked := player.tersure.active && player.tersure.kind == randomKindWawa
	player.mu.Unlock()
	if !locked {
		t.Fatal("娃娃锁未绑定 kind=wawa 会话")
	}

	// 先属性洗一轮（sub 8）落 pending，随后存（sub 9）使快照就绪——
	// intensify 直接在快照上重掷。
	if handled, err := handleTersureCustom(h, tersureMsg(8, int32(view), slot, -1)); !handled || err != nil {
		t.Fatalf("wawa prop wash: %v", err)
	}
	if handled, err := handleTersureCustom(h, tersureMsg(9, int32(view), slot)); !handled || err != nil {
		t.Fatalf("wawa prop save: %v", err)
	}

	// sub 21 强化（select=-1 全部未锁行）→ 预览帧 + 结果 sub 13。
	before := recorder.count()
	if handled, err := handleTersureCustom(h, tersureMsg(21, int32(view), slot, -1)); !handled || err != nil {
		t.Fatalf("wawa intensify: %v", err)
	}
	player.mu.Lock()
	pending := player.tersure.pendingProp
	player.mu.Unlock()
	if len(pending) == 0 {
		t.Fatal("强化后无 pending 候选")
	}
	frames := recorder.framesSince(before)
	var hasNotify13 bool
	for _, frame := range frames {
		if frame[0] == 0x1E && bytes.Contains(frame, []byte("on_custom_show_tersuer_result")) {
			// sub 13 int 值（0x0D）在帧内。
			if bytes.Contains(frame, []byte{13, 0, 0, 0}) {
				hasNotify13 = true
			}
		}
	}
	if !hasNotify13 {
		t.Error("强化结果未回执 sub 13")
	}

	// sub 20 替换两行（0,1）→ 结果 sub 11。
	before = recorder.count()
	if handled, err := handleTersureCustom(h, tersureMsg(20, int32(view), slot, 0, 1)); !handled || err != nil {
		t.Fatalf("wawa replace: %v", err)
	}
	frames = recorder.framesSince(before)
	var hasNotify11 bool
	for _, frame := range frames {
		if frame[0] == 0x1E && bytes.Contains(frame, []byte("on_custom_show_tersuer_result")) &&
			bytes.Contains(frame, []byte{11, 0, 0, 0}) {
			hasNotify11 = true
		}
	}
	if !hasNotify11 {
		t.Error("替换结果未回执 sub 11")
	}

	// 宝物走强化被拒（kind 门）。
	player.mu.Lock()
	player.tersure.kind = randomKindTreaure
	player.mu.Unlock()
	if handled, _ := handleTersureCustom(h, tersureMsg(21, int32(view), slot, -1)); !handled {
		t.Fatal("handled=false")
	}

	// sub 15 娃娃解锁。
	if handled, err := handleTersureCustom(h, tersureMsg(15, int32(view), slot)); !handled || err != nil {
		t.Fatalf("wawa unlock: %v", err)
	}
	player.mu.Lock()
	still := player.tersure.active
	player.mu.Unlock()
	if still {
		t.Fatal("娃娃解锁后会话仍在")
	}
}

func TestWawaIntensifyRows(t *testing.T) {
	rows := []randomPropLine{{PropID: 1}, {PropID: 2}, {PropID: 3}}
	locked := map[int]bool{1: true}
	// select=-1：未锁行 0/2。
	if got := wawaIntensifyRows(locked, rows, -1); len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("all-unlocked=%v", got)
	}
	// 单行 select=1 被锁 → 空。
	if got := wawaIntensifyRows(locked, rows, 1); got != nil {
		t.Fatalf("locked single=%v", got)
	}
	// 单行 select=0。
	if got := wawaIntensifyRows(locked, rows, 0); len(got) != 1 || got[0] != 0 {
		t.Fatalf("single=%v", got)
	}
	// 越界。
	if got := wawaIntensifyRows(locked, rows, 9); got != nil {
		t.Fatalf("out-of-range=%v", got)
	}
}
