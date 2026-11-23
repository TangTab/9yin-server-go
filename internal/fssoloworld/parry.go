package fssoloworld

// C2S 54 手动格挡（含非战斗态）。方言按参考端 protocol-probe 820f79a
// 真机多轮定版移植（非官方外服六件套——那个方言在 fork 客户端原生链上
// 「按下即塌」，2026-11-12 P2 5971 条死锁实证；2026-11-23 早真机复核：
// 我方首版照官方抓包回 m169+速度族，客户端 415 条 m54 全程无反应）：
//
//   - 双形判定：值 0/1 直判（fork 协议模式：0=按下、非 0=松开，Java
//     handleManualParry engage==0 同义）；其它值=官服族状态上报（B 线
//     官方方言登录后客户端每帧广播：基调=登录后首条上报值=未格挡、
//     按压=基调+K——2026-11-15 57719 会话 6984 条同值定案 + 2026-11-23
//     官方女号完整引导抓包 197 条全量复核（六纪元同差 K=0x11D5DA29、
//     长按 ~1.3s 周期重发按压令牌、松开回基调并重发、令牌按副本切景
//     轮换且纯客户端生成，详见 docs/official/官方女号抓包m54手动格挡
//     全解.md）。既非基调也非基调+K 的异值=轮换新基调，采纳并复位
//     （旧「首条基调永久锁存」在轮换后会把闲时心跳全量误判为按下，
//     造成永不松开的幻影格挡）。我方服务端走 B 线官方登录，真机
//     64344 会话全程恒 1=基调闲时上报）。
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

// parryTokenDelta 官方客户端按压令牌与基调的恒定差（0x11D5DA29；
// 2026-11-23 官方女号抓包六个纪元同差定案，按压=基调+K 且长按期间
// 周期重发同值——服务端据此区分「长按重发」与「切景轮换新基调」）。
const parryTokenDelta int64 = 299227689

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
// 恒 1、~1/s），按压发异值；0=按下的 fork 二值方言由 0/1 直判分支覆盖。
// 无变迁恒零帧（官服对长按期间的按压令牌重发同样零应答）。
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

// parryStateReport 官服族状态上报解码（2026-11-23 官方女号抓包 197 条
// 全量复核）：基调=首条上报值（未格挡）；按压令牌恒为基调+K（六纪元
// 同差定案，且长按期间 ~1.3s 周期重发按压令牌、松开回基调并重发）；
// 令牌按副本切景轮换且纯客户端生成（双向流零下发）——既非基调也非
// 基调+K 的异值只可能是轮换后的新基调，采纳之并复位格挡态。否则轮换
// 后闲时心跳全部误判为按下，造成永不松开的幻影格挡（按下即塌同族）。
// 值域 {0,1} 走 fork 二值方言直判（0=按下，与基调采集无序：基调若先
// 锁 0 也不会反判）。
func (s *Server) parryStateReport(pl *Player, value int64) bool {
	pl.parryMu.Lock()
	defer pl.parryMu.Unlock()
	if pl.parryBaseline == 0 {
		if value == 0 {
			// fork 二值方言首报即按下（哨兵 0 与按下值撞车）：锚定
			// 空闲值 1 后直判，避免基调锁 0 造成永久反判。
			pl.parryBaseline = 1
			return true
		}
		pl.parryBaseline = value
		return false
	}
	if value == pl.parryBaseline {
		return false
	}
	if value == pl.parryBaseline+parryTokenDelta {
		return true
	}
	if value == 0 || value == 1 {
		return value == 0
	}
	pl.parryBaseline = value
	return false
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
