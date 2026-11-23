package fssoloworld

// npc_patrol_test.go 巡逻集成测试：目录绑定（mappath 路线命中）、物化
// 采样（不闪现回出生点）、帧族序（State 先于 0x21/批量上限/水位）、
// AOI 半径对账、op0x0A 位移包解析。路线真值在仓根 testdata/mappath。

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// TestSceneNPCCatalogPatrolBound born03 巡逻绑定：mappath 路线表命中
// （walkers>0），目录缓存内 spawns 携带时间表。
func TestSceneNPCCatalogPatrolBound(t *testing.T) {
	s := newNpcTestServer(t)
	catalog := s.sceneNPCCatalog("born03")
	if len(catalog.spawns) == 0 {
		t.Fatal("born03 目录为空（真档缺席——见日志）")
	}
	if catalog.bind.Walkers == 0 {
		t.Fatalf("巡逻绑定 0 命中（routes 未装载？bind=%+v）", catalog.bind)
	}
	walkers := 0
	for _, spawn := range catalog.spawns {
		if spawn.Schedule != nil {
			walkers++
		}
	}
	if walkers != catalog.bind.Walkers {
		t.Fatalf("时间表持有 %d ≠ bind.Walkers %d", walkers, catalog.bind.Walkers)
	}
	t.Logf("born03 巡逻：walkers=%d stand=%d missingPath=%d far=%d",
		catalog.bind.Walkers, catalog.bind.Stand, catalog.bind.MissingPath, catalog.bind.FarOrShort)
}

// resetNpcCatalogCache 清场景目录缓存（测试面：切 env 后重建绑定态）。
func resetNpcCatalogCache() {
	npcCatalogMu.Lock()
	npcCatalogCache = map[string]*npcCatalog{}
	npcCatalogMu.Unlock()
}

// TestSceneNPCCatalogPatrolOff NINEYIN_NPC_PATROL=off：目录不绑定（全部
// 站立，坐标保持摆放值）。目录缓存按场景键进程级共享，先清缓存防
// 前例绑定态串场。
func TestSceneNPCCatalogPatrolOff(t *testing.T) {
	t.Setenv("NINEYIN_NPC_PATROL", "off")
	resetNpcCatalogCache()
	t.Cleanup(resetNpcCatalogCache) // 还原 env 后清缓存，后续测试重建绑定态
	s := newNpcTestServer(t)
	catalog := s.sceneNPCCatalog("born03")
	if catalog.bind.Walkers != 0 {
		t.Fatalf("PATROL=off 仍有绑定 walkers=%d", catalog.bind.Walkers)
	}
	for _, spawn := range catalog.spawns {
		if spawn.Schedule != nil {
			t.Fatal("PATROL=off 不应携带时间表")
		}
	}
}

// TestPatrolSamplingMovesMaterialization 物化采样：采样副本随时刻推进
// 位移（60s 窗口内必有），且不污染目录缓存。
func TestPatrolSamplingMovesMaterialization(t *testing.T) {
	s := newNpcTestServer(t)
	catalog := s.sceneNPCCatalog("born03")
	var walker *npcfeed.Spawn
	for i := range catalog.spawns {
		if catalog.spawns[i].Schedule != nil {
			walker = &catalog.spawns[i]
			break
		}
	}
	if walker == nil {
		t.Fatal("born03 无巡逻 NPC（绑定失败——见日志）")
	}
	snapX, snapZ := walker.X, walker.Z
	nowMs := time.Now().UnixMilli()
	moved := false
	for offset := int64(0); offset < 60_000 && !moved; offset += 500 {
		sampled := sampleSpawns([]npcfeed.Spawn{*walker}, nowMs+offset)
		if math.Abs(float64(sampled[0].X-snapX)) > 1 || math.Abs(float64(sampled[0].Z-snapZ)) > 1 {
			moved = true
		}
	}
	if !moved {
		t.Fatal("60s 窗口内采样位置无位移（时间表死表？）")
	}
	if walker.X != snapX || walker.Z != snapZ {
		t.Fatal("sampleSpawns 改动了目录缓存坐标")
	}
}

// TestPatrolFramesOrderAndBatch 巡逻帧族：State 翻转帧先于 0x21；批量
// ≤8 条/帧；同拍二次组帧无重复 State（水位），0x21 每拍全量。
func TestPatrolFramesOrderAndBatch(t *testing.T) {
	s := newNpcTestServer(t)
	catalog := s.sceneNPCCatalog("born03")
	if catalog.bind.Walkers == 0 {
		t.Fatal("born03 巡逻绑定 0 命中")
	}
	visible := map[uint32]bool{}
	for _, spawn := range catalog.spawns {
		visible[spawn.ObjectID] = true
	}
	moveState := map[uint32]bool{}
	nowMs := time.Now().UnixMilli()
	frames := patrolFrames(catalog, visible, moveState, nowMs, nil)
	if len(frames) == 0 {
		t.Fatal("巡逻帧族为空")
	}
	first21, last10 := -1, -1
	for i, f := range frames {
		switch f[0] {
		case 0x10:
			last10 = i
		case 0x21:
			if first21 < 0 {
				first21 = i
			}
		default:
			t.Fatalf("帧 %d 形态异常 op=%#x（只应有 0x10/0x21）", i, f[0])
		}
	}
	if last10 > first21 {
		t.Fatalf("State 帧（%d）应先于 0x21（%d）", last10, first21)
	}
	for i, f := range frames {
		if f[0] == 0x21 {
			if n := int(binary.LittleEndian.Uint16(f[1:])); n > patrolBroadcastBatch {
				t.Fatalf("0x21 帧 %d 记录数 %d 超上限 %d", i, n, patrolBroadcastBatch)
			}
		}
	}
	again := patrolFrames(catalog, visible, moveState, nowMs, nil)
	for _, f := range again {
		if f[0] == 0x10 {
			t.Fatal("同拍无翻转却重复下发 State 帧（水位失效）")
		}
	}
	if len(again) == 0 {
		t.Fatal("第二拍应仍有 0x21 全量广播")
	}
}

