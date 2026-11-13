package main

// 副本实例生命周期（dungeon lifecycle）。
//
// enter_clone 创建实例并经 0x0C 重入序列进入副本场景；实例运行时以
// sceneKey.instance 隔离（同 config 多实例并存互不干扰），managed 常
// 驻——末成员离开不自动销毁，供 reenter_clone 反复进入；destroy_clone
// 显式销毁并回跳进本前位置。本期只做生命周期：副本 NPC 按共享场景
// 同款 populate 全量注册，仅以 noRespawn 保证杀完不复活；波次逐级
// 揭示（creator.NpcProgress）属单个副本内容层，后续接入。
//
// 位置持久化口径：进副本期间不持久化 clone 场景位置（防重启后落进
// 共享态副本场景），存档锚点保持 returnTo；destroy 回城后正常落库。
// 官服进副本播报（m1 c1=3）协议未还原，本期不发。

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// dungeonInstance 是一次打开的副本实例。run 在成功进入后回填
//（attachRun）；期间为 nil 表示登记后进入失败/尚未完成。
type dungeonInstance struct {
	id         string        // 实例 ID（即 sceneKey.instance）
	key        sceneKey      // 完整场景键（config/resource/instance）
	difficulty int           // 难度档 1=普通 2=困难 3=地狱
	ownerRole  role.RoleID   // 一期单人归属（无组队系统）
	returnTo   role.Location // 进本前位置（销毁回跳与存档锚点）
	entry      role.Position // 实例入口落点（reenter 用）
	createdAt  time.Time
	finished   bool // 末波全灭（引擎 finishHook 置位；属主会话 ticker 执行回城+销毁）
	run        *sceneRun
}

// dungeonManager 登记每个角色当前打开的副本实例。实例运行时本身已
// 在全局 sceneRegistry（按 sceneKey 含 instance），manager 只持有生命
// 周期侧的归属与回跳信息。
type dungeonManager struct {
	mu      sync.Mutex
	nextID  uint64
	byOwner map[role.RoleID]*dungeonInstance
}

var dungeons = &dungeonManager{byOwner: map[role.RoleID]*dungeonInstance{}}

// open 返回该角色当前打开的实例；无则 nil。
func (m *dungeonManager) open(owner role.RoleID) *dungeonInstance {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byOwner[owner]
}

// create 登记新实例并分配实例 ID；该角色已有打开实例时拒绝（前端
// 文案口径：先销毁再进入）。run 留空，由 attachRun 在成功进入后回填。
func (m *dungeonManager) create(owner role.RoleID, scene role.Scene, difficulty int, returnTo role.Location, entry role.Position) (*dungeonInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byOwner[owner]; exists {
		return nil, fmt.Errorf("role %d already has an open dungeon (destroy it first)", owner)
	}
	m.nextID++
	id := fmt.Sprintf("dn-%d-%d", owner, m.nextID)
	instance := &dungeonInstance{
		id:         id,
		key:        sceneKey{config: scene.Config, resource: scene.Resource, instance: id},
		difficulty: difficulty,
		ownerRole:  owner,
		returnTo:   returnTo,
		entry:      entry,
		createdAt:  time.Now(),
	}
	m.byOwner[owner] = instance
	return instance, nil
}

// attachRun 在成功进入副本后回填运行时并置实例标记（managed 常驻 +
// noRespawn 副本怪不复活）。调用方是属主会话 goroutine：begin/attach
// 与本调用之间不存在并发 detach 路径（成员只能被属主会话自身摘除）。
func (m *dungeonManager) attachRun(owner role.RoleID, run *sceneRun) *dungeonInstance {
	m.mu.Lock()
	defer m.mu.Unlock()
	instance, exists := m.byOwner[owner]
	if !exists || instance == nil {
		return nil
	}
	instance.run = run
	run.mu.Lock()
	run.managed = true
	run.noRespawn = true
	run.mu.Unlock()
	return instance
}

