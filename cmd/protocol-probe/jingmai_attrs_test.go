package main

// 官方式经脉对齐（2026-11-15）单测：阶梯上限、全套补齐、激活条数闸、
// 正逆互斥、官方逐层属性接线、快照向后兼容。数据链（varprop→proppack）
// 装载断言走真实资源（无 NINEYIN_SERVER_ROOT 时跳过）。

import (
	"testing"
	"time"
)

// 阶梯模型（用户拍板）：初始 9；每开 1 穴 +36 封顶目录上限；跷脉 225
// 特例 75/穴；开满即封顶（带脉 3 穴 36/72/180 缺口由全开兜齐）。
func TestJingMaiCapLadder(t *testing.T) {
	cases := []struct {
		maxLevel, opened, total, want int32
	}{
		{216, 0, 8, 9},
		{216, 1, 8, 36},
		{216, 2, 8, 72},
		{216, 3, 8, 108},
		{216, 4, 8, 144},
		{216, 5, 8, 180},
		{216, 6, 8, 216},
		{216, 7, 8, 216}, // 超阶梯封顶
		{216, 8, 8, 216}, // 开满=封顶
		{225, 1, 3, 75},  // 跷脉特例
		{225, 2, 3, 150},
		{225, 3, 3, 225},
		{180, 1, 3, 36}, // 带脉：第 3 穴开满直取 180
		{180, 2, 3, 72},
		{180, 3, 3, 180},
		{3, 0, 0, 3},   // 目录上限 ≤9 直取（stub 目录形态）
		{3, 2, 2, 3},   //
		{-1, 5, 5, 9},  // 无效目录回落 9
	}
	for _, c := range cases {
		if got := jingMaiCapFor(c.maxLevel, c.opened, c.total); got != c.want {
			t.Fatalf("capFor(max=%d, opened=%d, total=%d)=%d, want %d", c.maxLevel, c.opened, c.total, got, c.want)
		}
	}
}

// 全套开通：老档 14 条（F 盘 52795 抓包序）幂等补齐到全套，原行序保留、
// 新条目追加；jm_actLimit 快照缺省回填官方初始 3。
func TestEnsureJingMaiLearnedSupplementsFullSet(t *testing.T) {
	stubEmptyXueWeiCatalog(t)
	// 手工全套目录（starter 表全部命中）。
	full := &jingMaiCatalog{byID: map[string]int{}}
	for _, id := range starterJingMaiIDs {
		full.byID[id] = len(full.order)
		full.order = append(full.order, jingMaiDef{configID: id, itemType: 1003, maxLevel: 216})
	}
	player := newPlayerActor("补齐侠", 0)
	legacy := []string{
		"jm_dantian", "jm_zuyangming", "jm_zushaoyin", "jm_shoutaiyin",
		"jm_shoutaiyang", "jm_zutaiyin", "jm_zushaoyang", "jm_zujueyin",
		"jm_shoushaoyang", "jm_shouyangming", "jm_zutaiyang", "jm_shoushaoyin",
		"jm_yinqiao_wei", "jm_yangqiao_wei",
	}
	player.restoreJingMai(jingMaiSnapshot{Learned: legacy}, time.Now())
	player.ensureJingMaiLearned(full)
	snapshot := player.jingMaiSnapshotOf()
	if len(snapshot.Learned) != len(starterJingMaiIDs) {
		t.Fatalf("learned=%d, want full set %d", len(snapshot.Learned), len(starterJingMaiIDs))
	}
	for i, id := range legacy {
		if snapshot.Learned[i] != id {
			t.Fatalf("legacy order broken at %d: %v", i, snapshot.Learned)
		}
	}
	for _, id := range []string{"jm_shoujueyin", "jm_chong", "jm_dai", "jm_yinwei", "jm_yangwei", "jm_yinqiao", "jm_yangqiao"} {
		if !containsString(snapshot.Learned, id) {
			t.Fatalf("full set missing %s", id)
		}
	}
	// 逆脉/经筋不在全套：未学。
	if containsString(snapshot.Learned, "jm_shoushaoyin_ni") || containsString(snapshot.Learned, "jm_zutaiyang_jin") {
		t.Fatalf("full set must not include _ni/_jin: %v", snapshot.Learned)
	}
	// 条数上限缺省官方初始 3（restore 回填后快照即真值）。
	if snapshot.ActLimit != jingMaiActLimitDefault {
		t.Fatalf("snapshot actLimit=%d, want default %d", snapshot.ActLimit, jingMaiActLimitDefault)
	}
	if player.jingMaiActHeadroom() != int(jingMaiActLimitDefault) {
		t.Fatalf("headroom=%d, want default %d", player.jingMaiActHeadroom(), jingMaiActLimitDefault)
	}
	// 幂等：再跑一遍不重复。
	player.ensureJingMaiLearned(full)
	if again := player.jingMaiSnapshotOf(); len(again.Learned) != len(starterJingMaiIDs) {
		t.Fatalf("re-ensure learned=%d", len(again.Learned))
	}
}

