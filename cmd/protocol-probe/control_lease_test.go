package main

// 受控机制回归（规格书《击飞击退击倒机制规格书》§十二）：
// 帧型逐字节 / desc 映射与条件门 / 解析链 overlay>on_hit>desc /
// 击倒 State 对 / 击飞簇与 juggle / 击退不换态不冻结 / 玩家封招闸。

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/skillscript"
)

// decodeCustomIntMessage 解 0x1E 数值壳的值序列（含首个 message id）。
// 返回扁平的 (tag, payload) 序列，供布局断言。
type wireValue struct {
	tag    byte
	int32  int32
	f64    float64
	text   string
	obj    [2]uint32
}

func decodeCustomIntMessage(t *testing.T, frame []byte) []wireValue {
	t.Helper()
	if frame[0] != 0x1E {
		t.Fatalf("not 0x1E shell: %x", frame[:3])
	}
	count := int(binary.LittleEndian.Uint16(frame[1:]))
	out := make([]wireValue, 0, count)
	off := 3
	for i := 0; i < count; i++ {
		if off >= len(frame) {
			t.Fatalf("frame truncated at value %d: %x", i, frame)
		}
		tag := frame[off]
		off++
		value := wireValue{tag: tag}
		switch tag {
		case 2: // int32
			value.int32 = int32(binary.LittleEndian.Uint32(frame[off:]))
			off += 4
		case 3: // int64
			off += 8
		case 4: // float32
			off += 4
		case 5: // float64
			value.f64 = math.Float64frombits(binary.LittleEndian.Uint64(frame[off:]))
			off += 8
		case 6: // string
			n := int(binary.LittleEndian.Uint32(frame[off:]))
			off += 4
			value.text = string(frame[off : off+n-1])
			off += n
		case 7: // wide string
			n := int(binary.LittleEndian.Uint32(frame[off:]))
			off += 4 + n
		case 8: // object
			value.obj = [2]uint32{
				binary.LittleEndian.Uint32(frame[off:]),
				binary.LittleEndian.Uint32(frame[off+4:]),
			}
			off += 8
		default:
			t.Fatalf("unknown tag %d at %d", tag, off-1)
		}
		out = append(out, value)
	}
	return out
}

// countControlFrames 统计捕获帧里 msg24（[24, obj, c1, …] 布局）的 c1 子码帧数。
func countControlFrames(t *testing.T, frames [][]byte, c1 int32) int {
	t.Helper()
	count := 0
	for _, frame := range frames {
		if len(frame) < 22 || frame[0] != 0x1E {
			continue
		}
		values := decodeCustomIntMessage(t, frame)
		if len(values) >= 3 && values[0].tag == 2 && values[0].int32 == 24 &&
			values[2].tag == 2 && values[2].int32 == c1 {
			count++
		}
	}
	return count
}

// countControlStateFrames 统计 0x10 属性帧里 State(183)=指定整型值的推送
// （int 按 WireInt32 落位=官方 float 位型方言）。
func countControlStateFrames(frames [][]byte, state int32) int {
	count := 0
	for _, frame := range frames {
		if len(frame) < 18 || frame[0] != 0x10 {
			continue
		}
		props := int(binary.LittleEndian.Uint16(frame[10:]))
		off := 12
		for i := 0; i < props && off+6 <= len(frame); i++ {
			idx := int(binary.LittleEndian.Uint16(frame[off:]))
			if idx == int(controlStatePropertyIndex) &&
				int32(binary.LittleEndian.Uint32(frame[off+2:])) == state {
				count++
			}
			off += 6 // State 整型态恒 4B 载荷
		}
	}
	return count
}

