package fssoloworld

// sceneswitch.go 场景切换（2433 scene_transition.go SwitchScene 真机时序
// 移植，2026-11 定版三坑照抄）：
//
//	0x0C 单字节退出（客户端原生拆除对象/记录，服务端不逐个发 0x0E）
//	→ 750ms 间隙（旧 stage Lua on_exit_scene ClearAll 异步，紧跟的
//	  0x0B/0x0D 会被清掉——sceneSwitchGap）
//	→ 0x0B 进场（新场景 config/resource，seq 自增）
//	→ 0x0D 玩家重物化（目标 stage_main 需要本地 1-1 对象才能开图；
//	  0x1F 必须推迟，否则坐标被投影到旧地形）
//	→ 等客户端目标场景自己的 0x09（期间旧场景过期 0x0A 全丢弃）
//	→ 0x1F+vitals 大批 → 开图双记录 → NPC AOI 首拍 → Ready 整段重推
//	  → 邮箱重放 → 存档。
//
// 同场景传送（gm tp/传送门 gotodoor 族）只发 0x1F，不走本序列。

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// sceneSwitchGap 0x0C→0x0B 间隙（包级变量：测试可调短；2433 实测 750ms）。
var sceneSwitchGap = 750 * time.Millisecond

// sceneSwitchTimeout 就绪屏障兜底：0x0D 后该窗口内客户端未回 0x09 则
// 丢弃切换态（位移收编恢复；玩家可重走 GM 命令）。
const sceneSwitchTimeout = 30 * time.Second

// sceneSwitchState 一次切换的进行态（player.sceneSwitch 挂点）。
type sceneSwitchState struct {
	dest       sceneDestination
	startedAt  time.Time
	awaitReady bool   // 0x0B/0x0D 已发，等目标场景 0x09
	source     string // 触发源（gm scene/portal）——trace 用
}

// sceneSwitchInFlight 切换在途（位移收编守卫：旧场景过期 0x0A 丢弃）。
func (s *Server) sceneSwitchInFlight(pl *Player) bool {
	pl.sceneSwitchMu.Lock()
	defer pl.sceneSwitchMu.Unlock()
	return pl.sceneSwitch != nil
}

// pendingSceneSwitch 就绪屏障查询（clientReady 分支：awaitReady 才接管）。
func (s *Server) pendingSceneSwitch(pl *Player) *sceneSwitchState {
	pl.sceneSwitchMu.Lock()
	defer pl.sceneSwitchMu.Unlock()
	if pl.sceneSwitch != nil && pl.sceneSwitch.awaitReady {
		return pl.sceneSwitch
	}
	return nil
}

// transferPlayer 执行传送/切图（gm scene 与传送门共用执行器）：
// 同场景目标=仅 0x1F 落位；跨场景=完整重入序列（异步段 goroutine 发）。
func (s *Server) transferPlayer(pl *Player, dest sceneDestination, source string) error {
	if !pl.haveRoleState {
		return errors.New("no role state (not entered)")
	}
	if s.sceneSwitchInFlight(pl) {
		return errors.New("scene switch already in progress")
	}
	if dest.Config == "" || dest.Resource == "" {
		return fmt.Errorf("bad destination (config=%q resource=%q)", dest.Config, dest.Resource)
	}
	if dest.Resource == pl.roleState.Resource {
		// 同场景传送：0x1F 落位 + 存档（gm tp 同款；不动 NPC 视口）。
		pl.roleState.Position = dest.Position
		s.sendToClient(pl, entergen.LocationFrame(forkPlayerObjectID, forkPlayerOwnerID,
			float32(dest.Position[0]), float32(dest.Position[1]),
			float32(dest.Position[2]), float32(dest.Position[3])))
		s.sendRoleSave(pl)
		s.trace("teleport %s client=%d → %.2f,%.2f,%.2f (0x1F)", source, pl.Conn1,
			dest.Position[0], dest.Position[1], dest.Position[2])
		return nil
	}
	// 跨场景：先落权威态（重登/中断兜底也指向新场景），开图点亮新场景位。
	pl.roleState.Scene = dest.Config
	pl.roleState.Resource = dest.Resource
	pl.roleState.Position = dest.Position
	sceneIndex := s.sceneGlobalIndexOf(dest.Resource)
	openBits := entergen.EffectiveOpenMapBits(pl.roleState, sceneIndex)
	pl.roleState.OpenMaps = openBits[:]
	if sceneIndex >= 0 {
		pl.sceneID = sceneIndex
	}
	state := &sceneSwitchState{dest: dest, startedAt: time.Now(), source: source}
	pl.sceneSwitchMu.Lock()
	if pl.sceneSwitch != nil {
		pl.sceneSwitchMu.Unlock()
		return errors.New("scene switch already in progress")
	}
	pl.sceneSwitch = state
	pl.sceneSwitchMu.Unlock()

	s.sendToClient(pl, entergen.ExitSceneFrame()) // 0x0C
	s.sendRoleSave(pl)                            // 2433 切图即存档口径
	s.trace("scene switch start client=%d %s → %s (%s) 0x0C sent", pl.Conn1,
		pl.roleState.Resource, dest.Resource, source)

	go func() {
		time.Sleep(sceneSwitchGap) // 旧 stage 清理异步窗口
		pl.sceneSwitchMu.Lock()
		current := pl.sceneSwitch
		pl.sceneSwitchMu.Unlock()
		if current != state {
			return // 已被超时清理/玩家离线
		}
		ctx := s.enterContext(pl, false) // 新场景 config/resource（seq 自增）
		entry, err := entergen.EntrySceneFrame(ctx.Identity, dest.Config, dest.Resource)
		if err != nil {
			s.trace("scene switch 0x0B failed client=%d: %v", pl.Conn1, err)
			return
		}
		appearance, err := entergen.AppearanceFrame(ctx)
		if err != nil {
			s.trace("scene switch 0x0D failed client=%d: %v", pl.Conn1, err)
			return
		}
		s.sendToClient(pl, entry)      // 0x0B
		s.sendToClient(pl, appearance) // 0x0D（0x1F 推迟到 0x09 后）
		pl.sceneSwitchMu.Lock()
		state.awaitReady = true
		pl.sceneSwitchMu.Unlock()
		time.AfterFunc(sceneSwitchTimeout, func() {
			pl.sceneSwitchMu.Lock()
			stale := pl.sceneSwitch == state
			if stale {
				pl.sceneSwitch = nil
			}
			pl.sceneSwitchMu.Unlock()
			if stale {
				s.trace("scene switch timeout client=%d (no 0x09 in %s), state cleared", pl.Conn1, sceneSwitchTimeout)
			}
		})
	}()
	return nil
}

