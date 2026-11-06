package sim

import (
	"sort"

	"github.com/local/9yin-go-server/internal/ecs"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// World 包装 ECS 世界，持有场景内全部已注册 NPC 的模拟态。
// 实体 id 直接复用 worldcore.EntityID 的数值（玩家保留 0/1），
// 因此与客户端 objectID 之间不需要任何映射表。
//
// 本类型不是线程安全的：所有访问都必须发生在 sceneLifecycle
// 的锁临界区内——这与被替换的平行 map 的纪律完全一致。
// 组件集在 spawn 时固定，此后只整体写回组件值、不增删组件，
// 因此不会触发 archetype 搬移，也不存在迭代中结构变更的风险。
type World struct {
	world *ecs.World
}

func New() *World {
	return &World{world: ecs.NewWorld()}
}

// Reset 丢弃全部实体，供场景世代切换使用。显式 id 不依赖
// 自增计数器，直接换一个新世界即可。
func (s *World) Reset() {
	s.world = ecs.NewWorld()
}

// SpawnNPC 注册一个无战斗 AI 的 NPC（平民），只带基础数值。
// 调用方负责查重——与旧 map 一样，重复 id 会覆盖旧实体。
func (s *World) SpawnNPC(id worldcore.EntityID, v Vitals) {
	ecs.Write(s.world, ecs.Id(id), ecs.C(v))
}

// SpawnCombatNPC 注册带战斗 AI 的敌对 NPC。
func (s *World) SpawnCombatNPC(id worldcore.EntityID, v Vitals, b Brain) {
	ecs.Write(s.world, ecs.Id(id), ecs.C(v), ecs.C(b))
}

// Remove 注销 NPC；id 不存在时静默返回，与旧 map 的 delete 行为一致。
func (s *World) Remove(id worldcore.EntityID) {
	ecs.Delete(s.world, ecs.Id(id))
}

// Raw 暴露底层 ECS 世界，供调用方把本地组件类型（如 protocol-probe
// 的 sceneEntity）挂到同一批实体上。约定：调用方的注册必须在一次
// ecs.Write 中写入全部组件——分次补写会让实体在 archetype 间来回
// 搬移。除注册外，请优先使用本类型的类型化访问器。
func (s *World) Raw() *ecs.World {
	return s.world
}

func (s *World) Vitals(id worldcore.EntityID) (Vitals, bool) {
	return ecs.Read[Vitals](s.world, ecs.Id(id))
}

func (s *World) WriteVitals(id worldcore.EntityID, v Vitals) {
	ecs.Write(s.world, ecs.Id(id), ecs.C(v))
}

func (s *World) Brain(id worldcore.EntityID) (Brain, bool) {
	return ecs.Read[Brain](s.world, ecs.Id(id))
}

func (s *World) WriteBrain(id worldcore.EntityID, b Brain) {
	ecs.Write(s.world, ecs.Id(id), ecs.C(b))
}

// CombatCount 返回携带 Brain 的 NPC 数量，供 combatTick 的
// 空场景短路使用。
func (s *World) CombatCount() int {
	count := 0
	ecs.Query1[Brain](s.world).MapId(func(_ ecs.Id, _ *Brain) {
		count++
	})
	return count
}

type combatEntry struct {
	id     worldcore.EntityID
	vitals *Vitals
	brain  *Brain
}

type combatTransformEntry struct {
	id        worldcore.EntityID
	vitals    *Vitals
	brain     *Brain
	transform *worldcore.Transform
}

// EachCombatNPCTransform 在 EachCombatNPC 之上附带 Transform 组件
// 指针，供选靶几何等需要坐标的战斗路径单次遍历使用，
// 避免逐 ID Read 的双哈希查找。契约同 EachCombatNPC。
func (s *World) EachCombatNPCTransform(fn func(id worldcore.EntityID, v *Vitals, b *Brain, t *worldcore.Transform)) {
	entries := make([]combatTransformEntry, 0)
	ecs.Query3[Brain, Vitals, worldcore.Transform](s.world).MapId(func(id ecs.Id, b *Brain, v *Vitals, t *worldcore.Transform) {
		entries = append(entries, combatTransformEntry{id: worldcore.EntityID(id), vitals: v, brain: b, transform: t})
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
	for i := range entries {
		fn(entries[i].id, entries[i].vitals, entries[i].brain, entries[i].transform)
	}
}

// EachCombatNPC 以 EntityID 升序回调全部敌对 NPC，回调收到
// 指向组件存储的指针——就地修改即可生效，无需写回。
// 排序保证线上字节确定：archetype 的迭代顺序是插入顺序，
// 而视口反复注册/注销之后它不再等于 id 顺序。
//
// 契约：回调期间（以及收集与回调之间）禁止一切结构性 ECS
// 操作（spawn/remove/增删组件）。combatTick 在同一把锁内
// 串行执行且只改组件值，天然满足；违反契约会使收集到的
// 指针失效。
func (s *World) EachCombatNPC(fn func(id worldcore.EntityID, v *Vitals, b *Brain)) {
	entries := make([]combatEntry, 0)
	ecs.Query2[Brain, Vitals](s.world).MapId(func(id ecs.Id, b *Brain, v *Vitals) {
		entries = append(entries, combatEntry{id: worldcore.EntityID(id), vitals: v, brain: b})
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
	for i := range entries {
		fn(entries[i].id, entries[i].vitals, entries[i].brain)
	}
}
