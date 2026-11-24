package fssoloworld

// npc_patrol_test.go 巡逻集成测试：目录绑定（mappath 路线命中，born03 与
// 其它场景）、物化采样（不闪现回出生点）、帧族序（State 先于 0x21/批量
// 上限/水位）、巡逻拍全链下发（sendToClient 观察缝）、路线目录缺席告警、
// AOI 半径对账、op0x0A 位移包解析。路线真值在仓根
// testdata/share/creator/mappath。

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
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
	moveState := map[uint32]string{}
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

// TestSceneNPCCatalogPatrolMultiScene born03 之外也绑定巡逻：全量 mappath
// 路线入数据根后，新手村 / 城市 / 门派各有 walkers。绑定 0 命中=数据根缺
// 该场景的路线档（2026-11-24 Res 层化后路线目录未随迁、全城站立的事故
// 形态），直接失败并指明原因。
func TestSceneNPCCatalogPatrolMultiScene(t *testing.T) {
	s := newNpcTestServer(t)
	for _, scene := range []string{"born01_mohezhen", "city03_jinling", "school07_wudang"} {
		catalog := s.sceneNPCCatalog(scene)
		if len(catalog.spawns) == 0 {
			t.Fatalf("%s 目录为空（真档缺席——见日志）", scene)
		}
		if catalog.bind.Walkers == 0 {
			t.Fatalf("%s 巡逻绑定 0 命中（bind=%+v）——数据根缺 mappath 路线档？", scene, catalog.bind)
		}
		t.Logf("%s 巡逻：walkers=%d stand=%d missingPath=%d far=%d",
			scene, catalog.bind.Walkers, catalog.bind.Stand, catalog.bind.MissingPath, catalog.bind.FarOrShort)
	}
}

// TestPatrolRoutesMissingTraced 路线目录缺席：绑定全站立，并 trace 一次
// 说清原因（多场景装载不重复刷）。
func TestPatrolRoutesMissingTraced(t *testing.T) {
	var lines []string
	s := newNpcTestServerLogf(t, func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	})
	absent := filepath.Join(t.TempDir(), "absent")
	t.Setenv("NINEYIN_MAPPATH_DIR", absent)
	resetNpcCatalogCache()
	t.Cleanup(resetNpcCatalogCache) // 还原 env 后清缓存，后续测试重建绑定态
	patrolRoutesMissingOnce = sync.Once{}
	t.Cleanup(func() { patrolRoutesMissingOnce = sync.Once{} })

	for _, scene := range []string{"born03", "city03"} {
		if catalog := s.sceneNPCCatalog(scene); catalog.bind.Walkers != 0 {
			t.Fatalf("%s 路线目录缺席仍有 walkers=%d", scene, catalog.bind.Walkers)
		}
	}
	traced := 0
	for _, line := range lines {
		if strings.Contains(line, "npc patrol: 巡逻路线目录缺席") && strings.Contains(line, absent) {
			traced++
		}
	}
	if traced != 1 {
		t.Fatalf("缺席 trace 应恰好 1 条（两个场景装载），得 %d", traced)
	}
}

// TestResolveMappathDirDefaultPath 路线目录缺省 = 数据根下
// share/creator/mappath（与客户端资源树同布局，2026-11-25 起；旧
// <数据根>/mappath 已废），env NINEYIN_MAPPATH_DIR 优先。
func TestResolveMappathDirDefaultPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Res")
	t.Setenv("NINEYIN_MAPPATH_DIR", "")
	if want := filepath.Join(root, "share", "creator", "mappath"); resolveMappathDir(root) != want {
		t.Fatalf("缺省路线目录 %s，应为 %s", resolveMappathDir(root), want)
	}
	override := filepath.Join(t.TempDir(), "elsewhere", "mappath")
	t.Setenv("NINEYIN_MAPPATH_DIR", override)
	if got := resolveMappathDir(root); got != override {
		t.Fatalf("NINEYIN_MAPPATH_DIR 覆盖未生效: %s", got)
	}
}

