package fssoloworld

// gmdummy_test.go GM 训练木桩（gmdummy.go）：生成 / 物化 / 登记、被击不
// 还手、打空即死且不重生、清除与离场注销、半径模式常驻可见、gm dummy
// 命令方言。
//
// 木桩取数据产物里的官方模板 mon_cclone_mz001（缺席跳过）；战斗用例复用
// 技能 / 战斗夹具（可拨时钟 + 下行帧捕获），事件由 CombatPump 手动推进。
// 断言只钉与木桩有关的帧（伤害 / 交手 / 移除 / 还手），不钉整段帧序——
// 命中帧簇的定版在 combat*_test.go。

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// dummySceneSeq 合成场景序号：每个用例一个独立场景键（数据根里不存在该
// 摆放目录 → 空目录，只放本用例的木桩）。
var dummySceneSeq int64

// newDummyFixture 战斗夹具 + 空场景：玩家站 (100,5,200) 朝 +X，场景目录里
// 只有本用例生成的木桩。
func newDummyFixture(t *testing.T) *combatFixture {
	t.Helper()
	f := newCombatFixture(t, 1, nil)
	catalog := f.s.sceneNPCCatalog(fmt.Sprintf("dummy_test_%d", atomic.AddInt64(&dummySceneSeq, 1)))
	f.pl.npcMu.Lock()
	f.pl.npcScene, f.pl.npcVisible = catalog.dir, map[uint32]bool{}
	f.pl.npcMu.Unlock()
	f.scene = catalog.dir
	return f
}

// spawn 生成 count 只木桩（血量 hp）；数据产物没有官方木桩模板时跳过。
func (f *combatFixture) spawn(t *testing.T, hp, count int) []npcfeed.Spawn {
	t.Helper()
	spawns, err := f.s.spawnDummies(f.pl, hp, count)
	if err != nil {
		if strings.Contains(err.Error(), "缺席") {
			t.Skipf("官方木桩模板缺席: %v", err)
		}
		t.Fatal(err)
	}
	if len(spawns) != count {
		t.Fatalf("生成 %d 只, want %d", len(spawns), count)
	}
	return spawns
}

// pumpAs 推进战斗时钟，期间的下行帧转成可读串（target 标 npc）。
func (f *combatFixture) pumpAs(d time.Duration, target *npcFighter) []string {
	f.advance(d)
	f.s.CombatPump()
	return combatText(f.s, f.pl, target, f.take())
}

func dummyKey(spawn npcfeed.Spawn) *uint64 {
	key := uint64(spawn.ObjectID) | uint64(spawn.OwnerID)<<32
	return &key
}

func dummyHP(f *npcFighter) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hp
}

// assertReleased 木桩已注销：登记表、战斗体、玩家视口簿记里都没有它。
func (f *combatFixture) assertReleased(t *testing.T, spawn npcfeed.Spawn) {
	t.Helper()
	if _, ok := f.s.dynamic.spawnOf(f.scene, spawn.ObjectID); ok {
		t.Error("登记表里仍有该木桩")
	}
	// npcByKey 只查不建（fighterFor 会为查到的摆放补建战斗体）。
	if f.s.npcByKey(uint64(spawn.ObjectID)|uint64(spawn.OwnerID)<<32) != nil {
		t.Error("战斗体表里仍有该木桩")
	}
	f.pl.npcMu.Lock()
	seen := f.pl.npcVisible[spawn.ObjectID]
	f.pl.npcMu.Unlock()
	if seen {
		t.Error("玩家视口簿记里仍有该木桩")
	}
}

func (f *combatFixture) relations() int {
	st := &f.pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.live.relations)
}