// TestControlFrameBuildersMatchOfficialLayouts 帧型逐字节（规格书 §三）：
// c56/c79/c78/State 整型/m18 动画变体。
func TestControlFrameBuildersMatchOfficialLayouts(t *testing.T) {
	marker, err := controlMarkerFrame(0x11, 7)
	if err != nil {
		t.Fatal(err)
	}
	values := decodeCustomIntMessage(t, marker)
	if len(values) != 3 || values[0].int32 != 24 || values[1].obj != [2]uint32{0x11, 7} || values[2].int32 != 56 {
		t.Fatalf("c56 values=%+v frame=%x", values, marker)
	}

	launch, err := launchControlFrame(0x11, 7, 0x22, 1, "hurt_f_1_l")
	if err != nil {
		t.Fatal(err)
	}
	values = decodeCustomIntMessage(t, launch)
	if len(values) != 5 || values[0].int32 != 24 || values[1].obj != [2]uint32{0x11, 7} ||
		values[2].int32 != 79 || values[3].obj != [2]uint32{0x22, 1} {
		t.Fatalf("c79 head values=%+v", values[:4])
	}
	if values[4].tag != 6 || values[4].text != "hurt_f_1_l" {
		t.Fatalf("c79 anim = %+v", values[4])
	}

	knock, err := knockbackControlFrame(0x11, 7, 0x22, 1, 109)
	if err != nil {
		t.Fatal(err)
	}
	values = decodeCustomIntMessage(t, knock)
	if len(values) != 8 || values[0].int32 != 24 || values[1].obj != [2]uint32{0x11, 7} ||
		values[2].int32 != 78 || values[3].obj != [2]uint32{0x22, 1} ||
		values[4].text != "" || values[5].text != "" {
		t.Fatalf("c78 values=%+v", values)
	}
	if !findInt(t, values, 109) {
		t.Fatalf("c78 angle missing: %+v", values)
	}
	if !anyF64(values, knockbackSpeedMarker) {
		t.Fatalf("c78 speed marker missing: %+v", values)
	}

	state, err := controlStateFrame(0x33, 1, controlStateDown)
	if err != nil {
		t.Fatal(err)
	}
	if state[0] != 0x10 || countControlStateFrames([][]byte{state}, controlStateDown) != 1 {
		t.Fatalf("state 12 frame=%x", state)
	}
	// 整型 12 按 float32 位型落位（群战 1.68156e-44 方言）。
	if bits := binary.LittleEndian.Uint32(state[14:18]); bits != 12 {
		t.Fatalf("state 12 raw bits=%#x, want 12", bits)
	}

	hurt, err := skillHurtActionFrameAnim(0x11, 7, 0x22, 1, "hurt_f_1_s")
	if err != nil {
		t.Fatal(err)
	}
	values = decodeCustomIntMessage(t, hurt)
	if len(values) != 5 || values[0].int32 != 18 || values[1].obj != [2]uint32{0x11, 7} ||
		values[2].text != "hurt_f_1_s" || values[3].int32 != 0 || values[4].obj != [2]uint32{0x22, 1} {
		t.Fatalf("m18 values=%+v", values)
	}
	fallback, _ := skillHurtActionFrameAnim(0x11, 7, 0x22, 1, "")
	values = decodeCustomIntMessage(t, fallback)
	if values[2].text != "hurt_half" {
		t.Fatalf("m18 default anim=%+v", values[2])
	}
}

func findInt(t *testing.T, values []wireValue, want int32) bool {
	t.Helper()
	for _, value := range values {
		if value.tag == 2 && value.int32 == want {
			return true
		}
	}
	return false
}

func anyF64(values []wireValue, want float64) bool {
	for _, value := range values {
		if value.tag == 5 && value.f64 == want {
			return true
		}
	}
	return false
}

// TestControlSpecFromDescRule desc 层映射表（§五②）。
func TestControlSpecFromDescRule(t *testing.T) {
	if spec := controlSpecFromDescRule(&skillscript.DescRule{ControlKind: "launch", ControlDurMS: 1500, Condition: &skillscript.DescCondition{Unparried: true}, Chance: 0.3}); spec == nil ||
		spec.Kind != "launch" || spec.DurMS != 1500 || spec.Condition != "unparried" || spec.Chance != 0.3 {
		t.Fatalf("launch rule = %+v", spec)
	}
	if spec := controlSpecFromDescRule(&skillscript.DescRule{DisplacementKind: "push", DisplacementDistanceM: 2.5}); spec == nil || spec.Kind != "push" || spec.DistM != 2.5 {
		t.Fatalf("push rule = %+v", spec)
	}
	for _, kind := range []string{"slow", "root", "silence", "ban_qinggong", "stiffen"} {
		if spec := controlSpecFromDescRule(&skillscript.DescRule{ControlKind: kind}); spec != nil {
			t.Fatalf("%s rule should be nil, got %+v", kind, spec)
		}
	}
	if spec := controlSpecFromDescRule(&skillscript.DescRule{DisplacementKind: "pull"}); spec != nil {
		t.Fatalf("pull is phase 2, got %+v", spec)
	}
	if controlSpecFromDescRule(nil) != nil {
		t.Fatal("nil rule must map to nil")
	}
}

