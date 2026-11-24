package fssoloworld

// npc.go 场景 NPC 下发薄壳 + 巡逻广播：npcfeed 目录缓存（含巡逻时间表
// 绑定）→ 半径/上限过滤 → 采样物化帧。数据链与帧配方见 internal/npcfeed
// 包注释及 docs/official/NPC服务端XML构建链与黑金样板.md；巡逻配方见
// npcfeed/patrol.go 注释（mappath 路线 + 每秒 0x21/State 广播）。

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/npcdata"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

const (
	// patrolFirstBroadcastDelay 巡逻首拍延迟：物化落地窗口内不发 0x21，
	// 防巡逻帧把刚物化的 NPC 拽回出生点（2433 定版 5s）。
	patrolFirstBroadcastDelay = 5 * time.Second
	// patrolBroadcastBatch 单帧 0x21 记录上限（官方抓包最多 5 条，留余量）。
	patrolBroadcastBatch = 8
	// npcViewportHysteresis AOI 视口迟滞圈（米）：NINEYIN_NPC_RADIUS>0 时
	// 出圈多留一圈防边缘抖动。
	npcViewportHysteresis = 150.0
)

// npcPushEnabled NINEYIN_NPC_PUSH=off 一键关停（方言演进期保护，
// 渲染异常时归因用）。
func npcPushEnabled() bool { return os.Getenv("NINEYIN_NPC_PUSH") != "off" }

// npcForm 物化帧形态：NINEYIN_NPC_FORM（"0x28"/"0x0d"/"dual"）显式
// 选择；否则 NINEYIN_NPC_0X28=off 判别切裸 0x0D；缺省 Form0x28Only。
//
// ⚠️ dual 双发已实证挤爆客户端 RoleUserData 池（2026-11-22 16:37 真机：
// 0x0D×1675+0x28×1675 两通道各自建档→can't create any more 回潮→功能
// NPC 被挤出消失）——2433 定论「0x28 对象与 0x0D 对象零重叠」，同对象
// 双通道是重复建档，勿再作缺省。
func npcForm() npcfeed.Form {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_NPC_FORM")); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "0x28", "28":
			return npcfeed.Form0x28Only
		case "0x0d", "0d":
			return npcfeed.Form0DOnly
		case "dual":
			return npcfeed.FormDual
		}
	}
	if os.Getenv("NINEYIN_NPC_0X28") == "off" {
		return npcfeed.Form0DOnly
	}
	return npcfeed.Form0x28Only
}

// npcPatrolEnabled NINEYIN_NPC_PATROL=off 关停巡逻（路线绑定与每秒
// 0x21/State 广播；默认开——2026-11-22「同时只显示一个 NPC」疑云已定案
// 为 MaxVisuals=230 池满并修复，见 62a07c4）。
func npcPatrolEnabled() bool { return os.Getenv("NINEYIN_NPC_PATROL") != "off" }

// npcCatalog 场景物化目录（联查命中集，进程级缓存；巡逻绑定后 spawns
// 携带时间表与吸附坐标）。dir=解析后的摆放目录名（帧/巡逻装载共用）。
type npcCatalog struct {
	dir    string
	spawns []npcfeed.Spawn
	stats  npcfeed.Stats
	bind   npcfeed.PatrolBindStats
	// sceneID scenes.ini 场景号（-1=未收录）：官服形态 ident 的 obj 高字节
	// 来源，玩家进场分配与 NPC 共用。
	sceneID int
	// ident 官服形态分配器（静态 NPC 之后的续分起点，玩家进场取号用）：
	// NPC 走官服形态时必有；固定段下仅玩家走官服形态时存在（空推过静态
	// 目录），否则 nil。
	ident *npcfeed.SceneIdent
	// owners obj → own（0x0E 移除帧取 own：官服形态 own 是独立序列号，
	// 不可由 obj 反推）。
	owners map[uint32]uint32
}

// ownerOf 对象的 own（目录外对象回落固定段公式）。
func (c *npcCatalog) ownerOf(objectID uint32) uint32 {
	if own, ok := c.owners[objectID]; ok {
		return own
	}
	return npcfeed.OwnerIDOf(objectID)
}

var (
	npcCatalogMu    sync.Mutex
	npcCatalogCache = map[string]*npcCatalog{}
)

