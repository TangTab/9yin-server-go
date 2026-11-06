package main

import "fmt"

const (
	// 当前 FxGameLogic 的 GameMessageHandler 绑定：
	// 槽位 843 / 原生 sub_11044B98 => (843-15)/2 == 消息 ID 414。
	serverSwitchControlMessage int32 = 414

	// sub_127F9930 在这个 SwitchManager 能力被启用之前
	// 拒绝进入原生招式动作路径。
	skillActionInputSwitch int32 = 1223
)

// skillActionSwitchFrame 只更新开关 1223。接收端的 "sync"
// 命令会先清除整个原生开关树，因此使用 "reset"
// 可以避免禁用无关的客户端能力。
func skillActionSwitchFrame() ([]byte, error) {
	return serverModernCustomIntMessage(
		serverSwitchControlMessage,
		customString("reset"),
		customInt(skillActionInputSwitch),
		customInt(1),
	)
}

func enableSkillActionSwitch(link sceneMessageConnection) error {
	frame, err := skillActionSwitchFrame()
	if err != nil {
		return fmt.Errorf("encode skill-action switch: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return fmt.Errorf("write skill-action switch: %w", err)
	}
	return nil
}
