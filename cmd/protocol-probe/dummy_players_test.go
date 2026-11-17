package main

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// ---- 侠恶狂邪数值（官服公式移植）----

func TestDummyAlignmentValuesDeterministic(t *testing.T) {
	for _, character := range []int{1, 2, 3, 4} {
		first := []int32{}
		for pass := 0; pass < 3; pass++ {
			justice, evil, freeBase, charValue := dummyAlignmentValues("红袖姑娘", character)
			if pass == 0 {
				first = []int32{justice, evil, freeBase, charValue}
				continue
			}
			if justice != first[0] || evil != first[1] || freeBase != first[2] || charValue != first[3] {
				t.Fatalf("character=%d pass=%d 数值不确定：%v vs %v", character, pass,
					[]int32{justice, evil, freeBase, charValue}, first)
			}
		}
	}
}

func TestDummyAlignmentValuesInvariants(t *testing.T) {
	// 无标签：三围总和 ≤ 100（官服阈值之内）。
	for i := 0; i < 64; i++ {
		justice, evil, freeBase, charValue := dummyAlignmentValues("路人甲", 0)
		if justice+evil+freeBase > 100 {
			t.Fatalf("无标签三围超阈：%d+%d+%d", justice, evil, freeBase)
		}
		if charValue != 0 {
			t.Fatalf("无标签 charValue 应为 0")
		}
	}
	// 侠：侠>恶，满足侠恶条件（侠+恶 > 桀骜×2 且 侠/恶 > 2）。
	for i := 0; i < 64; i++ {
		justice, evil, freeBase, _ := dummyAlignmentValues("测试侠客"+string(rune('A'+i%26)), 1)
		if justice <= evil {
			t.Fatalf("侠标签 justice(%d) 应大于 evil(%d)", justice, evil)
		}
		if justice+evil <= freeBase*2 {
			t.Fatalf("侠恶条件不满足：j+e=%d free=%d", justice+evil, freeBase)
		}
		if justice < 100+1 {
			t.Fatalf("侠标签计算值过低：%d", justice-evil)
		}
	}
	// 恶：恶>侠。
	for i := 0; i < 64; i++ {
		justice, evil, _, _ := dummyAlignmentValues("测试恶人"+string(rune('A'+i%26)), 2)
		if evil <= justice {
			t.Fatalf("恶标签 evil(%d) 应大于 justice(%d)", evil, justice)
		}
	}
	// 狂/邪：侠恶条件不满足（大值/小值 ≤ 2 → 判狂邪）。
	for i := 0; i < 64; i++ {
		justice, evil, freeBase, _ := dummyAlignmentValues("测试狂人"+string(rune('A'+i%26)), 3)
		big, small := justice, evil
		if small > big {
			big, small = small, big
		}
		if big > small*2 {
			t.Fatalf("狂标签不应满足侠恶条件：j=%d e=%d", justice, evil)
		}
		if (justice+evil)/2+freeBase < 100 {
			t.Fatalf("狂标签狂邪值过低：free=%d", freeBase)
		}
	}
}

// ---- 配置语义 ----

func TestDummyMatchesScene(t *testing.T) {
	dummy := dummyPlayer{Scene: "City01"}
	if !dummy.matchesScene("ini\\scene\\city01", "city01") {
		t.Fatal("resource 名匹配失败")
	}
	if dummy.matchesScene("ini\\scene\\city02", "city02") {
		t.Fatal("不应匹配其它场景")
	}
	dummy.Scene = ""
	if !dummy.matchesScene("whatever", "") {
		t.Fatal("空 scene 应匹配全部场景")
	}
}

func TestDummyMoveSemantics(t *testing.T) {
	runner := dummyPlayer{Patrol: []dummyPosition{{}, {}}}
	walker := dummyPlayer{PatrolRandom: true, MoveMode: "walk"}
	yufeng := dummyPlayer{PatrolRandom: true, Yufeng: true}
	stander := dummyPlayer{}
	if !dummyMoves(runner) || !dummyMoves(walker) || dummyMoves(stander) {
		t.Fatal("dummyMoves 判定错误")
	}
	if got := dummyMoveSpeed(runner); got != dummyPatrolSpeed {
		t.Fatalf("默认配速 = %f，应为 %f", got, dummyPatrolSpeed)
	}
	if got := dummyMoveSpeed(walker); got != dummyWalkSpeed {
		t.Fatalf("步行配速 = %f，应为 %f", got, dummyWalkSpeed)
	}
	if got := dummyMoveSpeed(yufeng); got != dummyQingGongSpeed {
		t.Fatalf("御风配速 = %f，应为 %f", got, dummyQingGongSpeed)
	}
}

