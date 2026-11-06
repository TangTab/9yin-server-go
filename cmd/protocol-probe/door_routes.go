package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/local/9yin-go-server/internal/role"
)

var defaultDoorDataPath = filepath.Join(defaultModernShareRoot, "rule", "door_data.ini")

// scenePortalRoute describes only routes that are fully present in the
// installed modern resources. Adventure/instance doors intentionally do not
// use this type: their clone admission contract is not yet recovered.
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
		// GotoDoor's installed PlayerTransferPosition carries an exact local
		// transform and no target scene. It is a same-scene portal, not a clone
		// entry or a generic external transport.
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
	file, err := os.Open(path)
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

// resourceForDoorScene derives the installed terrain name from the modern
// scene config. DoorData's TargetScene carries only the config; EntryScene
// additionally requires the terrain resource. Mainline city/school/born
// maps use their numeric prefix, while clone maps use their full config name.
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
	// Clone and special scenes generally name terrain after their whole config.
	return lower, nil
}