// patrolEntry 0x21 帧解码出的一条巡逻记录。
type patrolEntry struct {
	objectID, ownerID uint32
	speed, rotate     float32
	flags             uint32
}

// decodePatrolTick 把一拍下发的帧解码：0x21 → objectID → 记录；返回
// State（0x10）帧数。其余 op 原样计数返回（调用方断言应为 0）。
func decodePatrolTick(frames [][]byte) (entries map[uint32]patrolEntry, stateFrames, others int) {
	entries = make(map[uint32]patrolEntry)
	for _, f := range frames {
		switch f[0] {
		case 0x10:
			stateFrames++
		case 0x21:
			n := int(binary.LittleEndian.Uint16(f[1:]))
			if len(f) != 3+n*44 {
				panic(fmt.Sprintf("0x21 帧长 %d 与记录数 %d 不符", len(f), n))
			}
			for i := 0; i < n; i++ {
				e := f[3+i*44:]
				entries[binary.LittleEndian.Uint32(e[0:])] = patrolEntry{
					objectID: binary.LittleEndian.Uint32(e[0:]),
					ownerID:  binary.LittleEndian.Uint32(e[4:]),
					speed:    math.Float32frombits(binary.LittleEndian.Uint32(e[24:])),
					rotate:   math.Float32frombits(binary.LittleEndian.Uint32(e[28:])),
					flags:    binary.LittleEndian.Uint32(e[40:]),
				}
			}
		default:
			others++
		}
	}
	return entries, stateFrames, others
}

// checkPatrolEntry 一条记录的槽位须与时间表在该时刻的动作成组：行进
// spd>0、flags=1、rotate=+2π；停留 spd=0、flags=0、rotate=0。
func checkPatrolEntry(t *testing.T, id uint32, e patrolEntry, motion npcfeed.Motion) {
	t.Helper()
	if e.ownerID == 0 {
		t.Fatalf("walker %d 的 0x21 记录缺席", id)
	}
	if motion.Moving {
		if e.flags != 1 || e.speed <= 0 || math.Abs(float64(e.rotate-2*math.Pi)) > 1e-6 {
			t.Fatalf("行进槽位异常 id=%d: %+v", id, e)
		}
		return
	}
	if e.flags != 0 || e.speed != 0 || e.rotate != 0 {
		t.Fatalf("停留槽位异常 id=%d: %+v", id, e)
	}
}

// isolateTalkPaused 对话暂停表是包级状态：交互用例点过的行人可能仍挂着
// 暂停。换入空表、用例结束还原，巡逻拍用例互不串场。
func isolateTalkPaused(t *testing.T) {
	t.Helper()
	talkPauseMu.Lock()
	saved := talkPaused
	talkPaused = map[uint32]int64{}
	talkPauseMu.Unlock()
	t.Cleanup(func() {
		talkPauseMu.Lock()
		talkPaused = saved
		talkPauseMu.Unlock()
	})
}

