package fssoloworld

// ident_test.go 玩家 wire ident 单一来源与进场分配（官服形态开关）+ 会话
// serial 对齐。官服锚点（四抓包，2026-11-24；详见
// docs/fork-era/id-system-analysis.md 文首「官服规则」）：obj 高 8 位=场景
// ID 低字节、低 24 位=场景内序号，own=场景内序列号；0x0B 的 [1:5]=0x04 的
// serial@0x21=登录会话 serial。构建产物在仓根 testdata/ini（场景目录与
// scenes.ini），缺数据根时场景相关用例跳过。

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// resetNpcCatalog 测试独占场景目录缓存（进程级；进场分配计数器挂在上面）。
func resetNpcCatalog(t *testing.T) {
	t.Helper()
	npcCatalogMu.Lock()
	saved := npcCatalogCache
	npcCatalogCache = map[string]*npcCatalog{}
	npcCatalogMu.Unlock()
	t.Cleanup(func() {
		npcCatalogMu.Lock()
		npcCatalogCache = saved
		npcCatalogMu.Unlock()
	})
}

// sceneSpawnCount 场景摆放目录的物化条数（0=数据根缺席/无摆放）。
func (s *Server) sceneSpawnCount(dir string) int {
	return len(s.sceneNPCCatalog(dir).spawns)
}

// sceneIdent 官服形态期望值：场景 sceneID、count 个静态 NPC 之后的进场玩家。
func sceneIdent(sceneID, count int) wireIdent {
	return wireIdent{
		objectID: npcfeed.SceneObjectIDAt(sceneID, 2+5*count),
		ownerID:  uint32(7 + 5*count),
	}
}

// frameIdent 取未压缩对象帧的 ident；视图对象/全局域（sub/flag≠0）与非
// 对象帧返回 ok=false。
func frameIdent(f []byte) (objectID, ownerID uint32, ok bool) {
	le32 := func(off int) uint32 { return binary.LittleEndian.Uint32(f[off:]) }
	switch f[0] {
	case 0x0B:
		return le32(5), le32(9), true
	case 0x0D, 0x0E, 0x1F:
		return le32(1), le32(5), true
	case 0x10, 0x11, 0x12, 0x13, 0x14:
		if f[1] != 0 {
			return 0, 0, false
		}
		return le32(2), le32(6), true
	}
	return 0, 0, false
}

// assertIdentChain 帧序里每个玩家域对象帧都挂 want；0x0B 的 [1:5] 须等于
// serial。返回是否见到 0x0B（重入簇不重发 0x0B，由调用方判定期望）。
func assertIdentChain(t *testing.T, frames [][]byte, want wireIdent, serial uint32, what string) bool {
	t.Helper()
	sawEntry := false
	for i, f := range frames {
		objectID, ownerID, ok := frameIdent(f)
		if !ok {
			continue
		}
		if objectID != want.objectID || ownerID != want.ownerID {
			t.Fatalf("%s 帧 #%d op=0x%02X ident=%08X/%08X，应为 %08X/%08X",
				what, i, f[0], objectID, ownerID, want.objectID, want.ownerID)
		}
		if f[0] == 0x0B {
			sawEntry = true
			if got := binary.LittleEndian.Uint32(f[1:]); got != serial {
				t.Fatalf("%s 0x0B [1:5]=%#x，应为登录会话 serial %#x", what, got, serial)
			}
		}
	}
	return sawEntry
}

// TestPlayerIdentFixedDefault 缺省方言：未分配的会话回落 fork 固定身份
// 1-1（assignPlayerIdent 空转），enterContext/属性帧同源，Sequence 取登录
// 会话 serial。
func TestPlayerIdentFixedDefault(t *testing.T) {
	s, pl := newCustomTestServer(t)
	pl.Conn2 = 0x94B
	s.assignPlayerIdent(pl)
	if got := pl.wireIdent(); got != (wireIdent{forkPlayerObjectID, forkPlayerOwnerID}) {
		t.Fatalf("fixed ident=%+x", got)
	}
	ctx := s.enterContext(pl, false)
	want := entergen.Identity{ObjectID: 1, OwnerID: 1, Sequence: 0x94B}
	if ctx.Identity != want {
		t.Fatalf("enterContext identity=%+x want %+x", ctx.Identity, want)
	}
	// 帧构造器：会话构造取会话 ident，零值直构回落固定身份。
	if got := newFrameBuilder(pl).playerIdent(); got != (wireIdent{1, 1}) {
		t.Fatalf("frameBuilder ident=%+x", got)
	}
	b := &frameBuilder{}
	b.prop("OperateMode", entergen.Int32Value(1))
	if len(b.frames) != 1 || b.frames[0][0] != 0x10 {
		t.Fatalf("prop frame: % x", opOf(b.frames))
	}
	if objectID, ownerID, _ := frameIdent(b.frames[0]); objectID != 1 || ownerID != 1 {
		t.Fatalf("零值 frameBuilder ident=%x/%x want 1/1", objectID, ownerID)
	}
}