// 单只在面朝方向前方（普攻射程 2 米内），多只环绕等距、首只正对朝向，
// 全部面朝玩家。
func TestDummySpot(t *testing.T) {
	near := func(got, want float64) bool { return math.Abs(got-want) < 1e-3 }
	x, z, orient := dummySpot([4]float64{100, 5, 200, math.Pi / 2}, 0, 1)
	if !near(float64(x), 101.8) || !near(float64(z), 200) || !near(float64(orient), -math.Pi/2) {
		t.Errorf("朝 +X 单只: (%.3f, %.3f, %.3f), want (101.8, 200, -π/2)", x, z, orient)
	}
	x, z, _ = dummySpot([4]float64{100, 5, 200, math.Pi}, 0, 1)
	if !near(float64(x), 100) || !near(float64(z), 198.2) {
		t.Errorf("朝 -Z 单只: (%.3f, %.3f), want (100, 198.2)", x, z)
	}
	player := [4]float64{100, 5, 200, 0} // 朝 +Z
	for i, want := range [][2]float64{{100, 203}, {103, 200}, {100, 197}, {97, 200}} {
		x, z, orient = dummySpot(player, i, 4)
		if !near(float64(x), want[0]) || !near(float64(z), want[1]) {
			t.Errorf("环绕第 %d 只: (%.3f, %.3f), want (%.1f, %.1f)", i, x, z, want[0], want[1])
		}
		// 面朝玩家：朝向单位向量 (sin, cos) 指回玩家。
		dx, dz := player[0]-float64(x), player[2]-float64(z)
		if norm := math.Hypot(dx, dz); !near(math.Sin(float64(orient)), dx/norm) || !near(math.Cos(float64(orient)), dz/norm) {
			t.Errorf("环绕第 %d 只没有面朝玩家: orient=%.3f", i, orient)
		}
	}
}

func TestDummySpawnMaterializesAndRegisters(t *testing.T) {
	f := newDummyFixture(t)
	spawns := f.spawn(t, 500000, 2)

	// 下行 = 这两条摆放的 0x28 物化帧（缺省 0x28-only），逐字节一致。
	frames := f.take()
	want, err := npcfeed.Frames(spawns, npcfeed.Form0x28Only)
	if err != nil || len(frames) != len(want) {
		t.Fatalf("物化帧 %d 条（编码 err=%v）, want %d", len(frames), err, len(want))
	}
	for i, frame := range frames {
		if frame[0] != 0x28 || frame[1] != 0x46 || !bytes.Equal(frame, want[i]) {
			t.Errorf("物化帧 %d 与 npcfeed.Frames 的输出不一致: op=%#x mode=%#x", i, frame[0], frame[1])
		}
	}
	for i, spawn := range spawns {
		if spawn.ObjectID < dynamicObjectBase || spawn.OwnerID != npcfeed.OwnerIDOf(spawn.ObjectID) {
			t.Errorf("木桩 %d 对象号不在动态段: %X/%X", i, spawn.ObjectID, spawn.OwnerID)
		}
		if i > 0 && spawn.ObjectID == spawns[0].ObjectID {
			t.Error("两只木桩对象号重复")
		}
		// 环绕 3 米、首只正对朝向（朝 +X → (103, 200)）。
		if x, z := float64(spawn.X)-100, float64(spawn.Z)-200; math.Abs(math.Hypot(x, z)-3) > 1e-3 {
			t.Errorf("木桩 %d 落点 (%.3f, %.3f) 不在 3 米环上", i, spawn.X, spawn.Z)
		}
		if !passive(spawn) || !attackable(spawn) || !spawn.Once || spawn.MaxHP != 500000 {
			t.Errorf("木桩 %d 属性不符: passive=%t attackable=%t once=%t maxHP=%d",
				i, passive(spawn), attackable(spawn), spawn.Once, spawn.MaxHP)
		}
		// 登记：本场景查得到，换场景查不到；战斗体血量 = 自定义血量。
		if _, ok := f.s.findNPCSpawn(f.scene, spawn.ObjectID); !ok {
			t.Errorf("findNPCSpawn 查不到木桩 %d", i)
		}
		if _, ok := f.s.findNPCSpawn(f.scene+"_elsewhere", spawn.ObjectID); ok {
			t.Errorf("换场景不应查到木桩 %d", i)
		}
		f.pl.npcMu.Lock()
		seen := f.pl.npcVisible[spawn.ObjectID]
		f.pl.npcMu.Unlock()
		if !seen {
			t.Errorf("玩家视口簿记没有木桩 %d", i)
		}
		if hp := dummyHP(f.s.fighterFor(f.scene, spawn)); hp != 500000 {
			t.Errorf("木桩 %d 战斗体血量 = %d, want 500000", i, hp)
		}
	}
}

