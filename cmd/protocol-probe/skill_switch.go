package main

import "fmt"

const (
	// Current FxGameLogic GameMessageHandler binding:
	// slot 843 / native sub_11044B98 => (843-15)/2 == message ID 414.
	serverSwitchControlMessage int32 = 414

	// sub_127F9930 refuses to enter the native skill-action path until this
	// SwitchManager capability is enabled.
	skillActionInputSwitch int32 = 1223
)

// skillActionSwitchFrame updates only switch 1223. The receiver's "sync"
// command clears the complete native switch tree first, so using "reset"
// avoids disabling unrelated client capabilities.
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
