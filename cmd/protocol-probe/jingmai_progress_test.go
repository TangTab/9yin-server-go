package main

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// 经脉（msg150/161）：Java JingMaiCustomHandler/JingMaiMessages 全链对齐
// （已学模型 + 首修自动激活 + starter 种子）。目录从 stub INI 装载（无
// NINEYIN_SERVER_ROOT 依赖）；结算速度 43 真气/分钟、750/周天；每日
// 真气硬闸默认关闭不限量（GM set_zhenqi_cap 开闸才按阈值截断）。

func stubJingMaiCatalog(t *testing.T) *jingMaiCatalog {
	t.Helper()
	dir := t.TempDir()
	jingmaiINI := "[jm_a]\nItemType=1003\nStaticData=10\n\n[jm_b]\nItemType=1003\nStaticData=11\n"
	staticINI := "[10]\nMinVarPropNo=1\nMaxVarPropNo=3\n\n[11]\nMinVarPropNo=100\nMaxVarPropNo=324\n"
	if err := os.WriteFile(filepath.Join(dir, "jingmai.ini"), []byte(jingmaiINI), 0o644); err != nil {
		t.Fatalf("write jingmai.ini: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "jingmai_static.ini"), []byte(staticINI), 0o644); err != nil {
		t.Fatalf("write jingmai_static.ini: %v", err)
	}
	originalJingMai, originalStatic := defaultJingMaiINIPath, defaultJingMaiStaticINIPath
	defaultJingMaiINIPath = filepath.Join(dir, "jingmai.ini")
	defaultJingMaiStaticINIPath = filepath.Join(dir, "jingmai_static.ini")
	t.Cleanup(func() {
		defaultJingMaiINIPath, defaultJingMaiStaticINIPath = originalJingMai, originalStatic
	})
	catalog, err := loadJingMaiCatalog(defaultJingMaiINIPath, defaultJingMaiStaticINIPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	return catalog
}

// seedJingMaiLearned 直填已学授予序（等价快照回放，重置结算锚点）。
func seedJingMaiLearned(player *playerActor, ids ...string) {
	player.restoreJingMai(jingMaiSnapshot{Learned: ids}, time.Now())
}

func cultivateMsg(sub int32, id string) clientCustomMessage {
	values := []clientCustomValue{
		{Type: 2, Int32: clientCustomJingMaiCultivate},
		{Type: 2, Int32: sub},
	}
	if id != "" || sub == jingMaiSubSetCultivation {
		values = append(values, clientCustomValue{Type: 6, Text: id})
	}
	return clientCustomMessage{Values: values}
}

func activeMsg(sub int32, id string) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomJingMaiActive},
		{Type: 2, Int32: sub},
		{Type: 6, Text: id},
	}}
}

func TestLoadJingMaiCatalogRangeWidths(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	if len(catalog.order) != 2 {
		t.Fatalf("defs=%d", len(catalog.order))
	}
	def, ok := catalog.def("jm_a")
	if !ok || def.maxLevel != 3 || def.staticData != 10 || def.itemType != 1003 {
		t.Fatalf("jm_a def=%+v", def)
	}
	def, ok = catalog.def("jm_b")
	if !ok || def.maxLevel != 225 {
		t.Fatalf("jm_b def=%+v (want MaxVarPropNo-MinVarPropNo+1=225)", def)
	}
}