// TestNpcPatrolTickBroadcast 巡逻拍全链（sendToClient 观察缝）：物化落地
// 窗口内零巡逻帧 → 过闸后每拍下发 State 翻转帧（先）+ 0x21 批量帧（后），
// 记录恰为视口内全部巡逻 NPC → 同拍二次不重复 State（水位）→ 对话暂停
// 的 walker 置 stand、速度 0，菜单关闭即恢复，60s 兜底过期后按时间表
// 续走 → timerCheck 尾部挂接（Tick 驱动同样下发）。
func TestNpcPatrolTickBroadcast(t *testing.T) {
	s := newNpcTestServer(t)
	isolateTalkPaused(t)
	pl := s.players.AddPlayer(31, 1, "patrol")
	s.players.SetPlayerState(pl, PlOnline)
	frames := captureFrames(t)

	s.sceneNPCEnter(pl, "born03", [4]float64{482.5, 13.7, 1008, 1.56})
	catalog := s.sceneNPCCatalog("born03")
	walkers := map[uint32]npcfeed.Spawn{}
	for _, spawn := range catalog.spawns {
		if spawn.Schedule != nil {
			walkers[spawn.ObjectID] = spawn
		}
	}
	if len(walkers) == 0 {
		t.Fatal("born03 无巡逻 NPC（绑定失败——见日志）")
	}

	// 物化落地窗口内（首拍延迟）：不下发巡逻帧。
	s.npcPatrolTick(time.Now())
	if len(*frames) != 0 {
		t.Fatalf("首拍延迟闸内下发了 %d 帧（ops=% x）", len(*frames), opOf(*frames))
	}

	ready := time.Now().Add(patrolFirstBroadcastDelay + time.Second)
	tick := func(at time.Time) map[uint32]patrolEntry {
		t.Helper()
		*frames = nil
		s.npcPatrolTick(at)
		entries, _, others := decodePatrolTick(*frames)
		if others != 0 {
			t.Fatalf("巡逻拍混入其它帧 %d 条（ops=% x）", others, opOf(*frames))
		}
		return entries
	}
	entries := tick(ready)
	if len(entries) != len(walkers) {
		t.Fatalf("0x21 记录 %d 条 ≠ 视口内巡逻 NPC %d", len(entries), len(walkers))
	}
	lastState, first21 := -1, -1
	for i, f := range *frames {
		if f[0] == 0x10 {
			lastState = i
		} else if first21 < 0 {
			first21 = i
		}
	}
	if lastState < 0 || lastState > first21 {
		t.Fatalf("State 翻转帧应先于 0x21（末 State=%d 首 0x21=%d）", lastState, first21)
	}
	for id, spawn := range walkers {
		e, ok := entries[id]
		if !ok {
			t.Fatalf("walker %d 缺 0x21 记录", id)
		}
		if e.ownerID != spawn.OwnerID {
			t.Fatalf("walker %d own=%#x ≠ 目录 %#x", id, e.ownerID, spawn.OwnerID)
		}
		checkPatrolEntry(t, id, e, spawn.Schedule.MotionAt(ready.UnixMilli()))
	}

	// 同拍二次：0x21 仍全量，State 不重复（水位）。
	if again := tick(ready); len(again) != len(walkers) {
		t.Fatalf("第二拍 0x21 记录 %d 条", len(again))
	}
	for _, f := range *frames {
		if f[0] == 0x10 {
			t.Fatal("同拍无翻转却重复下发 State 帧（水位失效）")
		}
	}

	// 对话暂停：挑目录序第一个此刻在走的 walker。停下的状态是它自己的
	// 静息态——出生状态带姿态的巡逻 NPC（巡街捕快之类）停回姿态，不一定
	// 是 stand。
	var pausedID uint32
	var paused npcfeed.Spawn
	for i := range catalog.spawns {
		if spawn := &catalog.spawns[i]; spawn.Schedule != nil && spawn.Schedule.MotionAt(ready.UnixMilli()).Moving {
			pausedID, paused = spawn.ObjectID, *spawn
			break
		}
	}
	if pausedID == 0 {
		t.Fatal("born03 过闸首拍无行进中的 walker（64 条路线全在停留？时间表异常）")
	}
	pauseNpcTalk(pausedID)
	if e := tick(ready)[pausedID]; e.flags != 0 || e.speed != 0 {
		t.Fatalf("对话暂停的 walker 应停走: %+v", e)
	}
	hold, err := entergen.NpcStateFrame(pausedID, paused.OwnerID, paused.RestAction())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(*frames, func(f []byte) bool { return bytes.Equal(f, hold) }) {
		t.Fatalf("暂停应下发静息态 State 帧 %q", paused.RestAction())
	}
	// 菜单关闭：恢复巡逻（回到 walk）。
	resumeNpcTalk(pausedID)
	checkPatrolEntry(t, pausedID, tick(ready)[pausedID], paused.Schedule.MotionAt(ready.UnixMilli()))
	walk, err := entergen.NpcStateFrame(pausedID, paused.OwnerID, "walk")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(*frames, func(f []byte) bool { return bytes.Equal(f, walk) }) {
		t.Fatal("恢复巡逻应下发 walk State 帧")
	}
	// 60s 兜底过期：菜单异常未关，暂停不再生效，按时间表续走。
	pauseNpcTalk(pausedID)
	if e := tick(ready)[pausedID]; e.flags != 0 || e.speed != 0 {
		t.Fatalf("对话暂停的 walker 应停走: %+v", e)
	}
	late := ready.Add(61 * time.Second)
	checkPatrolEntry(t, pausedID, tick(late)[pausedID], paused.Schedule.MotionAt(late.UnixMilli()))

	// timerCheck 尾部挂接：过闸后 Tick 驱动同样下发巡逻帧。
	pl.npcMu.Lock()
	pl.patrolReadyAt = time.Now().Add(-time.Second)
	pl.npcMu.Unlock()
	*frames = nil
	s.Tick()
	if entries, _, _ := decodePatrolTick(*frames); len(entries) != len(walkers) {
		t.Fatalf("timerCheck 驱动的巡逻拍 0x21 记录 %d 条 ≠ %d", len(entries), len(walkers))
	}
}

