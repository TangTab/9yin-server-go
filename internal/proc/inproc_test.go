package proc

import "testing"

func TestInprocRouting(t *testing.T) {
	bus := NewInprocBus()
	defer bus.Close()

	var gotWorld, gotAll, gotOther []string
	bus.Subscribe(RoleWorld, "world.scene.1", func(e Envelope) { gotWorld = append(gotWorld, string(e.Payload)) })
	bus.Subscribe(RoleLog, TopicWildcard, func(e Envelope) { gotAll = append(gotAll, e.Topic) })
	bus.Subscribe(RoleWorld, "world.scene.2", func(e Envelope) { gotOther = append(gotOther, string(e.Payload)) })

	bus.Publish(Envelope{From: RoleEntry, Topic: "world.scene.1", Payload: []byte("a")})
	bus.Publish(Envelope{From: RoleWorld, Topic: "no.where", Payload: []byte("x")})

	if len(gotWorld) != 1 || gotWorld[0] != "a" {
		t.Fatalf("精确路由错误：%v", gotWorld)
	}
	if len(gotOther) != 0 {
		t.Fatalf("不应路由到其他场景：%v", gotOther)
	}
	if len(gotAll) != 2 {
		t.Fatalf("广播订阅应收两条，实际 %v", gotAll)
	}
}

func TestInprocOrder(t *testing.T) {
	bus := NewInprocBus()
	defer bus.Close()

	var seq []int
	bus.Subscribe(RoleWorld, "tick", func(e Envelope) { seq = append(seq, len(e.Payload)) })
	for i := 1; i <= 5; i++ {
		bus.Publish(Envelope{From: RoleMember, Topic: "tick", Payload: make([]byte, i)})
	}
	for i, want := range []int{1, 2, 3, 4, 5} {
		if seq[i] != want {
			t.Fatalf("顺序错误：%v", seq)
		}
	}
}

func TestInprocClose(t *testing.T) {
	bus := NewInprocBus()
	bus.Close()
	if err := bus.Publish(Envelope{Topic: "x"}); err == nil {
		t.Fatal("关闭后 Publish 应报错")
	}
	if err := bus.Subscribe(RoleWorld, "x", func(Envelope) {}); err == nil {
		t.Fatal("关闭后 Subscribe 应报错")
	}
}
