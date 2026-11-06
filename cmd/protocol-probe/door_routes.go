package main

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/local/9yin-go-server/internal/role"
)

var defaultDoorDataPath = filepath.Join(defaultModernShareRoot, "rule", "door_data.ini")

// scenePortalRoute 只描述在已安装的现代资源中完整存在的路线。
// 奇遇/副本传送门有意不
// 使用此类型：它们的副本准入契约尚未被还原。
type scenePortalRoute struct {
	sameScene     bool
	destination   sceneDestination
	position      role.Position
	trigger       role.Position
	triggerRadius float32
	source        string
}

type doorDataTarget struct {
	config    string
	position  role.Position
	movePoint role.Position
}

var modernDoorTargets struct {
	once    sync.Once
	targets map[string]doorDataTarget
	err     error
}

func doorPortalRoute(npc npcSpawn) (scenePortalRoute, bool, error) {
	switch strings.ToLower(strings.TrimSpace(npc.resolved.ScriptClass)) {
	case "door":
		packageID, _ := effectiveNPCBusinessValue(npc, "DoorPackageID")
		if packageID == "" || packageID == "0" {
			return scenePortalRoute{}, false, nil
		}
		targets, err := loadModernDoorTargets()
		if err != nil {
			return scenePortalRoute{}, false, err
		}
		target, exists := targets[packageID]
		if !exists {
			return scenePortalRoute{}, false, nil
		}
		resource, err := resourceForDoorScene(target.config)
		if err != nil {
			return scenePortalRoute{}, false, fmt.Errorf("door package %s: %w", packageID, err)
		}
		return scenePortalRoute{
			destination: sceneDestination{location: role.Location{
				Scene:    role.Scene{Config: target.config, Resource: resource},
				Position: target.position,
			}},
			trigger:       target.movePoint,
			triggerRadius: portalTriggerRadius(npc),
			source:        "DoorPackageID=" + packageID,
		}, true, nil
	case "gotodoor":
		// GotoDoor 安装的 PlayerTransferPosition 携带精确的本地
		// transform 且不含目标场景。它是同场景传送门，不是副本
		// 入口，也不是通用的外部传送。
		positionText, _ := effectiveNPCBusinessValue(npc, "PlayerTransferPosition")
		position, ok := parseDoorPosition(positionText)
		if !ok {
			return scenePortalRoute{}, false, nil
		}
		return scenePortalRoute{
			sameScene:     true,
			position:      position,
			trigger:       role.Position{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient},
			triggerRadius: portalTriggerRadius(npc),
			source:        "PlayerTransferPosition",
		}, true, nil
	default:
		return scenePortalRoute{}, false, nil
	}
}

func portalTriggerRadius(npc npcSpawn) float32 {
	if radius := npc.float32Property("SpringRange"); radius > 0 && radius <= 100 {
		return radius
	}
	return 8
}

func loadModernDoorTargets() (map[string]doorDataTarget, error) {
	modernDoorTargets.once.Do(func() {
		modernDoorTargets.targets, modernDoorTargets.err = loadDoorTargets(defaultDoorDataPath)
	})
	return modernDoorTargets.targets, modernDoorTargets.err
}

func loadDoorTargets(path string) (map[string]doorDataTarget, error) {
	file, err := openResourceReader(path)
	if err != nil {
		return nil, fmt.Errorf("open door data %s: %w", path, err)
	}
	defer file.Close()

	sections := make(map[string]map[string]string)
	var current map[string]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			current = make(map[string]string)
			sections[name] = current
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found && current != nil {
			current[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read door data %s: %w", path, err)
	}

	targets := make(map[string]doorDataTarget, len(sections))
	for id, values := range sections {
		config := strings.TrimSpace(values["TargetScene"])
		if config == "" {
			continue
		}
		position, targetOK := parseDoorCoordinates(values["TargetX"], values["TargetY"], values["TargetZ"], values["TargetOrient"])
		movePoint, moveOK := parseDoorCoordinates(values["MoveTargetX"], values["MoveTargetY"], values["MoveTargetZ"], "0")
		if !targetOK || !moveOK {
			continue
		}
		targets[id] = doorDataTarget{config: config, position: position, movePoint: movePoint}
	}
	return targets, nil
}

func parseDoorPosition(value string) (role.Position, bool) {
	value = strings.Trim(strings.TrimSpace(value), "\"")
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return role.Position{}, false
	}
	return parseDoorCoordinates(parts[0], parts[1], parts[2], parts[3])
}

func parseDoorCoordinates(x, y, z, orient string) (role.Position, bool) {
	values := []string{x, y, z, orient}
	parsed := [4]float32{}
	for index, value := range values {
		floatValue, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
		if err != nil {
			return role.Position{}, false
		}
		parsed[index] = float32(floatValue)
	}
	return role.Position{X: parsed[0], Y: parsed[1], Z: parsed[2], Orient: parsed[3]}, true
}

// resourceForDoorScene 从现代场景配置推导已安装的地形名。
// DoorData 的 TargetScene 只携带配置名；EntryScene
// 还额外需要地形资源。主线的 city/school/born
// 地图使用其数字前缀，而副本地图使用完整配置名。
func resourceForDoorScene(config string) (string, error) {
	config = strings.ReplaceAll(strings.TrimSpace(config), "/", `\`)
	base := strings.TrimSuffix(filepath.Base(config), filepath.Ext(config))
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("empty target scene")
	}
	lower := strings.ToLower(base)
	for _, prefix := range []string{"city", "school", "born", "scene"} {
		if strings.HasPrefix(lower, prefix) {
			if separator := strings.IndexByte(lower, '_'); separator > 0 {
				return lower[:separator], nil
			}
		}
	}
	// 副本地图和特殊场景通常直接以完整配置名作为地形名。
	return lower, nil
}