// TestNpcPatrolTickViewportConfig 部署形态（env.server：半径 150 / 上限
// 230；start-single.bat：官服 ident）：巡逻拍只含视口内的巡逻 NPC，记录的
// obj 须是进场物化帧登记过的对象、own 取目录分配值（官服形态 own 是
// 场景内序列号，不可由 obj 反推——必须与物化同源）。
func TestNpcPatrolTickViewportConfig(t *testing.T) {
	t.Setenv("NINEYIN_NPC_RADIUS", "150")
	t.Setenv("NINEYIN_NPC_LIMIT", "230")
	t.Setenv("NINEYIN_NPC_IDENT", "scene")
	resetNpcCatalogCache()
	t.Cleanup(resetNpcCatalogCache) // 还原 env 后清缓存，后续测试重建缺省形态
	s := newNpcTestServer(t)
	isolateTalkPaused(t)
	pl := s.players.AddPlayer(32, 1, "patrol-viewport")
	s.players.SetPlayerState(pl, PlOnline)
	frames := captureFrames(t)

	enterFrames := s.sceneNPCEnter(pl, "born03", [4]float64{482.5, 13.7, 1008, 1.56})
	catalog := s.sceneNPCCatalog("born03")
	if catalog.sceneID < 0 {
		t.Fatal("born03 不在 scenes.ini——官服 ident 回落固定段，无法验证")
	}
	pl.npcMu.Lock()
	visible := pl.npcVisible
	pl.patrolReadyAt = time.Now().Add(-time.Second)
	pl.npcMu.Unlock()
	if len(visible) == 0 || len(visible) > 230 {
		t.Fatalf("视口物化 %d 条（应 1..230）", len(visible))
	}
	materialized := make(map[uint32]bool, len(enterFrames))
	for _, f := range enterFrames {
		if f[0] == 0x28 {
			materialized[binary.LittleEndian.Uint32(f[10:14])] = true
		}
	}
	walkers := map[uint32]npcfeed.Spawn{}
	for i := range catalog.spawns {
		spawn := &catalog.spawns[i]
		if spawn.Schedule == nil || !visible[spawn.ObjectID] {
			continue
		}
		walkers[spawn.ObjectID] = *spawn
		if !materialized[spawn.ObjectID] {
			t.Fatalf("视口内 walker %#x 不在进场物化帧里", spawn.ObjectID)
		}
		if spawn.ObjectID>>24 != uint32(catalog.sceneID&0xFF) {
			t.Fatalf("官服形态 obj 高字节应为场景号低字节：id=%#x sceneID=%d", spawn.ObjectID, catalog.sceneID)
		}
	}
	if len(walkers) == 0 {
		t.Fatal("出生点 150m 视口内无巡逻 NPC")
	}

	*frames = nil
	now := time.Now()
	s.npcPatrolTick(now)
	entries, _, others := decodePatrolTick(*frames)
	if others != 0 || len(entries) != len(walkers) {
		t.Fatalf("0x21 记录 %d 条 ≠ 视口内巡逻 NPC %d（others=%d）", len(entries), len(walkers), others)
	}
	for id, spawn := range walkers {
		e := entries[id]
		if e.ownerID != spawn.OwnerID {
			t.Fatalf("walker %#x 0x21 own=%#x ≠ 物化 own %#x", id, e.ownerID, spawn.OwnerID)
		}
		checkPatrolEntry(t, id, e, spawn.Schedule.MotionAt(now.UnixMilli()))
	}
	t.Logf("部署形态：视口 %d 条，其中巡逻 %d 条", len(visible), len(walkers))
}

