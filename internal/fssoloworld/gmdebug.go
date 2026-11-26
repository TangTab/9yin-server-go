package fssoloworld

// gmdebug.go GM 调试工具（gmcmd.go 的 rage / move 子命令；面板「GM 测试台 →
// 调试工具」）：设置怒气与怒气自动回复、移动参数（移动速度 / 下落重力）。
// 移植自参考端 9yin-go-server2 GM 面板的同名功能，帧与数值按本线口径重做。
//
// 怒气 = SP（docs/combat/寒心恨雪与太极剑舞引导机制.md §四）：命中回怒、
// 怒招消耗读写的都是战斗会话态 live.sp（combatcore.go ensureLiveLocked），
// 所以这里落的也是它——GM 设的值与玩家实际能用的怒气是同一份；客户端经
// 0x10 SP 单属性帧热发。上限 = live.maxSP（登录面板 MaxSP，与登录批下发
// 给客户端的上限同源；内功档加成 MaxSPAdd 未建模，登录批固定 80）。
// 参考端的怒气是写死的常量 100，本线上限取面板值，故不照搬。怒气每次
// 登录从 0 起（进场帧取会话态，enterContext），不随档。
//
// 自动回复/秒是会话级调试开关（combatState.rageRegen；连接生命周期、不落
// 库，重登归零）：combatTick 每秒 +N 至上限，0 = 关闭，不设上限（官方
// 满档 5）。
//
// 移动参数是纯客户端物理：MoveSpeed / RunSpeed / Gravity 都在进场 0x0D 帧
// 里（缺省 3.9 / 3.9 / 9.8），热发 0x10 单属性帧即生效，Gravity 在下一次
// 起跳时读取。同是会话级：切场景重入会重发带缺省值的 0x0D 与人物数据大批
//（后者含 Gravity），reentryReady 在重入簇里重推覆写；重登回缺省。

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

const (
	// gmMoveMin / gmMoveMax 移动速度、下落重力的取值范围。
	gmMoveMin, gmMoveMax = 0.1, 100.0
	// gmDefaultMoveSpeed / gmDefaultGravity 进场缺省值，与 entergen 0x0D 帧
	// 同值（appearance.go 的 MoveSpeed/RunSpeed 与 Gravity）：gm move reset
	// 回落、查询显示用。TestGmMoveDefaultsMatchAppearanceFrame 守两处不漂。
	gmDefaultMoveSpeed float32 = 3.9
	gmDefaultGravity   float32 = 9.8
)

// gmMotionState GM 调试移动参数的会话覆写（挂 Player.gmMotion；零值 = 未改）。
type gmMotionState struct {
	mu      sync.Mutex
	speed   float32 // MoveSpeed 与 RunSpeed 同值；0 = 未改
	gravity float32 // Gravity；0 = 未改
}

func (m *gmMotionState) get() (speed, gravity float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.speed, m.gravity
}

func (m *gmMotionState) set(speed, gravity float32) {
	m.mu.Lock()
	m.speed, m.gravity = speed, gravity
	m.mu.Unlock()
}

// ---- 帧 ----

// moveFrames 移动参数 0x10 单属性帧（玩家对象；一属性一帧，与登录物理流、
// 战斗热发同形）：speed>0 → MoveSpeed + RunSpeed，gravity>0 → Gravity。
func moveFrames(ident wireIdent, speed, gravity float32) [][]byte {
	b := &frameBuilder{ident: ident}
	if speed > 0 {
		b.prop("MoveSpeed", entergen.Float32Value(speed))
		b.prop("RunSpeed", entergen.Float32Value(speed))
	}
	if gravity > 0 {
		b.prop("Gravity", entergen.Float32Value(gravity))
	}
	return b.frames
}

// gmMoveFrames 会话内已改的移动参数（切场景重入后重推；未改 = 零帧）。
func (s *Server) gmMoveFrames(pl *Player) [][]byte {
	speed, gravity := pl.gmMotion.get()
	return moveFrames(pl.wireIdent(), speed, gravity)
}