// 激活条数闸（jmActCount 上限，GM set_jingmai_act_count 1..8）：满员
// 静默拒（客户端无错误键），上调后放行；jmActCount 属性随 8 件套真值。
func TestJingMaiActLimitRejectsAndGMAdjusts(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	_ = catalog
	player := newPlayerActor("条数侠", 0)
	seedJingMaiLearned(player, "jm_a", "jm_b")
	conn := &captureMessageConnection{}
	for _, id := range []string{"jm_a", "jm_b"} {
		if handled, err := handleJingMaiActiveCustom(conn, player, activeMsg(jingMaiSubActivate, id), "test"); err != nil || !handled {
			t.Fatalf("activate %s: handled=%t err=%v", id, handled, err)
		}
	}
	if snapshot := player.jingMaiSnapshotOf(); len(snapshot.Activated) != 2 {
		t.Fatalf("activated=%v", snapshot.Activated)
	}
	// stub 目录只有 2 条已学，借 GM 上调条数后继续验证拒绝路径：
	player.setJingMaiActLimit(1)
	if headroom := player.jingMaiActHeadroom(); headroom != 0 {
		t.Fatalf("headroom=%d after limit 1 with 2 active, want 0", headroom)
	}
	// 已满员：重复激活既有的不占新位（already present 路径）；
	// 去活一条后余量恢复。
	if handled, err := handleJingMaiActiveCustom(conn, player, activeMsg(jingMaiSubDeactivate, "jm_b"), "test"); err != nil || !handled {
		t.Fatalf("deactivate: %v", err)
	}
	if headroom := player.jingMaiActHeadroom(); headroom != 0 {
		t.Fatalf("headroom=%d with limit 1 & 1 active, want 0", headroom)
	}
	if applied := player.setJingMaiActLimit(8); applied != 8 {
		t.Fatalf("applied=%d", applied)
	}
	if headroom := player.jingMaiActHeadroom(); headroom != 7 {
		t.Fatalf("headroom=%d, want 7", headroom)
	}
	// jmActCount 属性真值（旧硬编码 1 是数据异常根源）。
	player.mu.Lock()
	props := player.jingMaiPropsLocked()
	player.mu.Unlock()
	if value, ok := findJingMaiProp(props, propJMActCount); !ok || value.I32 != 8 {
		t.Fatalf("jmActCount=%+v ok=%t, want 8", value, ok)
	}
	// 快照往返保真 + 上限钳 1..8。
	if snapshot := player.jingMaiSnapshotOf(); snapshot.ActLimit != 8 {
		t.Fatalf("snapshot actLimit=%d", snapshot.ActLimit)
	}
	if applied := player.setJingMaiActLimit(99); applied != jingMaiActLimitMax {
		t.Fatalf("clamp high=%d", applied)
	}
	if applied := player.setJingMaiActLimit(0); applied != 1 {
		t.Fatalf("clamp low=%d", applied)
	}
}

