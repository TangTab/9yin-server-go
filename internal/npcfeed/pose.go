package npcfeed

// pose.go NPC 姿态/状态：静息态（不移动时的 State）解析与动作名校验。
//
// 官服物化帧定版（2026-11-25，姑苏 npc.pcapng + npc-点击对话.pcapng 两包
// 420 个 NPC）：State 与 BornAction 恒同值——姿态 NPC 两者都是出生状态的
// 动作名（ty_clap / ty_sitchair_drinkloop / ty_lie_sleep / sitloop…），普通
// 站立为 stand，行进中进视口为 State=walk + BornAction=stand。姿态 NPC
// 物化后不再收 State 帧（姿态由客户端保持）；点击对话也不发 State。
//
// 出生状态的数据来源（客户端表三种列名的列注释同为「出生状态」，逐层
// 覆盖）：摆放行 BornAction（creator 属性）> 模板 BornAction 列 > 模板
// BornActionRec 列（table@BornActionRec）。walk / run 表示「出生即在走」，
// 不是停留姿态。

import (
	"os"
	"regexp"
	"strings"
)

const (
	// StandAction 静息缺省态。
	StandAction = "stand"
	// WalkAction 行进态。
	WalkAction = "walk"
	// RunAction 跑动态（出生状态列里与 walk 同义：出生即在走）。
	RunAction = "run"
)

// actionNameRe 动作名形态：字母数字下划线。客户端表里混有贴图路径之类
// 的脏值（scene_commonnpc 的 BornAction 列有 icon\monster\*.png）。
var actionNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// bornActionOf 出生状态原始值：摆放行 > 模板 BornAction > 模板
// BornActionRec（都空返回空）。
func bornActionOf(placement, template map[string]string) string {
	for _, value := range []string{placement["BornAction"], template["BornAction"], template["BornActionRec"]} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// poseEnabled NINEYIN_NPC_POSE=off 关停出生姿态（静息态一律 stand、物化帧回到
// 此前的 stand/stand）——真机 A/B 开关：这次改动一次影响 1.5 万个 NPC 的出场
// 状态，某类动作在客户端表现不对时不必回滚代码即可对照。路点到达动作不受
// 影响。
func poseEnabled() bool { return os.Getenv("NINEYIN_NPC_POSE") != "off" }

// RestAction 静息态：不移动时的 State。出生状态是姿态动作名就用它；
// 缺省 / stand / walk / run / 非动作名 → stand。
func (s Spawn) RestAction() string {
	if !poseEnabled() {
		return StandAction
	}
	if action := specificAction(s.BornAction); action != "" {
		return action
	}
	return StandAction
}

// specificAction 特定停留动作：是动作名、且不是 stand / walk / run / nil /
// 0 时返回原值，否则返回空（=没有特定动作，按缺省站立）。
//
// 分号列表（摆放行偶见，60 条：`stand;sitloop`、`坐喝茶;坐拉二胡`）语义未知
// （候选随机取一？顺序播放？），官服抓包里没有样本——不猜：去空后恰剩一个
// 动作（`stand;` / `diedownloop;` 这类尾分号）就取它，多候选按没有特定动作。
func specificAction(action string) string {
	action = strings.TrimSpace(action)
	if strings.Contains(action, ";") {
		var candidates []string
		for _, part := range strings.Split(action, ";") {
			if part = strings.TrimSpace(part); part != "" {
				candidates = append(candidates, part)
			}
		}
		if len(candidates) != 1 {
			return ""
		}
		action = candidates[0]
	}
	if action == "" || !actionNameRe.MatchString(action) {
		return ""
	}
	switch strings.ToLower(action) {
	case StandAction, WalkAction, RunAction, "nil", "0":
		return ""
	}
	return action
}

// State 该动作下的 State：行进为 walk；停留中路点带特定到达动作
// （ActionName 列：ty_saodiloop 扫地 / interact50 …）就用它，否则用
// rest（调用方给的静息态）。
func (m Motion) State(rest string) string {
	if m.Moving {
		return WalkAction
	}
	if arrival := specificAction(m.Arrival); arrival != "" {
		return arrival
	}
	return rest
}

// materialState 物化时的 State：调用方已按时刻标了 State（巡逻采样时在
// 走 = walk）就用它，否则静息态。
func (s Spawn) materialState() string {
	if s.State != "" {
		return s.State
	}
	return s.RestAction()
}