// ---- 0x0D 归档 ----

func TestDummyAddObjectSmoke(t *testing.T) {
	dummy := dummyPlayer{
		Name:   "红袖姑娘",
		Sex:    1,
		School: "school_emei",
		Level:  42,
		Yufeng: true,
		Scene:  "city01",
		Pos:    dummyPosition{X: 10, Y: 1, Z: -3, Orient: 1.5},
		Look: dummyAppearanceCfg{
			Cloth: "cloth_g0003",
			Hair:  `obj\char\g_hair\g_hair9`,
		},
	}
	payload := dummyAddObject(dummyObjectIDBase+7, dummy, time.Now())
	if payload[0] != 0x0D {
		t.Fatalf("opcode = %#x，应为 0x0D", payload[0])
	}
	if got := binaryLE(payload[1:5]); got != dummyObjectIDBase+7 {
		t.Fatalf("objectID = %#x", got)
	}
	if binaryLE(payload[5:9]) != playerOwnerID {
		t.Fatalf("ownerID 不符")
	}
	if binaryLE16(payload[0x3d:]) == 0 {
		t.Fatal("属性计数为 0")
	}
	// 御风 buff 与门派串必须进档（按字节扫描即可验证链路）。
	if !bytes.Contains(payload, []byte("buf_ride_yufeng")) {
		t.Fatal("御风 buff 未入 BufferListStr")
	}
	if !bytes.Contains(payload, []byte("school_emei")) {
		t.Fatal("门派串未入档")
	}
	// 服装经 equipmentModels 按性别解析后入档（表缺时原样透传）。
	clothModel := equipmentModels.resolve("cloth_g0003", 1)
	if !bytes.Contains(payload, []byte(clothModel)) {
		t.Fatalf("服装模型 %q 未入档", clothModel)
	}
}

// TestDummyAddObjectWrapsAsOfficial0x28 锁定 OFFICIAL 档假人物化方言：
// 假人是玩家形大档（0x0D），物化时必须走 wrapObjectAddAsOp28 包装成
// 0x28 mode=0x46 明文帧，而不是被 officialNPC0x28Frame 重写成
// ConfigID="dummy" 的 NPC 壳（npc 目录无此配置，官方客户端查不到模型
// 不渲染——2026-11-17 23:34 真机实证）。
func TestDummyAddObjectWrapsAsOfficial0x28(t *testing.T) {
	dummy := dummyPlayer{
		Name:   "岳隐锋",
		Sex:    0,
		School: "school_wudang",
		Level:  44,
		Scene:  "city01",
		Pos:    dummyPosition{X: 610, Y: -94, Z: 439, Orient: 0.179},
	}
	payload := dummyAddObject(dummyObjectIDBase+1, dummy, time.Now())
	if payload[0] != 0x0D {
		t.Fatalf("opcode = %#x，wrap 前必须是 0x0D 大档", payload[0])
	}
	frame := wrapObjectAddAsOp28(payload)
	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("包装帧头 = %x，应为 28 46", frame[:2])
	}
	body := frame[10:]
	if len(body) != len(payload)-1 {
		t.Fatalf("体长 %d，应为 %d（0x0D 平移-1）", len(body), len(payload)-1)
	}
	// 外服 0x28 内层布局锚：值对数 u16@0x3C 非零（大档字段在）。
	if got := binaryLE16(body[0x3C:]); got == 0 {
		t.Fatal("体 [0x3C] 值对数为 0——大档字段丢失")
	}
	// 玩家形身份锚：Type=2 值对必须仍在体内（字节级存在性即可）。
	typeIdx := make([]byte, 2)
	typeEntry, ok := clientdata.PlayerProperty("Type")
	if !ok {
		t.Fatal("Type 属性在当前表缺席")
	}
	binary.LittleEndian.PutUint16(typeIdx, typeEntry.Index)
	if !bytes.Contains(body, typeIdx) {
		t.Fatal("Type 属性值对不在包装体内")
	}
}

