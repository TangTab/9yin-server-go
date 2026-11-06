package main

import (
	"fmt"
	"log"
	"time"
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

// handleBlockStateCustom 还原手动格挡（Java handleManualParry +
// sendParryStateFrames 同源）：状态变化时挂/清槽 11 的 BuffInParry，
// 回 msg169（BuffInParry 事件，双宽串角色名）+ 0x10 InParry 属性帧。
// 重复按下静默；重复松开仍回帧（客户端状态重同步，78 发按值去重
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
	// fork 的 tag2/tag3 双编码：i32 与 i64 都按数值比较 0。
	state := custom.Values[1]
	engage := false
	switch state.Type {
	case 2:
		engage = state.Int32 == 0
	case 3:
		engage = state.Int64 == 0
	case 4, 5:
		number, ok := state.exactInt32()
		if !ok {
			return fmt.Errorf("block state value is not integral: %s", state)
		}
		engage = number == 0
	default:
		return fmt.Errorf("block state value must be numeric, got %s", state)
	}

	alreadyParrying := player.parryingSnapshot()
	if alreadyParrying == engage {
		if !engage {
			// 游离的重复松开：客户端可能错过上一次回帧，重推空态。
			if frame, err := player.parryStateFrame(false, ""); err == nil {
				if writeErr := link.WriteFrame(frame); writeErr != nil {
					return writeErr
				}
			} else {
				return err
			}
			log.Printf("%s: ack stray manual parry release (client state resync)", remote)
		}
		return nil
	}

	now := time.Now().UTC()
	buffInfo := player.releaseParryBuff()
	if engage {
		buffInfo = player.engageParryBuff(now)
	}
	name := player.snapshotName()
	notice, err := serverCustomIntMessage(169,
		customInt(boolInt32(!engage)),
		customString("BuffInParry"),
		customObject(playerObjectID, playerOwnerID),
		customObject(playerObjectID, playerOwnerID),
		customWideString(name),
		customWideString(name),
		customInt64(now.UnixMilli()),
		customInt(0),
	)
	if err != nil {
		return fmt.Errorf("encode parry notice: %w", err)
	}
	if err := link.WriteFrame(notice); err != nil {
		return err
	}
	frame, err := player.parryStateFrame(engage, buffInfo)
	if err != nil {
		return fmt.Errorf("encode parry state: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return err
	}
	log.Printf("%s: manual parry engage=%t buff_slot=%d", remote, engage, parryBuffSlot)
	return nil
}
