package main

import (
	"fmt"
	"log"
	"time"

	"github.com/local/9yin-go-server/internal/entity"
)

// CLIENT_CUSTOMMSG_ACTIVE_PARRY 是现代客户端用于主动招架控制的共享 C->S 入口。
// 它被刻意与 211 分开：
// 各个套路用于提交攻击数据，而招架则是玩家的战斗状态请求，
// 该请求必须等到后续处理来袭攻击的命中判定时
// 再进行求值。
const clientCustomActiveParry int32 = 218

// activeParryRequest 刻意对 custom_sender 已知的两个字段做无损保留：
// 子类型（subtype）及其随附参数（argument）。我们保留原始的
// TVarList 值，而不是把第二个值强转成猜测的
// 枚举；其确切的游戏内含义将由实机抓包来判定。
type activeParryRequest struct {
	Subtype  int32
	Argument clientCustomValue
	At       time.Time
}

// handleActiveParryCustom 确立权威的入口边界。
// 它接受现代 TVarList 写入方使用的所有整数编码（门派
// UI 已经证明某些表单会把枚举值以 double 类型发出）。
// 它不虚构视觉动作，也不虚构固定的招架窗口：
// 这些取决于原生的命中结果路径，而该路径尚未得到验证。
func handleActiveParryCustom(player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomActiveParry {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("active parry before scene player exists")
	}
	if len(custom.Values) < 3 {
		return true, fmt.Errorf("active parry requires subtype and argument")
	}
	subtype, ok := custom.Values[1].exactInt32()
	if !ok {
		return true, fmt.Errorf("active parry subtype must be an integral TVarList value, got %s", custom.Values[1])
	}
	request := activeParryRequest{Subtype: subtype, Argument: custom.Values[2], At: time.Now().UTC()}
	player.setActiveParry(request)
	log.Printf("%s: accepted active parry subtype=%d argument=%s", remote, subtype, request.Argument)
	return true, nil
}

// CLIENT_CUSTOMMSG_BLOCK_STATE（fork handleBlockStateCustom：值 0=按下
// 格挡、非 0=松开；Java handleManualParry engage==0 同义）。
const clientCustomBlockState int32 = 54

// propLogicState 是预警切换回帧的单属性索引（官方表 463 号，byte 型）。
const propLogicState uint16 = 463