// TestControlAllowedGates 条件门（§4.5）：nil/无 kind 恒拒；parry_broken
// 对 NPC 恒拒；几率掷确定性（同 OID 同秒同结果）。
func TestControlAllowedGates(t *testing.T) {
	now := time.Unix(2000000, 0)
	if controlAllowed(nil, 81, now) {
		t.Fatal("nil spec must be rejected")
	}
	if controlAllowed(&skillscript.ControlSpec{}, 81, now) {
		t.Fatal("empty kind must be rejected")
	}
	if !controlAllowed(&skillscript.ControlSpec{Kind: "knockdown"}, 81, now) {
		t.Fatal("plain spec must pass")
	}
	spec := &skillscript.ControlSpec{Kind: "knockdown", Condition: "parry_broken"}
	if controlAllowed(spec, 81, now) {
		t.Fatal("parry_broken on NPC must fail")
	}
	chanced := &skillscript.ControlSpec{Kind: "knockdown", Chance: 0.5}
	if controlAllowed(chanced, 81, now) != controlAllowed(chanced, 81, now) {
		t.Fatal("chance roll is not deterministic")
	}
}

// newControlTestHarness 构造带战斗 NPC 的生命周期与捕获连接。
func newControlTestHarness(t *testing.T) (*sceneLifecycle, *captureMessageConnection) {
	t.Helper()
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 81)
	return lifecycle, conn
}

