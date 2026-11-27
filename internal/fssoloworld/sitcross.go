package fssoloworld

// sitcross.go 原生打坐：C2S 240 态机 + 每秒回复结算。移植参考端
// official-2433-b-line（cmd/protocol-probe handleSitcrossCustom /
// sitcrossTick·sitcrossRecoveryFrames、sim.Vitals.RestoreWhileSit）。
//
//	进入  客户端把已学的 zs_default_01 识别为原生动作，按快捷键即发
//	      [240, 1]（不走 211 战斗路线）→ BufferInfo2 槽串（buf_baosd_02,
//	      StaticData 1E37）+ m169(0,"buf_baosd_02") 挂 buff——官方客户端
//	      的打坐态键控在这对帧上（男号线 #1293/#1294 实证；旧 fork 方言
//	      LogicState=102 + m24 dazuo_old 官方全程不发，真机打坐不生效
//	      根因，2026-11-27 撤）。按住按键期间客户端重发 start：同态请求
//	      空转（不当切换开关，也不拿重复帧淹没客户端）。
//	退出  [240, 0] 正常起立 / [240, 2] 本地被打断（含被敌方命中）→
//	      m169(1) 撤簇首 + BufferInfo2 清空（登录清理 #79/#80 同形）。
//	回复  combatTick 每秒一跳：HP/MP 各 +ceil(上限/10)——存活角色十秒
//	      回满，不看装备/内功推导出的总量。帧序 HPRatio → HP → m24
//	      治疗/回蓝飘字（c1=16/18），MP 同形。回复帧簇不带 LogicState：
//	      整帧重推会让客户端重入坐姿动画。
//	打断  战斗活动把 LogicState 顶成战斗态即离开打坐（noteFightLocked，
//	      脱战链照常），此后客户端补发的 [240, 2] 因已不在坐姿而空转。
//	      战斗/脱战过渡期内拒绝进入——否则到点的脱战检查会把刚坐下的
//	      角色顶回常态。

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

const (
	// sitcrossFullRecoverySeconds 打坐回满节拍数：每跳回复 ceil(上限/10)。
	sitcrossFullRecoverySeconds = 10
	// sitcrossSkillID 原生打坐技能（starter 快捷栏条目；飘字来源位）。
	sitcrossSkillID = "zs_default_01"
	// m24 子码：16 治疗飘字、18 回蓝飘字（回复 tick 用；进出簇不发
	// m24——官服方言走 buff 注册对）。
	msg24HealHP, msg24HealMP = 16, 18
)

// sitting 玩家是否在打坐。
func (pl *Player) sitting() bool {
	st := &pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.live.ready && st.live.logic == logicSitcross
}

// sitRecoveryPerSecond 打坐每跳回复量 ceil(上限/10)（与登录面板 *UpSpeed
// 基础位同式）：向上取整，不能整除的推导上限也在十跳内回满。
func sitRecoveryPerSecond(maximum int) int {
	if maximum <= 0 {
		return 0
	}
	return (maximum + sitcrossFullRecoverySeconds - 1) / sitcrossFullRecoverySeconds
}

// handleSitcross C2S 240 原生打坐切换 [240, int startOrStop]（文件头）。
func (s *Server) handleSitcross(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	flag, ok := message.Values[1].ExactInt32()
	if !ok {
		return
	}
	enter := flag == 1
	if !enter && flag != 0 && flag != 2 {
		s.trace("sitcross: 未知 start_or_stop=%d client=%d", flag, pl.Conn1)
		return
	}
	st := &pl.combat
	st.mu.Lock()
	s.ensureLiveLocked(st, pl)
	from := st.live.logic
	switch {
	case enter && from == logicSitcross, !enter && from != logicSitcross:
		// 重复 start / 已不在坐姿（多为受击后客户端补发的打断）。
		st.mu.Unlock()
		return
	case enter && !st.live.fightAt.IsZero():
		st.mu.Unlock()
		s.trace("sitcross: 战斗/脱战过渡中拒绝进入 client=%d", pl.Conn1)
		return
	}
	to := byte(logicNormal)
	if enter {
		to = logicSitcross
	}
	st.live.logic = int(to)
	// 官服进入/退出簇（男号线 #1293/#1294 与登录清理 #79/#80 实证）：
	// 进入 = BufferInfo 槽 CSV（打坐 buff buf_baosd_02，StaticData 1E37）
	// → m169(0,"buf_baosd_02")；退出 = m169(1) 簇首 → 槽清空。官方客户端
	// 的打坐态/回血显示键控在这对帧上；LogicState=102 + m24 dazuo_old
	// 是 fork 线方言（官方全程不发 LogicState，真机 2026-11-27 打坐不生
	// 效根因）。逻辑位 logicSitcross 仍是服务端回复 tick 的内部闸门。
	id := pl.wireIdent()
	b := newFrameBuilder(pl)
	if enter {
		b.prop("BufferInfo2", entergen.StringValue(
			fmt.Sprintf("%s,%X,%X,0,0,0", sitcrossBuffStatic, id.objectID, id.ownerID)))
		b.buffNotice(true, pl.Name)
	} else {
		b.buffNotice(false, pl.Name)
		b.prop("BufferInfo2", entergen.StringValue(""))
	}
	frames := b.frames
	st.mu.Unlock()
	s.sendFrames(pl, frames)
	s.trace("sitcross: native state %d -> %d via buf_baosd_02 registration client=%d", from, to, pl.Conn1)
}

