package scene

import (
	"encoding/json"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/component"
	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/proc"
)

// TestSpawnerSerializesBusToTick 合流语义：Bus handler（任意 goroutine）
// 只入队，Tick 消费建实体——玩家挂 InternalsPlayer+SceneMember，场景实体
// 按 resource 去重。
func TestSpawnerSerializesBusToTick(t *testing.T) {
	spawner := NewSpawner()
	world := ecs.New()

	// 模拟两条 Bus 投递（同场景两名玩家）
	for _, name := range []string{"小虾米", "阿紫"} {
		payload, _ := json.Marshal(EnterSceneEvent{
			Account: "acct:v1:x", RoleName: name,
			Config: BirthConfig, Resource: "school09",
		})
		spawner.Handle(proc.Envelope{Topic: TopicEnterScene, Payload: payload})
	}
	// 未消费前不建实体
	if world.EntityCount() != 0 {
		t.Fatal("Tick 前不应有实体")
	}

	spawner.Tick(world, 0)
	if got := world.Count[component.InternalsInternalsPlayer](); got != 2 {
		t.Fatalf("玩家实体 %d want 2", got)
	}
	if got := world.Count[component.InternalsInternalsScene](); got != 1 {
		t.Fatalf("场景实体 %d want 1（按 resource 去重）", got)
	}

	// 玩家场景归属指向同一场景实体
	world.Each1(func(_ ecs.EntityID, m *SceneMember) {
		if m.Resource != "school09" || m.Config != BirthConfig {
			t.Fatalf("归属 %+v", m)
		}
		scene, ok := world.Get[component.InternalsInternalsScene](m.Scene)
		if !ok {
			t.Fatal("场景实体缺失 InternalsScene")
		}
		if scene.Name != "school09" {
			t.Fatalf("场景名 %q", scene.Name)
		}
	})

	// 非本 Topic 的事件忽略
	spawner.Handle(proc.Envelope{Topic: "other", Payload: []byte("{}")})
	spawner.Tick(world, 0)
	if world.EntityCount() != 3 {
		t.Fatalf("无关事件不应建实体：%d", world.EntityCount())
	}
}

// BirthConfig 与 login.BirthScene 同值（移花宫）——测试里避免 import cycle
// 直接写字面量。
const BirthConfig = `ini\scene\school09_YiHuaGong`
