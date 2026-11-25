package fssoloworld

// gmdummy.go GM 训练木桩：在玩家面前运行时生成纯沙包 NPC，用于测输出 /
// 测技能（gm dummy <who> [血量|clear] [数量]）。
//
// 数据链与静态 NPC 完全相同（模板行 + 战斗行 → npcfeed.Spawn → 0x28 物化
// 帧 → 命中 / 伤害 / 击杀走 combat*.go），区别只在四处：
//
//	对象号   现分（静态段之后的动态段），登记进 dynamic 登记表——
//	         findNPCSpawn 与范围技索敌从登记表补查；
//	可见性   只对生成它的玩家物化（登记进该玩家的 npcVisible）；离场 /
//	         下线 / 清除即注销；
//	不还手   官方行本就 CantAttack=1（passive，另钉死覆写）：被击只有交手
//	         帧与伤害，不追击、不起手、脱战也不归位回血；
//	一次性   Spawn.Once：自定义血量打空即死，尸体消失后注销，不重生。
//
// 模板取哪一根要看客户端**当前**数据里的敌对 / 可击杀列：客户端的血条
// 颜色、能否攻击、模型都按 ConfigID 查它自己的表。选错的教训见
// dummyConfigID。

import (
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

const (
	// dummyConfigID 官方新手练习木桩（AttackNpc，模型 npc\itemnpc004，烟雨庄
	// 里就摆着它）：客户端当前数据里它 NotPositive=1（敌对——自动选敌认）、
	// CantBeKilled=0、CantAttack=1、固定不动。
	//
	// ⚠️ 别换回 mon_cclone_mz001：参考端数据里它叫「蜃境木桩（一阶）」，现行
	// 客户端表里是「测试木桩0豁免」——NotPositive=0、CantBeKilled=1、
	// CantFight=1。客户端按 ConfigID 查它自己的表，不把这种对象当可攻击
	// 目标，按攻击不会自动选中（2026-11-26 真机）。
	dummyConfigID = "funnpcborn003043"
	// dummyDefaultHP 缺省血量。
	dummyDefaultHP = 100000
	// dummySpawnMax 单次生成上限，也是单个玩家场上保有上限。
	dummySpawnMax = 20
	// dummyAhead 单只木桩离玩家的距离（米，面朝方向正前方）：普攻射程 2
	// 米，留在射程内；木桩碰撞半径缩到 0.5，与玩家的 0.9 不重叠。
	dummyAhead = 1.8
	// dummyRingRadius 多只木桩环绕玩家的半径（米，给范围技用）。
	dummyRingRadius = 3.0
	// dynamicObjectBase 固定段方言的动态对象号起点：静态 NPC 自
	// 0x0E000102 起按目录序连排（最大的场景不到 7 千条），动态段留出 6.5
	// 万号。高字节 0x0E = 客户端交互管线路由的场景段（npcfeed/join.go）。
	dynamicObjectBase uint32 = 0x0E010000
	// dynamicObjectSpan 固定段动态对象号容量。
	dynamicObjectSpan uint32 = 0x10000
)

// dummyTemplateOverrides 木桩的模板列覆写：不还手（官方行本就 CantAttack=1，
// 这里钉死防数据快照差异）、头顶名、碰撞半径（官方行 1.0，离玩家 1.8 米时会
// 重叠）。
var dummyTemplateOverrides = map[string]string{
	"CantAttack":    "1",
	"Name":          "训练木桩",
	"CollideRadius": "0.5",
}

// dummyCombatOverrides 木桩的战斗行列覆写：1 级、不闪避、外防清零——打出的
// 数字就是面板公式值。
var dummyCombatOverrides = map[string]string{
	"Level":             "1",
	"Dodge":             "0",
	"PhysicsShieldBase": "0",
}

// dummyTemplateID 木桩用的模板 ID：NINEYIN_DUMMY_CONFIG 覆盖——真机上换
// 别的木桩模板对照不必重编。
func dummyTemplateID() string {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_DUMMY_CONFIG")); v != "" {
		return v
	}
	return dummyConfigID
}

