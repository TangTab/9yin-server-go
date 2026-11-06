package main

import (
	"math"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func TestDoorPackageRouteUsesInstalledDoorData(t *testing.T) {
	npc := npcSpawn{resolved: clientdata.ResolvedNPC{
		ConfigID:    "DoorNpc00013",
		ScriptClass: "Door",
		Extensions:  map[string]string{"template.DoorPackageID": "13"},
	}}
	route, handled, err := doorPortalRoute(npc)
	if err != nil || !handled {
		t.Fatalf("door route handled=%t err=%v", handled, err)
	}
	if route.sameScene || route.source != "DoorPackageID=13" {
		t.Fatalf("door route=%+v", route)
	}
	if got, want := route.destination.location.Scene.Config, `ini\scene\city04_LuoYang`; got != want {
		t.Fatalf("door target config=%q want %q", got, want)
	}
	if got, want := route.destination.location.Scene.Resource, "city04"; got != want {
		t.Fatalf("door target resource=%q want %q", got, want)
	}
	if math.Abs(float64(route.destination.location.Position.X-796.758)) > 0.001 {
		t.Fatalf("door target position=%+v", route.destination.location.Position)
	}
	if math.Abs(float64(route.trigger.X-727.905)) > 0.001 || math.Abs(float64(route.trigger.Z-1355.668)) > 0.001 {
		t.Fatalf("door trigger=%+v", route.trigger)
	}
}

func TestPortalAtUsesMoveTargetRadiusAndCooldown(t *testing.T) {
	world := newSceneLifecycle("test", nil)
	world.spawnSceneEntityLocked(2, sim.NewVitals(10, 10), worldcore.Transform{}, sceneEntity{id: 2, ownerID: 1, portal: &scenePortalRoute{
		trigger: role.Position{X: 100, Z: 200}, triggerRadius: 8, source: "DoorPackageID=13",
	}}, nil)
	now := time.Date(2026, 7, 26, 18, 30, 0, 0, time.UTC)
	if _, entered := world.portalAt(role.Position{X: 108.01, Z: 200}, now); entered {
		t.Fatal("position outside trigger radius entered portal")
	}
	route, entered := world.portalAt(role.Position{X: 106, Z: 205}, now)
	if !entered || route.source != "DoorPackageID=13" {
		t.Fatalf("entered=%t route=%+v", entered, route)
	}
	if _, entered := world.portalAt(role.Position{X: 100, Z: 200}, now.Add(time.Second)); entered {
		t.Fatal("portal cooldown did not suppress duplicate entry")
	}
	if _, entered := world.portalAt(role.Position{X: 100, Z: 200}, now.Add(2*time.Second)); !entered {
		t.Fatal("portal did not reactivate after cooldown")
	}
}

func TestGotoDoorRouteIsSameScenePosition(t *testing.T) {
	npc := npcSpawn{resolved: clientdata.ResolvedNPC{
		ConfigID:    "GotoDoorCity04a",
		ScriptClass: "GotoDoor",
		Extensions:  map[string]string{"template.PlayerTransferPosition": `"-306.917,30.925,-403.589,5.79"`},
	}}
	route, handled, err := doorPortalRoute(npc)
	if err != nil || !handled {
		t.Fatalf("goto route handled=%t err=%v", handled, err)
	}
	if !route.sameScene || route.source != "PlayerTransferPosition" {
		t.Fatalf("goto route=%+v", route)
	}
	if math.Abs(float64(route.position.Z+403.589)) > 0.001 || math.Abs(float64(route.position.Orient-5.79)) > 0.001 {
		t.Fatalf("goto position=%+v", route.position)
	}
}

func TestDoorResourceFromModernSceneConfig(t *testing.T) {
	for config, want := range map[string]string{
		`ini\scene\city04_LuoYang`:     "city04",
		`ini\scene\school07_WuDang`:    "school07",
		`ini\scene\born03_YanYuZhuang`: "born03",
		`ini\scene\clone004_wl`:        "clone004_wl",
	} {
		got, err := resourceForDoorScene(config)
		if err != nil || got != want {
			t.Fatalf("resource(%q)=%q err=%v want %q", config, got, err, want)
		}
	}
}