// patrolStatesOf 一拍里的 0x10 State 帧解成状态名序列（NpcStateFrame 的
// 单属性布局：12B 头 + idx u16 + u32 线长 + 串）。
func patrolStatesOf(frames [][]byte) []string {
	var out []string
	for _, f := range frames {
		if f[0] != 0x10 || len(f) < 18 {
			continue
		}
		n := int(binary.LittleEndian.Uint32(f[14:]))
		if len(f) < 18+n || n < 1 {
			continue
		}
		out = append(out, strings.TrimRight(string(f[18:18+n]), "\x00"))
	}
	return out
}

// TestPatrolStateTimeline 巡逻状态时间线（合成路线）：去程 walk → 到点
// 停留播放路点到达动作（ty_saodiloop 扫地）→ 回程 walk → 回到起点站立；
// 状态不变不发 State 帧；水位记状态名。出生状态 walk / run 的巡逻 NPC
// 停留时回 stand 而不是 walk。
func TestPatrolStateTimeline(t *testing.T) {
	route := &npcfeed.Route{PathID: "AI_test", Waypoints: []npcfeed.Waypoint{
		{X: 0, Y: 0, Z: 0, Arrival: "stand", DwellMs: 2000},
		{X: 10, Y: 0, Z: 0, Arrival: "ty_saodiloop", DwellMs: 4000},
	}}
	spawn := npcfeed.Spawn{ObjectID: 0x0E000102, OwnerID: 0x1002, BornAction: "walk",
		Schedule: npcfeed.BuildSchedule(route, 2, 4, 0)}
	catalog := &npcCatalog{spawns: []npcfeed.Spawn{spawn}}
	visible := map[uint32]bool{spawn.ObjectID: true}
	moveState := map[uint32]string{}
	// 0..5s 去程 → 5..9s 在 w1 停留 → 9..14s 回程 → 14..16s 在 w0 站立。
	steps := []struct {
		at   int64
		want string // 空=状态未变，不应有 State 帧
	}{
		{2_000, "walk"}, {6_000, "ty_saodiloop"}, {7_000, ""}, {11_000, "walk"}, {15_000, "stand"},
	}
	for _, step := range steps {
		frames := patrolFrames(catalog, visible, moveState, step.at, nil)
		states := patrolStatesOf(frames)
		if step.want == "" {
			if len(states) != 0 {
				t.Fatalf("t=%dms 状态未变却发了 State 帧 %q", step.at, states)
			}
			continue
		}
		if len(states) != 1 || states[0] != step.want {
			t.Fatalf("t=%dms State 帧 %q，应为 [%q]", step.at, states, step.want)
		}
		if moveState[spawn.ObjectID] != step.want {
			t.Fatalf("t=%dms 水位 %q ≠ %q", step.at, moveState[spawn.ObjectID], step.want)
		}
	}
	// 对话暂停：停在静息态（stand），不是出生状态的 walk。
	talkPauseMu.Lock()
	saved := talkPaused
	talkPaused = map[uint32]int64{spawn.ObjectID: 1_000}
	talkPauseMu.Unlock()
	t.Cleanup(func() {
		talkPauseMu.Lock()
		talkPaused = saved
		talkPauseMu.Unlock()
	})
	if states := patrolStatesOf(patrolFrames(catalog, visible, map[uint32]string{}, 2_000, talkPausedSnapshot())); len(states) != 1 || states[0] != "stand" {
		t.Fatalf("对话暂停应置静息态 stand，得 %q", states)
	}
	// 出生状态是真姿态的巡逻 NPC（如巡街捕快 ty_bukuai_guard）：停留回姿态；
	// 路点自带到达动作时仍以到达动作为先。
	guard := spawn
	guard.BornAction = "ty_bukuai_guard"
	guardCatalog := &npcCatalog{spawns: []npcfeed.Spawn{guard}}
	guardState := map[uint32]string{}
	for _, c := range []struct {
		at   int64
		want string
	}{{2_000, "walk"}, {6_000, "ty_saodiloop"}, {11_000, "walk"}, {15_000, "ty_bukuai_guard"}} {
		if states := patrolStatesOf(patrolFrames(guardCatalog, visible, guardState, c.at, nil)); len(states) != 1 || states[0] != c.want {
			t.Fatalf("带姿态的巡逻 NPC t=%dms State 帧 %q，应为 [%q]", c.at, states, c.want)
		}
	}
}

