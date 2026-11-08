package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// 九阴志全局门控状态机：语义逐条对齐 Java 蓝本 NinJiuZhi.java
//（9yin-server git 3d1df4a），Go 差异只有初始缺省阶段 6（官服
// gamestep 对齐）与落盘路径随 server root。

func TestJyzGateSetStageMonotonic(t *testing.T) {
	gate := newJyzGate(6, 2)
	reason, old, current := gate.setStage(5)
	if reason == "" || old != 6 || current != 6 {
		t.Fatalf("stage regression accepted: reason=%q old=%d new=%d", reason, old, current)
	}
	if !strings.Contains(reason, "阶段只增不减") {
		t.Fatalf("rejection reason should explain monotonic rule: %q", reason)
	}
	reason, old, current = gate.setStage(9)
	if reason != "" || old != 6 || current != 9 {
		t.Fatalf("setStage(9) rejected: reason=%q old=%d new=%d", reason, old, current)
	}
	if reason, _, _ := gate.setStage(9); reason == "" {
		t.Fatal("equal target should be rejected as non-increasing")
	}
	for _, bad := range []int{0, -1, 10, 999} {
		if reason, _, _ := gate.setStage(bad); reason == "" {
			t.Fatalf("stage %d should be out of range", bad)
		}
	}
	if gate.currentStage() != 9 {
		t.Fatalf("stage=%d want 9", gate.currentStage())
	}
}

func TestJyzGateEventThresholdsAutoAdvance(t *testing.T) {
	gate := newJyzGate(1, 2)
	for i := 0; i < 9; i++ {
		gate.addEvent("kill")
	}
	if gate.currentStage() != 1 {
		t.Fatalf("9 points should stay stage 1, got %d", gate.currentStage())
	}
	gate.addEvent("kill")
	if gate.currentStage() != 2 {
		t.Fatalf("10 points should reach stage 2, got %d", gate.currentStage())
	}
	// 连跳：默认累计阈值 30/60/…/360 一次灌满直达封顶。
	for i := 0; i < 350; i++ {
		gate.addEvent("task")
	}
	if gate.currentStage() != 9 {
		t.Fatalf("360 points should multi-jump to stage 9, got %d", gate.currentStage())
	}
	// 封顶后事件点继续累计但阶段不再涨。
	gate.addEvent("kill")
	if gate.currentStage() != 9 {
		t.Fatalf("capped stage moved: %d", gate.currentStage())
	}
}

func TestJyzGateResetAndRevisions(t *testing.T) {
	gate := newJyzGate(6, 2)
	baseRev := gate.stageRevision()
	gate.setStage(7)
	if got := gate.stageRevision(); got != baseRev+1 {
		t.Fatalf("stage revision after advance = %d, want %d", got, baseRev+1)
	}
	for i := 0; i < 3; i++ {
		gate.addEvent("kill")
	}
	// 点数变化不推 stageRevision（ticker 不补推）。
	if got := gate.stageRevision(); got != baseRev+1 {
		t.Fatalf("point-only change bumped stage revision: %d", got)
	}
	old, current := gate.reset()
	if old != 7 || current != 6 {
		t.Fatalf("reset %d -> %d, want 7 -> 6", old, current)
	}
	if got := gate.stageRevision(); got != baseRev+2 {
		t.Fatalf("stage revision after reset = %d, want %d", got, baseRev+2)
	}
	if !strings.Contains(gate.status(), "累计 0 点") {
		t.Fatalf("reset should clear points: %s", gate.status())
	}
	// initial==当前时 reset 不动阶段版本。
	gate.reset()
	if got := gate.stageRevision(); got != baseRev+2 {
		t.Fatalf("no-op reset bumped revision: %d", got)
	}
}

func TestJyzGateStatus(t *testing.T) {
	gate := newJyzGate(6, 2)
	want := "九阴志：阶段 6/9（5内后期），距下一阶还差 210 事件点；累计 0 点；门控覆写 0 条"
	if got := gate.status(); got != want {
		t.Fatalf("status=%q want %q", got, want)
	}
	gate.addEvent("kill")
	gate.addEvent("task")
	gate.addEvent("task")
	if got := gate.status(); !strings.Contains(got, "（kill:1 task:2）") {
		t.Fatalf("status should itemize events: %s", got)
	}
	// 封顶阶段无“距下一阶”句。
	top := newJyzGate(9, 2)
	if got := top.status(); strings.Contains(got, "距下一阶") || !strings.Contains(got, "阶段 9/9（6内）") {
		t.Fatalf("capped status=%q", got)
	}
}