// sitcrossBuffName / sitcrossBuffStatic 打坐 buff（StaticData 0x1E37；
// BufferInfo2 槽串首列，登录基线同源）。
const (
	sitcrossBuffName   = "buf_baosd_02"
	sitcrossBuffStatic = "1E37"
)

// buffNotice m169 打坐 buff 挂/撤（无时长：到期段 0；目标/施法者=本人，
// 名字对=角色名——男号线 #1294 九值同形）。
func (b *frameBuilder) buffNotice(add bool, name string) {
	flag := int32(1)
	if add {
		flag = 0
	}
	b.custom(entergen.MsgBuff,
		entergen.CustomInt(flag),
		entergen.CustomString(sitcrossBuffName),
		b.selfValue(),
		b.selfValue(),
		entergen.CustomString(name),
		entergen.CustomString(name),
		entergen.CustomInt64(0),
		entergen.CustomInt(0))
}

// sitTickLocked 打坐回复一跳（combatTick 每秒调，调用方持 combatState.mu）。
// 只在坐姿且存活时结算；返回本跳回复帧簇，nil = 不在打坐或已回满。
func (s *Server) sitTickLocked(pl *Player, st *combatState, maxMP int) [][]byte {
	if !st.live.ready || st.live.logic != logicSitcross || st.live.hp <= 0 {
		return nil
	}
	b := newFrameBuilder(pl)
	if hp := st.live.hp; hp < st.live.maxHP {
		st.live.hp = min(hp+sitRecoveryPerSecond(st.live.maxHP), st.live.maxHP)
		b.prop("HPRatio", entergen.Int32Value(hpRatio(st.live.hp, st.live.maxHP)))
		b.prop("HP", entergen.Int32Value(int32(st.live.hp)))
		sitFloat(b, msg24HealHP, st.live.hp-hp, sitRecoveryPerSecond(st.live.maxHP)-(st.live.hp-hp), pl.Name)
	}
	if mp := st.mp; mp < maxMP {
		st.mp = min(mp+sitRecoveryPerSecond(maxMP), maxMP)
		b.prop("MPRatio", entergen.Int32Value(mpRatio(st.mp, maxMP)))
		b.prop("MP", entergen.Int32Value(int32(st.mp)))
		sitFloat(b, msg24HealMP, st.mp-mp, sitRecoveryPerSecond(maxMP)-(st.mp-mp), pl.Name)
	}
	return b.frames
}

// sitFloat 打坐回复飘字 m24（10 值）：
//
//	[24, 自己, c1(16 治疗/18 回蓝), 实际回复, 溢出截断, 0,
//	 窄串"zs_default_01", 窄串"SkillNormal", 宽串受者名, 宽串受者名]
//
// 未触顶跳 (回复量, 0)；补满末跳 (实际, 溢出)——两者之和恒为名义回复量。
// 宽名双写为施法者/受者两位，打坐自愈时同为本人。
func sitFloat(b *frameBuilder, kind, applied, wasted int, name string) {
	b.custom(24, b.selfValue(), entergen.CustomInt(int32(kind)), entergen.CustomInt(int32(applied)),
		entergen.CustomInt(int32(wasted)), entergen.CustomInt(0), entergen.CustomString(sitcrossSkillID),
		entergen.CustomString("SkillNormal"), entergen.CustomWideString(name), entergen.CustomWideString(name))
}