// reentryReady 目标场景 0x09 → 重入簇（clientReady 分支接管）：0x1F+vitals
// → 开图双记录 → NPC AOI 首拍 → Ready 整段重推 → 邮箱重放 → 存档。
func (s *Server) reentryReady(pl *Player, pending *sceneSwitchState) {
	pl.sceneSwitchMu.Lock()
	pl.sceneSwitch = nil
	pl.sceneSwitchMu.Unlock()

	ctx := s.enterContext(pl, false)
	pos := pl.roleState.Position
	sent := 0
	// 0x1F 落位 + 0x10 人物数据大批（0x0D→0x1F→vitals 登录实证序）。
	s.sendToClient(pl, entergen.LocationFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID,
		float32(pos[0]), float32(pos[1]), float32(pos[2]), float32(pos[3])))
	sent++
	if vital, err := entergen.PlayerVitalBatchFrame(ctx); err == nil {
		s.sendToClient(pl, vital)
		sent++
	} else {
		s.trace("reentry vital frame failed client=%d: %v", pl.Conn1, err)
	}
	// 开图双记录：新场景迷雾行 + 玩家级累积位图（当前场景位已点亮）。
	if maps, err := entergen.MapRecordsFrames(ctx, ctx.SceneIndex); err == nil {
		for _, frame := range maps {
			s.sendToClient(pl, frame)
			sent++
		}
	} else {
		s.trace("reentry map records failed client=%d: %v", pl.Conn1, err)
	}
	// NPC AOI 首拍：视口/巡逻簿记重置（patrolReadyAt+5s 首拍缓冲）。
	for _, frame := range s.sceneNPCEnter(pl, pl.roleState.Resource, pos) {
		s.sendToClient(pl, frame)
		sent++
	}
	// 不重推 ReadyFrames 全家（2026-11-22 晚对齐 2433 切图重入实证
	// 61036#8499：0x0C→0x0B→0x0D→0x1F→0x11 行海→视口流——无表重发、
	// 无视图簇/战斗属性/自定义配置整段。0x0C 清空后需恢复的域由上方
	// 0x1F/vitals/开图/邮箱各段精确覆盖；705 连接级不随切图重置（2433
	// 定论）。
	// 邮箱重放（0x0C 清玩家对象上挂的记录，邮件行随重入簇恢复）。
	sent += s.pushLetterRows(pl)
	s.sendRoleSave(pl)
	s.trace("scene switch reentry ready client=%d scene=%s (%s) %d frames",
		pl.Conn1, pl.roleState.Resource, pending.source, sent)
}

// gmSceneCommand gm scene 子命令：gm scene <who> <key> [x y z [orient]]。
// key=别名/resource 短名/config|resource 直填；裸 `gm scene` 列可用 key。
func (s *Server) gmSceneCommand(nc net.Conn, pl *Player, who string, fields []string) {
	if len(fields) < 3 {
		for _, line := range strings.Split(scenePresetKeys(), "\n") {
			fmt.Fprintf(nc, ">%s\r", line)
		}
		return
	}
	dest, ok := resolveSceneDestination(fields[2])
	if !ok {
		fmt.Fprintf(nc, ">gm scene: unknown key %q (bare `gm scene` lists keys)\r", fields[2])
		return
	}
	// 坐标覆盖：x y z [orient]（直填形态必须给全三轴）。
	if len(fields) >= 6 {
		var pos [4]float64
		for i := 0; i < 3; i++ {
			f, err := strconv.ParseFloat(fields[3+i], 64)
			if err != nil {
				fmt.Fprintf(nc, ">gm scene: bad coord %q\r", fields[3+i])
				return
			}
			pos[i] = f
		}
		pos[3] = dest.Position[3]
		if len(fields) >= 7 {
			if f, err := strconv.ParseFloat(fields[6], 64); err == nil {
				pos[3] = f
			}
		}
		dest.Position = pos
	} else if strings.Contains(fields[2], "|") && dest.Position == [4]float64{} {
		fmt.Fprint(nc, ">gm scene: direct config|resource form requires explicit x y z\r")
		return
	}
	if err := s.transferPlayer(pl, dest, "gm scene"); err != nil {
		fmt.Fprintf(nc, ">gm scene: %v\r", err)
		return
	}
	fmt.Fprintf(nc, ">gm scene %s: → %s (%s) %.2f,%.2f,%.2f switching (0x0C→0x0B→0x0D→0x09)\r",
		who, dest.Resource, dest.Config, dest.Position[0], dest.Position[1], dest.Position[2])
}