// 木桩模板定版：必须是客户端眼里的真木桩——服务端产物看得到的列里，战斗
// 类、可被攻击、自己不攻击、打得死、固定不动。曾误选「测试木桩0豁免」
// （mon_cclone_mz001：CantBeKilled=1、非敌对），客户端不把它当攻击目标。
// 客户端是否敌对（NotPositive=1）产物不带这列——换模板时去客户端
// npcconfig 里核对。
func TestDummyTemplateIsRealStake(t *testing.T) {
	spawn, err := stakeSpawn(1000, 0, 0, 0, 0)
	if err != nil {
		if strings.Contains(err.Error(), "缺席") {
			t.Skipf("木桩模板缺席: %v", err)
		}
		t.Fatal(err)
	}
	if spawn.ConfigID != dummyConfigID || !passive(spawn) || !attackable(spawn) {
		t.Fatalf("木桩模板 %s 属性不符: passive=%t attackable=%t",
			spawn.ConfigID, passive(spawn), attackable(spawn))
	}
	for name, want := range map[string]string{"CantBeKilled": "0", "CantMove": "1", "Fixed": "1"} {
		if got := spawn.Template[name]; got != want {
			t.Errorf("模板 %s 的 %s = %q, want %q（测试物 / 豁免物不是木桩）", spawn.ConfigID, name, got, want)
		}
	}
	if spawn.Template["Resource"] == "" || spawn.RestAction() != "stand" || spawn.Combat == nil {
		t.Errorf("木桩模板 %s 缺模型 / 姿态 / 战斗行: resource=%q rest=%q combat=%v",
			spawn.ConfigID, spawn.Template["Resource"], spawn.RestAction(), spawn.Combat)
	}
}

// NINEYIN_DUMMY_CONFIG 换模板（真机对照用，不必重编）。
func TestDummyTemplateOverride(t *testing.T) {
	t.Setenv("NINEYIN_DUMMY_CONFIG", "mon_cclone_mz001")
	spawn, err := stakeSpawn(100, 0, 0, 0, 0)
	if err != nil {
		if strings.Contains(err.Error(), "缺席") {
			t.Skipf("对照模板缺席: %v", err)
		}
		t.Fatal(err)
	}
	if spawn.ConfigID != "mon_cclone_mz001" || !passive(spawn) {
		t.Errorf("覆盖未生效或覆写丢失: config=%s passive=%t", spawn.ConfigID, passive(spawn))
	}
}

func TestDummyTakesDamageNeverFightsBack(t *testing.T) {
	f := newDummyFixture(t)
	t.Setenv("NINEYIN_NPC_AI", "on") // NPC AI 开着也不能还手
	spawns := f.spawn(t, 100, 1)
	f.take()
	target := f.s.fighterFor(f.scene, spawns[0])

	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, dummyKey(spawns[0]), 101.8, 5, 200))
	f.take()
	joined := strings.Join(f.pumpAs(467*time.Millisecond, target), " | ")
	// 首击：交手簇 + 伤害帧（普攻公式 6，木桩不闪避、外防清零）。
	for _, want := range []string{"npc:LogicState=1", "m24(npc,i:37,s:npc_foundplayer)",
		fmt.Sprintf("m24(npc,i:17,i:6,i:0,i:0,s:zs_normal_00,s:SkillNormal,s:%s,", spawns[0].ConfigID)} {
		if !strings.Contains(joined, want) {
			t.Errorf("首击缺 %q:\n%s", want, joined)
		}
	}
	if hp := dummyHP(target); hp != 94 {
		t.Fatalf("首击后木桩 HP = %d, want 94", hp)
	}
	f.pumpAs(533*time.Millisecond, target)
	if hp := dummyHP(target); hp != 88 {
		t.Fatalf("末击后木桩 HP = %d, want 88", hp)
	}

	// 之后 12 秒：不起手、不追击、不位移，玩家不掉血；脱战只回常态——
	// 不归位，也就不回血。
	joined = strings.Join(f.pumpAs(12*time.Second, target), " | ")
	for _, unwanted := range []string{"npc:CurSkillID", "npc:State=", "loc(npc", "m24(me,i:17"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("木桩不应还手 / 移动，却出现 %q:\n%s", unwanted, joined)
		}
	}
	for _, want := range []string{"npc:LogicState=0", "me:rec114-row0"} {
		if !strings.Contains(joined, want) {
			t.Errorf("脱战缺 %q:\n%s", want, joined)
		}
	}
	if hp := dummyHP(target); hp != 88 {
		t.Errorf("脱战回血了: HP = %d, want 88", hp)
	}
	target.mu.Lock()
	pos := target.pos
	target.mu.Unlock()
	if math.Abs(pos[0]-101.8) > 1e-3 || math.Abs(pos[2]-200) > 1e-3 {
		t.Errorf("木桩位移了: %v", pos)
	}
}

