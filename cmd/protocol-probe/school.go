package main

import (
	"fmt"
	"log"
	"strings"
)

// 门派加入（msg973 新手指南 / msg997 门派消息；fork school.go:64-99
// 反汇编还原；docs/fork-p3-exchange-school.md）。
//
// fork 两个 handler 都收敛到 joinFaction = lookupFactionByID + setFaction：
//   - 973 GUIDE_ADD_SCHOOL：`[973, str(仅窄串 tag6) schoolKey]`；
//   - 997 SCHOOL：遍历 values[] 取首个窄串作为 schoolKey（lua 侧
//     custom_send_Request_Declare_Fight 等多变体共用一个入口）。
//
// lookupFactionByID：TrimSpace → 空 id "empty school id." → 对 faction
// 全 ID（school_shaolin / force_* / newschool_*）EqualFold 匹配 → 未命中
// "unknown school id %q"。setFaction 产出 School@189/Force/NewSchool
// 三属性单帧（factionProperties 同构）。
// fork 的 C2S 路径**会话级不落库**（无 persist faction 字符串证据；GM
// set_faction 才持久化）——门派跨登录存续走 GM 通道，本处对齐 fork。
const (
	clientCustomGuideAddSchool int32 = 973
	clientCustomSchoolMessage  int32 = 997
)

// lookupFactionByID 按 faction 全 ID 大小写不敏感查封闭预设表。
func lookupFactionByID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("empty school id.")
	}
	for _, preset := range gmFactionPresets {
		if strings.EqualFold(preset.Faction, id) {
			return preset.Faction, nil
		}
	}
	return "", fmt.Errorf("unknown school id %q", id)
}

// joinFaction 校验并切换门派，返回待写 0x10 帧（fork 同名同构）。
func (p *playerActor) joinFaction(id string) ([]byte, error) {
	faction, err := lookupFactionByID(id)
	if err != nil {
		return nil, err
	}
	return p.setFaction(faction)
}

func writeJoinedFaction(link sceneMessageConnection, player *playerActor, schoolKey, remote string) error {
	frame, err := player.joinFaction(schoolKey)
	if err != nil {
		log.Printf("%s: guide add school rejected: %v", remote, err)
		return nil
	}
	if err := link.WriteFrame(frame); err != nil {
		return fmt.Errorf("guide add school: %w", err)
	}
	log.Printf("%s: joined faction %s", remote, schoolKey)
	return nil
}

// handleGuideAddSchoolCustom `[973, str schoolKey]`（仅窄串；fork :72
// CMPB $0x6 单一形态）。
func handleGuideAddSchoolCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomGuideAddSchool {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: guide add school before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) < 2 || custom.Values[1].Type != 6 {
		log.Printf("%s: guide add school rejected: malformed values=%v", remote, custom.Values)
		return true, nil
	}
	if err := writeJoinedFaction(link, player, custom.Values[1].Text, remote); err != nil {
		return true, err
	}
	return true, nil
}

// handleSchoolCustom msg997：取首个窄串作为门派键（fork :91-:93 值扫描
// 循环 + joinFaction）。无玩家返回错误（fork :88-89 errorf 同口径，
// 由分发层记日志）。
func handleSchoolCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomSchoolMessage {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("school message before player spawn")
	}
	schoolKey := ""
	for _, value := range custom.Values[1:] {
		if value.Type == 6 {
			schoolKey = value.Text
			break
		}
	}
	if schoolKey == "" {
		log.Printf("%s: school message rejected: no school id in values=%v", remote, custom.Values)
		return true, nil
	}
	if err := writeJoinedFaction(link, player, schoolKey, remote); err != nil {
		return true, err
	}
	return true, nil
}
