package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/luascript"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// waveTestNPC 带波次刻度（creator.NpcProgress）的目录 AttackNpc。
func waveTestNPC(configID string, wave int, x, z float32) npcSpawn {
	npc := testNPCSpawn()
	npc.resolved.ConfigID = configID
	npc.resolved.ScriptClass = "AttackNpc"
	if wave > 0 {
		npc.resolved.Extensions["creator.NpcProgress"] = strconv.Itoa(wave)
	}
	npc.x, npc.y, npc.z = x, 0, z
	return npc
}

// newWaveTestRun 构造波次副本运行时：waves {2,3,5} + 一个无波次常驻
// 平民；成员视图挂靠（waveTick 的在场闸门需要）。
func newWaveTestRun(t *testing.T, scripts *luascript.Host) (*sceneRun, *sceneLifecycle, *captureMessageConnection) {
	t.Helper()
	conn := &captureMessageConnection{}
	view := newSceneLifecycle("test", conn)
	view.radius = 50
	catalog := []npcSpawn{
		waveTestNPC("mon_a", 2, 1, 0),
		waveTestNPC("mon_b", 2, -1, 0),
		waveTestNPC("mon_c", 3, 5, 5),
		waveTestNPC("mon_d", 5, -5, -5),
		testNPCSpawn(), // CommonNpc 无波次：常驻
	}
	run := newSceneRun(sceneKey{config: "clone-test", resource: "clone-test", instance: "dn-t"}, nil)
	if err := run.populate(catalog); err != nil {
		t.Fatal(err)
	}
	run.mu.Lock()
	run.scripts = scripts
	run.managed = true
	run.noRespawn = true
	run.mu.Unlock()
	view.run = run
	view.attachMember(newPlayerActor("tester", 0), role.Position{X: 1}, roleVisual{})
	run.activateMember(view.wireID)
	view.refreshVisibility(catalog, role.Position{X: 1})
	t.Cleanup(func() {
		view.close()
		run.mu.Lock()
		if !run.closed {
			run.shutdownLocked()
		}
		run.mu.Unlock()
	})
	return run, view, conn
}

// waveAddObjectSeen 扫捕获帧：是否出现过携带该 wire ID 的 0x0D
// ServerAddObject 物化帧（新波 NPC 对账到视口的直接证据）。
func waveAddObjectSeen(conn *captureMessageConnection, id uint32) bool {
	needle := []byte{byte(id), byte(id >> 8), byte(id >> 16), byte(id >> 24)}
	for _, frame := range conn.Frames() {
		if len(frame) == 0 || frame[0] != 0x0D {
			continue
		}
		if bytes.Contains(frame, needle) {
			return true
		}
	}
	return false
}

func waveNPCRegistered(t *testing.T, run *sceneRun, id uint32) bool {
	t.Helper()
	run.mu.Lock()
	defer run.mu.Unlock()
	_, exists := run.npcMeta(worldcore.EntityID(id))
	return exists
}

// TestWavePopulateSplit 波次引擎装填：副本实例只注册首波（NpcProgress
// 最小值）与无波次常驻 NPC；其余波次留待推进。共享场景（instance
// 空）同目录全量注册，行为不变。
func TestWavePopulateSplit(t *testing.T) {
	run, _, _ := newWaveTestRun(t, nil)
	run.mu.Lock()
	waveMode, currentWave := run.waveMode, run.currentWave
	pending := append([]int(nil), run.pendingWaves...)
	alive := len(run.waveAlive)
	run.mu.Unlock()

	if !waveMode || currentWave != 2 || alive != 2 {
		t.Fatalf("wave state mode=%v current=%d alive=%d, want (true,2,2)", waveMode, currentWave, alive)
	}
	if len(pending) != 2 || pending[0] != 3 || pending[1] != 5 {
		t.Fatalf("pending waves = %v, want [3 5]", pending)
	}
	idA, idB, idC, idD, idCommon := forkNPCID(2), forkNPCID(3), forkNPCID(4), forkNPCID(5), forkNPCID(6)
	for _, id := range []uint32{idA, idB, idCommon} {
		if !waveNPCRegistered(t, run, id) {
			t.Fatalf("NPC %d should be registered at populate", id)
		}
	}
	for _, id := range []uint32{idC, idD} {
		if waveNPCRegistered(t, run, id) {
			t.Fatalf("NPC %d must wait for its wave", id)
		}
	}

	// 对照：共享场景同目录全量注册（instance 为空不开波次引擎）。
	shared := newSceneRun(sceneKey{config: "clone-test", resource: "clone-test"}, nil)
	if err := shared.populate([]npcSpawn{
		waveTestNPC("mon_a", 2, 1, 0), waveTestNPC("mon_c", 3, 5, 5),
	}); err != nil {
		t.Fatal(err)
	}
	shared.mu.Lock()
	waveMode = shared.waveMode
	shared.mu.Unlock()
	if waveMode {
		t.Fatal("shared scene must not enable the wave engine")
	}
	for _, id := range []uint32{forkNPCID(2), forkNPCID(3)} {
		if !waveNPCRegistered(t, shared, id) {
			t.Fatalf("shared scene NPC %d must register", id)
		}
	}
	shared.mu.Lock()
	shared.shutdownLocked()
	shared.mu.Unlock()
}

