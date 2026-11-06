package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// sendEntryScene is the current-client 0x0B scene archive used for both the
// initial role entry and a live 0x0C re-entry.  The property schema (0x09) is
// connection-scoped and is deliberately not resent during a re-entry.
func sendEntryScene(conn sceneMessageConnection, scene role.Scene) error {
	entry := make([]byte, 15, 96)
	entry[0] = 0x0B
	binary.LittleEndian.PutUint32(entry[1:], playerObjectID)
	binary.LittleEndian.PutUint32(entry[5:], playerOwnerID)
	binary.LittleEndian.PutUint32(entry[9:], 1)
	binary.LittleEndian.PutUint16(entry[13:], 3)
	entry = appendStringProperty(entry, 7, scene.Config)
	entry = appendStringProperty(entry, 8, scene.Resource)
	entry = binary.LittleEndian.AppendUint16(entry, 99)
	entry = binary.LittleEndian.AppendUint32(entry, 1000000)
	return conn.WriteFrame(entry)
}

func encodeExitScene() []byte { return []byte{0x0C} }

type sceneDestination struct{ location role.Location }

// parseSceneDestination takes config,resource,x,y,z,orient.  It is a
// deliberately explicit test/GM input; no client-side action is guessed.
func parseSceneDestination(value string) (sceneDestination, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 6 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return sceneDestination{}, fmt.Errorf("want config,resource,x,y,z,orient")
	}
	values := [4]float32{}
	for i := range values {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[i+2]), 32)
		if err != nil {
			return sceneDestination{}, fmt.Errorf("coordinate %d: %w", i, err)
		}
		values[i] = float32(parsed)
	}
	return sceneDestination{location: role.Location{
		Scene:    role.Scene{Config: strings.TrimSpace(parts[0]), Resource: strings.TrimSpace(parts[1])},
		Position: role.Position{X: values[0], Y: values[1], Z: values[2], Orient: values[3]},
	}}, nil
}

// parseScenePosition parses the GM-only same-scene teleport input. Scene
// identity is deliberately omitted: TeleportWithinScene always keeps the
// currently active scene and rejects use during a re-entry.
func parseScenePosition(value string) (role.Position, error) {
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

// sceneRuntime owns the re-entry sequence for one active player.  It keeps
// the NPC registry and lifecycle generation in lock-step with the client.
type sceneRuntime struct {
	conn       sceneMessageConnection
	world      *sceneLifecycle
	machine    interface{ ReenterScene() error }
	registry   *sceneNPCRegistry
	staticNPCs []npcSpawn
	radius     float32
	activeNPCs map[int]struct{}
	player     *playerActor
	activeRole *role.RoleSnapshot
	npcCatalog []npcSpawn
	// The first 0x0A after 0x0C is stale activity from the old scene. The
	// player must already be present when the target stage runs, but NPCs must
	// wait for the later target 0x09 readiness notification.
	awaitingStableReentryReady bool
}

func (r *sceneRuntime) catalogForCurrentScene() ([]npcSpawn, error) {
	if r.registry == nil {
		return r.staticNPCs, nil
	}
	catalog, _, err := r.registry.catalogFor(r.activeRole.Location.Scene)
	if err != nil {
		return nil, err
	}
	return catalog, nil
}

// SwitchScene starts a live scene replacement.  The old scene's Lua teardown
// runs asynchronously after 0x0C, so 0x0B must not immediately follow it:
// otherwise its delayed ClearAll removes the replacement player. The target
// PlayerAddObject must arrive immediately after EntryScene because target
// stage_main requires the local 1-1 object to open the target scene. Its
// separate ServerLocation motion update is deferred until final 0x09
// ClientReady, otherwise the client visibly projects the destination onto the
// old scene while terrain is still changing.
func (r *sceneRuntime) SwitchScene(destination sceneDestination) error {
	if r.player == nil || r.activeRole == nil {
		return fmt.Errorf("switch scene without an active player")
	}
	destination.location.Scene = normalizeClientScene(destination.location.Scene)
	previous := r.activeRole.Location
	// parseSceneDestination deliberately has no persistence information.  Keep
	// the current optimistic-concurrency version when replacing the scene or
	// the subsequent SaveLocation would always use version zero and fail.
	destination.location.Version = previous.Version
	r.activeRole.Location = destination.location
	catalog, err := r.catalogForCurrentScene()
	if err != nil {
		r.activeRole.Location = previous
		return err
	}
	if err := r.machine.ReenterScene(); err != nil {
		return err
	}
	if err := r.conn.WriteFrame(encodeExitScene()); err != nil {
		r.activeRole.Location = previous
		return fmt.Errorf("write ExitScene: %w", err)
	}
	if err := r.world.begin(destination.location.Scene.Config, destination.location.Scene.Resource); err != nil {
		return fmt.Errorf("begin replacement scene: %w", err)
	}
	// Observed client trace: EntryScene arrived 12ms after ExitScene. That is
	// inside the previous stage's cleanup and reliably loses the new player.
	// Keep the network order, but let on_exit_scene finish its ClearAll first.
	time.Sleep(750 * time.Millisecond)
	if err := sendEntryScene(r.conn, destination.location.Scene); err != nil {
		return fmt.Errorf("write EntryScene: %w", err)
	}
	if err := sendPlayerAddObject(r.conn, r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values)); err != nil {
		return fmt.Errorf("write player re-entry archive: %w", err)
	}
	clear(r.activeNPCs)
	if _, _, err := synchronizeNPCViewport(r.world, catalog, r.activeNPCs, destination.location.Position, r.radius); err != nil {
		return fmt.Errorf("register replacement NPC viewport: %w", err)
	}
	r.npcCatalog = catalog
	r.awaitingStableReentryReady = true
	return nil
}

// TeleportWithinScene is the instantaneous server-authoritative relocation
// path. It intentionally sends only ServerLocation (0x1F): no ExitScene,
// EntryScene, PlayerAddObject, or readiness barrier belongs to a coordinate
// change inside an already-loaded map.
func (r *sceneRuntime) TeleportWithinScene(position role.Position) error {
	if r.player == nil || r.activeRole == nil {
		return fmt.Errorf("teleport without an active player")
	}
	if r.awaitingStableReentryReady {
		return fmt.Errorf("target scene is still loading")
	}
	previous := r.activeRole.Location.Position
	r.activeRole.Location.Position = position
	if err := r.conn.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {
		r.activeRole.Location.Position = previous
		return fmt.Errorf("write same-scene ServerLocation: %w", err)
	}
	_, changed, err := synchronizeNPCViewport(r.world, r.npcCatalog, r.activeNPCs, position, r.radius)
	if err != nil {
		return fmt.Errorf("synchronize teleported NPC viewport: %w", err)
	}
	if changed {
		if err := r.world.reconcileViewport(); err != nil {
			return fmt.Errorf("reconcile teleported NPC viewport: %w", err)
		}
	}
	return nil
}
