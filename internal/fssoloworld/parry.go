package fssoloworld

// C2S 54 手动格挡（含非战斗态）。方言按参考端 protocol-probe 820f79a
// 真机多轮定版移植，并经 2026-11-24 官服抓包（初始账号登陆-任务-入门2）
// 逐帧复核。
//
// ★非战斗格挡的总开关在客户端本地，服务端开不了这道门：设置界面
// 「调节战斗的自由度」里的「非战斗状态可招架」复选框（systeminfo.ini
// [Config] fightparry，默认 0；即 107/705 个性配置串第 94 字节，见
// docs/official/非战斗格挡开关与705个性配置字段表.md）。未勾选时客户端
// 本地拒绝进入格挡，m54 恒发基调——2026-11-23 早真机 415 条全程无按压
// 令牌即此。此前「按下即塌／需 LogicState=1 撑持」都是在该开关关闭下
// 的误判：撑持只是把客户端强推进战斗态才让它肯格挡。
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
//   - 不碰 LogicState。官服非战斗按压簇（2026-11-24 抓包 7 次连按）=
//     InParry=1 → ParryUpSpeedAdd=-3 → m169(0,"BuffInParry") →
//     InManualParry=1，松开全清，全程无 LogicState 帧。旧「按下升 1／
//     松开回 0 撑持」会让客户端瞬时进入战斗状态，松开回 0 还会误清真实
//     战斗架势——开关勾选后架势由客户端自己维持，撑持已撤。

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// parryMoveSpeedBase / parrySlowFactor / parryRunAddFactor / parryUpSpeedDelta
// 格挡减速参数（2026-11-24 官服抓包：MoveSpeed 3.9→1.17 = ×0.3、
// RunSpeedAdd 0→-2.73 = ×-0.7、ParryUpSpeedAdd 0→-1；松开全回基值）。
const (
	parryMoveSpeedBase = 3.9
	parrySlowFactor    = 0.3
	parryRunAddFactor  = -0.7
	parryUpSpeedDelta  = -1
)

// parryBuffName 格挡 buff（StaticData 4558=0x11CE，BufferInfo2 槽串首列；
// 无时长租约——挂/撤全走 m169 事件对与槽串登记）。
const parryBuffName = "BuffInParry"

// clientCustomParryState C2S msg54 格挡状态 [54, numeric]。
const clientCustomParryState int32 = 54

// parryTokenDelta 官方客户端按压令牌与基调的恒定差（0x11D5DA29；
// 2026-11-23 官方女号抓包六个纪元同差定案，按压=基调+K 且长按期间
// 周期重发同值——服务端据此区分「长按重发」与「切景轮换新基调」）。
const parryTokenDelta int64 = 299227689

// parryValueTotal 招架耐力总量（应答帧口径；无存档会话按新号基线 72）。
func (s *Server) parryValueTotal(pl *Player) int32 {
	if pl.haveRoleState {
		if total := int32(pl.roleState.Parry); total > 0 {
			return total
		}
	}
	return 72
}

// handleParryState C2S 54：纯基调状态机 + 三属性权威帧（不碰 LogicState）。
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

// sendParryStateFrame 官服应答簇（逐属性单帧，docs/official/初始女号
// 抓包m54手动格挡全解.md §四 逐帧钉死）：非战斗只置/清 InManualParry
//（输入状态；官服簇尾方言）；战斗上下文按压 = MoveSpeed 减速 →
// RunSpeedAdd → InParry → ParryUpSpeedAdd → BufferInfo2 槽串 →
// m169(0,"BuffInParry") → InManualParry；松开 m169(1) 簇首 →
// BufferInfo2 清空 → 减速族与三标志全清零。ParryValue 是按值变更推送
// 的耐力 meter（parryRegen），不入簇；全程不碰 LogicState。
func (s *Server) sendParryStateFrame(pl *Player) {
	pl.parryMu.Lock()
	parrying := pl.parrying
	pl.parryMu.Unlock()
	st := &pl.combat
	st.mu.Lock()
	inCombat := st.live.ready && st.live.logic == logicFighting
	st.mu.Unlock()
	if !inCombat {
		s.sendParryProp(pl, entergen.NamedProperty("InManualParry",
			entergen.Int32Value(boolToInt(parrying))))
		return
	}
	id := pl.wireIdent()
	if parrying {
		s.sendParryProp(pl, entergen.NamedProperty("MoveSpeed",
			entergen.Float32Value(float32(parryMoveSpeedBase*parrySlowFactor))))
		s.sendParryProp(pl, entergen.NamedProperty("RunSpeedAdd",
			entergen.Float32Value(float32(parryMoveSpeedBase*parryRunAddFactor))))
		s.sendParryProp(pl, entergen.NamedProperty("InParry", entergen.Int32Value(1)))
		s.sendParryProp(pl, entergen.NamedProperty("ParryUpSpeedAdd",
			entergen.Int32Value(parryUpSpeedDelta)))
		s.sendParryProp(pl, entergen.NamedProperty("BufferInfo2", entergen.StringValue(
			fmt.Sprintf("11CE,%X,%X,1,0,0", id.objectID, id.ownerID))))
		s.sendParryBuffNotice(pl, true)
		s.sendParryProp(pl, entergen.NamedProperty("InManualParry", entergen.Int32Value(1)))
		return
	}
	s.sendParryBuffNotice(pl, false)
	s.sendParryProp(pl, entergen.NamedProperty("BufferInfo2", entergen.StringValue("")))
	s.sendParryProp(pl, entergen.NamedProperty("MoveSpeed",
		entergen.Float32Value(parryMoveSpeedBase)))
	s.sendParryProp(pl, entergen.NamedProperty("RunSpeedAdd", entergen.Float32Value(0)))
	s.sendParryProp(pl, entergen.NamedProperty("InParry", entergen.Int32Value(0)))
	s.sendParryProp(pl, entergen.NamedProperty("ParryUpSpeedAdd", entergen.Int32Value(0)))
	s.sendParryProp(pl, entergen.NamedProperty("InManualParry", entergen.Int32Value(0)))
}

// sendParryProp 单属性 0x10 帧（官服簇逐属性方言）。
func (s *Server) sendParryProp(pl *Player, prop entergen.IndexedProperty) {
	if frame, err := playerPropertyFrame(pl, prop); err == nil {
		s.sendToClient(pl, frame)
	} else {
		s.trace("parry prop frame encode failed client=%d: %v", pl.Conn1, err)
	}
}

// sendParryBuffNotice m169 格挡 buff 挂/撤（BuffInParry=StaticData 4558
// 无时长租约：到期段恒 0；目标/施法者=本人，名字对=角色名）。
func (s *Server) sendParryBuffNotice(pl *Player, add bool) {
	id := pl.wireIdent()
	flag := int32(1)
	if add {
		flag = 0
	}
	frame, err := entergen.CustomMessageFrame(entergen.MsgBuff,
		entergen.CustomInt(flag),
		entergen.CustomString(parryBuffName),
		entergen.CustomObject(id.objectID, id.ownerID),
		entergen.CustomObject(id.objectID, id.ownerID),
		entergen.CustomString(pl.Name),
		entergen.CustomString(pl.Name),
		entergen.CustomInt64(0),
		entergen.CustomInt(0))
	if err != nil {
		s.trace("parry buff notice encode failed client=%d: %v", pl.Conn1, err)
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
