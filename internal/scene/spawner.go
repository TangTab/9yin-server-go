package scene

import (
	"encoding/json"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/component"
	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/proc"
)

// TopicEnterScene 登录链 → world 的进场事件（entry 角色发布，world 订阅）。
const TopicEnterScene = "login.enter_scene"

// EnterSceneEvent 进场事件载荷（login 侧 0x0B 已发出后发布；
// JSON 形态为合流最小演示，正式化随 proc 协议规格线走 proto.VarList）。
type EnterSceneEvent struct {
	Account  string  `json:"account"`
	RoleName string  `json:"role"`
	Config   string  `json:"config"`
	Resource string  `json:"resource"`
}

// Spawner 是 login 链与 ECS 内核的合流点：Bus 订阅 handler 只入队，
// Tick 在主循环内消费队列物化实体——World 非并发安全，跨 goroutine
// 事件必须经此收拢到 tick 侧（对应原版 world 串行场景线程语义）。
type Spawner struct {
	mu    sync.Mutex
	queue []EnterSceneEvent

	scenes map[string]ecs.EntityID // resource → 场景实体（去重）

	// OnSpawn 每次物化后的回调（日志/观测挂点）。
	OnSpawn func(world *ecs.World, entity ecs.EntityID, ev EnterSceneEvent)
}

// NewSpawner 创建物化器。
func NewSpawner() *Spawner {
	return &Spawner{scenes: make(map[string]ecs.EntityID)}
}

// Handle 是 Bus.Subscribe 的 handler（任意 goroutine 调用，仅入队）。
func (s *Spawner) Handle(env proc.Envelope) {
	if env.Topic != TopicEnterScene {
		return
	}
	var ev EnterSceneEvent
	if err := json.Unmarshal(env.Payload, &ev); err != nil {
		return
	}
	s.mu.Lock()
	s.queue = append(s.queue, ev)
	s.mu.Unlock()
}

// Tick 消费队列：玩家实体挂 InternalsInternalsPlayer；所在场景实体按
// resource 去重挂 InternalsInternalsScene（主城单实例语义的最小形态）。
// 注册方式：sched.AddFunc("spawn_players", spawner.Tick)。
func (s *Spawner) Tick(w *ecs.World, _ float64) {
	s.mu.Lock()
	pending := s.queue
	s.queue = nil
	s.mu.Unlock()

	for _, ev := range pending {
		sceneEnt := s.sceneEntity(w, ev.Resource)
		player := w.Create()
		w.Add(player, component.InternalsInternalsPlayer{Name: ev.RoleName})
		w.Add(player, SceneMember{Scene: sceneEnt, Config: ev.Config, Resource: ev.Resource})
		if s.OnSpawn != nil {
			s.OnSpawn(w, player, ev)
		}
	}
}

func (s *Spawner) sceneEntity(w *ecs.World, resource string) ecs.EntityID {
	if ent, ok := s.scenes[resource]; ok && w.Alive(ent) {
		return ent
	}
	ent := w.Create()
	w.Add(ent, component.InternalsInternalsScene{Name: resource})
	s.scenes[resource] = ent
	return ent
}

// SceneMember 玩家的场景归属组件（struct 生成物尚无此薄层，手写最小
// 形态；struct 模板扩展后可替换为生成物）。
type SceneMember struct {
	Scene    ecs.EntityID
	Config   string
	Resource string
}