// TestNpcViewportDiffRadius AOI 半径模式：近点首账只增、迟滞圈内保持、
// 远点全删。
func TestNpcViewportDiffRadius(t *testing.T) {
	s := newNpcTestServer(t)
	catalog := s.sceneNPCCatalog("born03")
	anchor := catalog.spawns[0]
	radius := 60.0
	near := [4]float64{float64(anchor.X), float64(anchor.Y), float64(anchor.Z), 0}
	nowMs := time.Now().UnixMilli()

	adds, removes, visible := npcViewportDiff(catalog, near, nowMs, map[uint32]bool{}, radius)
	if len(adds) == 0 || len(removes) != 0 {
		t.Fatalf("近点首账应只增：%+d -%d", len(adds), len(removes))
	}
	if !visible[anchor.ObjectID] {
		t.Fatal("锚点 NPC 不在圈内")
	}
	edge := [4]float64{float64(anchor.X) + radius + 30, float64(anchor.Y), float64(anchor.Z), 0}
	_, removes2, visible2 := npcViewportDiff(catalog, edge, nowMs, visible, radius)
	if !visible2[anchor.ObjectID] || len(removes2) != 0 {
		t.Fatal("迟滞圈内应保持可见且不删")
	}
	far := [4]float64{float64(anchor.X) + 5000, float64(anchor.Y), float64(anchor.Z), 0}
	_, removes3, visible3 := npcViewportDiff(catalog, far, nowMs, visible, radius)
	if len(visible3) != 0 || len(removes3) != len(visible) {
		t.Fatalf("远点应全删：visible=%d removes=%d（原 %d）", len(visible3), len(removes3), len(visible))
	}
}

// TestSceneNPCEnterBookkeeping 进场簿记：sceneNPCEnter 初始化可见集/
// 场景/首拍时刻（+5s），物化帧非空。
func TestSceneNPCEnterBookkeeping(t *testing.T) {
	s := newNpcTestServer(t)
	pl := s.Players().AddPlayer(21, 1, "acct")
	frames := s.sceneNPCEnter(pl, "born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(frames) == 0 {
		t.Fatal("进场物化帧族为空")
	}
	pl.npcMu.Lock()
	visible, scene, ready := pl.npcVisible, pl.npcScene, pl.patrolReadyAt
	pl.npcMu.Unlock()
	if len(visible) == 0 || scene != "born03_yanyuzhuang" {
		t.Fatalf("视口簿记未初始化 visible=%d scene=%q（应存解析目录名）", len(visible), scene)
	}
	if remaining := time.Until(ready); remaining <= 0 || remaining > patrolFirstBroadcastDelay+time.Second {
		t.Fatalf("首拍延迟异常 %v（应约 %v）", remaining, patrolFirstBroadcastDelay)
	}
}

// TestParseMotionPosition op0x0A 67B 位移包定版：四 tag=4 槽 + 限值校验。
func TestParseMotionPosition(t *testing.T) {
	msg := make([]byte, 67)
	const offset = 0x26
	put := func(off int, v float32) {
		msg[off] = 4
		binary.LittleEndian.PutUint32(msg[off+1:], math.Float32bits(v))
	}
	put(offset, 482.5)
	put(offset+5, 13.7)
	put(offset+10, 1008)
	put(offset+15, 1.56)
	x, y, z, orient, ok := parseMotionPosition(msg)
	if !ok || x != 482.5 || y != 13.7 || z != 1008 || orient != 1.56 {
		t.Fatalf("解析失败 ok=%v (%v,%v,%v,%v)", ok, x, y, z, orient)
	}
	if _, _, _, _, ok := parseMotionPosition(msg[:66]); ok {
		t.Fatal("66B 包不应命中")
	}
	bad := append([]byte(nil), msg...)
	bad[offset+5] = 3
	if _, _, _, _, ok := parseMotionPosition(bad); ok {
		t.Fatal("tag 破坏包不应命中")
	}
	bad2 := append([]byte(nil), msg...)
	binary.LittleEndian.PutUint32(bad2[offset+1:], math.Float32bits(20000))
	if _, _, _, _, ok := parseMotionPosition(bad2); ok {
		t.Fatal("越界坐标不应命中")
	}
}