// TestWaveDefaultAdvanceAndFinish 默认引擎：逐波全灭推进（新波物化到
// 视口连接），末波全灭置 finished（收尾定时器由 shutdown 清场）。
func TestWaveDefaultAdvanceAndFinish(t *testing.T) {
	run, _, conn := newWaveTestRun(t, nil)
	idA, idB, idC, idD := forkNPCID(2), forkNPCID(3), forkNPCID(4), forkNPCID(5)

	killAndRecord(run, idA, time.Now())
	killAndRecord(run, idB, time.Now())
	conn.resetFrames()
	run.tickOnce(time.Now())
	if !waveNPCRegistered(t, run, idC) {
		t.Fatal("wave 3 should spawn after wave 2 is cleared")
	}
	if !waveAddObjectSeen(conn, idC) {
		t.Fatal("spawned wave NPC must be materialized into the member viewport")
	}
	run.mu.Lock()
	alive, kills := len(run.waveAlive), run.dungeonKills
	run.mu.Unlock()
	if alive != 1 || kills != 2 {
		t.Fatalf("after advance alive=%d kills=%d, want (1,2)", alive, kills)
	}

	killAndRecord(run, idC, time.Now())
	run.tickOnce(time.Now())
	if !waveNPCRegistered(t, run, idD) {
		t.Fatal("wave 5 should spawn after wave 3 is cleared")
	}

	killAndRecord(run, idD, time.Now())
	run.tickOnce(time.Now())
	run.mu.Lock()
	finished := run.finished
	pending := len(run.pendingWaves)
	run.mu.Unlock()
	if !finished || pending != 0 {
		t.Fatalf("final wave cleared: finished=%v pending=%d, want (true,0)", finished, pending)
	}
}

