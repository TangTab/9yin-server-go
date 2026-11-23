package fssoloworld

// C2S 54 手动格挡（含非战斗态）。方言按参考端 protocol-probe 820f79a
// 真机多轮定版移植（非官方外服六件套——那个方言在 fork 客户端原生链上
// 「按下即塌」，2026-11-12 P2 5971 条死锁实证；2026-11-23 早真机复核：
// 我方首版照官方抓包回 m169+速度族，客户端 415 条 m54 全程无反应）：
//
//   - 双形判定：值 0/1 直判（fork 协议模式：0=按下、非 0=松开，Java
//     handleManualParry engage==0 同义）；其它值=官服族状态上报（B 线
//     官方方言登录后客户端每帧广播：基调=登录后首条上报值=未格挡、
//     值≠基调=格挡中——2026-11-15 57719 会话 6984 条同值定案。我方
//     服务端走 B 线官方登录，真机 64344 会话全程恒 1=基调闲时上报）。
//   - 无变迁零帧（状态上报回帧会自激成洪流）；游离重复松开回帧重同步。
//   - 应答=三属性权威帧 InParry/InManualParry/ParryValue（server2
//     manualParryStateUpdate 同构）——fork 客户端格挡认知锚在 fighter
//     资源上；active 门槛 = parrying && ParryValue>0。
//   - 非战斗架势由 LogicState=1 撑住（2026-11-16 真机定版：战斗段 6s
//     保持 vs 预警段 50ms 即弃；三属性帧全删撑持曾致 6 按塌 5 回退）。
//     帧形=单属性帧但 idx 必须动态解析（我方表=465；旧 463 硬编错位
//     =「按下即塌」根因）。松开回写原值 0。

import (
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// clientCustomParryState C2S msg54 格挡状态 [54, numeric]。
const clientCustomParryState int32 = 54

// parryIdleValue 无在格挡时的 LogicState 还原位（我们未建 Z 拔刀权威态，
// 恒按空闲 0 回写；后续接 m292 时改为记 0/2 原值）。
const parryIdleValue byte = 0

// parryValueTotal 招架耐力总量（应答帧口径；无存档会话按新号基线 72）。
func (s *Server) parryValueTotal(pl *Player) int32 {
	if pl.haveRoleState {
		if total := int32(pl.roleState.Parry); total > 0 {
			return total
		}
	}
	return 72
}

// handleParryState C2S 54：纯基调状态机 + 三属性权威帧 + LogicState 撑持。
// 基调语义统一覆盖双方言：B 线登录客户端闲时恒发基调（真机 64344 会话
// 恒 1、~1/s），按压发异值（官方两抓包=成对 u64 令牌）；0=按下的 fork
// 二值方言在基调≠0 时同样落入「异值=按下」分支。无变迁恒零帧。
func (s *Server) handleParryState(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	value, ok := parryNumericValue(message.Values[1])
	if !ok {
		return
	}
	engage := s.parryStateReport(pl, value)

	pl.parryMu.Lock()
	already := pl.parrying
	if already == engage {
		pl.parryMu.Unlock()
		return // 无变迁零帧（回帧自激成洪流）
	}
	pl.parrying = engage
	pl.parryMu.Unlock()

	s.sendParryStateFrame(pl)
	// 非战斗架势撑持：按下升 1，松开回 0（单属性帧、idx 动态解析）。
	if engage {
		s.sendParryLogicState(pl, 1)
	} else {
		s.sendParryLogicState(pl, parryIdleValue)
	}
	s.trace("manual parry engage=%t value=%d baseline=%d client=%d",
		engage, value, pl.parryBaseline, pl.Conn1)
}

// parryStateReport 官服族状态上报解码：基调=首条上报值（未格挡），
// 值≠基调=格挡中，回基调=取消。
func (s *Server) parryStateReport(pl *Player, value int64) bool {
	pl.parryMu.Lock()
	defer pl.parryMu.Unlock()
	if pl.parryBaseline == 0 {
		pl.parryBaseline = value
	}
	return value != pl.parryBaseline
}

// sendParryStateFrame 三属性权威帧（InParry/InManualParry/ParryValue）。
func (s *Server) sendParryStateFrame(pl *Player) {
	pl.parryMu.Lock()
	parrying := pl.parrying
	pl.parryMu.Unlock()
	value := s.parryValueTotal(pl)
	active := int32(0)
	if parrying && value > 0 {
		active = 1
	}
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0, []entergen.IndexedProperty{
		entergen.NamedProperty("InParry", entergen.Int32Value(active)),
		entergen.NamedProperty("InManualParry", entergen.Int32Value(active)),
		entergen.NamedProperty("ParryValue", entergen.Int32Value(value)),
	})
	if err != nil {
		s.trace("parry state frame encode failed client=%d: %v", pl.Conn1, err)
		return
	}
	s.sendToClient(pl, frame)
}

// sendParryLogicState 架势撑持帧（LogicState 单属性；idx 走表动态解析）。
func (s *Server) sendParryLogicState(pl *Player, state byte) {
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty("LogicState", entergen.ByteValue(state))})
	if err != nil {
		s.trace("parry logic state frame encode failed client=%d: %v", pl.Conn1, err)
		return
	}
	s.sendToClient(pl, frame)
}

// parryNumericValue m54 值域统一提取（tag2/3/4/5 积分形态）。
func parryNumericValue(value entergen.CustomInboundValue) (int64, bool) {
	switch value.Type {
	case 2:
		return int64(value.Int32), true
	case 3:
		return value.Int64, true
	case 4, 5:
		number, ok := value.ExactInt32()
		if !ok {
			return 0, false
		}
		return int64(number), true
	default:
		return 0, false
	}
}