// TestEnterChainIdentConsistency 进场链全链同源：0x0B 宣告的 ident 与其
// 后 0x0D/0x10/0x11/0x1F 等玩家域帧逐帧一致。fixed=恒 1-1；scene=官服形态
// （born03=scenes.ini [3] → 0x03 段，序号=静态目录之后）。
func TestEnterChainIdentConsistency(t *testing.T) {
	for _, tc := range []struct {
		name, env string
		scene     bool
	}{
		{name: "fixed"},
		{name: "scene", env: "scene", scene: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetNpcCatalog(t)
			t.Setenv("NINEYIN_PLAYER_IDENT", tc.env)
			s, pl := newCustomTestServer(t)
			// 建号默认已进新手副本，本用例对拍 born03 进场链 ident
			//（段号 3）——显式落 born03 模拟存量号。
			pl.roleState.Scene, pl.roleState.Resource = `ini\scene\born03_YanYuZhuang`, "born03"
			n := s.sceneSpawnCount("born03")
			if tc.scene && n == 0 {
				t.Skip("数据根缺席（testdata/ini）")
			}
			want := wireIdent{forkPlayerObjectID, forkPlayerOwnerID}
			if tc.scene {
				want = sceneIdent(3, n)
			}
			frames := captureFrames(t)
			p := make([]byte, 12)
			binary.LittleEndian.PutUint32(p[0:], 7)
			binary.LittleEndian.PutUint32(p[4:], 1)
			binary.LittleEndian.PutUint32(p[8:], 1)
			if !s.storeLoadEnter(0, p) {
				t.Fatal("storeLoadEnter rejected")
			}
			if got := pl.wireIdent(); got != want {
				t.Fatalf("进场后 ident=%08X/%08X want %08X/%08X（静态 NPC %d 条）",
					got.objectID, got.ownerID, want.objectID, want.ownerID, n)
			}
			if !assertIdentChain(t, *frames, want, pl.Conn2, "登录进场") {
				t.Fatalf("登录进场缺 0x0B（前 40 帧 ops=% x）", opOf((*frames)[:min(40, len(*frames))]))
			}
			// 同场景第二个玩家：序号/序列号续分（fixed 恒 1-1）。
			pl2 := s.Players().AddPlayer(8, 2, "acct2")
			pl2.roleState, pl2.haveRoleState = pl.roleState, true
			s.assignPlayerIdent(pl2)
			want2 := want
			if tc.scene {
				want2 = wireIdent{want.objectID + 1, want.ownerID + 1}
			}
			if got := pl2.wireIdent(); got != want2 {
				t.Fatalf("第二个玩家 ident=%08X/%08X want %08X/%08X", got.objectID, got.ownerID, want2.objectID, want2.ownerID)
			}
		})
	}
}