// ---- 登记表 ----

// dynamicObject 登记表里的一条运行时对象。
type dynamicObject struct {
	spawn npcfeed.Spawn
	owner uint64 // 生成它的玩家（连接键 persistKey）
	scene string // 摆放目录名（与 Player.npcScene 同口径）
}

// dynamicRegistry 运行时对象登记表（零值可用，首条登记时建表）。方法都
// 只在持锁期间读写表，不回调出去——调用方可以拿着别的锁调它。
type dynamicRegistry struct {
	mu      sync.Mutex
	byID    map[uint32]*dynamicObject
	byOwner map[uint64]map[uint32]bool
	next    uint32 // 固定段方言的下一个动态对象号（0 = 未起算）
}

func (r *dynamicRegistry) add(d *dynamicObject) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byID == nil {
		r.byID = map[uint32]*dynamicObject{}
		r.byOwner = map[uint64]map[uint32]bool{}
	}
	r.byID[d.spawn.ObjectID] = d
	if r.byOwner[d.owner] == nil {
		r.byOwner[d.owner] = map[uint32]bool{}
	}
	r.byOwner[d.owner][d.spawn.ObjectID] = true
}

// spawnOf 按对象号查（scene 不符 = 不在这个场景）。
func (r *dynamicRegistry) spawnOf(scene string, objectID uint32) (npcfeed.Spawn, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d := r.byID[objectID]; d != nil && d.scene == scene {
		return d.spawn, true
	}
	return npcfeed.Spawn{}, false
}

// ownedSet 玩家的全部对象号。
func (r *dynamicRegistry) ownedSet(owner uint64) map[uint32]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[uint32]bool, len(r.byOwner[owner]))
	for id := range r.byOwner[owner] {
		out[id] = true
	}
	return out
}

// ownedSpawns 玩家在 scene 里的全部摆放。
func (r *dynamicRegistry) ownedSpawns(owner uint64, scene string) []npcfeed.Spawn {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []npcfeed.Spawn
	for id := range r.byOwner[owner] {
		if d := r.byID[id]; d != nil && d.scene == scene {
			out = append(out, d.spawn)
		}
	}
	return out
}

// remove 摘除并返回（不在表里返回 nil）。
func (r *dynamicRegistry) remove(objectID uint32) *dynamicObject {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.byID[objectID]
	if d == nil {
		return nil
	}
	delete(r.byID, objectID)
	delete(r.byOwner[d.owner], objectID)
	if len(r.byOwner[d.owner]) == 0 {
		delete(r.byOwner, d.owner)
	}
	return d
}

// allocFixed 固定段方言取号（own 与静态 NPC 同一纯函数）。
func (r *dynamicRegistry) allocFixed() (uint32, uint32, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.next == 0 {
		r.next = dynamicObjectBase
	}
	if r.next >= dynamicObjectBase+dynamicObjectSpan {
		return 0, 0, fmt.Errorf("动态对象号耗尽（重启 world 进程）")
	}
	objectID := r.next
	r.next++
	return objectID, npcfeed.OwnerIDOf(objectID), nil
}

// ---- 生成 ----

// allocDynamicIdent 给 scene 的一条运行时对象取对象号：官服形态（目录带
// 分配器）与进场玩家共用场景内序号；固定段方言走动态段。
func (s *Server) allocDynamicIdent(scene string) (uint32, uint32, error) {
	if catalog := s.sceneNPCCatalog(scene); catalog.ident != nil {
		npcCatalogMu.Lock()
		objectID, ownerID, ok := catalog.ident.Alloc(1)
		npcCatalogMu.Unlock()
		if ok {
			return objectID, ownerID, nil
		}
	}
	return s.dynamic.allocFixed()
}

