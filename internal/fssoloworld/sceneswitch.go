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
//	→ 0x1F+vitals 大批 → 开图双记录 → 0x0C 清场重建簇（AutoTrace 寻路
//	  闸门/基础属性/Task·轻功记录行/就绪信息/buff/View 40/43/46/战斗
//	  属性/键位·快捷栏）→ NPC AOI 首拍 → 邮箱重放 → 存档。
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
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
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
		ident := pl.wireIdent()
		s.sendToClient(pl, entergen.LocationFrame(ident.objectID, ident.ownerID,
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
		// 进场分配先于本次进场的任何帧；0x0B/0x0D 生成失败即撤销（客户端
		// 仍停在旧场景，旧 ident 继续有效）。
		prevIdent := pl.wireIdent()
		s.assignPlayerIdent(pl)
		ctx := s.enterContext(pl, false) // 新场景 config/resource + 本次进场 ident
		entry, err := entergen.EntrySceneFrame(ctx.Identity, dest.Config, dest.Resource)
		if err != nil {
			pl.setWireIdent(prevIdent)
			s.trace("scene switch 0x0B failed client=%d: %v", pl.Conn1, err)
			return
		}
		appearance, err := entergen.AppearanceFrame(ctx)
		if err != nil {
			pl.setWireIdent(prevIdent)
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
// → 开图双记录 → 0x0C 清场重建簇（AutoTrace/基础属性/记录行/就绪信息/
// buff/视图族/战斗属性/键位快捷栏）→ NPC AOI 首拍 → 邮箱重放 → 存档。
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
	// ★0x0C 清场重建簇（参考端 stable re-entry 整表定案：ServerExitScene
	// 连玩家对象一起拆除，挂对象的一切——属性视图/记录行/View 40/43/46——
	// 全部随清，重入后须按登录就绪簇同序整推重建）。各域缺席的真机症状：
	// AutoTrace 缺=点地图不发起寻路（msg266 不发）；View43 缺=faculty
	// 普通开始按钮禁用；战斗属性簇缺=角色栏信息空白；键位/快捷栏缺=自定义
	// 配置全丢。705 连接级不随切图重置（2433 定论），不重发。
	if autoTrace, err := entergen.AutoTraceFrame(ctx); err == nil {
		s.sendToClient(pl, autoTrace)
		sent++
	}
	if base, err := entergen.BasePropertyFrames(ctx); err == nil {
		for _, frame := range base {
			s.sendToClient(pl, frame)
			sent++
		}
	} else {
		s.trace("reentry base props failed client=%d: %v", pl.Conn1, err)
	}
	// 记录行族（Task_Completed 位图=功能解锁域 / Task_Accepted / QingGongRec
	// 18 行——参考端 grantFunctionUnlock+grantStarterQingGong 对应物）。
	if rows, err := entergen.MountSequenceFrames(ctx); err == nil {
		for _, frame := range rows {
			s.sendToClient(pl, frame)
			sent++
		}
	} else {
		s.trace("reentry record rows failed client=%d: %v", pl.Conn1, err)
	}
	// 就绪信息簇 m607/m52/m586 + 登录 buff m169（官方切图=第二爆发整段
	// 重演，wholesale 两相位定案——随 0x0C 清对象丢失，整推重建）。
	if info, err := entergen.ReadyInfoFrames(ctx, time.Now()); err == nil {
		for _, frame := range info {
			s.sendToClient(pl, frame)
			sent++
		}
	} else {
		s.trace("reentry ready info failed client=%d: %v", pl.Conn1, err)
	}
	if buffs, err := entergen.LoginBuffFrames(ctx); err == nil {
		for _, frame := range buffs {
			s.sendToClient(pl, frame)
			sent++
		}
	} else {
		s.trace("reentry login buffs failed client=%d: %v", pl.Conn1, err)
	}
	// 视图族重建（每帧自带容器声明）：View40 武学 / View43 内功 / View46 轻功。
	for _, build := range []func(rolestate.RoleState) ([][]byte, error){
		entergen.SkillViewFrames, entergen.NeiGongViewFrames, entergen.QingGongViewFrames,
	} {
		frames, err := build(pl.roleState)
		if err != nil {
			s.trace("reentry view frames failed client=%d: %v", pl.Conn1, err)
			continue
		}
		for _, frame := range frames {
			s.sendToClient(pl, frame)
			sent++
		}
	}
	// 战斗属性单帧族（真机「切图后角色栏信息没有下发」定案：随 0x0C 丢失）。
	if combat, err := entergen.CombatAttributeFrames(ctx); err == nil {
		for _, frame := range combat {
			s.sendToClient(pl, frame)
			sent++
		}
	} else {
		s.trace("reentry combat attrs failed client=%d: %v", pl.Conn1, err)
	}
	// 键位域 + 快捷栏行（挂玩家对象——PlayerObjectConfigFrames 不含 705）。
	if cfg, err := entergen.PlayerObjectConfigFrames(ctx); err == nil {
		for _, frame := range cfg {
			s.sendToClient(pl, frame)
			sent++
		}
	} else {
		s.trace("reentry player config failed client=%d: %v", pl.Conn1, err)
	}
	// NPC AOI 首拍：视口/巡逻簿记重置（patrolReadyAt+5s 首拍缓冲）。
	for _, frame := range s.sceneNPCEnter(pl, pl.roleState.Resource, pos) {
		s.sendToClient(pl, frame)
		sent++
	}
	// 表帧（0x09 属性/记录全表）与 0x28 化身/物理流不重发（连接级存活，
	// 61036#8499 实证无表重发）；705 亦连接级不重发。挂玩家对象的域已由
	// 上方清场重建簇整推。
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