// TestSceneSwitchReassignsIdent 切图换 ident（官服：男号同会话 3D0000F3
// → 040036B5）：born03 [3] → born01 [4]，0x0B 宣告新 ident，重入簇帧与
// 之后的实时帧全部改挂新 ident。
func TestSceneSwitchReassignsIdent(t *testing.T) {
	resetNpcCatalog(t)
	t.Setenv("NINEYIN_PLAYER_IDENT", "scene")
	s, pl := newCustomTestServer(t)
	if s.sceneSpawnCount("born03") == 0 {
		t.Skip("数据根缺席（testdata/ini）")
	}
	prevGap := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prevGap })

	frames := captureFrames(t)
	dest, ok := resolveSceneDestination("born01")
	if !ok {
		t.Fatal("born01 resolve failed")
	}
	dest.Position = [4]float64{990, 6.66, 532, 4.7}
	if err := s.transferPlayer(pl, dest, "test"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	// 切换在途（gap 内）ident 未变；gap 后 0x0B/0x0D 生成成功才落定。
	deadline := time.Now().Add(2 * time.Second)
	for s.pendingSceneSwitch(pl) == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.pendingSceneSwitch(pl) == nil {
		t.Fatal("awaitReady not armed after gap")
	}
	want := sceneIdent(4, s.sceneSpawnCount(s.resolveSceneDir(dest.Config, dest.Resource)))
	if got := pl.wireIdent(); got != want {
		t.Fatalf("切图后 ident=%08X/%08X want %08X/%08X", got.objectID, got.ownerID, want.objectID, want.ownerID)
	}
	if !assertIdentChain(t, *frames, want, pl.Conn2, "切图进场") {
		t.Fatal("切图进场缺 0x0B")
	}
	mark := len(*frames)
	s.clientReady(pl)
	if s.sceneSwitchInFlight(pl) {
		t.Fatal("switch state not cleared after reentry ready")
	}
	saw := assertIdentChain(t, (*frames)[mark:], want, pl.Conn2, "切图重入簇")
	if saw {
		t.Fatal("重入簇不应再发 0x0B")
	}
	// 之后的实时帧（技能 MP 单属性）挂新 ident。
	s.sendFrames(pl, mpFrame(pl.wireIdent(), 50))
	if objectID, ownerID, ok := frameIdent((*frames)[len(*frames)-1]); !ok || objectID != want.objectID || ownerID != want.ownerID {
		t.Fatalf("切图后实时帧 ident=%x/%x want %x/%x", objectID, ownerID, want.objectID, want.ownerID)
	}
	// 同场景传送不换 ident（官服同场景无 0x0C/0x0B）。
	same := dest
	same.Position = [4]float64{1, 2, 3, 4}
	if err := s.transferPlayer(pl, same, "test"); err != nil {
		t.Fatalf("same-scene transfer: %v", err)
	}
	if got := pl.wireIdent(); got != want {
		t.Fatalf("同场景传送后 ident 变为 %08X/%08X", got.objectID, got.ownerID)
	}
}

// TestSceneIdentUnknownScene 场景不在 scenes.ini：无场景号可取，ident 保持
// 不变（不发出半套官服形态）。
func TestSceneIdentUnknownScene(t *testing.T) {
	resetNpcCatalog(t)
	t.Setenv("NINEYIN_PLAYER_IDENT", "scene")
	s, pl := newCustomTestServer(t)
	pl.roleState.Scene, pl.roleState.Resource = "ini\\scene\\no_such_scene_xyz", "no_such_scene_xyz"
	before := pl.wireIdent()
	s.assignPlayerIdent(pl)
	if got := pl.wireIdent(); got != before {
		t.Fatalf("未知场景 ident 被改为 %08X/%08X", got.objectID, got.ownerID)
	}
}