// stakeSpawn 合成一条木桩摆放（对象号未定）。
func stakeSpawn(hp int32, x, y, z, orient float32) (npcfeed.Spawn, error) {
	root := npcfeed.ResolveRoot()
	if root == "" {
		return npcfeed.Spawn{}, fmt.Errorf("NPC 数据根缺席")
	}
	templates, combat, err := npcfeed.LoadStore(root)
	if err != nil {
		return npcfeed.Spawn{}, err
	}
	configID := dummyTemplateID()
	spawn, err := npcfeed.SpawnFromTemplate(templates, combat, configID,
		dummyTemplateOverrides, dummyCombatOverrides, hp, x, y, z, orient)
	if err != nil {
		return npcfeed.Spawn{}, err
	}
	if !attackable(spawn) {
		return npcfeed.Spawn{}, fmt.Errorf("模板 %s 不可被攻击（script=%s CantBeAttack=%s）",
			configID, spawn.Script, spawn.Template["CantBeAttack"])
	}
	return spawn, nil
}

// dummySpot 第 i 只（共 count 只）木桩的落点与朝向：单只在玩家面朝方向前方
// dummyAhead 米；多只环绕玩家等距分布，首只正对玩家朝向。全部面朝玩家。
// 朝向约定 atan2(dx, dz)（与技能作用范围同口径）。
func dummySpot(player [4]float64, i, count int) (x, z, orient float32) {
	radius, angle := float64(dummyAhead), player[3]
	if count > 1 {
		radius = dummyRingRadius
		angle = player[3] + float64(i)*2*math.Pi/float64(count)
	}
	x = float32(player[0] + radius*math.Sin(angle))
	z = float32(player[2] + radius*math.Cos(angle))
	return x, z, float32(math.Atan2(player[0]-float64(x), player[2]-float64(z)))
}

// spawnDummies 在玩家面前生成 count 只木桩（血量 hp），物化帧即时下发。
// 返回已登记的摆放。
func (s *Server) spawnDummies(pl *Player, hp, count int) ([]npcfeed.Spawn, error) {
	if count < 1 || count > dummySpawnMax {
		return nil, fmt.Errorf("数量须在 1~%d 之间", dummySpawnMax)
	}
	owner := persistKey(pl.Conn1, pl.Conn2)
	pl.npcMu.Lock()
	scene, pos := pl.npcScene, pl.roleState.Position
	entered := pl.npcVisible != nil
	pl.npcMu.Unlock()
	if scene == "" || !entered {
		return nil, fmt.Errorf("角色尚未进入场景")
	}
	if live := len(s.dynamic.ownedSet(owner)); live+count > dummySpawnMax {
		return nil, fmt.Errorf("场上木桩已达上限 %d 只（先清除）", dummySpawnMax)
	}
	spawns := make([]npcfeed.Spawn, 0, count)
	for i := 0; i < count; i++ {
		x, z, orient := dummySpot(pos, i, count)
		spawn, err := stakeSpawn(int32(hp), x, float32(pos[1]), z, orient)
		if err != nil {
			return nil, err
		}
		if spawn.ObjectID, spawn.OwnerID, err = s.allocDynamicIdent(scene); err != nil {
			return nil, err
		}
		spawns = append(spawns, spawn)
	}
	frames, err := npcfeed.Frames(spawns, npcForm())
	if err != nil {
		return nil, err
	}
	// 登记与视口簿记在同一把 npcMu 下落地：期间玩家若已换场景（进场会
	// 注销其全部木桩）则不落地。
	pl.npcMu.Lock()
	if pl.npcScene != scene || pl.npcVisible == nil {
		pl.npcMu.Unlock()
		return nil, fmt.Errorf("角色已离开场景")
	}
	for i := range spawns {
		s.dynamic.add(&dynamicObject{spawn: spawns[i], owner: owner, scene: scene})
		pl.npcVisible[spawns[i].ObjectID] = true
	}
	pl.npcMu.Unlock()
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
	s.trace("gm dummy: client=%d scene=%s spawned=%d hp=%d config=%s obj=%X",
		pl.Conn1, scene, count, hp, spawns[0].ConfigID, spawns[0].ObjectID)
	return spawns, nil
}

