package npcfeed

// join.go 联查与场景目录：摆放行 → NpcConfig 模板行（FightID）→
// NpcTemplate 战斗行；注册闸门与 ident 分配。
//
// 闸门移植 2433 npcSpawnEligible：EventNpc（任务/事件触发器）、
// TaskSubId（任务触发生成）、SubSceneNo+战斗类（活动/分线态战斗怪）
// 不注册——不物化、不进目录。

import (
	"fmt"
	"sort"
	"strings"
)

// Spawn 一条联查完毕的待物化 NPC（帧生成输入）。
type Spawn struct {
	ObjectID   uint32
	OwnerID    uint32
	X, Y, Z    float32
	Orient     float32
	ConfigID   string            // 摆放 id（=模板联查键）
	File       string            // 摆放类别
	Script     string            // 模板物化类名
	PathID     string            // 摆放 PathId（巡逻路线键；缺席回落模板 PathID）
	BornAction string            // 出生状态原始值：摆放 BornAction > 模板 BornAction > 模板 BornActionRec（空=stand，见 pose.go）
	State      string            // 物化时的 State（巡逻采样在走=walk；空=静息态）
	Template   map[string]string // NpcConfig 模板行（非 nil）
	Combat     *CombatRow        // 战斗行（FightID 未命中为 nil）
	Schedule   *Schedule         // 巡逻时间表（BindPatrol 绑定；nil=站立）
}

// Stats 目录装载统计（诊断/测试面）。
type Stats struct {
	Placements int      // 白名单+坐标合法的摆放行数
	Resolved   int      // 模板联查命中数
	Skipped    int      // 闸门拦施数
	CombatHit  int      // 战斗行命中数
	Samples    []string // 未解析/被闸样本（≤12 条）
}

// IdentScheme NPC wire ident 分配方言（场景目录装载时选定）。
type IdentScheme int

const (
	// IdentFixed 固定段（fork 客户端真机验证形态，缺省）：obj=0x0E000100+
	// (idx+2)、own=0x1000+低 16 位。0x0E 是姑苏（scenes.ini [14]）的场景号
	// ——fork 0x11 段在外服版客户端只建视觉层，交互管线按 0x0E 段路由
	// （npc.pcapng 实证官方 NPC obj=0x0E001CB7/0x0E00147A）。0x0E000000/1 保留。
	IdentFixed IdentScheme = iota
	// IdentScene 官服形态（四抓包实证，2026-11-24；见
	// docs/fork-era/id-system-analysis.md 文首「官服规则」）：obj 高 8 位=
	// 场景 ID 低字节（scenes.ini 节号——玩家与同场景全部对象共用）、低
	// 24 位=场景内序号；own=场景内序列号。静态 NPC 序号自 2 起每个占 5 个
	// 单位，序列号滞后 5。
	IdentScene
)

// ObjectIDAt 固定段 obj（目录下标 idx 0 起）。
func ObjectIDAt(index int) uint32 { return 0x0E000100 + uint32(index+2) }

// OwnerIDOf 固定段 own：obj 的纯函数（官方风格小值；仅固定段成立——官服
// 形态 own 是独立序列号，重建后同 obj 会换新值，不可由 obj 反推）。
func OwnerIDOf(objectID uint32) uint32 { return 0x00001000 + (objectID & 0xFFFF) }

// 官服静态对象序号/序列号节律：首个可见对象序号 2、序列号 7，其后每个
// NPC 占 5 个序号单位并推进 5 个序列号——born_single_01 两台服务器同为
// 0x3E/0x43、0x4D/0x52、0x75/0x7A（女号 8032 线与男号 7114 线逐值相同）；
// born01 WorldNpc10627 起 0x861/0x866、0x86B/0x870… 步长 5；姑苏
// WorldNpc01248 起 0x140C/0x1430 步长 5。own = 序号 + 5 为场景加载初值，
// 官服随个别 NPC 多耗 1~2 个序列号缓慢爬升（born01 分段常数
// +5→+11→+105→+111→+215→+303），此处取初值。
const (
	sceneIdentIndexBase = 2
	sceneIdentSerialLag = 5
	// sceneIdentNPCUnits 静态 NPC 占的序号单位数。
	sceneIdentNPCUnits = 5
	// sceneIndexLimit 序号 24 位上限（越界会溢进场景 ID 字节）。
	sceneIndexLimit = 1 << 24
)