func TestJingMaiCultivateSetsCurLastAutoActivatesAndWritesPropsFrame(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	player := newPlayerActor("经脉侠", 0)
	seedJingMaiLearned(player, "jm_a", "jm_b")
	conn := &captureMessageConnection{}

	// 首次开修（Java setCultivating）：自动补 active_jingmai_rec 行 + 面板帧。
	if handled, err := handleJingMaiCultivateCustom(conn, player, catalog, cultivateMsg(jingMaiSubSetCultivation, "jm_a"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 2 {
		t.Fatalf("frames=%d, want record add + props", len(frames))
	}
	if frames[0][0] != 0x11 || !strings.Contains(string(frames[0]), "jm_a") {
		t.Fatalf("auto active record=%x", frames[0])
	}
	if frames[1][0] != 0x10 {
		t.Fatalf("props frame=%#x", frames[1][0])
	}
	// 换修：已有激活行 → 仅面板帧。
	if handled, err := handleJingMaiCultivateCustom(conn, player, catalog, cultivateMsg(jingMaiSubSetCultivation, "jm_b"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	snapshot := player.jingMaiSnapshotOf()
	// 显示契约（客户端 get_jingmai_item 用 LastJingMai 定位 View45 行）：
	// 修炼中 last 与 cur 同值，首修/换修后 tips 四件套立即有数据。
	if snapshot.Cur != "jm_b" || snapshot.Last != "jm_b" {
		t.Fatalf("cur=%q last=%q", snapshot.Cur, snapshot.Last)
	}
	// 停修：空 id 清修炼位。
	if handled, err := handleJingMaiCultivateCustom(conn, player, catalog, cultivateMsg(jingMaiSubSetCultivation, ""), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if snapshot = player.jingMaiSnapshotOf(); snapshot.Cur != "" || snapshot.Last != "jm_b" {
		t.Fatalf("cleared cur=%q last=%q", snapshot.Cur, snapshot.Last)
	}
	// 2（首修）+1（换修）+1（停修）=4；除首帧记录行外均为 0x10。
	frames = conn.Frames()
	if len(frames) != 4 {
		t.Fatalf("frames=%d, want 4", len(frames))
	}
	for _, frame := range frames[1:] {
		if frame[0] != 0x10 {
			t.Fatalf("props frame=%#x", frame[0])
		}
	}
}

func TestJingMaiCultivateRejectsUnknownUnlearnedAndUnhandledSubs(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	player := newPlayerActor("经脉侠", 0)
	conn := &captureMessageConnection{}

	if handled, err := handleJingMaiCultivateCustom(conn, player, catalog, cultivateMsg(jingMaiSubSetCultivation, "jm_unknown"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 目录已知但未学：Java「未学经脉静默拒」。
	if handled, err := handleJingMaiCultivateCustom(conn, player, catalog, cultivateMsg(jingMaiSubSetCultivation, "jm_a"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if handled, err := handleJingMaiCultivateCustom(conn, player, catalog, cultivateMsg(21, "jm_a"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if snapshot := player.jingMaiSnapshotOf(); snapshot.Cur != "" {
		t.Fatalf("cur moved on reject: %q", snapshot.Cur)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("rejects emitted %d frames", len(frames))
	}
}

func TestJingMaiActiveAddsAndRemovesRecordRows(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	_ = catalog
	player := newPlayerActor("经脉侠", 0)
	conn := &captureMessageConnection{}

	if handled, err := handleJingMaiActiveCustom(conn, player, activeMsg(jingMaiSubActivate, "jm_a"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 2 {
		t.Fatalf("frames=%d, want record add + props", len(frames))
	}
	addFrame := frames[0]
	if addFrame[0] != 0x11 || binary.LittleEndian.Uint16(addFrame[10:]) != recordActiveJingMai {
		t.Fatalf("record add=%x (want idx %d active_jingmai_rec)", addFrame, recordActiveJingMai)
	}
	if !strings.Contains(string(addFrame), "jm_a") {
		t.Fatalf("record cell missing jm_a: %x", addFrame)
	}

	// 重复激活：不重复加行，仅面板帧。
	if handled, err := handleJingMaiActiveCustom(conn, player, activeMsg(jingMaiSubActivate, "jm_a"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}

	// 去活：0x12 清 0 基行 + 面板帧。
	if handled, err := handleJingMaiActiveCustom(conn, player, activeMsg(jingMaiSubDeactivate, "jm_a"), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()
	delFrame := frames[len(frames)-2]
	if delFrame[0] != 0x12 || binary.LittleEndian.Uint16(delFrame[12:]) != 0 {
		t.Fatalf("record del=%x (want row 0)", delFrame)
	}
	if snapshot := player.jingMaiSnapshotOf(); len(snapshot.Activated) != 0 {
		t.Fatalf("activated=%v after deactivate", snapshot.Activated)
	}
}

func TestJingMaiCultivateTickAccruesAndLevelsUp(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	player := newPlayerActor("经脉侠", 0)
	anchor := time.Date(2026, 11, 7, 10, 0, 0, 0, time.Local)
	player.restoreJingMai(jingMaiSnapshot{Learned: []string{"jm_a"}, Cur: "jm_a", ZhenQi: jingMaiZhenQiPool{Day: anchor.Format("2006-01-02")}}, anchor)

	// 2 分钟 = 86 真气：未满 750 周天 → 进度帧 + 玩家属性帧（ZQPLValue
	// 等刷新，Java meridianTickOnce 变化即补 playerPropsFrame 同构）。
	frames := player.jingMaiCultivateTick(catalog, anchor.Add(2*time.Minute))
	if len(frames) != 2 {
		t.Fatalf("frames=%d, want progress + props", len(frames))
	}
	snapshot := player.jingMaiSnapshotOf()
	if got := snapshot.Progress["jm_a"]; got.Fill != 86 || got.Level != 0 || got.Total != 86 {
		t.Fatalf("progress=%+v, want fill=86 level=0 total=86", got)
	}
	if snapshot.ZhenQi.Used != 86 {
		t.Fatalf("zhenQi used=%d, want 86", snapshot.ZhenQi.Used)
	}
	propsIndices, err := propertyFrameIndices(frames[len(frames)-1])
	if err != nil || !containsIndex(propsIndices, propZQPLValue) {
		t.Fatalf("tick props frame missing ZQPLValue: indices=%v err=%v", propsIndices, err)
	}

	// 再 8 分钟 = 344 真气 → 86+344=430 仍未升级，进度帧 + 属性帧。
	frames = player.jingMaiCultivateTick(catalog, anchor.Add(10*time.Minute))
	if len(frames) != 2 {
		t.Fatalf("frames=%d", len(frames))
	}

	// 追加至 707 后 1 分钟 +43=750 → 升级帧（Level+TotalFillValue）+
	// 属性帧（JingMaiTotalLevel 14→15 随帧刷新）。
	snapshot = player.jingMaiSnapshotOf()
	snapshot.Progress["jm_a"] = jingMaiProgressRow{Level: 0, Fill: 707, Total: 707}
	player.restoreJingMai(snapshot, anchor.Add(11*time.Minute))
	frames = player.jingMaiCultivateTick(catalog, anchor.Add(12*time.Minute))
	if len(frames) != 2 {
		t.Fatalf("frames=%d, want level-up + props", len(frames))
	}
	propsIndices, err = propertyFrameIndices(frames[len(frames)-1])
	if err != nil || !containsIndex(propsIndices, propJingMaiTotLevel) {
		t.Fatalf("level-up props frame missing JingMaiTotalLevel: indices=%v err=%v", propsIndices, err)
	}
	snapshot = player.jingMaiSnapshotOf()
	if got := snapshot.Progress["jm_a"]; got.Level != 1 || got.Fill != 0 || got.Total != 750 {
		t.Fatalf("leveled=%+v, want level=1 fill=0 total=750", got)
	}
}

func TestJingMaiCultivateTickRespectsDailyBudgetAndMaxLevel(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	player := newPlayerActor("经脉侠", 0)
	anchor := time.Date(2026, 11, 7, 10, 0, 0, 0, time.Local)
	// jm_a maxLevel=3；硬闸 500 开启且已用 499/500 → 1 分钟只能补 1 点
	//（开闸截断；Java applyZhenQi headroom 同构）。
	player.restoreJingMai(jingMaiSnapshot{
		Learned: []string{"jm_a"},
		Cur:     "jm_a",
		Progress: map[string]jingMaiProgressRow{
			"jm_a": {Level: 0, Fill: 10, Total: 10},
		},
		Cap:    jingMaiDailyCap{Enabled: true, Value: 500},
		ZhenQi: jingMaiZhenQiPool{Day: anchor.Format("2006-01-02"), Used: 499},
	}, anchor)
	frames := player.jingMaiCultivateTick(catalog, anchor.Add(1*time.Minute))
	snapshot := player.jingMaiSnapshotOf()
	if got := snapshot.Progress["jm_a"]; got.Fill != 11 {
		t.Fatalf("fill=%d, want 11 (budget 1)", got)
	}
	if snapshot.ZhenQi.Used != 500 {
		t.Fatalf("used=%d, want capped 500", snapshot.ZhenQi.Used)
	}
	// 额度耗尽后不再产帧（零帧静默）。
	if frames = player.jingMaiCultivateTick(catalog, anchor.Add(2*time.Minute)); len(frames) != 0 {
		t.Fatalf("frames=%d after budget exhausted", len(frames))
	}
	// 关闸不限量（Java 默认行为）：同日 500 之上照常入账。
	player.setJingMaiDailyCap(false, 500)
	if frames = player.jingMaiCultivateTick(catalog, anchor.Add(3*time.Minute)); len(frames) != 2 {
		t.Fatalf("frames=%d with cap disabled", len(frames))
	}
	if snapshot = player.jingMaiSnapshotOf(); snapshot.ZhenQi.Used != 543 {
		t.Fatalf("used=%d, want uncapped 543 (+43/min)", snapshot.ZhenQi.Used)
	}

	// 满级截断：jm_b maxLevel=225 级 225 → 不再累计。
	player2 := newPlayerActor("经脉侠2", 0)
	player2.restoreJingMai(jingMaiSnapshot{
		Learned: []string{"jm_b"},
		Cur:     "jm_b",
		Progress: map[string]jingMaiProgressRow{
			"jm_b": {Level: 225, Fill: 0, Total: 225 * 750},
		},
		ZhenQi: jingMaiZhenQiPool{Day: anchor.Format("2006-01-02")},
	}, anchor)
	if frames := player2.jingMaiCultivateTick(catalog, anchor.Add(5*time.Minute)); len(frames) != 0 {
		t.Fatalf("frames=%d at max level", len(frames))
	}
	if snapshot := player2.jingMaiSnapshotOf(); snapshot.ZhenQi.Used != 0 {
		t.Fatalf("used=%d at max level (不应耗真气)", snapshot.ZhenQi.Used)
	}
}

func TestJingMaiViewFramesAndSnapshotRoundtrip(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	player := newPlayerActor("经脉侠", 0)
	player.restoreJingMai(jingMaiSnapshot{
		Learned:   []string{"jm_a", "jm_b"},
		Cur:       "jm_a",
		Activated: []string{"jm_a"},
		Progress: map[string]jingMaiProgressRow{
			"jm_a": {Level: 1, Fill: 12, Total: 762},
		},
	}, time.Now())

	frames, err := player.jingMaiViewFrames(catalog)
	if err != nil {
		t.Fatalf("view frames: %v", err)
	}
	// Java viewFrames：容器 + 已学行（learned 序，与目录全量无关）。
	if len(frames) != 3 {
		t.Fatalf("frames=%d, want 1 create + 2 learned rows", len(frames))
	}
	if frames[0][0] != 0x15 {
		t.Fatalf("container opcode=%#x (want 0x15 create view)", frames[0][0])
	}
	if !strings.Contains(string(frames[1]), "jm_a") {
		t.Fatalf("row frame missing jm_a: %x", frames[1])
	}

	// 快照 JSON 往返保真（learned 授予序不丢）。
	rendered := player.jingMaiSnapshotOf()
	raw, err := json.Marshal(rendered)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded jingMaiSnapshot
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	restored := newPlayerActor("回档侠", 0)
	restored.restoreJingMai(decoded, time.Now())
	if again := restored.jingMaiSnapshotOf(); again.Cur != "jm_a" || len(again.Activated) != 1 || again.Progress["jm_a"].Level != 1 {
		t.Fatalf("roundtrip=%+v", again)
	}
	if len(rendered.Learned) != 2 || rendered.Learned[0] != "jm_a" || rendered.Learned[1] != "jm_b" {
		t.Fatalf("learned roundtrip=%v", rendered.Learned)
	}
}

func TestPushJingMaiFramesReplaysViewRecordsAndProps(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	player := newPlayerActor("经脉侠", 0)
	player.restoreJingMai(jingMaiSnapshot{
		Learned:   []string{"jm_a", "jm_b"},
		Cur:       "jm_a",
		Activated: []string{"jm_a"},
		Progress:  map[string]jingMaiProgressRow{"jm_a": {Level: 1, Fill: 12, Total: 762}},
	}, time.Now())
	conn := &captureMessageConnection{}
	if err := pushJingMaiFrames(conn, player, catalog); err != nil {
		t.Fatalf("push: %v", err)
	}
	frames := conn.Frames()
	// 1 容器 + 2 已学行 + 1 记录行 + 1 面板帧 = 5（Java 授予块顺序：
	// View45 → active_jingmai_rec → 玩家属性帧）。
	if len(frames) != 5 {
		t.Fatalf("frames=%d, want 5", len(frames))
	}
	if frames[3][0] != 0x11 || binary.LittleEndian.Uint16(frames[3][10:]) != recordActiveJingMai {
		t.Fatalf("record replay=%x", frames[3])
	}
	if frames[4][0] != 0x10 {
		t.Fatalf("props frame=%#x", frames[4][0])
	}
}

// 首次进世界种子（Java PlayerMeridian.create）：starter 表命中目录才
// 入学、每条 1 级 0 真气；View45 只发已学行，cur 空 → 无记录行。
func TestPushJingMaiFramesSeedsStarterLearned(t *testing.T) {
	dir := t.TempDir()
	jingmaiINI := "[jm_dantian]\nItemType=1003\nStaticData=1\n\n[jm_ni]\nItemType=1003\nStaticData=2\n"
	staticINI := "[1]\nMinVarPropNo=141001\nMaxVarPropNo=141180\n\n[2]\nMinVarPropNo=1\nMaxVarPropNo=9\n"
	if err := os.WriteFile(filepath.Join(dir, "jingmai.ini"), []byte(jingmaiINI), 0o644); err != nil {
		t.Fatalf("write jingmai.ini: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "jingmai_static.ini"), []byte(staticINI), 0o644); err != nil {
		t.Fatalf("write jingmai_static.ini: %v", err)
	}
	catalog, err := loadJingMaiCatalog(filepath.Join(dir, "jingmai.ini"), filepath.Join(dir, "jingmai_static.ini"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if got := len(catalog.starterOrder()); got != 1 {
		t.Fatalf("starter order=%d, want 1 (jm_dantian only)", got)
	}

	player := newPlayerActor("萌新侠", 0)
	conn := &captureMessageConnection{}
	if err := pushJingMaiFrames(conn, player, catalog); err != nil {
		t.Fatalf("push: %v", err)
	}
	frames := conn.Frames()
	// 容器 + 1 已学行（jm_ni 未学不发）+ 面板帧 = 3。
	if len(frames) != 3 {
		t.Fatalf("frames=%d, want 3", len(frames))
	}
	if !strings.Contains(string(frames[1]), "jm_dantian") {
		t.Fatalf("starter row missing jm_dantian: %x", frames[1])
	}
	snapshot := player.jingMaiSnapshotOf()
	if len(snapshot.Learned) != 1 || snapshot.Learned[0] != "jm_dantian" {
		t.Fatalf("learned=%v, want [jm_dantian]", snapshot.Learned)
	}
	if got := snapshot.Progress["jm_dantian"]; got.Level != 1 || got.Fill != 0 {
		t.Fatalf("seeded progress=%+v, want level=1 fill=0", got)
	}
	// 再推一次幂等：不重复种子、不重复行。
	if err := pushJingMaiFrames(conn, player, catalog); err != nil {
		t.Fatalf("repush: %v", err)
	}
	if frames = conn.Frames(); len(frames) != 6 {
		t.Fatalf("frames=%d after repush, want 6", len(frames))
	}
}

// findJingMaiProp 按属性索引取值（jingMaiPropsLocked 的行查询辅助）。
func findJingMaiProp(properties []clientdata.IndexedProperty, index uint16) (clientdata.Value, bool) {
	for _, property := range properties {
		if property.Index == index {
			return property.Value, true
		}
	}
	return clientdata.Value{}, false
}

// GM set_zhenqi_cap：ZhenQiDayValue 展示口径随开关变化（开闸=阈值/
// 关闸=官方缺省 500），阈值关闭时仍保存、重开沿用；快照 roundtrip 与
// 旧档（无 cap 节点）兼容（Java setDailyCap/zhenQiDayValue 同构）。
func TestJingMaiDailyCapDisplayAndPersistence(t *testing.T) {
	player := newPlayerActor("硬闸侠", 0)
	seedJingMaiLearned(player, "jm_a")

	player.mu.Lock()
	properties := player.jingMaiPropsLocked()
	player.mu.Unlock()
	if value, ok := findJingMaiProp(properties, propZhenQiDayValue); !ok || value.I32 != zhenQiDayValueMax {
		t.Fatalf("default ZhenQiDayValue=%+v ok=%t, want %d", value, ok, zhenQiDayValueMax)
	}

	player.setJingMaiDailyCap(true, 3000)
	player.mu.Lock()
	properties = player.jingMaiPropsLocked()
	player.mu.Unlock()
	if value, _ := findJingMaiProp(properties, propZhenQiDayValue); value.I32 != 3000 {
		t.Fatalf("capped ZhenQiDayValue=%d, want 3000", value.I32)
	}

	// 关闸回落 500，但阈值 3000 仍在快照里（重开沿用）。
	player.setJingMaiDailyCap(false, 0)
	snapshot := player.jingMaiSnapshotOf()
	if snapshot.Cap.Enabled || snapshot.Cap.Value != 3000 {
		t.Fatalf("snapshot cap=%+v, want disabled but value kept 3000", snapshot.Cap)
	}
	player.mu.Lock()
	properties = player.jingMaiPropsLocked()
	player.mu.Unlock()
	if value, _ := findJingMaiProp(properties, propZhenQiDayValue); value.I32 != zhenQiDayValueMax {
		t.Fatalf("uncapped ZhenQiDayValue=%d, want %d", value.I32, zhenQiDayValueMax)
	}

	// 快照 JSON 往返保真。
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded jingMaiSnapshot
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Cap.Value != 3000 || decoded.Cap.Enabled {
		t.Fatalf("roundtrip cap=%+v", decoded.Cap)
	}

	// 旧档无 cap 节点（零值）：关闸 + 阈值回落官方缺省。
	legacy := []byte(`{"cur":"","last":"","activated":null,"progress":{},"zhenQi":{"day":"","used":0}}`)
	var legacySnapshot jingMaiSnapshot
	if err := json.Unmarshal(legacy, &legacySnapshot); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	restored := newPlayerActor("回档侠", 0)
	restored.restoreJingMai(legacySnapshot, time.Now())
	if again := restored.jingMaiSnapshotOf(); again.Cap.Enabled || again.Cap.Value != zhenQiDayValueMax {
		t.Fatalf("legacy cap=%+v, want disabled with default %d", again.Cap, zhenQiDayValueMax)
	}
}

// GM set_jingmai_tier：直设周天（钳 1..目录上限、fill 归零走新周天
// 起点）、未学拒绝、经脉总等级随动（Java setLevel 同构）。
func TestSetJingMaiLevelClampsAndResetsFill(t *testing.T) {
	catalog := stubJingMaiCatalog(t)
	player := newPlayerActor("周天侠", 0)
	player.restoreJingMai(jingMaiSnapshot{
		Learned: []string{"jm_a", "jm_b"},
		Progress: map[string]jingMaiProgressRow{
			"jm_a": {Level: 1, Fill: 100, Total: 100},
			"jm_b": {Level: 1},
		},
	}, time.Now())

	// jm_a maxLevel=3：99 钳到 3，fill 归零，行号=授予序 1。
	old, row, slot, ok := player.setJingMaiLevel(catalog, "jm_a", 99)
	if !ok || old != 1 || row.Level != 3 || row.Fill != 0 || slot != 1 {
		t.Fatalf("set 99: old=%d row=%+v slot=%d ok=%t", old, row, slot, ok)
	}
	// 经脉总等级随动：jm_a=3 + jm_b=1 → JingMaiTotalLevel=4。
	player.mu.Lock()
	total := 0
	for _, id := range player.jingmai.learned {
		total += int(player.jingmai.progress[id].Level)
	}
	player.mu.Unlock()
	if total != 4 {
		t.Fatalf("totalLevel=%d, want 4", total)
	}
	// 0 钳到 1。
	if _, row, _, ok := player.setJingMaiLevel(catalog, "jm_a", 0); !ok || row.Level != 1 {
		t.Fatalf("set 0: row=%+v ok=%t (want clamp to 1)", row, ok)
	}
	// 未学经脉拒绝（先在客户端开通）。
	if _, _, _, ok := player.setJingMaiLevel(catalog, "jm_ni", 36); ok {
		t.Fatalf("unlearned jm_ni accepted")
	}
	// 目录外 id 同样拒绝（行号=0）。
	if _, _, _, ok := player.setJingMaiLevel(catalog, "jm_missing", 36); ok {
		t.Fatalf("unknown jm_missing accepted")
	}
}