// sceneNPCEnter 进场物化帧族（过滤+采样+编码）并初始化玩家视口簿记
// （可见集/场景/巡逻首拍时刻）。场景键经 resolveSceneDir 解析为权威摆放
// 目录（变体目录字典序陷阱见其注释），npcScene 存目录名。
func (s *Server) sceneNPCEnter(pl *Player, scene string, origin [4]float64) [][]byte {
	dir := s.resolveSceneDir(pl.roleState.Scene, scene)
	frames, visible := s.sceneNPCViewFrames(dir, origin)
	pl.npcMu.Lock()
	pl.npcPos = origin
	pl.npcScene = dir
	pl.npcVisible = visible
	pl.patrolReadyAt = time.Now().Add(patrolFirstBroadcastDelay)
	pl.npcMoveState = nil
	pl.npcMu.Unlock()
	return frames
}

// npcEnterAborted 进场物化未落地（native 生成失败回退 golden——golden
// 无 NPC 帧）：撤销视口簿记，防 tick 对客户端从未物化的 NPC 广播 0x21。
func (s *Server) npcEnterAborted(pl *Player) {
	pl.npcMu.Lock()
	pl.npcVisible = nil
	pl.npcMoveState = nil
	pl.npcMu.Unlock()
}

// sceneNPCFrames 场景 NPC 物化帧族（装载+联查+ident+过滤+采样+编码；
// 无玩家簿记——测试/无会话面）。key=目录名或场景键（born03 前缀兜底）。
func (s *Server) sceneNPCFrames(key string, origin [4]float64) [][]byte {
	frames, _ := s.sceneNPCViewFrames(s.sceneNPCCatalog(key).dir, origin)
	return frames
}

// sceneNPCViewFrames 视口子集帧族 + 可见 ident 集：过滤（半径/上限）→
// 资源 override → 巡逻坐标采样 → 编码。装载失败返回空——NPC 是增强
// 下发，不阻塞进场。dir=已解析摆放目录名。
func (s *Server) sceneNPCViewFrames(dir string, origin [4]float64) ([][]byte, map[uint32]bool) {
	if !npcPushEnabled() {
		return nil, nil
	}
	spawns := s.sceneNPCSpawns(dir, origin)
	frames, err := npcfeed.Frames(spawns, npcForm())
	if err != nil {
		s.trace("npc frames: %v", err)
		return nil, nil
	}
	visible := make(map[uint32]bool, len(spawns))
	for _, spawn := range spawns {
		visible[spawn.ObjectID] = true
	}
	return frames, visible
}

// sceneNPCSpawns 视口子集（坐标已采样到当前时刻）：全量模式（半径 0）
// 一次物化；半径模式按玩家平面位置过滤 + 距离升序截断。返回目录副本
// （不改动缓存）。dir=已解析摆放目录名。
func (s *Server) sceneNPCSpawns(dir string, origin [4]float64) []npcfeed.Spawn {
	catalog := s.sceneNPCCatalog(dir)
	spawns := catalog.spawns
	if radius := npcRadius(); radius > 0 {
		nowMs := time.Now().UnixMilli()
		spawns = filterBySampledRadius(spawns, origin[0], origin[2], radius, npcLimit(), nowMs)
	} else if limit := npcLimit(); limit > 0 && len(spawns) > limit {
		spawns = spawns[:limit]
	}
	spawns = applyResourceOverride(spawns)
	return sampleSpawns(spawns, time.Now().UnixMilli())
}

// sampleSpawns 巡逻 Spawn 坐标覆写为时间表当前位置（重进视口不闪现
// 回出生点；静止 Spawn 原样），State 标为采样时刻的状态——在走即 walk，
// 与官服一致：行进中进视口的 NPC 物化即 walk；停留即路点到达动作或
// 静息态。返回新切片（不改动目录缓存）。
func sampleSpawns(spawns []npcfeed.Spawn, nowMs int64) []npcfeed.Spawn {
	out := make([]npcfeed.Spawn, len(spawns))
	copy(out, spawns)
	for i := range out {
		schedule := out[i].Schedule
		if schedule == nil {
			continue
		}
		p := schedule.PositionAt(nowMs)
		out[i].X, out[i].Y, out[i].Z, out[i].Orient = p.X, p.Y, p.Z, p.Orient
		out[i].State = schedule.MotionAt(nowMs).State(out[i].RestAction())
	}
	return out
}