// 正逆互斥（jingmai_active_exclude.ini：跷↔跷·隐）：激活互斥对象时先摘
// 已在列的互斥经脉（0x12 删行帧 + 换行），不占新条数位。
func TestJingMaiActiveExcludeSwaps(t *testing.T) {
	stubEmptyXueWeiCatalog(t, [2]string{"jm_yinqiao", "jm_yinqiao_wei"}, [2]string{"jm_yinqiao_wei", "jm_yinqiao"})
	player := newPlayerActor("互斥侠", 0)
	seedJingMaiLearned(player, "jm_yinqiao_wei", "jm_yinqiao")
	conn := &captureMessageConnection{}
	if handled, err := handleJingMaiActiveCustom(conn, player, activeMsg(jingMaiSubActivate, "jm_yinqiao_wei"), "test"); err != nil || !handled {
		t.Fatalf("activate wei: %v", err)
	}
	if snapshot := player.jingMaiSnapshotOf(); len(snapshot.Activated) != 1 || snapshot.Activated[0] != "jm_yinqiao_wei" {
		t.Fatalf("activated=%v", snapshot.Activated)
	}
	// 激活互斥对象 jm_yinqiao：先摘 wei（0x12）再加 qiao（0x11）。
	if handled, err := handleJingMaiActiveCustom(conn, player, activeMsg(jingMaiSubActivate, "jm_yinqiao"), "test"); err != nil || !handled {
		t.Fatalf("activate qiao: %v", err)
	}
	snapshot := player.jingMaiSnapshotOf()
	if len(snapshot.Activated) != 1 || snapshot.Activated[0] != "jm_yinqiao" {
		t.Fatalf("after swap activated=%v, want [jm_yinqiao]", snapshot.Activated)
	}
	frames := conn.Frames()
	delFrames := 0
	for _, frame := range frames {
		if frame[0] == 0x12 {
			delFrames++
		}
	}
	if delFrames != 1 {
		t.Fatalf("record-del frames=%d, want 1 (exclusion swap)", delFrames)
	}
}

// 官方逐层属性接线：激活经脉 → meridianBonus 进 effectiveFive/
// derivedResourceAdds/maxParryAddTotal/derivedMinMagicDefAdd；升级/开穴
// 变化后重算；未激活零加成。
func TestJingMaiBonusWiring(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	// 手工给 jm_a 塞官方形态的逐层表（varprop 缺表时 perLevel 为空，
	// 这里直接构造验证接线与累积语义——每层段是累积值直取）。
	for i := range catalog.order {
		if catalog.order[i].configID != "jm_a" {
			continue
		}
		catalog.order[i].perLevel = map[int32]jingMaiAttrSet{
			1: {str: 2, sta: 2, maxHPAdd: 85, maxMPAdd: 20, maxParryAdd: 147, minMagicDefAdd: 5},
			3: {str: 6, sta: 6, maxHPAdd: 300, maxMPAdd: 60, maxParryAdd: 400, minMagicDefAdd: 15},
		}
		catalog.order[i].maxLevel = 3
	}
	player := newPlayerActor("加成侠", 0)
	seedJingMaiLearned(player, "jm_a")
	baseStr, baseSta, _, _, _ := player.progress.effectiveFive()
	baseHP, baseMP := player.progress.derivedResourceAdds()
	baseParry := player.progress.maxParryAddTotal()
	baseMagicDef := player.progress.derivedMinMagicDefAdd()

	// 未激活：零加成。
	player.refreshJingMaiBonus(catalog)
	if str, _, _, _, _ := player.progress.effectiveFive(); str != baseStr {
		t.Fatalf("inactive bonus leaked: str %d -> %d", baseStr, str)
	}

	// 激活 1 级：五维 +2、HP Add +85（属性公式分量 2·2+7·2+85）、
	// MP +20（jm_a 无 ing/spi 分量）、招架 +147、内防 +5。
	player.activateJingMai("jm_a")
	player.mu.Lock()
	player.jingmai.progress["jm_a"] = jingMaiProgressRow{Level: 1}
	player.recomputeJingMaiBonusLocked(catalog)
	str, sta, _, _, _ := player.progress.effectiveFive()
	hpAdd, mpAdd := player.progress.derivedResourceAdds()
	parry := player.progress.maxParryAddTotal()
	magicDef := player.progress.derivedMinMagicDefAdd()
	player.mu.Unlock()
	if str != baseStr+2 || sta != baseSta+2 {
		t.Fatalf("five: str %d->%d sta %d->%d", baseStr, str, baseSta, sta)
	}
	if hpAdd != baseHP+2*2+7*2+85 {
		t.Fatalf("hpAdd %d -> %d (want +%d)", baseHP, hpAdd, 2*2+7*2+85)
	}
	if mpAdd != baseMP+20 {
		t.Fatalf("mpAdd %d -> %d (want +20)", baseMP, mpAdd)
	}
	if parry != baseParry+147 {
		t.Fatalf("parryAdd=%d, want base+%d=%d", parry, 147, baseParry+147)
	}
	if magicDef != baseMagicDef+5 {
		t.Fatalf("minMagicDefAdd=%d, want base+5=%d", magicDef, baseMagicDef+5)
	}

	// 直设 3 级（满层）：取满层累积段。
	player.setJingMaiLevel(catalog, "jm_a", 3)
	player.refreshJingMaiBonus(catalog)
	player.mu.Lock()
	hpAdd, _ = player.progress.derivedResourceAdds()
	player.mu.Unlock()
	if hpAdd != baseHP+6*2+6*7+300 {
		t.Fatalf("maxed hpAdd=%d, want +%d", hpAdd, 6*2+6*7+300)
	}

	// 去活归零。
	player.deactivateJingMai("jm_a")
	player.refreshJingMaiBonus(catalog)
	if hpAdd, _ = player.progress.derivedResourceAdds(); hpAdd != baseHP {
		t.Fatalf("deactivated hpAdd=%d, want base %d", hpAdd, baseHP)
	}
}