// gmFloat 移动参数的应答/日志写法（float32 最短定形：3.9 / 25 / 9.8）。
func gmFloat(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

// ---- 怒气 ----

// rageRegenLocked 怒气自动回复一拍（combatTick 每秒调；调用方持
// combatState.mu）：速率 > 0 且未满时 +速率至上限，出 SP 单属性帧。
func (s *Server) rageRegenLocked(pl *Player, st *combatState) [][]byte {
	if st.rageRegen <= 0 || !st.live.ready || st.live.sp >= st.live.maxSP {
		return nil
	}
	st.live.sp = min(st.live.sp+st.rageRegen, st.live.maxSP)
	b := newFrameBuilder(pl)
	b.prop("SP", entergen.Int32Value(int32(st.live.sp)))
	return b.frames
}

// gmRageEdit gm rage 一次设置的落值（parseGMRage）。
type gmRageEdit struct {
	sp    int  // 设置后的怒气（已按上限钳过）
	setSP bool // 本次是否改了怒气（regen 形态只改速率）
	regen int  // 设置后的自动回复/秒
}

// parseGMRage gm rage 的设置形态（fields 自 sub 起）：<sp|max [regen]> 或
// regen <n>。参数不合法 ok=false（调用方回用法，不落值）。
func parseGMRage(fields []string, sp, maxSP, regen int) (gmRageEdit, bool) {
	e := gmRageEdit{sp: sp, regen: regen}
	regenAt := 0 // fields 里速率所在位（0 = 不改）
	switch {
	case strings.EqualFold(fields[2], "regen"):
		regenAt = 3
	case strings.EqualFold(fields[2], "max"):
		e.sp, e.setSP = maxSP, true
	default:
		n, err := strconv.Atoi(fields[2])
		if err != nil || n < 0 {
			return e, false
		}
		e.sp, e.setSP = min(n, maxSP), true
	}
	want := 3 // 应有字段数（rage/who/值）
	if regenAt > 0 || len(fields) > 3 {
		regenAt, want = 3, 4
	}
	if len(fields) != want {
		return e, false
	}
	if regenAt > 0 {
		// 速率不设上限（只挡溢出：一拍 +N 直接回满也算合法用法）。
		n, err := strconv.Atoi(fields[regenAt])
		if err != nil || n < 0 || n > math.MaxInt32 {
			return e, false
		}
		e.regen = n
	}
	return e, true
}

// gmRageCommand gm rage <who> [怒气|max [自动回复/秒]] | regen <自动回复/秒>：
// 设置怒气（0~上限，超出钳到上限；max = 回满），可同时带自动回复速率；
// regen 形态只改速率。裸 gm rage <who> 读当前值。
func (s *Server) gmRageCommand(nc net.Conn, pl *Player, who string, fields []string) {
	const usage = ">usage: gm rage <who> [sp|max [regen]] | regen <n> (regen: per-second, 0=off)\r"
	st := &pl.combat
	st.mu.Lock()
	s.ensureLiveLocked(st, pl)
	sp, maxSP, regen := st.live.sp, st.live.maxSP, st.rageRegen
	if len(fields) == 2 {
		st.mu.Unlock()
		fmt.Fprintf(nc, ">gm rage %s: sp=%d/%d regen=%d/s\r", who, sp, maxSP, regen)
		return
	}
	edit, ok := parseGMRage(fields, sp, maxSP, regen)
	if !ok {
		st.mu.Unlock()
		fmt.Fprint(nc, usage)
		return
	}
	st.live.sp, st.rageRegen = edit.sp, edit.regen
	st.mu.Unlock()
	note := "regen set"
	if edit.setSP {
		note = "sent (0x10 SP)"
		b := newFrameBuilder(pl)
		b.prop("SP", entergen.Int32Value(int32(edit.sp)))
		s.sendFrames(pl, b.frames)
	}
	s.trace("gm rage: client=%d sp=%d/%d regen=%d", pl.Conn1, edit.sp, maxSP, edit.regen)
	fmt.Fprintf(nc, ">gm rage %s: sp=%d/%d regen=%d/s %s\r", who, edit.sp, maxSP, edit.regen, note)
}

// ---- 移动参数 ----

// parseGmMoveValue 移动速度 / 下落重力取值：有限数，gmMoveMin~gmMoveMax。
func parseGmMoveValue(text string) (float32, bool) {
	v, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < gmMoveMin || v > gmMoveMax {
		return 0, false
	}
	return float32(v), true
}

// gmMoveCommand gm move <who> [speed|gravity <值> | reset]：移动速度同时更新
// MoveSpeed 与 RunSpeed；下落重力更新 Gravity（数值越大下落越快，下一次起跳
// 生效）；取值 0.1~100，reset 回进场缺省。裸 gm move <who> 读当前生效值。
func (s *Server) gmMoveCommand(nc net.Conn, pl *Player, who string, fields []string) {
	const usage = ">usage: gm move <who> [speed|gravity <0.1-100> | reset]\r"
	speed, gravity := pl.gmMotion.get()
	if len(fields) == 2 {
		if speed <= 0 {
			speed = gmDefaultMoveSpeed
		}
		if gravity <= 0 {
			gravity = gmDefaultGravity
		}
		fmt.Fprintf(nc, ">gm move %s: speed=%s gravity=%s\r", who, gmFloat(speed), gmFloat(gravity))
		return
	}
	// 发帧的那几项（reset = 两项都回落缺省并重发；0 = 不发）。
	sendSpeed, sendGravity := float32(0), float32(0)
	note := ""
	switch {
	case strings.EqualFold(fields[2], "reset") && len(fields) == 3:
		speed, gravity = 0, 0
		sendSpeed, sendGravity = gmDefaultMoveSpeed, gmDefaultGravity
		note = fmt.Sprintf("reset speed=%s gravity=%s sent (0x10)", gmFloat(sendSpeed), gmFloat(sendGravity))
	case len(fields) == 4 && strings.EqualFold(fields[2], "speed"):
		v, ok := parseGmMoveValue(fields[3])
		if !ok {
			fmt.Fprint(nc, usage)
			return
		}
		speed, sendSpeed = v, v
		note = "speed=" + gmFloat(v) + " sent (0x10 MoveSpeed+RunSpeed)"
	case len(fields) == 4 && strings.EqualFold(fields[2], "gravity"):
		v, ok := parseGmMoveValue(fields[3])
		if !ok {
			fmt.Fprint(nc, usage)
			return
		}
		gravity, sendGravity = v, v
		note = "gravity=" + gmFloat(v) + " sent (0x10 Gravity)"
	default:
		fmt.Fprint(nc, usage)
		return
	}
	pl.gmMotion.set(speed, gravity)
	s.sendFrames(pl, moveFrames(pl.wireIdent(), sendSpeed, sendGravity))
	s.trace("gm move: client=%d speed=%s gravity=%s", pl.Conn1, gmFloat(speed), gmFloat(gravity))
	fmt.Fprintf(nc, ">gm move %s: %s\r", who, note)
}