func TestDummyDiesOnceAndIsReleased(t *testing.T) {
	f := newDummyFixture(t)
	spawns := f.spawn(t, 12, 1) // 两击各 6 → 打空
	f.take()
	target := f.s.fighterFor(f.scene, spawns[0])
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, dummyKey(spawns[0]), 101.8, 5, 200))
	f.take()
	f.pumpAs(467*time.Millisecond, target)
	joined := strings.Join(f.pumpAs(533*time.Millisecond, target), " | ")
	if !strings.Contains(joined, "m24(npc,i:29,") || !target.isDead() || dummyHP(target) != 0 {
		t.Fatalf("末击应致死并走击杀簇: dead=%t hp=%d\n%s", target.isDead(), dummyHP(target), joined)
	}
	if strings.Contains(joined, "op0E") {
		t.Errorf("尸体停留期内不应移除:\n%s", joined)
	}

	// 尸体停留后：移除帧下发，随即注销。
	joined = strings.Join(f.pumpAs(corpseKeep, target), " | ")
	if !strings.Contains(joined, "op0E") {
		t.Errorf("尸体停留后应发移除帧:\n%s", joined)
	}
	f.assertReleased(t, spawns[0])

	// 再过一个重生延迟：不重生（没有物化帧，战斗体仍是死亡态）。
	joined = strings.Join(f.pumpAs(npcRespawnDelay+time.Second, target), " | ")
	if strings.Contains(joined, "op28") || !target.isDead() {
		t.Errorf("木桩不应重生: dead=%t\n%s", target.isDead(), joined)
	}
}

func TestDummyClearRemovesAndDropsRelation(t *testing.T) {
	f := newDummyFixture(t)
	spawns := f.spawn(t, 100, 3)
	f.take()
	target := f.s.fighterFor(f.scene, spawns[0])
	// 三只在 3 米环上，出了普攻射程：玩家走到第一只跟前再打。
	f.pl.roleState.Position = [4]float64{float64(spawns[0].X) - 1.8, 5, float64(spawns[0].Z), math.Pi / 2}
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, float32(f.pl.roleState.Position[0]), 5, float32(f.pl.roleState.Position[2]),
		1.571, dummyKey(spawns[0]), spawns[0].X, 5, spawns[0].Z))
	f.take()
	f.pumpAs(467*time.Millisecond, target)
	if n := f.relations(); n != 1 {
		t.Fatalf("交战中应有 1 条战斗关系, got %d", n)
	}

	// 清除：每只一条移除帧，交战中的关系行一并删掉。
	if n := f.s.clearDummies(f.pl); n != 3 {
		t.Fatalf("清除 %d 只, want 3", n)
	}
	joined := strings.Join(combatText(f.s, f.pl, target, f.take()), " | ")
	if got := strings.Count(joined, "op0E"); got != 3 {
		t.Errorf("移除帧 %d 条, want 3:\n%s", got, joined)
	}
	if !strings.Contains(joined, "me:rec114-row0") {
		t.Errorf("清除交战中的木桩应删关系行:\n%s", joined)
	}
	if n := f.relations(); n != 0 {
		t.Errorf("关系行应已清空, got %d", n)
	}
	for _, spawn := range spawns {
		f.assertReleased(t, spawn)
	}
	// 再清：零条、零帧。
	if n := f.s.clearDummies(f.pl); n != 0 || len(f.take()) != 0 {
		t.Errorf("重复清除应为空操作: n=%d", n)
	}
}