// take 摘除并返回该角色的打开实例；无则 nil。调用方随后自行 shutdown
// 返回实例的 run（shutdownInstance）。
func (m *dungeonManager) take(owner role.RoleID) *dungeonInstance {
	m.mu.Lock()
	defer m.mu.Unlock()
	instance := m.byOwner[owner]
	delete(m.byOwner, owner)
	return instance
}

// markFinished 标记实例通关（波次引擎 finishHook 在锁外调用）：只动
// manager 锁；回城+销毁由属主会话 ticker 的自动收尾分支执行。
func (m *dungeonManager) markFinished(owner role.RoleID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if instance, exists := m.byOwner[owner]; exists {
		instance.finished = true
	}
}

// shutdownInstance 幂等关闭实例运行时（停 tick/定时器、摘出注册表）。
// 已关闭或从未进入（run 为 nil）时安全返回。
func (m *dungeonManager) shutdownInstance(instance *dungeonInstance) {
	if instance == nil || instance.run == nil {
		return
	}
	instance.run.mu.Lock()
	if !instance.run.closed {
		instance.run.shutdownLocked()
	}
	instance.run.mu.Unlock()
}

// dropOnDisconnect 断线清场：摘除登记并空场关闭运行时。不回跳——
// 连接已断，副本位置本就不落库，重登自然回到 returnTo（存档锚点）。
func (m *dungeonManager) dropOnDisconnect(owner role.RoleID) {
	m.shutdownInstance(m.take(owner))
}

// cloneSceneOf 把 cloneinfo.ini 节名（ini\scene\clone003 原样）解析为
// 场景标识：地形名按副本口径取完整配置基名（与 resourceForDoorScene
// 同规则——city/school/born/scene 前缀取首段，副本图直接用全名）。
func cloneSceneOf(config string) (role.Scene, error) {
	resource, err := resourceForDoorScene(config)
	if err != nil {
		return role.Scene{}, err
	}
	return role.Scene{Config: config, Resource: resource}, nil
}

// cloneCatalogRow 在 cloneinfo.ini 目录（GM 目录层装载）中查找副本行。
// 目录未装载（资源树缺席）或无该节名时返回 nil。
func cloneCatalogRow(config string) *gmCloneRow {
	rows := loadGMCloneRows()
	for index := range rows {
		if rows[index].Config == config {
			return &rows[index]
		}
	}
	return nil
}

// dungeonDifficultyName 难度档的中文名（gmSession 回报文本用）。
func dungeonDifficultyName(difficulty int) string {
	switch difficulty {
	case 2:
		return "困难"
	case 3:
		return "地狱"
	default:
		return "普通"
	}
}

// dungeonEntryPosition 从副本 NPC 目录推导入口落点：全部合格战斗 NPC
//（attacknpc/bossnpc）的质心，向 X/Z 各偏 4m（约 5.6m，避免落点正压
// 在某只怪身上）。无战斗 NPC 时回落第一个合格出生点；空目录返回
// 零位。落点高度取质心均值（creator y 即地形高度）。
func dungeonEntryPosition(catalog []npcSpawn) role.Position {
	sumX, sumY, sumZ := float32(0), float32(0), float32(0)
	combat := 0
	for _, npc := range catalog {
		if !npcSpawnEligible(npc) {
			continue
		}
		class := strings.ToLower(strings.TrimSpace(npc.resolved.ScriptClass))
		if class != "attacknpc" && class != "bossnpc" {
			continue
		}
		sumX += npc.x
		sumY += npc.y
		sumZ += npc.z
		combat++
	}
	if combat > 0 {
		return role.Position{
			X: sumX/float32(combat) + 4,
			Y: sumY / float32(combat),
			Z: sumZ/float32(combat) + 4,
		}
	}
	for _, npc := range catalog {
		if npcSpawnEligible(npc) {
			return role.Position{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient}
		}
	}
	return role.Position{}
}
