package fssoloworld

// ident.go 玩家 wire ident（PERSISTID{objectID, ownerID}）的唯一来源：0x0B
// 进场帧 [5:9]/[9:13] 宣告，属性/记录/位置/自定义对象值全部挂这一对。
//
// 方言 NINEYIN_PLAYER_IDENT：
//
//	fixed（缺省）  fork 客户端真机验证形态，恒 1-1。
//	scene          官服形态（四抓包实证 2026-11-24，详见
//	               docs/fork-era/id-system-analysis.md 文首「官服规则」）：
//	               每次进场分配一套新 ident——
//	                   obj = 场景 ID 低字节<<24 | 场景内序号（玩家与同场景
//	                         NPC 共用高字节；序号 = 静态目录之后的下一个）
//	                   own = 场景内序列号（单调递增；重建对象同 obj 换新 own）
//	               官服样本：男号同会话 3D0000F3/0611（born_single_01）→
//	               040036B5/21AD1F（漠河镇）；姑苏 0E0D0447/04CEAA68。
//
// 分配点唯一（assignPlayerIdent）：每次进场调用一次，且先于该次进场的任何
// 帧；enterContext 一次进场会被调用多次，只读不分配。官服形态未经真机
// 验证，缺省不动现网行为。

import (
	"os"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// fork 客户端方言的玩家固定身份（1-1）：stage_main 开图/记录绑定锚。
const (
	forkPlayerObjectID uint32 = 1
	forkPlayerOwnerID  uint32 = 1
)

// wireIdent 玩家 wire ident：objectID=PERSISTID.ident、ownerID=PERSISTID.serial。
type wireIdent struct {
	objectID uint32
	ownerID  uint32
}

// orFixed 零值（未分配/测试直构的 Player）回落 fork 固定身份。
func (id wireIdent) orFixed() wireIdent {
	if id == (wireIdent{}) {
		return wireIdent{forkPlayerObjectID, forkPlayerOwnerID}
	}
	return id
}

// wireIdent 当前场景的玩家 ident。
func (pl *Player) wireIdent() wireIdent {
	pl.identMu.Lock()
	defer pl.identMu.Unlock()
	return pl.ident.orFixed()
}

// setWireIdent 进场分配落定（传回原值即撤销）。
func (pl *Player) setWireIdent(id wireIdent) {
	pl.identMu.Lock()
	pl.ident = id
	pl.identMu.Unlock()
}

// entergenIdentity entergen.Context.Identity：obj/own 取会话 ident；Sequence
// = 登录会话 serial——官服 0x0B [1:5] 实证等于 0x04 的 serial@0x21，整个
// 会话不变，不是进场流水号。
func (pl *Player) entergenIdentity() entergen.Identity {
	id := pl.wireIdent()
	return entergen.Identity{ObjectID: id.objectID, OwnerID: id.ownerID, Sequence: pl.Conn2}
}

// playerPropertyFrame 玩家对象 0x10 属性帧（ident 取会话当前值）。
func playerPropertyFrame(pl *Player, props ...entergen.IndexedProperty) ([]byte, error) {
	id := pl.wireIdent()
	return entergen.ObjectPropertyFrame(id.objectID, id.ownerID, 0, props)
}

// playerIdentScene NINEYIN_PLAYER_IDENT=scene 启用官服形态进场分配（缺省
// fixed）。真机 A/B 开关。
func playerIdentScene() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_PLAYER_IDENT")), "scene")
}

// npcIdentScheme NINEYIN_NPC_IDENT=scene 时 NPC 目录走官服形态 ident
// （obj 高字节=场景 ID，与玩家共用；缺省固定段 0x0E）。真机 A/B 开关。
func npcIdentScheme() npcfeed.IdentScheme {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_NPC_IDENT")), "scene") {
		return npcfeed.IdentScene
	}
	return npcfeed.IdentFixed
}

// assignPlayerIdent 进场分配（调用点：storeLoadEnter / transferPlayer 跨场景
// 分支，各每次进场一次）。官服形态下场景号未知或序号耗尽时保持原 ident
// 并留痕。
func (s *Server) assignPlayerIdent(pl *Player) {
	if !playerIdentScene() {
		return
	}
	id, ok := s.allocSceneIdent(pl)
	if !ok {
		s.trace("player ident: client=%d 场景 %s 无场景号或序号耗尽，ident 保持不变",
			pl.Conn1, pl.roleState.Resource)
		return
	}
	pl.setWireIdent(id)
}

// allocSceneIdent 官服形态分配：计数器挂在场景目录缓存上（npcCatalog.ident，
// 同场景玩家与 NPC 共用），持目录锁取号。每次进场占 1 个序号单位、序列号 +1。
func (s *Server) allocSceneIdent(pl *Player) (wireIdent, bool) {
	dir := s.resolveSceneDir(pl.roleState.Scene, pl.roleState.Resource)
	catalog := s.sceneNPCCatalog(dir)
	npcCatalogMu.Lock()
	objectID, ownerID, ok := catalog.ident.Alloc(1)
	sceneID := catalog.sceneID
	npcCatalogMu.Unlock()
	if !ok {
		return wireIdent{}, false
	}
	s.trace("player ident: client=%d scene=%s(%d) obj=%08X own=%08X",
		pl.Conn1, dir, sceneID, objectID, ownerID)
	return wireIdent{objectID: objectID, ownerID: ownerID}, true
}
