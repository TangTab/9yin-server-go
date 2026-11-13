package luascript

import (
	"os"
	"path/filepath"
	"testing"
)

// 副本事件上下文：ctx:dungeon() 快照表与 ctx:spawn_wave / ctx:after
// 能力回调的桥接（dungeon_wave.go 引擎侧消费）。
func TestCallDungeonContext(t *testing.T) {
	dir := t.TempDir()
	script := `
PROBE = {}
game.on("dungeon_probe", function(ctx)
	local d = ctx:dungeon()
	PROBE.dungeon = d.config .. "|" .. d.instance .. "|" .. d.wave .. "|" .. d.next_wave .. "|" .. d.killed .. "|" .. d.difficulty
	PROBE.spawn = ctx:spawn_wave(d.next_wave)
	PROBE.after = ctx:after(2.5, "later_event")
	return true
end)
`
	if err := os.WriteFile(filepath.Join(dir, "dungeon_probe.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	host := New(dir, func(string, ...any) {})
	if _, err := host.LoadDir(); err != nil {
		t.Fatal(err)
	}

	var spawnedWave int
	var afterCall struct {
		seconds float64
		event   string
	}
	ec := &EventContext{
		Dungeon: DungeonSnapshot{
			Config: `ini\scene\clone003`, Resource: "clone003", Instance: "dn-7-1",
			Difficulty: 2, Wave: 3, NextWave: 5, Killed: 12,
		},
		SpawnWave: func(wave int) error {
			spawnedWave = wave
			return nil
		},
		After: func(seconds float64, event string) error {
			afterCall.seconds, afterCall.event = seconds, event
			return nil
		},
	}
	handled, err := host.Call("dungeon_probe", ec)
	if err != nil || !handled {
		t.Fatalf("Call = (%v, %v), want (true, nil)", handled, err)
	}
	snapshot, evalErr := host.Eval(`return PROBE.dungeon .. "#" .. tostring(PROBE.spawn) .. "#" .. tostring(PROBE.after)`)
	if evalErr != nil {
		t.Fatalf("eval probe: %v", evalErr)
	}
	if want := `ini\scene\clone003|dn-7-1|3|5|12|2#true#true`; snapshot != want {
		t.Fatalf("probe = %q, want %q", snapshot, want)
	}
	if spawnedWave != 5 {
		t.Fatalf("spawn_wave callback wave = %d, want 5", spawnedWave)
	}
	if afterCall.seconds != 2.5 || afterCall.event != "later_event" {
		t.Fatalf("after callback = (%v, %q)", afterCall.seconds, afterCall.event)
	}
}

// spawn_wave/after 回调缺席（非副本事件路径）时调用应报错而不是静默：
// 错误经 Call 返回并触发 VM 重建，调用方按"已接手"口径处理。
func TestCallDungeonContextMissingCallbacks(t *testing.T) {
	dir := t.TempDir()
	script := `
game.on("dungeon_bad_spawn", function(ctx)
	ctx:spawn_wave(3)
	return true
end)
`
	if err := os.WriteFile(filepath.Join(dir, "bad.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	host := New(dir, func(string, ...any) {})
	if _, err := host.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ec := &EventContext{Dungeon: DungeonSnapshot{Config: "ini\\scene\\clone003", Wave: 2}}
	if _, err := host.Call("dungeon_bad_spawn", ec); err == nil {
		t.Fatal("spawn_wave without callback should error")
	}
}