// ---- 清除与注销 ----

// clearDummies 清除玩家场上的全部木桩（GM 清除）：木桩还在眼前，先发 0x0E
// 再注销。返回清除条数。
func (s *Server) clearDummies(pl *Player) int {
	cleared := 0
	for id := range s.dynamic.ownedSet(persistKey(pl.Conn1, pl.Conn2)) {
		if s.releaseDynamic(id, true) {
			cleared++
		}
	}
	return cleared
}

// releasePlayerDummies 玩家离场 / 下线：木桩随场景一起消失，不发移除帧。
func (s *Server) releasePlayerDummies(pl *Player) {
	for id := range s.dynamic.ownedSet(persistKey(pl.Conn1, pl.Conn2)) {
		s.releaseDynamic(id, false)
	}
}

// afterNpcDeath 排定死亡收尾：尸体停留后移除（npcCorpseRemove）；一次性
// 对象（Spawn.Once）随后注销，其余隔重生延迟复活。
func (s *Server) afterNpcDeath(f *npcFighter) {
	s.after(corpseKeep, func() { s.npcCorpseRemove(f) })
	if f.spawn.Once {
		s.after(corpseKeep, func() { s.releaseDynamic(f.spawn.ObjectID, false) })
		return
	}
	s.after(corpseKeep+npcRespawnDelay, func() { s.npcRespawn(f) })
}

// releaseDynamic 注销一条运行时对象：登记表、战斗体、玩家视口簿记、玩家
// 身上对它的战斗关系一并清掉。remove=true 时先给还看得见它的玩家发 0x0E
// （木桩活着被清除；尸体的移除帧 npcCorpseRemove 已发过，不重发）。对象
// 不在登记表里（已注销）返回 false。
func (s *Server) releaseDynamic(objectID uint32, remove bool) bool {
	d := s.dynamic.remove(objectID)
	if d == nil {
		return false
	}
	catalog := s.sceneNPCCatalog(d.scene)
	catalog.combatMu.Lock()
	f := catalog.combat[objectID]
	delete(catalog.combat, objectID)
	if f != nil {
		s.combatMu.Lock()
		delete(s.fighters, f.handle)
		s.combatMu.Unlock()
	}
	catalog.combatMu.Unlock()
	if f != nil {
		s.retireFighter(f)
	}
	frame := entergen.SceneRemoveObjectFrame(objectID, d.spawn.OwnerID)
	for _, pl := range s.players.Snapshot() {
		pl.npcMu.Lock()
		seen := pl.npcScene == d.scene && pl.npcVisible != nil && pl.npcVisible[objectID]
		if seen {
			delete(pl.npcVisible, objectID)
		}
		pl.npcMu.Unlock()
		if seen && remove {
			s.sendToClient(pl, frame)
		}
	}
	s.dropFightRelation(d.owner, objectID, d.spawn.OwnerID)
	s.trace("dynamic object %X released (remove=%t)", objectID, remove)
	return true
}

// retireFighter 战斗体下线：此后一律按死亡处理（在途的决策 / 命中 / 硬直
// 回调见 dead 即返回，buff 回调查不到句柄），起手与 buff 占用的辅助对象
// 摘除。
func (s *Server) retireFighter(f *npcFighter) {
	f.mu.Lock()
	var auxIDs []uint64
	if f.cast != nil {
		auxIDs = append(auxIDs, f.cast.aux)
	}
	for _, buff := range f.buffs {
		if buff.handle != 0 {
			auxIDs = append(auxIDs, buff.handle)
		}
	}
	f.dead, f.cast, f.buffs, f.logic, f.target = true, nil, nil, logicNormal, 0
	f.castSeq++ // 死前的余击作废
	f.stunSeq++ // 死前的硬直收尾回调作废
	f.mu.Unlock()
	if len(auxIDs) == 0 {
		return
	}
	s.combatMu.Lock()
	for _, id := range auxIDs {
		delete(s.aux, id)
	}
	s.combatMu.Unlock()
}