// TestWaveLuaTakeover Lua 接管节奏：dungeon_wave_cleared 处理器自行
// ctx:spawn_wave 下一波并返回 true——默认推进被抑制、新波由脚本物化。
func TestWaveLuaTakeover(t *testing.T) {
	dir := t.TempDir()
	script := `
game.on("dungeon_wave_cleared", function(ctx)
	local d = ctx:dungeon()
	if d.next_wave > 0 then
		ctx:spawn_wave(d.next_wave)
	end
	return true
end)
`
	if err := os.WriteFile(filepath.Join(dir, "takeover.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	host := luascript.New(dir, func(string, ...any) {})
	if _, err := host.LoadDir(); err != nil {
		t.Fatal(err)
	}

	run, _, _ := newWaveTestRun(t, host)
	idA, idB, idC := forkNPCID(2), forkNPCID(3), forkNPCID(4)

	killAndRecord(run, idA, time.Now())
	killAndRecord(run, idB, time.Now())
	run.tickOnce(time.Now())
	if !waveNPCRegistered(t, run, idC) {
		t.Fatal("script-driven spawn_wave should materialize wave 3")
	}
	run.mu.Lock()
	current, pending := run.currentWave, append([]int(nil), run.pendingWaves...)
	run.mu.Unlock()
	if current != 3 || len(pending) != 1 || pending[0] != 5 {
		t.Fatalf("takeover state current=%d pending=%v, want (3,[5])", current, pending)
	}
}

// TestDungeonScriptsLoad 真机脚本目录装载冒烟：scripts/host 下的副本
// 脚本（dungeon_clone003.lua）语法/装载错误在此拦截，并冒烟
// enter/wave_cleared 两事件的 clone003 分支语义。
func TestDungeonScriptsLoad(t *testing.T) {
	host := luascript.New("../../scripts/host", func(string, ...any) {})
	if _, err := host.LoadDir(); err != nil {
		t.Fatalf("load scripts/host: %v", err)
	}
	for _, event := range []string{luaEventDungeonEnter, luaEventDungeonNPCKilled, luaEventDungeonWaveCleared, luaEventDungeonDestroy, "clone003_advance"} {
		if !host.HasHandler(event) {
			t.Fatalf("expected handler for %q", event)
		}
	}

	// enter 冒烟：clone003 分支应广播进本剧情（notify 携带玩家名）。
	ec := &luascript.EventContext{
		Player:  luascript.PlayerSnapshot{Name: "冒烟侠"},
		Dungeon: luascript.DungeonSnapshot{Config: `ini\scene\clone003`, Instance: "dn-smoke", Wave: 2},
	}
	var notified string
	ec.Notify = func(brief, _ string) error { notified = brief; return nil }
	handled, err := host.Call(luaEventDungeonEnter, ec)
	if err != nil || !handled {
		t.Fatalf("dungeon_enter = (%v, %v), want (true, nil)", handled, err)
	}
	if !strings.Contains(notified, "冒烟侠") || !strings.Contains(notified, "暮色") {
		t.Fatalf("enter notify = %q, want clone003 narrative with player name", notified)
	}

	// wave_cleared 冒烟（波 3 → 接管分支）：ctx:after(2.5s) 排程后返回 true。
	var afterCall struct {
		seconds float64
		event   string
	}
	ec = &luascript.EventContext{Dungeon: luascript.DungeonSnapshot{Config: `ini\scene\clone003`, Instance: "dn-smoke", Wave: 3, NextWave: 5}}
	ec.Notify = func(string, string) error { return nil }
	ec.After = func(seconds float64, event string) error { afterCall.seconds, afterCall.event = seconds, event; return nil }
	handled, err = host.Call(luaEventDungeonWaveCleared, ec)
	if err != nil || !handled {
		t.Fatalf("dungeon_wave_cleared = (%v, %v), want (true, nil)", handled, err)
	}
	if afterCall.seconds != 2.5 || afterCall.event != "clone003_advance" {
		t.Fatalf("after = (%v, %q), want (2.5, clone003_advance)", afterCall.seconds, afterCall.event)
	}
}

// TestWaveNpcKilledEvent npc_killed 事件携带击杀者侧快照：计数与波次
// 刻度随击杀递进（脚本侧经 ctx:dungeon()/ctx:npc() 消费）。
func TestWaveNpcKilledEvent(t *testing.T) {
	dir := t.TempDir()
	script := `
PROBE = {}
game.on("dungeon_npc_killed", function(ctx)
	local d = ctx:dungeon()
	local n = ctx:npc()
	PROBE[#PROBE + 1] = n.config_id .. "@" .. d.wave .. "+" .. d.killed
	return true
end)
`
	if err := os.WriteFile(filepath.Join(dir, "killed.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	host := luascript.New(dir, func(string, ...any) {})
	if _, err := host.LoadDir(); err != nil {
		t.Fatal(err)
	}

	run, _, _ := newWaveTestRun(t, host)
	killAndRecord(run, forkNPCID(2), time.Now())
	killAndRecord(run, forkNPCID(3), time.Now())
	out, err := host.Eval(`return table.concat(PROBE, ",")`)
	if err != nil {
		t.Fatal(err)
	}
	if want := "mon_a@2+1,mon_b@2+2"; out != want {
		t.Fatalf("npc_killed probe = %q, want %q", out, want)
	}
}