// SceneObjectIDAt 官服形态 obj：高 8 位=场景 ID 低字节，低 24 位=场景内序号
// （sceneID ≥ 256 取低字节：官服 born_single_01 节号 317=0x13D → 0x3D 段）。
func SceneObjectIDAt(sceneID, index int) uint32 {
	return uint32(sceneID&0xFF)<<24 | uint32(index)&0xFFFFFF
}

// SceneIdent 官服形态 ident 分配器：场景内序号与序列号两个计数器各自
// 单调递增、创建时各取一次当前值（官服语义——序号可复用、序列号不复用）。
// 静态 NPC（场景目录）与进场玩家共用同一实例：玩家拿到的是静态目录之后
// 的下一个序号（官服 born_single_01 两服玩家序号同为 0xF3）。非并发安全，
// 调用方持锁。
type SceneIdent struct {
	sceneID int
	index   uint32 // 下一个对象的场景内序号
	serial  uint32 // 下一个对象的序列号
}

// NewSceneIdent 场景 ID = scenes.ini 节号（npcdata.SceneIndexOfDir）。
func NewSceneIdent(sceneID int) *SceneIdent {
	return &SceneIdent{
		sceneID: sceneID,
		index:   sceneIdentIndexBase,
		serial:  sceneIdentIndexBase + sceneIdentSerialLag,
	}
}

// Alloc 分配一个对象 ident：obj=场景 ID 低字节<<24|序号，own=序列号；两个
// 计数器各推进 units（静态 NPC 取 sceneIdentNPCUnits，动态对象取 1）。
// 序号越出 24 位返回 ok=false（调用方回落或报错）。
func (a *SceneIdent) Alloc(units int) (objectID, ownerID uint32, ok bool) {
	if a == nil || a.index >= sceneIndexLimit {
		return 0, 0, false
	}
	objectID, ownerID = SceneObjectIDAt(a.sceneID, int(a.index)), a.serial
	a.index += uint32(units)
	a.serial += uint32(units)
	return objectID, ownerID, true
}

// SkipStatic 空推 count 个静态 NPC：NPC 目录走固定段方言、只有动态对象
// （进场玩家）走官服形态时，把续分起点挪到静态目录之后。
func (a *SceneIdent) SkipStatic(count int) {
	for ; count > 0 && a.index < sceneIndexLimit; count-- {
		a.index += sceneIdentNPCUnits
		a.serial += sceneIdentNPCUnits
	}
}

// eligible 注册闸门（2433 npcSpawnEligible 同款，判据取摆放行原始属性
// 与模板类名）。
func eligible(p Placement, scriptClass string) bool {
	if strings.EqualFold(strings.TrimSpace(scriptClass), "EventNpc") {
		return false
	}
	if v := strings.TrimSpace(p.Attrs["TaskSubId"]); v != "" {
		return false
	}
	if v := strings.TrimSpace(p.Attrs["SubSceneNo"]); v != "" {
		switch strings.ToLower(strings.TrimSpace(scriptClass)) {
		case "attacknpc", "bossnpc":
			return false
		}
	}
	return true
}

// BuildCatalog 场景目录（场景键形态）：按 SceneDirOf 字典序前缀匹配摆放
// 目录。⚠️ 同前缀多变体目录（school01_fangong 抢在 school01_JinYiWei 前）
// 会选错——调用方持有权威场景配置路径时必须走 BuildCatalogDir。
func BuildCatalog(root, scene string) ([]Spawn, Stats, error) {
	if root == "" {
		return nil, Stats{}, fmt.Errorf("npcfeed: 数据根缺席")
	}
	sceneDir := SceneDirOf(root, scene)
	if sceneDir == "" {
		return nil, Stats{}, fmt.Errorf("npcfeed: 场景 %s 无摆放目录", scene)
	}
	return BuildCatalogDir(root, sceneDir)
}

// BuildCatalogDir 场景目录（精确目录形态，固定段 ident）：dir=ini/Scene 下
// 确切目录名（roleState.Scene 末段等权威来源；大小写不敏感匹配实际目录名）。
func BuildCatalogDir(root, dir string) ([]Spawn, Stats, error) {
	spawns, stats, _, err := BuildCatalogDirAs(root, dir, IdentFixed, 0)
	return spawns, stats, err
}