// filterBySampledRadius 半径过滤按巡逻采样位置（radius 模式下 NPC 走动
// 后以实际位置入圈）；其余语义同 npcfeed.FilterByRadius。
func filterBySampledRadius(spawns []npcfeed.Spawn, ox, oz, radius float64, limit int, nowMs int64) []npcfeed.Spawn {
	type distSpawn struct {
		d float64
		s npcfeed.Spawn
	}
	r2 := radius * radius
	kept := make([]distSpawn, 0, len(spawns))
	for _, spawn := range spawns {
		x, z := float64(spawn.X), float64(spawn.Z)
		if spawn.Schedule != nil {
			p := spawn.Schedule.PositionAt(nowMs)
			x, z = float64(p.X), float64(p.Z)
		}
		dx, dz := ox-x, oz-z
		if d := dx*dx + dz*dz; d <= r2 {
			kept = append(kept, distSpawn{d: d, s: spawn})
		}
	}
	// 稳定排序：目录序（ObjectID 升序）打底，按距离升序。
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].d < kept[j].d })
	if limit > 0 && len(kept) > limit {
		kept = kept[:limit]
	}
	out := make([]npcfeed.Spawn, len(kept))
	for i, k := range kept {
		out[i] = k.s
	}
	return out
}

// applyResourceOverride NINEYIN_NPC_RESOURCE_OVERRIDE 判别实验开关：
// 强制全 NPC 用同一模型串（模板层克隆改写，不污染目录缓存）。判别
// 「客户端模型库缺失（缺模型的 NPC 不渲染/缺动画呆立）」vs「下发建档
// 问题」——全同模型后若全城显示且会动，坐实前者（2433 线实验设计）。
func applyResourceOverride(spawns []npcfeed.Spawn) []npcfeed.Spawn {
	override := strings.TrimSpace(os.Getenv("NINEYIN_NPC_RESOURCE_OVERRIDE"))
	if override == "" {
		return spawns
	}
	patched := make([]npcfeed.Spawn, len(spawns))
	copy(patched, spawns)
	for i := range patched {
		tpl := make(map[string]string, len(patched[i].Template)+1)
		for k, v := range patched[i].Template {
			tpl[k] = v
		}
		tpl["Resource"] = override
		tpl["Material"] = "" // 防 Material 分支干扰
		patched[i].Template = tpl
	}
	return patched
}