// dropFightRelation 玩家身上与该对象的战斗关系行（木桩在交战中被清除）。
func (s *Server) dropFightRelation(owner uint64, objectID, ownerID uint32) {
	pl := s.playerByKey(owner)
	if pl == nil {
		return
	}
	key := uint64(objectID) | uint64(ownerID)<<32
	st := &pl.combat
	st.mu.Lock()
	row := relationIndex(st.live.relations, key)
	if row >= 0 {
		st.live.relations = append(st.live.relations[:row], st.live.relations[row+1:]...)
	}
	st.mu.Unlock()
	if row < 0 {
		return
	}
	ident := pl.wireIdent().orFixed()
	s.sendToClient(pl, entergen.RecordDelRowFrame(ident.objectID, ident.ownerID,
		clientdata.MustRecordIndexByName("fight_relation_rec"), uint16(row)))
}

// skillTargetSpawns 范围技的索敌范围：场景摆放表 + 本人生成的运行时对象
// （别的玩家的木桩对他不可见，不参与）。没有运行时对象时直接给摆放表
// （零拷贝）。
func (s *Server) skillTargetSpawns(pl *Player, scene string) []npcfeed.Spawn {
	spawns := s.sceneNPCCatalog(scene).spawns
	own := s.dynamic.ownedSpawns(persistKey(pl.Conn1, pl.Conn2), scene)
	if len(own) == 0 {
		return spawns
	}
	return append(append([]npcfeed.Spawn(nil), spawns...), own...)
}

// pinDynamics 半径模式的视口对账只看摆放表：本人生成的运行时对象不在表
// 里，会被当成「已出圈」——从移除项里摘掉并保持可见。
func (s *Server) pinDynamics(pl *Player, removes []uint32, nowVisible map[uint32]bool) []uint32 {
	own := s.dynamic.ownedSet(persistKey(pl.Conn1, pl.Conn2))
	if len(own) == 0 {
		return removes
	}
	kept := removes[:0]
	for _, id := range removes {
		if own[id] {
			nowVisible[id] = true
			continue
		}
		kept = append(kept, id)
	}
	return kept
}

// ---- GM 子命令 ----

// gmDummyCommand gm dummy <who> [血量|clear] [数量]：在玩家面前生成训练
// 木桩；clear 清除其场上全部木桩。
func (s *Server) gmDummyCommand(nc net.Conn, pl *Player, who string, fields []string) {
	if len(fields) >= 3 && strings.EqualFold(fields[2], "clear") {
		fmt.Fprintf(nc, ">gm dummy %s: cleared %d stake(s)\r", who, s.clearDummies(pl))
		return
	}
	hp, count := dummyDefaultHP, 1
	var err error
	if len(fields) >= 3 {
		if hp, err = strconv.Atoi(fields[2]); err != nil || hp < 1 || hp > math.MaxInt32 {
			fmt.Fprintf(nc, ">usage: gm dummy <who> [hp|clear] [count 1-%d]\r", dummySpawnMax)
			return
		}
	}
	if len(fields) >= 4 {
		if count, err = strconv.Atoi(fields[3]); err != nil || count < 1 || count > dummySpawnMax {
			fmt.Fprintf(nc, ">usage: gm dummy <who> [hp|clear] [count 1-%d]\r", dummySpawnMax)
			return
		}
	}
	spawns, err := s.spawnDummies(pl, hp, count)
	if err != nil {
		fmt.Fprintf(nc, ">gm dummy: %v\r", err)
		return
	}
	ids := make([]string, len(spawns))
	for i, spawn := range spawns {
		ids[i] = fmt.Sprintf("%X", spawn.ObjectID)
	}
	fmt.Fprintf(nc, ">gm dummy %s: spawned %d stake(s) hp=%d obj=%s\r",
		who, len(spawns), hp, strings.Join(ids, ","))
}