// 修炼升级钳当前阶梯上限：已达 cap 的经脉真气不入账（满级截断语义与
// 旧目录满级一致，只是上限换成阶梯值）。
func TestJingMaiCultivateRespectsCapLadder(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	// jm_b maxLevel=225（跷脉形态）——无穴 cap=9。
	player := newPlayerActor("封顶侠", 0)
	anchor := time.Date(2026, 11, 15, 10, 0, 0, 0, time.Local)
	player.restoreJingMai(jingMaiSnapshot{
		Learned:  []string{"jm_b"},
		Cur:      "jm_b",
		Progress: map[string]jingMaiProgressRow{"jm_b": {Level: 9, Fill: 700, Total: 9*750 + 700}},
	}, anchor)
	if frames := player.jingMaiCultivateTick(catalog, anchor.Add(10*time.Minute)); len(frames) != 0 {
		t.Fatalf("frames=%d at cap", len(frames))
	}
	snapshot := player.jingMaiSnapshotOf()
	if got := snapshot.Progress["jm_b"]; got.Level != 9 || got.Fill != 700 {
		t.Fatalf("capped progress=%+v, want level=9 fill=700 (no accrual)", got)
	}
	if snapshot.ZhenQi.Used != 0 {
		t.Fatalf("used=%d at cap (不应耗真气)", snapshot.ZhenQi.Used)
	}
}

// GM open_xuewei all：穴位地图全开 → 阶梯顶格（View44 首建整推 + View45
// MaxLevel 帧语义在真机验证，这里验证状态与上限）。
func TestGMOpenXueWeiAllRaisesCap(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	stubEmptyXueWeiCatalog(t)
	def := catalog.order[1] // jm_b maxLevel=225
	// 注入穴位地图 3 穴 + 穴位目录。
	for i := range catalog.order {
		if catalog.order[i].configID == "jm_b" {
			catalog.order[i].xueWei = []string{"xw_b1", "xw_b2", "xw_b3"}
		}
	}
	player := newPlayerActor("开穴侠", 0)
	seedJingMaiLearned(player, "jm_b")
	if cap := player.jingMaiCapOf(catalog, "jm_b"); cap != jingMaiInitialCap {
		t.Fatalf("initial cap=%d, want %d", cap, jingMaiInitialCap)
	}
	frames, fresh, err := player.gmOpenJingMaiXueWei(catalog, "jm_b", "all")
	if err != nil || len(fresh) != 3 {
		t.Fatalf("open all: fresh=%v err=%v", fresh, err)
	}
	// View44 容器未建（首开）→ 只产 View45 cap 帧（1 帧）。
	if len(frames) != 1 {
		t.Fatalf("frames=%d, want cap update only (container via full replay)", len(frames))
	}
	if cap := player.jingMaiCapOf(catalog, "jm_b"); cap != def.maxLevel {
		t.Fatalf("capped-open cap=%d, want %d", cap, def.maxLevel)
	}
	// 幂等：再开全无新增。
	if _, fresh, err := player.gmOpenJingMaiXueWei(catalog, "jm_b", "all"); err != nil || len(fresh) != 0 {
		t.Fatalf("re-open fresh=%v err=%v", fresh, err)
	}
}