// sceneNPCCatalog 场景目录（缓存；装载时顺带巡逻绑定——epoch=装载时刻，
// 同场景所有连接共享同一时间表，NPC 位置跨连接一致）。key=摆放目录名或
// 场景键：精确目录名命中（大小写不敏感）优先，否则场景键前缀匹配兜底
// （born03 → born03_yanyuzhuang）。多变体前缀（school01_fangong 抢在
// school01_JinYiWei 前）须由 resolveSceneDir 先解析出权威目录再进来。
func (s *Server) sceneNPCCatalog(key string) *npcCatalog {
	npcCatalogMu.Lock()
	defer npcCatalogMu.Unlock()
	if cached, ok := npcCatalogCache[strings.ToLower(key)]; ok {
		return cached
	}
	catalog := &npcCatalog{dir: key, sceneID: -1}
	root := npcfeed.ResolveRoot()
	if root == "" {
		s.trace("npc catalog: 数据根缺席（NINEYIN_DATA_ROOT / Res / testdata），场景 %s NPC 不下发", key)
		npcCatalogCache[strings.ToLower(key)] = catalog
		return catalog
	}
	s.traceDataSource(root)
	dir := key
	if canonical := npcfeed.CanonicalSceneDir(root, dir); canonical != "" {
		dir = canonical
	} else if prefix := npcfeed.SceneDirOf(root, dir); prefix != "" {
		dir = prefix
	}
	catalog.dir = dir
	scheme := npcIdentScheme()
	if id, ok := npcdata.SceneIndexOfDir(root, dir); ok {
		catalog.sceneID = int(id)
	} else if playerIdentScene() || scheme == npcfeed.IdentScene {
		s.trace("npc catalog: 场景 %s 不在 scenes.ini，ident 回落固定段", dir)
	}
	spawns, stats, alloc, err := npcfeed.BuildCatalogDirAs(root, dir, scheme, catalog.sceneID)
	catalog.ident = alloc
	if catalog.ident == nil && playerIdentScene() && catalog.sceneID >= 0 {
		// NPC 走固定段（或本场景无摆放目录）而玩家走官服形态：动态对象
		// 仍从静态目录之后起算（2+5N 与固定段 [0x102, 0x102+N) 不相交）。
		catalog.ident = npcfeed.NewSceneIdent(catalog.sceneID)
		catalog.ident.SkipStatic(len(spawns))
	}
	if err != nil {
		s.trace("npc catalog: %v", err)
		npcCatalogCache[strings.ToLower(key)] = catalog
		return catalog
	}
	catalog.owners = make(map[uint32]uint32, len(spawns))
	for _, spawn := range spawns {
		catalog.owners[spawn.ObjectID] = spawn.OwnerID
	}
	// 巡逻绑定：mappath 路线表（<root>/share/creator/mappath，场景短名前缀
	// ini）→ 出生点吸附 + 时间表（目录缓存内一次完成）。
	if npcPatrolEnabled() {
		mappathDir := resolveMappathDir(root)
		routes := npcfeed.LoadPatrolRoutes(mappathDir, dir)
		catalog.bind = npcfeed.BindPatrol(spawns, routes, time.Now().UnixMilli())
		s.tracePatrolRoutesMissing(mappathDir)
	}
	catalog.spawns, catalog.stats = spawns, stats
	s.trace("npc catalog: 场景 %s 摆放 %d 联查 %d 战斗命中 %d 闸拦 %d → 物化 %d 条（巡逻 walkers=%d stand=%d missingPath=%d far=%d）",
		dir, stats.Placements, stats.Resolved, stats.CombatHit, stats.Skipped, len(spawns),
		catalog.bind.Walkers, catalog.bind.Stand, catalog.bind.MissingPath, catalog.bind.FarOrShort)
	for _, sample := range stats.Samples {
		s.trace("npc catalog: 样本 %s", sample)
	}
	npcCatalogCache[strings.ToLower(key)] = catalog
	return catalog
}

// resolveSceneDir 权威摆放目录解析（2026-11-23 修复：SceneDirOf 字典序
// 首匹配在同前缀多变体目录上选错——八大门派全命中 school0X_fangong
// 反攻副本（摆放 300~400 行 vs 正图 3400~6900 行）、襄阳坡/塞外大漠命中
// _2_ 变体（scene08 仅剩 3 行≈空图），切换这些地图 NPC 缺失/寥寥）。
// 解析序：① roleState.Scene 末段（建号/切图落的精确 config 路径
// ini\scene\<dir>，目录存在才算）② BornCJ FindSceneOption(resource)
// .Config 末段（旧档 Scene 缺席兜底）③ SceneDirOf 字典序前缀（末路——
// 无权威信息的场景才走，变体目录会选错）④ resource 原样（装载报错空目录）。
func (s *Server) resolveSceneDir(scenePath, resource string) string {
	root := npcfeed.ResolveRoot()
	if root == "" {
		return resource
	}
	if dir := npcfeed.CanonicalSceneDir(root, lastPathSegment(scenePath)); dir != "" {
		return dir
	}
	if opt, ok := rolestate.FindSceneOption(resource); ok {
		if dir := npcfeed.CanonicalSceneDir(root, lastPathSegment(opt.Config)); dir != "" {
			return dir
		}
	}
	if dir := npcfeed.SceneDirOf(root, resource); dir != "" {
		return dir
	}
	return resource
}

// lastPathSegment 路径末段（\ 与 / 皆容）。
func lastPathSegment(path string) string {
	if i := strings.LastIndexAny(path, `\/`); i >= 0 {
		return path[i+1:]
	}
	return path
}

// resolveMappathDir mappath 巡逻路线目录：env NINEYIN_MAPPATH_DIR >
// <root>/share/creator/mappath——与客户端资源树同布局（客户端 share 整树
// 并入数据根后路线就在这个位置，无须另行搬运；正式根 Res\share\creator\
// mappath、仓内 testdata 同路径）。
func resolveMappathDir(root string) string {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_MAPPATH_DIR")); v != "" {
		return v
	}
	return filepath.Join(root, "share", "creator", "mappath")
}

