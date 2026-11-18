package engine

import (
	"context"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/proc"
)

func TestRunTicksUntilCancel(t *testing.T) {
	app := New(Options{TPS: 5000})

	ticks := 0
	app.Sched.AddFunc("count", func(w *ecs.World, dt float64) { ticks++ })

	started := false
	app.OnStart(func(*App) { started = true })

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	app.Run(ctx)

	if !started {
		t.Fatal("OnStart 未执行")
	}
	if ticks < 10 {
		t.Fatalf("80ms @5000TPS 应至少 10 帧，实际 %d", ticks)
	}
}

func TestOnStopAndBusClose(t *testing.T) {
	app := New(Options{TPS: 100})
	stopped := false
	app.OnStop(func(*App) { stopped = true })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	app.Run(ctx)

	if !stopped {
		t.Fatal("OnStop 未执行")
	}
	if err := app.Bus.Publish(proc.Envelope{Topic: "x"}); err == nil {
		t.Fatal("Run 返回后总线应已关闭")
	}
}

func TestTickOnce(t *testing.T) {
	app := New(Options{})
	var frame uint64
	app.Sched.AddFunc("probe", func(w *ecs.World, dt float64) { frame = w.CurrentTick() })
	app.TickOnce(1.0 / 60)
	if frame != 1 {
		t.Fatalf("TickOnce 应推进一帧，实际 %d", frame)
	}
}