func TestJyzGatePersistRoundtrip(t *testing.T) {
	dir := t.TempDir()
	gate := loadJyzGate(dir, jyzConfig{step: 0, eventState: 2})
	if gate.currentStage() != jyzGateDefaultStage {
		t.Fatalf("fresh load stage=%d want default %d", gate.currentStage(), jyzGateDefaultStage)
	}
	if _, err := os.Stat(filepath.Join(dir, "ninjiuzhi.json")); !os.IsNotExist(err) {
		t.Fatal("fresh load should not have written state yet")
	}
	gate.setStage(8)
	for i := 0; i < 4; i++ {
		gate.addEvent("kill")
	}
	saved, err := os.ReadFile(filepath.Join(dir, "ninjiuzhi.json"))
	if err != nil {
		t.Fatalf("state file missing after mutation: %v", err)
	}
	if !bytes.Contains(saved, []byte(`"stage": 8`)) || !bytes.Contains(saved, []byte(`"kill": 4`)) {
		t.Fatalf("state file shape unexpected: %s", saved)
	}

	reloaded := loadJyzGate(dir, jyzConfig{step: 0, eventState: 2})
	if reloaded.currentStage() != 8 {
		t.Fatalf("reloaded stage=%d want 8", reloaded.currentStage())
	}
	if !strings.Contains(reloaded.status(), "kill:4") {
		t.Fatalf("reloaded points lost: %s", reloaded.status())
	}
	if !strings.Contains(reloaded.status(), "累计 4 点") {
		t.Fatalf("reloaded totals lost: %s", reloaded.status())
	}

	// 损坏档：装载失败保持缺省并拒绝覆写（loadFailed 闸）。
	corrupt := filepath.Join(dir, "ninjiuzhi.json")
	if err := os.WriteFile(corrupt, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	broken := loadJyzGate(dir, jyzConfig{step: 0, eventState: 2})
	if broken.currentStage() != jyzGateDefaultStage {
		t.Fatalf("corrupt load stage=%d want default", broken.currentStage())
	}
	broken.setStage(9)
	after, err := os.ReadFile(corrupt)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "{not json" {
		t.Fatalf("loadFailed gate should refuse overwrite, file=%s", after)
	}
}

func TestLoadJyzGateRequirements(t *testing.T) {
	dir := t.TempDir()
	requirements := `{
		"features": {"smelting": 5},
		"tasks": {"1346": 2},
		"npcs": {"npc_locked": 3},
		"scenes": {"book8": 2},
		"inner_books": {"ng_wd_001": 4, "bad_key": "x"},
		"initial_stage": 3,
		"thresholds": {"2": 2, "5": 3}
	}`
	if err := os.WriteFile(filepath.Join(dir, "ninjiuzhi_requirements.json"), []byte(requirements), 0644); err != nil {
		t.Fatal(err)
	}
	gate := loadJyzGate(dir, jyzConfig{step: 0, eventState: 2})
	if gate.currentStage() != 3 {
		t.Fatalf("requirements initial_stage stage=%d want 3", gate.currentStage())
	}
	if !strings.Contains(gate.status(), "门控覆写 5 条") {
		t.Fatalf("override count should be 5 (bad_key ignored): %s", gate.status())
	}
	// 门控查询：未列键恒开放；列键按 required_stage 比较。
	if gate.isOpen("features", "smelting") {
		t.Fatal("smelting requires stage 5, should be closed at 3")
	}
	if !gate.isOpen("npcs", "npc_locked") {
		t.Fatal("npc_locked requires stage 3, should be open at 3")
	}
	if !gate.isOpen("scenes", "book_missing") {
		t.Fatal("unlisted key must default open")
	}
	// thresholds {"2":2,"5":3}：键 2/5 覆写、键 3/4/6… 保持默认
	// 30/60/100/…。多事件点灌入验证连跳与默认键并存：
	//   2 点 → 阶段 2（覆写键 2）；
	//   30 点 → 阶段 3（默认键 3）；
	//   59 点 → 仍 3（默认键 4=60 未到）；
	//   60 点 → 3→4→5 连跳（默认键 4=60 达标、覆写键 5=3 顺带满足），
	//   停在 5（默认键 6=100）。
	step := newJyzGate(1, 2)
	if req := readJyzGateRequirements(filepath.Join(dir, "ninjiuzhi_requirements.json")); req != nil {
		step.thresholds = req.thresholds
	} else {
		t.Fatal("requirements unreadable")
	}
	step.addEvent("kill")
	if step.currentStage() != 1 {
		t.Fatalf("1 point should stay stage 1 (threshold 2), got %d", step.currentStage())
	}
	step.addEvent("kill")
	if step.currentStage() != 2 {
		t.Fatalf("2 points should reach stage 2, got %d", step.currentStage())
	}
	for i := 0; i < 28; i++ {
		step.addEvent("kill")
	}
	if step.currentStage() != 3 {
		t.Fatalf("30 points should reach stage 3 (default key), got %d", step.currentStage())
	}
	for i := 0; i < 29; i++ {
		step.addEvent("kill")
	}
	if step.currentStage() != 3 {
		t.Fatalf("59 points should stay stage 3 (default key 4=60), got %d", step.currentStage())
	}
	step.addEvent("kill")
	if step.currentStage() != 5 {
		t.Fatalf("60 points should multi-jump 3->4->5, got %d", step.currentStage())
	}

	// env NINEYIN_JYZ_STEP（initial.step）优先于 requirements initial_stage。
	env := loadJyzGate(dir, jyzConfig{step: 7, eventState: 2})
	if env.currentStage() != 7 {
		t.Fatalf("env stage override lost: %d", env.currentStage())
	}
}

func TestLoadJyzGateThresholdsArrayForm(t *testing.T) {
	dir := t.TempDir()
	// 数组按位覆写：升到 2..9 阶分别需要 2/4/6/…/16 累计点——每 2
	// 点恰好升一阶（不触发连跳）。
	if err := os.WriteFile(filepath.Join(dir, "ninjiuzhi_requirements.json"),
		[]byte(`{"initial_stage":1,"thresholds":[0,0,2,4,6,8,10,12,14,16]}`), 0644); err != nil {
		t.Fatal(err)
	}
	gate := loadJyzGate(dir, jyzConfig{step: 0, eventState: 2})
	for stage := 1; stage < 9; stage++ {
		gate.addEvent("kill")
		gate.addEvent("kill")
		if gate.currentStage() != stage+1 {
			t.Fatalf("array-form thresholds: after %d kills stage=%d want %d", 2*(stage+1), gate.currentStage(), stage+1)
		}
	}
}

// pushJyzStageChange 三帧组：419 单 int → 414 'gamestep' 收尾 →
// worldInfo(0) 推进提示。data_record 事件行不随阶段重发。
func TestPushJyzStageChangeFrames(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := pushJyzStageChange(conn, 8); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("frames=%d want 3", len(frames))
	}
	legacy := []byte{0x1e, 0x02, 0x00, 0x02, 0xa3, 0x01, 0x00, 0x00, 0x02, 0x08, 0x00, 0x00, 0x00}
	if !bytes.Equal(frames[0], legacy) {
		t.Fatalf("419 frame=% X want % X", frames[0], legacy)
	}
	gamestep := frames[1]
	if gamestep[0] != 0x1e || !bytes.Contains(gamestep, []byte("gamestep\x00")) {
		t.Fatalf("414 gamestep frame malformed: % X", gamestep)
	}
	if !bytes.Contains(gamestep, []byte{0x02, 0x08, 0x00, 0x00, 0x00}) {
		t.Fatalf("414 gamestep step value missing: % X", gamestep)
	}
	notice := frames[2]
	if notice[0] != 0x05 || binary.LittleEndian.Uint16(notice[1:]) != 0 {
		t.Fatalf("worldInfo header malformed: % X", notice[:3])
	}
	units := utf16.Decode(bytesToUint16LE(notice[3:]))
	text := strings.TrimRight(string(units), "\x00")
	want := "九阴志推进：阶段 8（6内中期），江湖暗流涌动"
	if text != want {
		t.Fatalf("notice=%q want %q", text, want)
	}
}

func bytesToUint16LE(raw []byte) []uint16 {
	units := make([]uint16, len(raw)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(raw[i*2:])
	}
	return units
}

// config() 把全服阶段拼进既有帧配置；eventState 进程级不变。
func TestJyzGateConfigMergesStage(t *testing.T) {
	gate := newJyzGate(4, 44)
	cfg := gate.config()
	if cfg.step != 4 || cfg.eventState != 44 {
		t.Fatalf("config=%+v", cfg)
	}
	gate.setStage(5)
	if cfg = gate.config(); cfg.step != 5 || cfg.eventState != 44 {
		t.Fatalf("config after advance=%+v", cfg)
	}
}