// patrolRoutesMissingOnce 路线目录缺席只 trace 一次：每个场景目录装载都会
// 经过，缺目录=所有场景都没有巡逻，逐场景刷屏无益。
var patrolRoutesMissingOnce sync.Once

// npcSourceOnce 数据来源只在首个场景目录装载时 trace 一次。
var npcSourceOnce sync.Once

// traceDataSource 标明 NPC 三域的数据取自哪里（npcfeed.SourcesOf）：数据根
// 带客户端 share 树即直读（share），否则读构建产物（xml）——启动后看这一行
// 就知道 NPC 数据走的是哪条路；某域意外回落 xml 时在这里能看出来。share 树
// 因比产物旧而被判回落的，另起一行说明原因与办法（sharedata.StaleShareNote）。
func (s *Server) traceDataSource(root string) {
	npcSourceOnce.Do(func() {
		s.trace("npc data: 来源 %s（NINEYIN_DATA_SOURCE=%q；share=直读客户端 share 树，xml=构建产物）",
			npcfeed.SourcesOf(root), strings.TrimSpace(os.Getenv("NINEYIN_DATA_SOURCE")))
		if note := sharedata.StaleShareNote(root); note != "" {
			s.trace("npc data: %s", note)
		}
	})
}

// tracePatrolRoutesMissing 路线目录缺席告警。目录缺失时 LoadPatrolRoutes
// 静默返回空表、带 PathId 的 NPC 全部站立——2026-11-24 Res 层化后路线
// 目录没随迁，巡逻「消失」而日志里只有一行 walkers=0，补一条说清原因的
// trace。
func (s *Server) tracePatrolRoutesMissing(mappathDir string) {
	if _, err := res.Stat(mappathDir); err == nil {
		return
	}
	patrolRoutesMissingOnce.Do(func() {
		s.trace("npc patrol: 巡逻路线目录缺席 %s —— 带 PathId 的 NPC 全部站立（把客户端 share/creator/mappath 投放数据根同路径，或 go run ./tools/datagen patrol；NINEYIN_MAPPATH_DIR 可改路径）", mappathDir)
	})
}

// npcReconcile 视口对账（op0x0A 玩家位移驱动）：diff 全量集（按玩家
// 位置与当前时刻）——新进视口发物化帧（采样坐标）、出迟滞圈发 0x0E
// 移除帧；每包重算（μs 级），帧只在变化时发。全量模式（半径 0）直通。
func (s *Server) npcReconcile(pl *Player, pos [4]float64) {
	if !npcPushEnabled() {
		return
	}
	radius := npcRadius()
	pl.npcMu.Lock()
	if pl.npcVisible == nil || radius <= 0 { // 全量模式：一次物化、永不增删
		pl.npcPos = pos
		pl.npcMu.Unlock()
		return
	}
	scene, visible := pl.npcScene, pl.npcVisible
	pl.npcPos = pos
	pl.npcMu.Unlock()

	catalog := s.sceneNPCCatalog(scene)
	nowMs := time.Now().UnixMilli()
	adds, removes, nowVisible := npcViewportDiff(catalog, pos, nowMs, visible, radius)
	pl.npcMu.Lock()
	pl.npcVisible = nowVisible
	pl.npcMu.Unlock()
	for _, spawn := range adds {
		if frames, err := npcfeed.Frames(sampleSpawns([]npcfeed.Spawn{spawn}, nowMs), npcForm()); err == nil {
			for _, frame := range frames {
				s.sendToClient(pl, frame)
			}
		}
	}
	for _, id := range removes {
		s.sendToClient(pl, entergen.SceneRemoveObjectFrame(id, catalog.ownerOf(id)))
	}
	if len(adds)+len(removes) > 0 {
		s.trace("npc viewport reconcile: client=%d +%d -%d visible=%d", pl.Conn1, len(adds), len(removes), len(nowVisible))
	}
}