// handleChangeWarningCustom 处理 C2S m292（Z 拔/收刀，零参）。fork 客户端
// 「绝不自行发明状态」——Z 的执行完全依赖服务端回帧（9yin-go-server2
// handleChangeWarningCustom 权威方言，2026-11-17 七轮排查真机定版）：
// 受理仅翻 LogicState 0↔2，翻态后回 vitalUpdate 全量帧（FxGameLogic 收
// 面板里的 LogicState=2 播拔刀挂剑、收 0 播收刀）；战斗(1)/打坐(102)/
// 死亡(120)等态忽略。单属性 463 帧形态客户端不认（21:19 会话振荡实证）。
func handleChangeWarningCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomChangeWarning {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject change warning before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 1 {
		log.Printf("%s: ignore change warning with extra payload nvalues=%d", remote, len(custom.Values)-1)
		return true, nil
	}
	current := player.logicStateSnapshot()
	var next uint8
	switch current {
	case entity.LogicStateWarning:
		next = entity.LogicStateNormal
	case entity.LogicStateNormal:
		next = entity.LogicStateWarning
	default:
		// 客户端只在空闲静态/移动态发起切换；战斗(1)/打坐(102)/死亡(120)
		// 等态忽略请求，维持服务端权威状态（server2 同口径）。
		log.Printf("%s: ignore change warning from logic state %d", remote, current)
		return true, nil
	}
	player.setLogicState(next)
	frame, err := player.vitalUpdate()
	if err != nil {
		return true, fmt.Errorf("encode change warning state=%d: %w", next, err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("send change warning state=%d: %w", next, err)
	}
	log.Printf("%s: change warning %d -> %d via vitalUpdate", remote, current, next)
	return true, nil
}

// handleBlockStateCustom 还原手动格挡（Java handleManualParry 同源，
// clientCustomBlockState=54 即 C2S m54）。应答=外服私服六件套序
// （writeOfficialParryTokenAck，a.pcapng 5.2/5.7s 定版：按下
// MoveSpeed+RunSpeedAdd+InParry+ParryUpSpeedAdd+BufferInfo4 无时长槽串
// +m169 添加+InManualParry；松开反向+m169 移除）——此前 fork 四属性合帧
// 应答缺速度族，客户端 m54 连发 5971 条等不到外服私服方言（2026-11-12 P2）。
// 按下方言双形：fork 协议模式 0/1 按值（0=按下）；官服族客户端发令牌版
// （i64 成对随机令牌，2026-11-15 真机 58929 会话实证），按 parryTokenTransition
// 状态机判定。重复按下/松开均回帧（客户端状态重同步，78 发按值去重
// =50 沿=50 回帧的 F 盘实证）。非战斗也可按格挡（不切 LogicState）。
func handleBlockStateCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) error {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomBlockState {
		return nil
	}
	if player == nil {
		return fmt.Errorf("block state before scene player exists")
	}
	if len(custom.Values) < 2 {
		return fmt.Errorf("block state requires a value, got %d values", len(custom.Values))
	}
	state := custom.Values[1]
	// 虚招破防后的不可格挡窗：m54 全部静默吞并记住散逸令牌（格挡.pcapng
	// 定版：破防后客户端发新令牌×4 重试约 6s，官方零回应——窗后同值到达
	// 继续静默防误判成新按下）；真实按下（异值）走正常受理并清散逸记忆。
	if player.cantParrySnapshot() {
		player.noteParryStrayToken(state)
		return nil
	}
	if player.parryStraySnapshot() != 0 {
		if player.parryStrayMatches(state) {
			return nil
		}
		player.clearParryStrayToken()
	}
	// 双方言判定：fork 协议模式客户端发 0/1（0=按下、非 0=松开；Java
	// handleManualParry engage==0 同义）；官服族客户端（B 线官方方言登录
	// 后）的 m54 是**每帧状态上报**——值=基调（首条上报，未格挡）、
	// 值≠基调=格挡中（parryStateReport；2026-11-15 真机 57719 会话
	// 6984 条同值零间隙定案）。0/1 按值直判。
	engage := false
	tokenDialect := false
	stateReport := func(value int64) bool {
		tokenDialect = true
		return player.parryStateReport(value)
	}
	switch state.Type {
	case 2:
		if state.Int32 == 0 || state.Int32 == 1 {
			engage = state.Int32 == 0
		} else {
			engage = stateReport(int64(state.Int32))
		}
	case 3:
		if state.Int64 == 0 || state.Int64 == 1 {
			engage = state.Int64 == 0
		} else {
			engage = stateReport(state.Int64)
		}
	case 4, 5:
		number, ok := state.exactInt32()
		if !ok {
			return fmt.Errorf("block state value is not integral: %s", state)
		}
		if number == 0 || number == 1 {
			engage = number == 0
		} else {
			engage = stateReport(int64(number))
		}
	default:
		return fmt.Errorf("block state value must be numeric, got %s", state)
	}

	alreadyParrying := player.parryingSnapshot()
	if alreadyParrying == engage {
		if tokenDialect {
			// 状态上报无变迁：零帧（每帧上报，回任何帧都会自激成洪流）。
			return nil
		}
		if !engage {
			// 游离的重复松开：客户端清理被中断的动作时会连发多个松开，
			// 权威回帧保持其 fighter 资源对齐（server2「Echo even
			// duplicate release events」同口径，2026-11-17）。
			frame, err := player.manualParryStateUpdate()
			if err != nil {
				return err
			}
			if err := link.WriteFrame(frame); err != nil {
				return err
			}
			log.Printf("%s: ack stray manual parry release (client state resync)", remote)
		}
		return nil
	}

	player.releaseParryBuff()
	if engage {
		player.engageParryBuff()
		// 格挡不清怒(2026-11-07 撤销):Java handleManualParry 不动 SP,
		// 55866 复核四次 SP→0 单帧全部嵌在出招序列(294/337/20/24/CanUse)
		// 里、无一先于 msg169 格挡通告(按下帧甚至更靠后),且战斗态/
		// 非战斗态各占一半——"格挡开启清怒"系误读。格挡态本身按
		// "格挡被揍 +2"攒怒。
	}
	// 应答=三属性权威帧（InParry/InManualParry/ParryValue，server2
	// manualParryStateUpdate 同构）：fork 客户端格挡认知锚在 fighter
	// 资源上，六件套速度族/m169/BufferInfo4 是外服私服方言。
	frame, err := player.manualParryStateUpdate()
	if err != nil {
		return err
	}
	if err := link.WriteFrame(frame); err != nil {
		return err
	}
	// 格挡期间临时战斗姿态（2026-11-16 定版策略；2026-11-17 曾按
	// server2「零 463」口径删除，52506 会话 6 按塌 5 实证回退，恢复）：
	// 非战斗下客户端格挡架势由 LogicState=1 撑住（真机实证战斗段 6s
	// 保持 vs 预警段 50ms 即弃）。Z 持剑切换（m292）所有权不侵占——
	// 按下时记下 0/2 原值临时升 1，松开/破防原值回写。官方按下簇本身
	// 不带 463（a.pcapng），此帧是我方让客户端架势立住的撑持方言。
	if engage {
		if state := player.logicStateSnapshot(); state == 0 || state == 2 {
			player.setParryPreBlockState(state)
			player.setLogicState(1)
			player.writeLogicStateFrame(link, 1)
		}
	} else if pre := player.takeParryPreBlockState(); pre != 0xFF {
		player.setLogicState(pre)
		player.writeLogicStateFrame(link, pre)
	}
	log.Printf("%s: manual parry engage=%t", remote, engage)
	return nil
}

// ---- m54 外服私服令牌格挡问答（a.pcapng 5.144/5.695 定版，2026-11-12 P2） ----