func TestDummyReleasedOnSceneEnterAndLeave(t *testing.T) {
	f := newDummyFixture(t)
	spawns := f.spawn(t, 100, 1)
	f.take()

	// 换场景进场：旧场景的木桩静默注销（客户端随切图清场，不发移除帧）。
	f.s.sceneNPCEnter(f.pl, "dummy_test_elsewhere", [4]float64{1, 2, 3, 0})
	if frames := f.take(); len(frames) != 0 {
		t.Errorf("进场注销不应发帧, got %d", len(frames))
	}
	f.assertReleased(t, spawns[0])

	// 新场景再生成 → 下线：同样静默注销。
	spawns = f.spawn(t, 100, 1)
	f.take()
	f.s.playerLeave(f.pl)
	if frames := f.take(); len(frames) != 0 {
		t.Errorf("下线注销不应发帧, got %d", len(frames))
	}
	f.assertReleased(t, spawns[0])
	if f.s.players.FindByConnect(f.pl.Conn1, f.pl.Conn2) != nil {
		t.Error("下线后玩家应已离开玩家表")
	}
}

// 半径模式的视口对账只看摆放表：本人生成的木桩不在表里，不能被当成出圈
// 移除——走近走远都常驻可见。
func TestDummyPinnedInRadiusReconcile(t *testing.T) {
	f := newDummyFixture(t)
	t.Setenv("NINEYIN_NPC_RADIUS", "50")
	spawns := f.spawn(t, 100, 1)
	f.take()
	for _, pos := range [][4]float64{{100, 5, 200, 1.571}, {400, 5, 900, 0}} {
		f.s.npcReconcile(f.pl, pos)
		if frames := f.take(); len(frames) != 0 {
			t.Errorf("位置 %v 的对账不应发帧（木桩常驻可见）, got %d", pos, len(frames))
		}
	}
	f.pl.npcMu.Lock()
	seen := f.pl.npcVisible[spawns[0].ObjectID]
	f.pl.npcMu.Unlock()
	if !seen {
		t.Error("对账后木桩应仍在视口簿记里")
	}
}

func TestGmDummyCommand(t *testing.T) {
	f := newDummyFixture(t)
	f.s.players.SetupPlayerName(f.pl.Conn1, f.pl.Conn2, "木桩侠")
	gm := func(fields ...string) string {
		t.Helper()
		client, server := net.Pipe()
		defer client.Close()
		done := make(chan struct{})
		go func() {
			// gm 命令族挂在维护控制台上（echo.go）；不开监听，直接调用。
			(&EchoConsole{srv: f.s}).gmCommand(server, fields)
			server.Close()
			close(done)
		}()
		out, _ := io.ReadAll(client)
		<-done
		return string(out)
	}

	got := gm("dummy", "木桩侠", "500", "2")
	if !strings.Contains(got, ">gm dummy 木桩侠: spawned 2 stake(s) hp=500 obj=") || !strings.HasSuffix(got, ">end\r") {
		t.Fatalf("生成应答: %q", got)
	}
	f.take()
	for _, args := range [][]string{{"0"}, {"abc"}, {"500", "0"}, {"500", "21"}, {"2147483648"}} {
		if got := gm(append([]string{"dummy", "木桩侠"}, args...)...); !strings.Contains(got, ">usage: gm dummy") {
			t.Errorf("参数 %v 应回用法, got %q", args, got)
		}
	}
	// 场上已有 2 只：再生成 19 只超过上限 20。
	if got := gm("dummy", "木桩侠", "100", "19"); !strings.Contains(got, "上限") {
		t.Errorf("超过上限应拒绝: %q", got)
	}
	// 未进场（视口簿记未建）拒绝。
	f.pl.npcMu.Lock()
	f.pl.npcVisible = nil
	f.pl.npcMu.Unlock()
	if got := gm("dummy", "木桩侠", "100"); !strings.Contains(got, "尚未进入场景") {
		t.Errorf("未进场应拒绝: %q", got)
	}
	f.pl.npcMu.Lock()
	f.pl.npcVisible = map[uint32]bool{}
	f.pl.npcMu.Unlock()
	if got := gm("dummy", "木桩侠", "clear"); !strings.Contains(got, "cleared 2 stake(s)") {
		t.Errorf("清除应答: %q", got)
	}
	if got := gm("dummy", "木桩侠", "clear"); !strings.Contains(got, "cleared 0 stake(s)") {
		t.Errorf("重复清除应答: %q", got)
	}
}