// npcViewportDiff 视口对账 diff（纯函数）：按玩家平面位置与当前时刻
// （巡逻 NPC 按采样位置）——圈内未物化的为增、出迟滞圈的为删、圈内
// 已见/迟滞圈内保持可见。
func npcViewportDiff(catalog *npcCatalog, pos [4]float64, nowMs int64, visible map[uint32]bool, radius float64) (adds []npcfeed.Spawn, removes []uint32, nowVisible map[uint32]bool) {
	outer := radius + npcViewportHysteresis
	nowVisible = make(map[uint32]bool, len(visible))
	for i := range catalog.spawns {
		spawn := &catalog.spawns[i]
		x, z := float64(spawn.X), float64(spawn.Z)
		if spawn.Schedule != nil {
			p := spawn.Schedule.PositionAt(nowMs)
			x, z = float64(p.X), float64(p.Z)
		}
		dx, dz := pos[0]-x, pos[2]-z
		d2 := dx*dx + dz*dz
		if d2 <= radius*radius {
			nowVisible[spawn.ObjectID] = true
			if !visible[spawn.ObjectID] {
				adds = append(adds, *spawn)
			}
		} else if d2 <= outer*outer && visible[spawn.ObjectID] {
			nowVisible[spawn.ObjectID] = true // 迟滞圈内保持可见
		}
	}
	for id := range visible {
		if !nowVisible[id] {
			removes = append(removes, id)
		}
	}
	return adds, removes, nowVisible
}

// npcPatrolTick 巡逻广播（每秒，挂 timerCheck 尾部；2433 patrolTick
// 同构）：对每个在线玩家——视口内巡逻 NPC 按时刻采样组 0x21 批量帧
// （≤8 条/帧）+ State(183) walk/stand 翻转帧（先于 0x21——先置动画
// 状态再同步运动目的地）。
func (s *Server) npcPatrolTick(now time.Time) {
	if !npcPushEnabled() || !npcPatrolEnabled() {
		return
	}
	nowMs := now.UnixMilli()
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline {
			continue
		}
		pl.npcMu.Lock()
		if pl.npcVisible == nil || now.Before(pl.patrolReadyAt) {
			pl.npcMu.Unlock()
			continue // 未进场或物化落地窗口内（首拍延迟防拽回出生点）
		}
		visible, moveState, scene := pl.npcVisible, pl.npcMoveState, pl.npcScene
		if moveState == nil {
			moveState = make(map[uint32]string)
			pl.npcMoveState = moveState
		}
		pl.npcMu.Unlock()
		for _, frame := range patrolFrames(s.sceneNPCCatalog(scene), visible, moveState, nowMs, talkPausedSnapshot()) {
			s.sendToClient(pl, frame)
		}
	}
}

// ---- 对话暂停（点击行人 NPC 弹菜单时停走） ----

var (
	talkPauseMu sync.Mutex
	talkPaused  = map[uint32]int64{} // objectID → 暂停时刻 ms（60s 自动过期兜底）
)

// pauseNpcTalk 对话暂停：巡逻 NPC 停在暂停时刻采样位（60s 兜底过期，
// 菜单异常未关不放永久停走）。
func pauseNpcTalk(objectID uint32) {
	talkPauseMu.Lock()
	talkPaused[objectID] = time.Now().UnixMilli()
	talkPauseMu.Unlock()
}

// resumeNpcTalk 菜单关闭恢复巡逻。
func resumeNpcTalk(objectID uint32) {
	talkPauseMu.Lock()
	delete(talkPaused, objectID)
	talkPauseMu.Unlock()
}

func talkPausedSnapshot() map[uint32]int64 {
	talkPauseMu.Lock()
	defer talkPauseMu.Unlock()
	if len(talkPaused) == 0 {
		return nil
	}
	snapshot := make(map[uint32]int64, len(talkPaused))
	for id, at := range talkPaused {
		snapshot[id] = at
	}
	return snapshot
}