// 官方数据链装载（jingmai_varprop.ini→proppack.ini）：VarPropNo 101001
// = 手少阴 L1 = {sta2,spi2,HP85,MP20,parry147,minMagic5}（参考服
// data/经脉.json 同源逐位一致）。缺真实资源时跳过。
func TestJingMaiVarPropTablesOfficialSample(t *testing.T) {
	loadJingMaiTablesOnce()
	if len(jingMaiVarPropPacks) == 0 || len(jingMaiPropPackRows) == 0 {
		t.Skip("official varprop/proppack tables unavailable (no NINEYIN_SERVER_ROOT resources)")
	}
	set, ok := jingMaiAttrSetForVarProp(101001)
	if !ok {
		t.Fatalf("varprop 101001 missing")
	}
	if set.sta != 2 || set.spi != 2 || set.maxHPAdd != 85 || set.maxMPAdd != 20 || set.maxParryAdd != 147 || set.minMagicDefAdd != 5 {
		t.Fatalf("101001 attrSet=%+v", set)
	}
	// 逐层累积语义抽查：手少阴 L216（127216? no—101216）满层段有值且 ≥ L1。
	full, ok := jingMaiAttrSetForVarProp(101216)
	if !ok || full.maxHPAdd <= set.maxHPAdd {
		t.Fatalf("101216 attrSet=%+v (cumulative must exceed L1)", full)
	}
}

// 开穴规则收敛：已学内功达层即开（activerule 只增不减）。
func TestReEvaluateXueWeiByNeiGongRules(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	// 注入规则：ng_wd_001 20 层开 xw_w1、ng_yh_001 30 层开 xw_w2。
	setXueWeiCatalogForTest(&xueWeiCatalog{
		defs:     map[string]xueWeiDef{"xw_w1": {configID: "xw_w1", staticData: 1}, "xw_w2": {configID: "xw_w2", staticData: 2}},
		byStatic: map[int32]string{1: "xw_w1", 2: "xw_w2"},
		rules: map[string][]xueWeiRule{
			"ng_wd_001": {{level: 20, xueWeiID: "xw_w1"}},
			"ng_yh_001": {{level: 30, xueWeiID: "xw_w2"}},
		},
		excludes: map[string][]string{},
		total:    2,
	})
	player := newPlayerActor("规则侠", 0)
	// newPlayerProgress 默认书：ng_wd_001 30 层、ng_yh_001 30 层 → 两穴齐开。
	frames, changed := player.jingMaiXueWeiReEval(catalog)
	if !changed {
		t.Fatalf("re-eval reported no change with 30-level default books")
	}
	if len(frames) == 0 {
		t.Fatalf("no incremental frames for 2 fresh xuewei")
	}
	player.mu.Lock()
	opened := append([]string(nil), player.jingmai.openedXueWei...)
	player.mu.Unlock()
	if len(opened) != 2 || !containsString(opened, "xw_w1") || !containsString(opened, "xw_w2") {
		t.Fatalf("opened=%v, want both xw_w1/xw_w2", opened)
	}
	// 幂等：再评无新增。
	if _, changed := player.jingMaiXueWeiReEval(catalog); changed {
		t.Fatalf("re-eval changed on second pass")
	}
}