// TestNpcCatalogSceneIdent NINEYIN_NPC_IDENT=scene：NPC 目录走官服形态
// （姑苏 city02=scenes.ini [14] → 0x0E 段，与 npc.pcapng 官方 NPC 同段；
// own=序号+5），与同场景进场玩家共用高字节；缺省仍是固定段。场景号与
// 方言无关照常解析（供玩家进场分配）。
func TestNpcCatalogSceneIdent(t *testing.T) {
	resetNpcCatalog(t)
	s := newNpcTestServer(t)
	fixed := s.sceneNPCCatalog("city02_SuZhou")
	if len(fixed.spawns) == 0 {
		t.Skip("数据根缺席（testdata/ini）")
	}
	if fixed.sceneID != 14 || fixed.ident != nil {
		t.Fatalf("缺省目录 sceneID=%d ident=%v want 14/nil", fixed.sceneID, fixed.ident)
	}
	if got := fixed.spawns[0].ObjectID; got != npcfeed.ObjectIDAt(0) {
		t.Fatalf("缺省方言首条 ident=%x（应保持固定段）", got)
	}

	resetNpcCatalog(t)
	t.Setenv("NINEYIN_NPC_IDENT", "scene")
	scene := newNpcTestServer(t).sceneNPCCatalog("city02_SuZhou")
	n := len(scene.spawns)
	first, last := scene.spawns[0], scene.spawns[n-1]
	if scene.sceneID != 14 || scene.ident == nil {
		t.Fatalf("官服形态目录 sceneID=%d ident=%v", scene.sceneID, scene.ident)
	}
	if first.ObjectID != 0x0E000002 || first.OwnerID != 7 ||
		last.ObjectID != npcfeed.SceneObjectIDAt(14, 2+5*(n-1)) || last.OwnerID != uint32(7+5*(n-1)) {
		t.Fatalf("官服形态目录首/末 ident=%x/%x … %x/%x", first.ObjectID, first.OwnerID, last.ObjectID, last.OwnerID)
	}
	// own 取自目录而非由 obj 反推（固定段公式对不上）。
	if scene.ownerOf(first.ObjectID) != first.OwnerID || scene.ownerOf(first.ObjectID) == npcfeed.OwnerIDOf(first.ObjectID) {
		t.Fatalf("ownerOf(%x)=%x", first.ObjectID, scene.ownerOf(first.ObjectID))
	}
	// 物化帧体首 ident 与目录一致（0x28 明文体 [10:18]）。
	frames, _ := newNpcTestServer(t).sceneNPCViewFrames("city02_SuZhou", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(frames) != n || frames[0][0] != 0x28 || frames[0][1] != 0x46 {
		t.Fatalf("物化帧 n=%d head=%x", len(frames), opOf(frames[:min(2, len(frames))]))
	}
	for i, f := range frames {
		objectID := binary.LittleEndian.Uint32(f[10:])
		ownerID := binary.LittleEndian.Uint32(f[14:])
		if objectID != scene.spawns[i].ObjectID || ownerID != scene.spawns[i].OwnerID {
			t.Fatalf("物化帧 #%d ident=%x/%x want %x/%x", i, objectID, ownerID, scene.spawns[i].ObjectID, scene.spawns[i].OwnerID)
		}
	}
	// 同场景进场玩家：共用 0x0E 高字节，序号在静态目录之后（不与任何 NPC 撞号）。
	t.Setenv("NINEYIN_PLAYER_IDENT", "scene")
	player := newNpcTestServer(t)
	pl := player.Players().AddPlayer(21, 1, "acct")
	pl.roleState.Scene, pl.roleState.Resource = `ini\scene\city02_SuZhou`, "city02"
	player.assignPlayerIdent(pl)
	want := sceneIdent(14, n)
	if got := pl.wireIdent(); got != want || got.objectID>>24 != first.ObjectID>>24 {
		t.Fatalf("姑苏玩家 ident=%08X/%08X want %08X/%08X", got.objectID, got.ownerID, want.objectID, want.ownerID)
	}
}

// TestRolesFrameSessionSerial 0x04 的 serial@0x21=登录会话 serial（官服
// 抓包：男号 0x94B、女号 0x34EFE），与 0x0B 的 [1:5] 同源。
func TestRolesFrameSessionSerial(t *testing.T) {
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.Local)
	for _, roles := range [][]roleEntry{nil, {{name: "赵六"}}} {
		f := rolesFrame(0x94B, now, roles)
		if f[0] != 0x04 {
			t.Fatalf("op=%#x", f[0])
		}
		if got := binary.LittleEndian.Uint32(f[0x21:]); got != 0x94B {
			t.Fatalf("serial@0x21=%#x want 0x94b", got)
		}
		if got := binary.LittleEndian.Uint32(f[0x25:]); got != uint32(len(roles)) {
			t.Fatalf("roleCount@0x25=%d want %d", got, len(roles))
		}
	}
	s, pl := newCustomTestServer(t)
	pl.Conn2 = 0x34EFE
	entry, err := entergen.EntrySceneFrame(s.enterContext(pl, false).Identity, "cfg", "res")
	if err != nil {
		t.Fatalf("EntrySceneFrame: %v", err)
	}
	if got := binary.LittleEndian.Uint32(entry[1:]); got != 0x34EFE {
		t.Fatalf("0x0B [1:5]=%#x want 会话 serial 0x34efe", got)
	}
}
