package main

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/local/9yin-go-server/internal/gmapi"
	"github.com/local/9yin-go-server/internal/role"
)

// sendEntryScene 0x0B 场景档案。2026-11-10 真机教训：**0x0B 属性 idx
// 按连接收到的 0x09 表解析**——官服样本（c/登陆.pcapng）用官服表 idx
// 1447/1448 写 ConfigID/Resource，但本服客户端收到的是本服表（idx7=
// ConfigID/idx8=Resource）；照抄官服 idx 会把 Config 解析成 ClineNpcFile
// → 场景加载失败（真机实证，已回滚）。fork 版布局在本服表下自洽。
// 官服版与本版的实质差异只剩：attrCount 4 vs 3（官服多 idx178 Name 常量
// 段+idx1468 u16）、尾部 99/MaxVisuals、场景号=每次进场新流水（非查表
// key，无害）。寻路不初始化的根因不在 0x0B 属性 idx——继续排查。
func sendEntryScene(conn sceneMessageConnection, scene role.Scene) error {
	entry := make([]byte, 15, 96)
	entry[0] = 0x0B
	// fork 版布局（抓包实证）：[1:5] 场景对象ID（常量 0x1F6A3），
	// [5:9]/[9:13] 即玩家的线缆 objectID/ownerID，与玩家 0x0D 档案、
	// 0x1F/0x10/0x11 严格同源——三处必须一起改，否则客户端对象表断裂。
	binary.LittleEndian.PutUint32(entry[1:], 0x1F6A3)
	binary.LittleEndian.PutUint32(entry[5:], playerObjectID)
	binary.LittleEndian.PutUint32(entry[9:], playerOwnerID)
	binary.LittleEndian.PutUint16(entry[13:], 3)
	entry = appendStringProperty(entry, 7, scene.Config)
	entry = appendStringProperty(entry, 8, scene.Resource)
	entry = binary.LittleEndian.AppendUint16(entry, 99)
	entry = binary.LittleEndian.AppendUint32(entry, 200)
	return conn.WriteFrame(entry)
}

func encodeExitScene() []byte { return []byte{0x0C} }

type sceneDestination struct{ location role.Location }

// parseSceneDestination 接受 config,resource,x,y,z,orient。它是刻意显式
// 的测试/GM 输入；不猜测任何客户端动作。解析逻辑的权威实现在
// internal/gmapi（GM 校验与之共用），此处保留场景运行时侧的薄包装。
func parseSceneDestination(value string) (sceneDestination, error) {
	location, err := gmapi.ParseDestination(value)
	if err != nil {
		return sceneDestination{}, err
	}
	return sceneDestination{location: location}, nil
}

// parseScenePosition 解析仅限 GM 使用的同场景传送输入。场景标识刻意
// 省略：TeleportWithinScene 总是保持当前活动场景，并在重进入期间拒绝
// 使用。权威实现在 internal/gmapi，此处为薄包装。
func parseScenePosition(value string) (role.Position, error) {
	return gmapi.ParsePosition(value)
}

// sceneRuntime 负责单个活动玩家的重进入序列。它让 NPC 注册表和生命
// 周期世代与客户端保持同步。
type sceneRuntime struct {
	conn       sceneMessageConnection
	world      *sceneLifecycle
	machine    interface{ ReenterScene() error }
	registry   *sceneNPCRegistry
	staticNPCs []npcSpawn
	radius     float32
	player     *playerActor
	activeRole *role.RoleSnapshot
	npcCatalog []npcSpawn
	// 0x0C 之后的第一个 0x0A 是旧场景的过期活动。目标 stage 运行时玩家
	// 必须已经存在，但 NPC 必须等待稍后的目标 0x09 就绪通知。
	awaitingStableReentryReady bool
}

func (r *sceneRuntime) catalogForCurrentScene() ([]npcSpawn, error) {
	if r.registry == nil {
		return r.staticNPCs, nil
	}
	catalog, _, err := r.registry.catalogFor(r.activeRole.Location.Scene)
	if err != nil {
		return nil, err
	}
	return catalog, nil
}

// SwitchScene 启动一次现场场景替换。旧场景的 Lua 清理在 0x0C 之后异步
// 运行，所以 0x0B 不能紧跟其后：否则其延迟的 ClearAll 会移除替换后的
// 玩家。目标的 PlayerAddObject 必须紧随 EntryScene 到达，因为目标
// stage_main 需要本地 1-1 对象才能打开目标场景。其单独的 ServerLocation
// 移动更新推迟到最终的 0x09 ClientReady，否则在地形仍在切换时，客户端
// 会明显地把目的地投影到旧场景上。
func (r *sceneRuntime) SwitchScene(destination sceneDestination) error {
	if r.player == nil || r.activeRole == nil {
		return fmt.Errorf("switch scene without an active player")
	}
	destination.location.Scene = normalizeClientScene(destination.location.Scene)
	previous := r.activeRole.Location
	// parseSceneDestination 刻意不含持久化信息。替换场景时保留当前的
	// 乐观并发版本号，否则随后的 SaveLocation 会始终使用版本零而
	// 失败。
	destination.location.Version = previous.Version
	r.activeRole.Location = destination.location
	catalog, err := r.catalogForCurrentScene()
	if err != nil {
		r.activeRole.Location = previous
		return err
	}
	if err := r.machine.ReenterScene(); err != nil {
		return err
	}
	if err := r.conn.WriteFrame(encodeExitScene()); err != nil {
		r.activeRole.Location = previous
		return fmt.Errorf("write ExitScene: %w", err)
	}
	if err := r.world.begin(destination.location.Scene.Config, destination.location.Scene.Resource, catalog); err != nil {
		// 共享场景下 begin 失败只回滚本视图的位置与世界挂靠。
		r.activeRole.Location = previous
		return fmt.Errorf("begin replacement scene: %w", err)
	}
	// 实测客户端抓包：EntryScene 在 ExitScene 之后 12ms 到达。这正处在
	// 前一 stage 的清理期，必然丢失新玩家。保持网络顺序不变，但让
	// on_exit_scene 先完成它的 ClearAll。
	time.Sleep(750 * time.Millisecond)
	if err := sendEntryScene(r.conn, destination.location.Scene); err != nil {
		return fmt.Errorf("write EntryScene: %w", err)
	}
	if err := sendPlayerAddObject(r.conn, r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values)); err != nil {
		return fmt.Errorf("write player re-entry archive: %w", err)
	}
	r.world.attachMember(r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values))
	r.world.refreshVisibility(catalog, destination.location.Position)
	r.npcCatalog = catalog
	r.awaitingStableReentryReady = true
	return nil
}

// TeleportWithinScene 是即时生效的服务器权威挪移路径。它刻意只发送
// ServerLocation (0x1F)：ExitScene、EntryScene、PlayerAddObject 或就绪
// 屏障都不属于已加载地图内的坐标变更。
func (r *sceneRuntime) TeleportWithinScene(position role.Position) error {
	if r.player == nil || r.activeRole == nil {
		return fmt.Errorf("teleport without an active player")
	}
	if r.awaitingStableReentryReady {
		return fmt.Errorf("target scene is still loading")
	}
	previous := r.activeRole.Location.Position
	r.activeRole.Location.Position = position
	if err := r.conn.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {
		r.activeRole.Location.Position = previous
		return fmt.Errorf("write same-scene ServerLocation: %w", err)
	}
	if changed := r.world.updateMemberPosition(position, r.npcCatalog); changed {
		if err := r.world.reconcileViewport(); err != nil {
			return fmt.Errorf("reconcile teleported NPC viewport: %w", err)
		}
	}
	return nil
}
