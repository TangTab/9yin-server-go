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
