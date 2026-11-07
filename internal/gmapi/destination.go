package gmapi

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/role"
)

// ParseDestination 接受 config,resource,x,y,z,orient。它是刻意显式
// 的测试/GM 输入；不猜测任何客户端动作。逻辑原属
// cmd/protocol-probe/scene_transition.go 的 parseSceneDestination，
// 因 GM 校验需要而移入契约层；游戏服侧保留薄包装。
func ParseDestination(value string) (role.Location, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 6 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return role.Location{}, fmt.Errorf("want config,resource,x,y,z,orient")
	}
	values := [4]float32{}
	for i := range values {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[i+2]), 32)
		if err != nil {
			return role.Location{}, fmt.Errorf("coordinate %d: %w", i, err)
		}
		values[i] = float32(parsed)
	}
	return role.Location{
		Scene:    role.Scene{Config: strings.TrimSpace(parts[0]), Resource: strings.TrimSpace(parts[1])},
		Position: role.Position{X: values[0], Y: values[1], Z: values[2], Orient: values[3]},
	}, nil
}

// ParsePosition 解析仅限 GM 使用的同场景传送输入。场景标识刻意
// 省略：TeleportWithinScene 总是保持当前活动场景，并在重进入期间拒绝
// 使用。原属 cmd/protocol-probe/scene_transition.go。
func ParsePosition(value string) (role.Position, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return role.Position{}, fmt.Errorf("want x,y,z,orient")
	}
	values := [4]float32{}
	for i := range values {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 32)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			if err == nil {
				err = fmt.Errorf("must be finite")
			}
			return role.Position{}, fmt.Errorf("coordinate %d: %w", i, err)
		}
		values[i] = float32(parsed)
	}
	return role.Position{X: values[0], Y: values[1], Z: values[2], Orient: values[3]}, nil
}