func binaryLE(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func binaryLE16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

// ---- 视口准入 / 快照过滤 / 移动步进 ----

func newDummyTestScene(t *testing.T) (*sceneRun, *sceneLifecycle, *captureMessageConnection) {
	t.Helper()
	conn := &captureMessageConnection{}
	view := newSceneLifecycle("dummy-test", conn)
	run := view.run
	if _, err := run.scene.Begin("scene", "resource"); err != nil {
		t.Fatalf("begin scene: %v", err)
	}
	view.radius = dummyViewportRadius * 4
	view.mu.Lock()
	view.center = role.Position{}
	view.mu.Unlock()
	return run, view, conn
}

func registerTestDummy(t *testing.T, run *sceneRun, id uint32, x float32, mover bool) {
	t.Helper()
	dummy := dummyPlayer{
		Name:  "测试假人",
		Pos:   dummyPosition{X: x, Y: 0, Z: 0},
		Level: 10,
	}
	if mover {
		dummy.Patrol = []dummyPosition{{X: x, Z: 0}, {X: x + 20, Z: 0}}
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	if !run.registerDummyAtLocked(id, dummy, time.Now()) {
		t.Fatalf("注册假人 %#x 失败", id)
	}
}

func TestDummyAdmissionBudgetAndPacing(t *testing.T) {
	run, view, _ := newDummyTestScene(t)
	// 30 只站桩 + 1 只移动假人，全部在准入半径内。
	for i := 0; i < 30; i++ {
		registerTestDummy(t, run, dummyObjectIDBase+uint32(i), float32(i), false)
	}
	registerTestDummy(t, run, dummyObjectIDBase+30, 5, true)

	run.mu.Lock()
	changed := view.dummyAdmissionLocked()
	run.mu.Unlock()
	if !changed {
		t.Fatal("首次准入应产生变化")
	}
	visible := 0
	for i := 0; i <= 30; i++ {
		if view.dummyVisibleOf(dummyObjectIDBase + uint32(i)) {
			visible++
		}
	}
	if visible != dummyViewportAdmissionPerRequest {
		t.Fatalf("首次准入 %d 只，应为限流值 %d", visible, dummyViewportAdmissionPerRequest)
	}
	// 移动优先：唯一移动假人必须已在首批准入。
	if !view.dummyVisibleOf(dummyObjectIDBase + 30) {
		t.Fatal("移动假人未获移动优先准入")
	}
	// 预算上限：反复准入后不得超过预算。
	for round := 0; round < 10; round++ {
		run.mu.Lock()
		view.dummyAdmissionLocked()
		run.mu.Unlock()
	}
	run.mu.Lock()
	count := len(view.dummyVisible)
	run.mu.Unlock()
	if count > dummyViewportObjectBudget {
		t.Fatalf("准入 %d 只超出预算 %d", count, dummyViewportObjectBudget)
	}
}

func TestDummyAdmissionHysteresisExit(t *testing.T) {
	run, view, _ := newDummyTestScene(t)
	registerTestDummy(t, run, dummyObjectIDBase, 0, false)

	run.mu.Lock()
	view.dummyAdmissionLocked()
	run.mu.Unlock()
	if !view.dummyVisibleOf(dummyObjectIDBase) {
		t.Fatal("半径内假人未准入")
	}
	// 移出 enter 半径但仍在 exit 半径内（迟滞保留）。
	view.mu.Lock()
	view.center = role.Position{X: dummyViewportRadius + dummyViewportHysteresis/2, Z: 0}
	view.mu.Unlock()
	run.mu.Lock()
	changed := view.dummyAdmissionLocked()
	run.mu.Unlock()
	if changed {
		t.Fatal("迟滞带内不应退场")
	}
	// 移出 exit 半径：退场。
	view.mu.Lock()
	view.center = role.Position{X: dummyViewportRadius + dummyViewportHysteresis*2, Z: 0}
	view.mu.Unlock()
	run.mu.Lock()
	changed = view.dummyAdmissionLocked()
	run.mu.Unlock()
	if !changed || view.dummyVisibleOf(dummyObjectIDBase) {
		t.Fatal("超出 exit 半径未退场")
	}
}

func TestFilterDummySnapshot(t *testing.T) {
	run, view, _ := newDummyTestScene(t)
	registerTestDummy(t, run, dummyObjectIDBase, 0, false)
	registerTestDummy(t, run, dummyObjectIDBase+1, 1, false)

	run.mu.Lock()
	snapshot := run.scene.Snapshot()
	run.mu.Unlock()
	if len(snapshot.Entities) < 2 {
		t.Fatalf("快照实体不足：%d", len(snapshot.Entities))
	}
	// 未准入：全部剔除。
	run.mu.Lock()
	filtered := view.filterDummySnapshot(snapshot)
	run.mu.Unlock()
	for _, entry := range filtered.Entities {
		if isDummyObjectID(uint32(entry.Entity.ID)) {
			t.Fatal("未准入假人漏进快照")
		}
	}
	// 准入一只后：仅该只在档。
	view.mu.Lock()
	view.dummyVisible = map[uint32]struct{}{dummyObjectIDBase: {}}
	view.mu.Unlock()
	run.mu.Lock()
	filtered = view.filterDummySnapshot(snapshot)
	run.mu.Unlock()
	dummyCount := 0
	for _, entry := range filtered.Entities {
		if isDummyObjectID(uint32(entry.Entity.ID)) {
			dummyCount++
			if uint32(entry.Entity.ID) != dummyObjectIDBase {
				t.Fatal("准入集外的假人漏进快照")
			}
		}
	}
	if dummyCount != 1 {
		t.Fatalf("准入假人过滤后 = %d，应为 1", dummyCount)
	}
}

func TestDummyRouteStepPingPong(t *testing.T) {
	motion := &dummyMotionState{
		pos:   worldcore.Transform{X: 0, Z: 0},
		speed: 6,
		route: []worldcore.Transform{{X: 0}, {X: 10}, {X: 20}},
	}
	seen := []int{}
	for step := 0; step < 6; step++ {
		destination, duration := dummyRouteStep(motion)
		seen = append(seen, motion.routeIndex)
		if duration < 500*time.Millisecond {
			t.Fatalf("步进时长过短：%s", duration)
		}
		motion.pos = destination
	}
	// ping-pong：1 2 1 0 1 2（索引推进后折返）。
	want := []int{1, 2, 1, 0, 1, 2}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("路点序列 = %v，应为 %v", seen, want)
		}
	}
}