// TestSampleSpawnsMarksState 物化采样标 State：采样时刻在走 → walk（官服
// 同款：行进中进视口的 NPC 物化即 walk），停留 → 路点到达动作 / 静息态；
// 不巡逻的 NPC 不标（物化帧取静息态）。
func TestSampleSpawnsMarksState(t *testing.T) {
	route := &npcfeed.Route{PathID: "AI_test", Waypoints: []npcfeed.Waypoint{
		{X: 0, Y: 0, Z: 0, Arrival: "interact50", DwellMs: 3000},
		{X: 20, Y: 0, Z: 0, Arrival: "stand", DwellMs: 3000},
	}}
	walker := npcfeed.Spawn{ObjectID: 0x0E000102, OwnerID: 0x1002,
		Schedule: npcfeed.BuildSchedule(route, 2, 4, 0)}
	sitter := npcfeed.Spawn{ObjectID: 0x0E000103, OwnerID: 0x1003, BornAction: "ty_sitchair_drinkloop"}
	spawns := []npcfeed.Spawn{walker, sitter}
	// 0..10s 去程 → 10..13s 停在 w1 → 13..23s 回程 → 23..26s 在 w1 起点停留。
	for _, c := range []struct {
		at        int64
		walker    string
		walkerPos float32
	}{{5_000, "walk", 10}, {11_000, "stand", 20}, {18_000, "walk", 10}, {24_000, "interact50", 0}} {
		out := sampleSpawns(spawns, c.at)
		if out[0].State != c.walker || out[0].X != c.walkerPos {
			t.Errorf("t=%dms walker State/位置 = %q/%v，应为 %q/%v", c.at, out[0].State, out[0].X, c.walker, c.walkerPos)
		}
		if out[1].State != "" || out[1].RestAction() != "ty_sitchair_drinkloop" {
			t.Errorf("姿态 NPC 不应被巡逻采样改状态: State=%q rest=%q", out[1].State, out[1].RestAction())
		}
	}
	if walker.State != "" {
		t.Fatal("sampleSpawns 改动了目录里的 Spawn")
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