// BuildCatalogDirAs 指定 ident 方言的场景目录。sceneID 仅 IdentScene 使用
// （scenes.ini 节号；<0 = 场景 ID 未知，回落固定段）。返回的分配器为官服
// 形态下静态 NPC 之后的续分起点（进场玩家共用同场景计数器；固定段/场景
// ID 未知为 nil）。
func BuildCatalogDirAs(root, dir string, scheme IdentScheme, sceneID int) ([]Spawn, Stats, *SceneIdent, error) {
	stats := Stats{}
	if root == "" {
		return nil, stats, nil, fmt.Errorf("npcfeed: 数据根缺席")
	}
	if canonical := CanonicalSceneDir(root, dir); canonical != "" {
		dir = canonical
	} else {
		return nil, stats, nil, fmt.Errorf("npcfeed: 摆放目录 %s 不存在", dir)
	}
	placements, err := LoadPlacements(root, dir)
	if err != nil {
		return nil, stats, nil, err
	}
	templates, combat, err := LoadStore(root)
	if err != nil {
		return nil, stats, nil, err
	}
	spawns := make([]Spawn, 0, len(placements))
	for _, p := range placements {
		stats.Placements++
		tpl, ok := templates[p.ID]
		if !ok || tpl.Attrs["Resource"] == "" {
			sample := fmt.Sprintf("%s:%s 模板缺席", p.File, p.ID)
			appendSample(&stats, sample)
			continue
		}
		if !eligible(p, tpl.Script) {
			stats.Skipped++
			appendSample(&stats, fmt.Sprintf("%s:%s 被闸（EventNpc/TaskSubId/SubSceneNo）", p.File, p.ID))
			continue
		}
		spawn := Spawn{
			X: float32(p.X), Y: float32(p.Y), Z: float32(p.Z),
			Orient:   float32(p.AY),
			ConfigID: p.ID, File: p.File,
			Script:     tpl.Script,
			PathID:     strings.TrimSpace(p.Attrs["PathId"]),
			BornAction: bornActionOf(p.Attrs, tpl.Attrs),
			Template:   tpl.Attrs,
		}
		if fightID := strings.TrimSpace(tpl.Attrs["FightID"]); fightID != "" {
			if row, hit := combat[fightID]; hit {
				spawn.Combat = &row
				stats.CombatHit++
			}
		}
		spawns = append(spawns, spawn)
		stats.Resolved++
	}
	// ident 按目录序分配（联查命中集）：官服形态每个 NPC 占 5 个序号单位，
	// 分配器原样返回供动态对象续分。
	var alloc *SceneIdent
	if scheme == IdentScene && sceneID >= 0 {
		alloc = NewSceneIdent(sceneID)
	}
	for i := range spawns {
		if alloc == nil {
			spawns[i].ObjectID = ObjectIDAt(i)
			spawns[i].OwnerID = OwnerIDOf(spawns[i].ObjectID)
			continue
		}
		objectID, ownerID, ok := alloc.Alloc(sceneIdentNPCUnits)
		if !ok {
			return nil, stats, nil, fmt.Errorf("npcfeed: 场景 %s 物化 %d 条超出官服形态序号 24 位上限", dir, i)
		}
		spawns[i].ObjectID, spawns[i].OwnerID = objectID, ownerID
	}
	return spawns, stats, alloc, nil
}

func appendSample(stats *Stats, sample string) {
	if len(stats.Samples) < 12 {
		stats.Samples = append(stats.Samples, sample)
	}
}

// FilterByRadius 半径过滤（平面 x/z 距离）+ 距离升序截断上限。
// radius<=0 不限；limit<=0 不截断。返回新切片（不改动目录序缓存）。
func FilterByRadius(spawns []Spawn, ox, oz float64, radius float64, limit int) []Spawn {
	type distSpawn struct {
		d float64
		s Spawn
	}
	r2 := radius * radius
	kept := make([]distSpawn, 0, len(spawns))
	for _, s := range spawns {
		d := dist2D(ox, oz, float64(s.X), float64(s.Z))
		if radius > 0 && d > r2 {
			continue
		}
		kept = append(kept, distSpawn{d, s})
	}
	// 稳定排序：目录序（ObjectID 升序）打底，按距离升序。
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].d < kept[j].d })
	if limit > 0 && len(kept) > limit {
		kept = kept[:limit]
	}
	out := make([]Spawn, len(kept))
	for i, k := range kept {
		out[i] = k.s
	}
	return out
}

func dist2D(ax, az, bx, bz float64) float64 {
	dx, dz := ax-bx, az-bz
	return dx*dx + dz*dz
}