func TestDummyRandomStepSnapAndDuration(t *testing.T) {
	run, _, _ := newDummyTestScene(t)
	defer func() { run.mu.Lock(); run.shutdownLocked(); run.mu.Unlock() }()
	run.mu.Lock()
	run.dummyPatrolPoints = []worldcore.Transform{
		{X: 30, Z: 0}, {X: -30, Z: 0}, {X: 0, Z: 30},
	}
	run.mu.Unlock()
	motion := &dummyMotionState{
		id:    dummyObjectIDBase,
		pos:   worldcore.Transform{X: 0, Z: 0},
		speed: dummyPatrolSpeed,
		rng:   rand.New(rand.NewSource(int64(dummyObjectIDBase))),
	}
	run.mu.Lock()
	destination, duration := run.nextDummyStepLocked(motion, time.Now())
	run.mu.Unlock()
	if horizontalDistance(motion.pos, destination) < 8 {
		t.Fatalf("随机目的地距离过近：%.2f", horizontalDistance(motion.pos, destination))
	}
	if duration < time.Second {
		t.Fatalf("随机步进时长 < 1s：%s", duration)
	}
	// 吸附语义：目的地应与池内某点一致（水平距离）。
	matched := false
	for _, point := range run.dummyPatrolPoints {
		if horizontalDistance(point, destination) < 0.01 {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("随机目的地未吸附官方 NPC 点：(%.1f,%.1f)", destination.X, destination.Z)
	}
}

func TestDummyMotionTickBroadcastsToAdmitted(t *testing.T) {
	run, view, conn := newDummyTestScene(t)
	registerTestDummy(t, run, dummyObjectIDBase, 0, true)
	// 广播按成员扇出：把视图登记为场景成员（无 player，不物化远程实体）。
	run.mu.Lock()
	run.members[2] = &sceneMember{objectID: 2, selfID: 1, view: view}
	run.mu.Unlock()

	// 准入该移动假人后驱动 tick：起步应广播 0x20 给该成员。
	view.mu.Lock()
	view.dummyVisible = map[uint32]struct{}{dummyObjectIDBase: {}}
	view.mu.Unlock()
	now := time.Now()
	run.mu.Lock()
	motion := run.dummyMotion[dummyObjectIDBase]
	motion.nextMove = now // 立即起步
	run.dummyMotionTickLocked(now)
	run.mu.Unlock()
	found0x20 := false
	for _, frame := range conn.Frames() {
		if frame[0] == 0x20 {
			found0x20 = true
		}
	}
	if !found0x20 {
		t.Fatal("起步未广播 0x20 serverMoving")
	}
	if !motion.moving {
		t.Fatal("步进状态未置 moving")
	}
	// 到达：应广播 0x1F 并转站桩。
	conn.resetFrames()
	arrival := motion.stepEnd.Add(time.Millisecond)
	run.mu.Lock()
	run.dummyMotionTickLocked(arrival)
	run.mu.Unlock()
	found0x1F := false
	for _, frame := range conn.Frames() {
		if frame[0] == 0x1F {
			found0x1F = true
		}
	}
	if !found0x1F {
		t.Fatal("到达未广播 0x1F serverLocation")
	}
	if motion.moving {
		t.Fatal("到达后应转站桩 dwell")
	}
}

func TestDummyObjectIDRange(t *testing.T) {
	if !isDummyObjectID(dummyObjectIDBase) || !isDummyObjectID(dummyObjectIDBase+31924) {
		t.Fatal("假人段判定失败")
	}
	if isDummyObjectID(1) || isDummyObjectID(0x11000002) || isDummyObjectID(forkNPCID(2)) {
		t.Fatal("非假人段误判")
	}
	// forkNPCID 直通（≥0x01000000 不再前缀），与 NPC 段永不混叠。
	if forkNPCID(dummyObjectIDBase) != dummyObjectIDBase {
		t.Fatal("假人 ID 经 forkNPCID 变形")
	}
}

// setDummyTestSnapshot 注入 DB 快照（测试直写 store；Cleanup 恢复原状）。
func setDummyTestSnapshot(t *testing.T, dummies []dummyPlayer) {
	t.Helper()
	installedDummyStore.mu.Lock()
	previousDummies, previousCards := installedDummyStore.dummies, installedDummyStore.cards
	previousLoaded, previousRows := installedDummyStore.loaded, installedDummyStore.rows
	installedDummyStore.dummies = dummies
	installedDummyStore.cards = map[int32]dummyCardEntry{}
	installedDummyStore.loaded = true
	installedDummyStore.rows = int64(len(dummies))
	installedDummyStore.mu.Unlock()
	t.Cleanup(func() {
		installedDummyStore.mu.Lock()
		installedDummyStore.dummies, installedDummyStore.cards = previousDummies, previousCards
		installedDummyStore.loaded, installedDummyStore.rows = previousLoaded, previousRows
		installedDummyStore.mu.Unlock()
	})
}

func TestDummyInstallLimitTrimAndRestore(t *testing.T) {
	defer setDummyInstallLimit(-1)
	dummies := make([]dummyPlayer, 5)
	for i := range dummies {
		dummies[i] = dummyPlayer{Name: "限量假人", Level: 1, Pos: dummyPosition{X: float32(i)}}
	}
	setDummyTestSnapshot(t, dummies)
	run, _, _ := newDummyTestScene(t)
	defer func() { run.mu.Lock(); run.shutdownLocked(); run.mu.Unlock() }()

	run.mu.Lock()
	run.installDummiesLocked(nil)
	installed := len(run.dummies)
	run.mu.Unlock()
	if installed != 5 {
		t.Fatalf("全量安装 = %d，应为 5", installed)
	}

	// 上限 2：立即裁剪到 2（确定性前缀，ID 不变）。
	setDummyInstallLimit(2)
	run.mu.Lock()
	run.applyDummyLimitLocked()
	trimmed := len(run.dummies)
	run.mu.Unlock()
	if trimmed != 2 {
		t.Fatalf("上限裁剪 = %d，应为 2", trimmed)
	}
	if run.dummies[0].id != dummyObjectIDBase || run.dummies[1].id != dummyObjectIDBase+1 {
		t.Fatalf("裁剪后 ID 漂移：%#x/%#x", run.dummies[0].id, run.dummies[1].id)
	}

	// 新上限低于已安装量时安装入口也直接生效。
	setDummyInstallLimit(3)
	run.mu.Lock()
	run.reinstallDummiesLocked()
	reinstalled := len(run.dummies)
	run.mu.Unlock()
	if reinstalled != 3 {
		t.Fatalf("按上限重装 = %d，应为 3", reinstalled)
	}

	// 恢复全量 / 禁用。
	setDummyInstallLimit(-1)
	run.mu.Lock()
	run.applyDummyLimitLocked()
	restored := len(run.dummies)
	run.mu.Unlock()
	if restored != 5 {
		t.Fatalf("恢复全量 = %d，应为 5", restored)
	}
	setDummyInstallLimit(0)
	run.mu.Lock()
	run.applyDummyLimitLocked()
	disabled := len(run.dummies)
	run.mu.Unlock()
	if disabled != 0 {
		t.Fatalf("禁用后 = %d，应为 0", disabled)
	}
}
