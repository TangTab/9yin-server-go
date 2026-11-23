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
	ObjectID uint32
	OwnerID  uint32
	X, Y, Z  float32
	Orient   float32
	ConfigID string            // 摆放 id（=模板联查键）
	File     string            // 摆放类别
	Script   string            // 模板物化类名
	PathID   string            // 摆放 PathId（巡逻路线键；缺席回落模板 PathID）
	Template map[string]string // NpcConfig 模板行（非 nil）
	Combat   *CombatRow        // 战斗行（FightID 未命中为 nil）
	Schedule *Schedule         // 巡逻时间表（BindPatrol 绑定；nil=站立）
}

// Stats 目录装载统计（诊断/测试面）。
type Stats struct {
	Placements int      // 白名单+坐标合法的摆放行数
	Resolved   int      // 模板联查命中数
	Skipped    int      // 闸门拦施数
	CombatHit  int      // 战斗行命中数
	Samples    []string // 未解析/被闸样本（≤12 条）
}

// ObjectIDAt 场景内 NPC wire objectID（目录下标 idx 0 起）：
// 官方 PERSISTID 风格 0x0E00xxxx（npc.pcapng 实证：官方 NPC obj=
// 0x0E001CB7/0x0E00147A，与玩家同 0x0E 段——fork 0x11 段在外服版
// 客户端只建视觉层，交互管线按 0x0E 段路由）。0x0E000000/1 保留。
func ObjectIDAt(index int) uint32 { return 0x0E000100 + uint32(index+2) }

// OwnerIDOf NPC owner：官方风格小值（npc.pcapng 实证 own=0x1CF9 等
// 递增小值，非 0x66xxxx 段）。
func OwnerIDOf(objectID uint32) uint32 { return 0x00001000 + (objectID & 0xFFFF) }

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

// BuildCatalogDir 场景目录（精确目录形态）：dir=ini/Scene 下确切目录名
//（roleState.Scene 末段等权威来源；大小写不敏感匹配实际目录名）。
func BuildCatalogDir(root, dir string) ([]Spawn, Stats, error) {
	stats := Stats{}
	if root == "" {
		return nil, stats, fmt.Errorf("npcfeed: 数据根缺席")
	}
	if canonical := CanonicalSceneDir(root, dir); canonical != "" {
		dir = canonical
	} else {
		return nil, stats, fmt.Errorf("npcfeed: 摆放目录 %s 不存在", dir)
	}
	placements, err := LoadPlacements(root, dir)
	if err != nil {
		return nil, stats, err
	}
	templates, combat, err := LoadStore(root)
	if err != nil {
		return nil, stats, err
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
			Script:   tpl.Script,
			PathID:   strings.TrimSpace(p.Attrs["PathId"]),
			Template: tpl.Attrs,
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
	// ident 按目录序分配（联查命中集）。
	for i := range spawns {
		spawns[i].ObjectID = ObjectIDAt(i)
		spawns[i].OwnerID = OwnerIDOf(spawns[i].ObjectID)
	}
	return spawns, stats, nil
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