// TestApplyControlSpecKnockdownStatePair 击倒：c56 + State 6/12 同簇、
// AI 冻结、租约到推 State=1（§4.2/§六）。
func TestApplyControlSpecKnockdownStatePair(t *testing.T) {
	lifecycle, conn := newControlTestHarness(t)
	now := time.Now()
	spec := &skillscript.ControlSpec{Kind: "knockdown", DurMS: 80}
	if !applyControlSpecFrames(conn, lifecycle, spec, nil, 81, 1, now, "test_skill") {
		t.Fatal("knockdown not applied")
	}
	frames := conn.Frames()
	if got := countControlFrames(t, frames, 56); got != 1 {
		t.Fatalf("c56 count=%d, want 1", got)
	}
	if got := countControlStateFrames(frames, controlStateAir); got != 1 {
		t.Fatalf("State 6 count=%d, want 1", got)
	}
	if got := countControlStateFrames(frames, controlStateDown); got != 1 {
		t.Fatalf("State 12 count=%d, want 1", got)
	}
	// AI 冻结：Brain.ControlledUntil 抬到租约终点。
	lifecycle.run.mu.Lock()
	lease := lifecycle.run.npcControl[81]
	brain, hasBrain := lifecycle.run.combat.Brain(81)
	lifecycle.run.mu.Unlock()
	if lease == nil || lease.kind != "knockdown" {
		t.Fatalf("lease = %+v", lease)
	}
	if hasBrain && !brain.ControlledUntil.After(now) {
		t.Fatalf("brain not frozen: %v", brain.ControlledUntil)
	}
	// 恢复定时：租约 80ms 后推 State=1。
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if countControlStateFrames(conn.Frames(), controlStateNormal) >= 1 {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatal("recovery State=1 frame never arrived")
}

// TestApplyControlSpecLaunchClusterAndJuggle 击飞：m18 hurt_f_1_s +
// c56×2 + c79(hurt_f_1_l) + State=13；空中追击=裸 c79 不刷租约（§4.1）。
func TestApplyControlSpecLaunchClusterAndJuggle(t *testing.T) {
	lifecycle, conn := newControlTestHarness(t)
	now := time.Now()
	spec := &skillscript.ControlSpec{Kind: "launch", DurMS: 600, AirMS: 250}
	if !applyControlSpecFrames(conn, lifecycle, spec, nil, 81, 1, now, "test_skill") {
		t.Fatal("launch not applied")
	}
	frames := conn.Frames()
	if got := countControlFrames(t, frames, 79); got != 1 {
		t.Fatalf("c79 count=%d, want 1", got)
	}
	if got := countControlFrames(t, frames, 56); got != 2 {
		t.Fatalf("c56 count=%d, want 2", got)
	}
	if got := countControlStateFrames(frames, controlStateFloat); got != 1 {
		t.Fatalf("State 13 count=%d, want 1", got)
	}
	var sawHurtF bool
	for _, frame := range frames {
		if frame[0] != 0x1E {
			continue
		}
		values := decodeCustomIntMessage(t, frame)
		for _, value := range values {
			if value.text == launchHurtAction || value.text == launchFloatAction {
				sawHurtF = true
			}
		}
	}
	if !sawHurtF {
		t.Fatal("hurt_f anims missing from launch cluster")
	}

	// juggle：租约内的第二次 launch → 仅裸 c79、租约不刷。
	lifecycle.run.mu.Lock()
	firstUntil := lifecycle.run.npcControl[81].until
	lifecycle.run.mu.Unlock()
	conn2 := &captureMessageConnection{}
	if !applyControlSpecFrames(conn2, lifecycle, &skillscript.ControlSpec{Kind: "launch", DurMS: 900}, nil, 81, 1, now.Add(50*time.Millisecond), "test_skill") {
		t.Fatal("juggle not applied")
	}
	lifecycle.run.mu.Lock()
	keptUntil := lifecycle.run.npcControl[81].until
	lifecycle.run.mu.Unlock()
	if !keptUntil.Equal(firstUntil) {
		t.Fatalf("juggle refreshed lease: %v -> %v", firstUntil, keptUntil)
	}
	if got := countControlFrames(t, conn2.Frames(), 79); got != 1 {
		t.Fatalf("juggle c79 count=%d, want 1", got)
	}
	if got := countControlStateFrames(conn2.Frames(), controlStateFloat); got != 0 {
		t.Fatalf("juggle must not re-push State 13, got %d", got)
	}
}

// TestApplyControlSpecPushNoStateNoFreeze 击退：c78 落帧、State 不变、
// AI 不冻结（§五 push 不封）。
func TestApplyControlSpecPushNoStateNoFreeze(t *testing.T) {
	lifecycle, conn := newControlTestHarness(t)
	now := time.Now()
	spec := &skillscript.ControlSpec{Kind: "push", DurMS: 800}
	if !applyControlSpecFrames(conn, lifecycle, spec, nil, 81, 1, now, "test_skill") {
		t.Fatal("push not applied")
	}
	frames := conn.Frames()
	if got := countControlFrames(t, frames, 78); got != 1 {
		t.Fatalf("c78 count=%d, want 1", got)
	}
	for _, state := range []int32{controlStateAir, controlStateDown, controlStateFloat} {
		if got := countControlStateFrames(frames, state); got != 0 {
			t.Fatalf("push pushed State %d, want none", state)
		}
	}
	lifecycle.run.mu.Lock()
	brain, hasBrain := lifecycle.run.combat.Brain(81)
	lifecycle.run.mu.Unlock()
	if hasBrain && brain.ControlledUntil.After(now) {
		t.Fatal("push must not freeze NPC brain")
	}
}

// TestControlCastBlockedGate 玩家封招闸：knockdown 封、push 不封、
// 免控/解控路径（§六/§7.2）。
func TestControlCastBlockedGate(t *testing.T) {
	player := newPlayerActor("tester", 0)
	now := time.Now()
	if blocked := player.controlCastBlocked(now); blocked != "" {
		t.Fatalf("fresh player blocked by %q", blocked)
	}
	player.mu.Lock()
	player.controlLease = playerControlLease{kind: "knockdown", until: now.Add(2 * time.Second)}
	player.mu.Unlock()
	if blocked := player.controlCastBlocked(now); blocked != "knockdown" {
		t.Fatalf("knockdown lease blocked=%q", blocked)
	}
	// 租约过期即放行。
	if blocked := player.controlCastBlocked(now.Add(3 * time.Second)); blocked != "" {
		t.Fatalf("expired lease blocked=%q", blocked)
	}
	// push 不封招。
	player.mu.Lock()
	player.controlLease = playerControlLease{kind: "push", until: now.Add(2 * time.Second)}
	player.mu.Unlock()
	if blocked := player.controlCastBlocked(now); blocked != "" {
		t.Fatalf("push lease blocked=%q", blocked)
	}
	// 解控清租约 + 免控窗。
	player.mu.Lock()
	player.controlLease = playerControlLease{kind: "knockdown", until: now.Add(2 * time.Second)}
	if !player.breakPlayerControlLocked(now) {
		t.Fatal("break_control should clear active lease")
	}
	if player.breakPlayerControlLocked(now) {
		t.Fatal("break_control on empty lease should report false")
	}
	player.applyPlayerControlImmune(3000, now)
	if !player.controlImmunedLocked(now.Add(time.Second)) {
		t.Fatal("control_immune window should be active")
	}
	if player.controlImmunedLocked(now.Add(4 * time.Second)) {
		t.Fatal("control_immune window should expire")
	}
	// Ctx 快照真值：封招族 controlled=true。
	player.controlLease = playerControlLease{kind: "coma", until: now.Add(1500 * time.Millisecond)}
	kind, remain, controlled := player.controlSnapshotLocked(now)
	player.mu.Unlock()
	if kind != "coma" || !controlled || remain <= 0 || remain > 1500 {
		t.Fatalf("control snapshot = (%q, %d, %t)", kind, remain, controlled)
	}
}

// TestControlSpecResolutionChain 解析链 overlay > on_hit > desc（§五）。
func TestControlSpecResolutionChain(t *testing.T) {
	dir := t.TempDir()
	overlaySrc := `local M = {}
function M.compile(def)
  return { control = { kind = "knockdown", dur_ms = 2600, condition = "unparried" } }
end
function M.on_hit(ctx)
  return { control = { kind = "stun", dur_ms = 1700 } }
end
return M
`
	if err := os.WriteFile(filepath.Join(dir, "cs_probe.lua"), []byte(overlaySrc), 0644); err != nil {
		t.Fatal(err)
	}
	descSrc := `local E = {}
E["cs_desc_only"] = {
  on_hit = {
    { displacement_kind = "push", displacement_distance_m = 3 },
  },
}
return E
`
	if err := os.WriteFile(filepath.Join(dir, "desc_effects.lua"), []byte(descSrc), 0644); err != nil {
		t.Fatal(err)
	}
	host := skillscript.New(dir, nil)
	if _, err := host.Load(); err != nil {
		t.Fatal(err)
	}
	previous := skillScripts
	skillScripts = host
	t.Cleanup(func() { skillScripts = previous })

	// ① compile 层（编译期注入 definition.control，resolve 期直读不回落）。
	compiled := resolveControlCompileOverride("cs_probe")
	if compiled == nil || compiled.Kind != "knockdown" || compiled.DurMS != 2600 || compiled.Condition != "unparried" {
		t.Fatalf("compile layer = %+v", compiled)
	}
	// definition.control 已存 → 恒胜（on_hit 同模块存在也不抢）。
	if spec := resolveControlSpec(combatSkillDefinition{id: "cs_probe", control: compiled}, nil, 81, time.Unix(1000, 0)); spec != compiled {
		t.Fatalf("definition.control should short-circuit, got %+v", spec)
	}
	// ② on_hit 运行期键（definition.control 为 nil 时生效）。
	if spec := resolveControlSpec(combatSkillDefinition{id: "cs_probe"}, nil, 81, time.Unix(1000, 0)); spec == nil ||
		spec.Kind != "stun" || spec.DurMS != 1700 {
		t.Fatalf("on_hit layer = %+v", spec)
	}
	// ③ desc 层 push 映射（无 overlay/on_hit 模块）。
	descOnly := resolveControlSpec(combatSkillDefinition{id: "cs_desc_only"}, nil, 82, time.Unix(1000, 0))
	if descOnly == nil || descOnly.Kind != "push" || descOnly.DistM != 3 {
		t.Fatalf("desc layer push = %+v", descOnly)
	}
	// 全 miss → nil。
	if got := resolveControlSpec(combatSkillDefinition{id: "cs_missing"}, nil, 83, time.Unix(1000, 0)); got != nil {
		t.Fatalf("missing skill should resolve nil, got %+v", got)
	}
	// desc 闸：NINEYIN_CONTROL_DESC=off 时 desc 层静默（on_hit 不受影响）。
	t.Setenv("NINEYIN_CONTROL_DESC", "off")
	if got := resolveControlSpec(combatSkillDefinition{id: "cs_desc_only"}, nil, 82, time.Unix(1000, 0)); got != nil {
		t.Fatalf("desc gate off should silence desc layer, got %+v", got)
	}
}

// TestControlDisabledByEnv 环境总闸（§八）：NINEYIN_CONTROL=off 全静默。
func TestControlDisabledByEnv(t *testing.T) {
	t.Setenv("NINEYIN_CONTROL", "off")
	lifecycle, conn := newControlTestHarness(t)
	if applyControlFrames(conn, lifecycle, combatSkillDefinition{id: "x"}, nil, 81, 1, time.Now()) {
		t.Fatal("master switch off must not apply control")
	}
	if len(conn.Frames()) != 0 {
		t.Fatalf("master switch off wrote %d frames", len(conn.Frames()))
	}
}