// patrolFrames 单玩家巡逻帧族（纯函数；就地更新 moveState 水位）：视口
// 内巡逻 NPC 采样 Motion——状态变了才发 0x10 State（在走 walk；停留是
// 路点到达动作或静息态，见 npcfeed.Motion.State），全体组 0x21 批量帧
// （≤8 条/帧；行进 spd/flags=1 且 rotate 恒 +2π，停留 spd=0/flags=0）。
// paused（对话暂停）的 NPC 冻结在暂停时刻采样位、置静息态、每拍发 spd=0
// 停留条目（防旧目的地残留）。不巡逻的 NPC（姿态 NPC）不经此处——
// 姿态由物化帧给出后客户端自持，官服同样不再发 State。
func patrolFrames(catalog *npcCatalog, visible map[uint32]bool, moveState map[uint32]string, nowMs int64, paused map[uint32]int64) [][]byte {
	batch := make([]entergen.AllDestEntry, 0, 8)
	var stateFrames [][]byte
	for i := range catalog.spawns {
		spawn := &catalog.spawns[i]
		if spawn.Schedule == nil || !visible[spawn.ObjectID] {
			continue
		}
		pauseAt, isPaused := paused[spawn.ObjectID]
		if isPaused && nowMs-pauseAt > 60*1000 { // 兜底过期
			isPaused = false
		}
		var motion npcfeed.Motion
		if isPaused {
			p := spawn.Schedule.PositionAt(pauseAt)
			motion = npcfeed.Motion{DestX: p.X, DestY: p.Y, DestZ: p.Z, DestOrient: p.Orient}
		} else {
			motion = spawn.Schedule.MotionAt(nowMs)
		}
		state := motion.State(spawn.RestAction())
		if last, seen := moveState[spawn.ObjectID]; !seen || last != state {
			moveState[spawn.ObjectID] = state
			if frame, err := entergen.NpcStateFrame(spawn.ObjectID, spawn.OwnerID, state); err == nil {
				stateFrames = append(stateFrames, frame)
			}
		}
		flags := uint32(0)
		rotate := float32(0)
		if motion.Moving {
			flags = 1
			rotate = float32(2 * math.Pi) // 官方步行条目 rotate 恒 +2π
		}
		batch = append(batch, entergen.AllDestEntry{
			ObjectID: spawn.ObjectID, OwnerID: spawn.OwnerID,
			DestX: motion.DestX, DestY: motion.DestY, DestZ: motion.DestZ,
			DestOrient: motion.DestOrient, Speed: motion.Speed,
			Rotate: rotate, MoveFlags: flags,
		})
	}
	frames := stateFrames // 先置动画状态，再同步运动目的地
	for start := 0; start < len(batch); start += patrolBroadcastBatch {
		end := start + patrolBroadcastBatch
		if end > len(batch) {
			end = len(batch)
		}
		if frame := entergen.ServerAllDestFrame(batch[start:end]); frame != nil {
			frames = append(frames, frame)
		}
	}
	return frames
}

// parseMotionPosition op0x0A 67B 位移包（2433 线 motion.go 定版）：@0x26
// 起四个 tag=4 的 f32（X/Y/Z/orient），限值校验——更长移动包绝不能
// 更新位置（扫描会拼出貌似合理却错误的坐标）。
func parseMotionPosition(msg []byte) (x, y, z, orient float32, ok bool) {
	if len(msg) != 67 {
		return 0, 0, 0, 0, false
	}
	const offset = 0x26
	if msg[offset] != 4 || msg[offset+5] != 4 || msg[offset+10] != 4 || msg[offset+15] != 4 {
		return 0, 0, 0, 0, false
	}
	x = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+1:]))
	y = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+6:]))
	z = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+11:]))
	orient = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+16:]))
	for _, v := range []float32{x, y, z, orient} {
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, 0, 0, 0, false
		}
	}
	if math.Abs(float64(x)) > 10000 || math.Abs(float64(y)) > 3000 ||
		math.Abs(float64(z)) > 10000 || math.Abs(float64(orient)) > 20 {
		return 0, 0, 0, 0, false
	}
	return x, y, z, orient, true
}

// npcRadius NINEYIN_NPC_RADIUS 半径过滤（<=0 全量；默认全量——当前
// 真机验证口径）。
func npcRadius() float64 { return envFloat("NINEYIN_NPC_RADIUS", 0) }

// npcLimit NINEYIN_NPC_LIMIT 截断上限（<=0 不截断）。
func npcLimit() int { return int(envFloat("NINEYIN_NPC_LIMIT", 0)) }

// envFloat env 数值（缺席/非法回落缺省）。TrimSpace 容忍 CRLF 尾巴——
// env.server 经 bat findstr+call 注入时 LF 行尾的行会带 \r 残留进值
// （2026-11-22 NINEYIN_NPC_RADIUS 静默失效教训）。
func envFloat(key string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
	}
	return def
}
